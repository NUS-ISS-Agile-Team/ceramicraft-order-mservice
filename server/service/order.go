package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/NUS-ISS-Agile-Team/ceramicraft-commodity-mservice/common/productpb"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/clients"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/log"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/pkg/consts"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/pkg/types"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/pkg/utils"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/repository/dao"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/repository/model"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-payment-mservice/common/paymentpb"
)

type OrderService interface {
	CreateOrder(ctx context.Context, orderInfo types.OrderInfo) (orderNo string, err error)
	ListOrders(ctx context.Context, req types.ListOrderRequest) (resp *types.ListOrderResponse, err error)
	GetOrderDetail(ctx context.Context, orderNo string) (detail *types.OrderDetail, err error)
	CustomerGetOrderDetail(ctx context.Context, orderNo string, userID int) (detail *types.OrderDetail, err error)
}

type OrderServiceImpl struct {
	lock                 sync.Mutex
	orderDao             dao.OrderDao
	orderProductDao      dao.OrderProductDao
	orderLogDao          dao.OrderLogDao
	productServiceClient productpb.ProductServiceClient
	paymentServiceClient paymentpb.PaymentServiceClient
	messageWriter        utils.Writer
	syncMode             bool
}

func GetOrderServiceInstance() *OrderServiceImpl {
	return &OrderServiceImpl{
		orderDao:             dao.GetOrderDao(),
		orderProductDao:      dao.GetOrderProductDao(),
		orderLogDao:          dao.GetOrderLogDao(),
		productServiceClient: clients.GetProductClient(),
		paymentServiceClient: clients.GetPaymentClient(),
		messageWriter:        utils.GetWriter(),
		syncMode:             false,
	}
}

func (o *OrderServiceImpl) CreateOrder(ctx context.Context, orderInfo types.OrderInfo) (orderNo string, err error) {
	userId := ctx.Value("userID").(int)
	o.lock.Lock()
	defer o.lock.Unlock()

	orderItemIds := make([]int64, len(orderInfo.OrderItemList))
	for idx, item := range orderInfo.OrderItemList {
		orderItemIds[idx] = int64(item.ProductID)
	}

	// log.Logger.Infof("CreateOrder: orderItemIds = %v", orderItemIds)

	// 1. rpc: call product service and check if all the related product's stock is enough
	productList, err := o.productServiceClient.GetProductList(ctx, &productpb.GetProductListRequest{
		Ids: orderItemIds,
	})
	if err != nil {
		log.Logger.Errorf("CreateOrder: get product list failed, err: %s", err.Error())
		return "", err
	}

	productId2StockMap := make(map[int]int)
	for _, product := range productList.Products {
		productId2StockMap[int(product.Id)] = int(product.Stock)
	}

	itemTotalAmount := 0
	for _, orderItem := range orderInfo.OrderItemList {
		if orderItem.Quantity > productId2StockMap[orderItem.ProductID] {
			err = fmt.Errorf("CreateOrder failed, do not have enough stock, product id: %d", orderItem.ProductID)
			log.Logger.Errorf(err.Error())
			return "", err
		}
		itemTotalAmount += (orderItem.Price * orderItem.Quantity)
	}

	shippingFee := CalculateShippingFee(itemTotalAmount)
	tax := CalculateTax(itemTotalAmount)

	// 2. local func: gen order ID
	orderId := utils.GenerateOrderID()

	// 3. save order Info to database
	// 3.1 save order Info
	_, err = o.orderDao.Create(ctx, &model.Order{
		OrderNo:           orderId,
		UserID:            userId,
		Status:            consts.CREATED,
		TotalAmount:       itemTotalAmount + shippingFee + tax,
		CreateTime:        time.Now(),
		UpdateTime:        time.Now(),
		ReceiverFirstName: orderInfo.ReceiverFirstName,
		ReceiverLastName:  orderInfo.ReceiverLastName,
		ReceiverPhone:     orderInfo.ReceiverPhone,
		ReceiverAddress:   orderInfo.ReceiverAddress,
		ReceiverCountry:   orderInfo.ReceiverCountry,
		ReceiverZipCode:   orderInfo.ReceiverZipCode,
		Remark:            orderInfo.Remark,
		ShippingFee:       shippingFee,
		Tax:               tax,
	})
	if err != nil {
		log.Logger.Errorf("CreateOrder: insert into db failed, err: %s", err.Error())
		return "", err
	}

	// 3.2 save order items
	for _, orderItem := range orderInfo.OrderItemList {
		_, err := o.orderProductDao.Create(ctx, &model.OrderProduct{
			OrderNo:     orderId,
			ProductID:   orderItem.ProductID,
			ProductName: orderItem.ProductName,
			Price:       orderItem.Price,
			Quantity:    orderItem.Quantity,
			TotalPrice:  (orderItem.Price * orderItem.Quantity),
			CreateTime:  time.Now(),
			UpdateTime:  time.Now(),
		})
		if err != nil {
			log.Logger.Errorf("CreateOrder: create order item failed, err: %s", err.Error())
			return "", err
		}
	}

	orderMsg, err := getOrderMsg(orderId, orderInfo, userId)
	if err != nil {
		log.Logger.Errorf("getOrderMsg: json encode failed, err %s", err.Error())
		return "", err
	}
	// 4. message queue: send msg -- order ID
	err = o.messageWriter.SendMsg(ctx, "order_created", orderId, orderMsg)
	if err != nil {
		log.Logger.Errorf("CreateOrder: send message failed, err %s", err.Error())
		return "", err
	}

	if o.syncMode {
		oscMsg, err := getOrderStatusChangedMsg(orderId, userId, "Created", 1)
		if err != nil {
			log.Logger.Errorf("get order status changed msg failed, err %s", err.Error())
		}
		err = o.messageWriter.SendMsg(ctx, "order_status_changed", orderId, oscMsg)
		if err != nil {
			log.Logger.Errorf("send message failed, err %s", err)
		}
	} else {
		go func() {
			oscMsg, err := getOrderStatusChangedMsg(orderId, userId, "Created", 1)
			if err != nil {
				log.Logger.Errorf("get order status changed msg failed, err %s", err.Error())
			}
			err = o.messageWriter.SendMsg(ctx, "order_status_changed", orderId, oscMsg)
			if err != nil {
				log.Logger.Errorf("send message failed, err %s", err)
			}
		}()
	}

	// 5. rpc: call product service and decrease stock
	for _, orderItem := range orderInfo.OrderItemList {
		_, _ = o.productServiceClient.UpdateStockWithCAS(ctx, &productpb.UpdateStockWithCASRequest{
			Id:   int64(orderItem.ProductID),
			Deta: int64(-1 * orderItem.Quantity),
		})
	}

	// 6. rpc: call payment service and pay
	// TODO
	payResp, err := o.paymentServiceClient.PayOrder(ctx, &paymentpb.PayOrderRequest{
		UserId: int32(userId),
		Amount: int32(itemTotalAmount + shippingFee + tax),
		BizId:  orderId,
	})

	// 6.2 payment failed
	if err != nil || payResp.Code != 0 {
		_ = o.messageWriter.SendMsg(ctx, "order_canceled", orderId, orderMsg)
		if err != nil {
			log.Logger.Errorf("CreateOrder: payment failed, err: %s", err.Error())
			return "", err
		} else {
			errMsg := payResp.ErrorMsg
			rpcErr := errors.New(*errMsg)
			log.Logger.Errorf("CreateOrder: payment failed, err: %s", rpcErr.Error())
			return "", rpcErr
		}
	}

	// 6.1 payment success: update order status
	err = o.orderDao.UpdateStatusAndPayment(ctx, orderId, consts.PAYED, time.Now())
	if err != nil {
		log.Logger.Errorf("CreateOrder: update status failed, err %s", err.Error())
		return "", err
	}

	if o.syncMode {
		oscMsg, err := getOrderStatusChangedMsg(orderId, userId, "Created --> Paid", 2)
		if err != nil {
			log.Logger.Errorf("get order status changed msg failed, err %s", err.Error())
		}
		err = o.messageWriter.SendMsg(ctx, "order_status_changed", orderId, oscMsg)
		if err != nil {
			log.Logger.Errorf("send message failed, err %s", err)
		}
	} else {
		go func() {
			oscMsg, err := getOrderStatusChangedMsg(orderId, userId, "Created --> Paid", 2)
			if err != nil {
				log.Logger.Errorf("get order status changed msg failed, err %s", err.Error())
			}
			err = o.messageWriter.SendMsg(ctx, "order_status_changed", orderId, oscMsg)
			if err != nil {
				log.Logger.Errorf("send message failed, err %s", err)
			}
		}()
	}

	return orderId, nil
}

func getOrderMsg(orderId string, orderInfo types.OrderInfo, userId int) (msg string, err error) {
	orderMessage := types.OrderMessage{
		UserID:            userId,
		OrderID:           orderId,
		ReceiverFirstName: orderInfo.ReceiverFirstName,
		ReceiverLastName:  orderInfo.ReceiverLastName,
		ReceiverPhone:     orderInfo.ReceiverPhone,
		ReceiverAddress:   orderInfo.ReceiverAddress,
		ReceiverCountry:   orderInfo.ReceiverCountry,
		ReceiverZipCode:   orderInfo.ReceiverZipCode,
		Remark:            orderInfo.Remark,
		OrderItemList:     orderInfo.OrderItemList,
	}
	orderMsgJson, err := utils.JSONEncode(orderMessage)
	return orderMsgJson, err
}

func getOrderStatusChangedMsg(orderNo string, userId int, remark string, curStatus int) (msg string, err error) {
	rawMsg := types.OrderStatusChangedMessage{
		OrderNo:       orderNo,
		UserId:        userId,
		Remark:        remark,
		CurrentStatus: curStatus,
	}
	return utils.JSONEncode(rawMsg)
}

func CalculateShippingFee(total int) int {
	const ShippingFee = 800
	const TotalThresh = 30000
	if total >= TotalThresh {
		return 0
	}
	return ShippingFee
}

// tax = total * 9%
func CalculateTax(total int) int {
	return total * 9 / 100
}

func (o *OrderServiceImpl) ListOrders(ctx context.Context, req types.ListOrderRequest) (resp *types.ListOrderResponse, err error) {
	// 构建查询条件
	query := dao.OrderQuery{
		UserID:      req.UserID,
		OrderStatus: req.OrderStatus,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		OrderNo:     req.OrderNo,
		Limit:       req.Limit,
		Offset:      req.Offset,
	}

	// 调用 DAO 层查询订单列表
	orders, err := o.orderDao.GetByOrderQuery(ctx, query)
	if err != nil {
		log.Logger.Errorf("ListOrders: query orders failed, err: %s", err.Error())
		return nil, err
	}

	// 转换为响应格式
	orderList := make([]*types.OrderInfoInList, len(orders))
	for idx, order := range orders {
		orderInfo := &types.OrderInfoInList{
			OrderNo:           order.OrderNo,
			ReceiverFirstName: order.ReceiverFirstName,
			ReceiverLastName:  order.ReceiverLastName,
			ReceiverPhone:     order.ReceiverPhone,
			CreateTime:        order.CreateTime,
			TotalAmount:       int(order.TotalAmount),
			Status:            getOrderStatusName(order.Status),
		}
		orderList[idx] = orderInfo
	}

	resp = &types.ListOrderResponse{
		Orders: orderList,
		Total:  len(orderList),
	}

	return resp, nil
}

// GetOrderDetail 根据订单号查询订单详情
func (o *OrderServiceImpl) GetOrderDetail(ctx context.Context, orderNo string) (detail *types.OrderDetail, err error) {
	// 1. 查询订单基本信息
	order, err := o.orderDao.GetByOrderNo(ctx, orderNo)
	if err != nil {
		log.Logger.Errorf("GetOrderDetail: get order failed, orderNo: %s, err: %s", orderNo, err.Error())
		return nil, err
	}

	// 2. 查询订单商品列表
	orderProducts, err := o.orderProductDao.GetByOrderNo(ctx, orderNo)
	if err != nil {
		log.Logger.Errorf("GetOrderDetail: get order products failed, orderNo: %s, err: %s", orderNo, err.Error())
		return nil, err
	}

	// 3. 查询订单状态日志
	orderLogs, err := o.orderLogDao.GetByOrderNo(ctx, orderNo)
	if err != nil {
		log.Logger.Errorf("GetOrderDetail: get order logs failed, orderNo: %s, err: %s", orderNo, err.Error())
		return nil, err
	}

	// 4. 转换订单商品信息
	orderItems := make([]*types.OrderItemDetail, 0, len(orderProducts))
	for _, product := range orderProducts {
		orderItem := &types.OrderItemDetail{
			ID:          product.ID,
			ProductID:   product.ProductID,
			ProductName: product.ProductName,
			Price:       product.Price,
			Quantity:    product.Quantity,
			TotalPrice:  product.TotalPrice,
			CreateTime:  product.CreateTime,
			UpdateTime:  product.UpdateTime,
		}
		orderItems = append(orderItems, orderItem)
	}

	// 5. 转换订单状态日志
	statusLogs := make([]*types.OrderStatusLogDetail, 0, len(orderLogs))
	for _, log := range orderLogs {
		statusLog := &types.OrderStatusLogDetail{
			ID:            log.ID,
			CurrentStatus: log.CurrentStatus,
			StatusName:    getOrderStatusName(log.CurrentStatus),
			Remark:        log.Remark,
			CreateTime:    log.CreateTime,
		}
		statusLogs = append(statusLogs, statusLog)
	}

	// 6. 构建订单详情响应
	detail = &types.OrderDetail{
		// 基本订单信息
		OrderNo:      order.OrderNo,
		UserID:       order.UserID,
		Status:       order.Status,
		StatusName:   getOrderStatusName(order.Status),
		TotalAmount:  int(order.TotalAmount),
		PayAmount:    int(order.PayAmount),
		ShippingFee:  int(order.ShippingFee),
		Tax:          int(order.Tax),
		PayTime:      order.PayTime,
		CreateTime:   order.CreateTime,
		UpdateTime:   order.UpdateTime,
		DeliveryTime: order.DeliveryTime,
		ConfirmTime:  order.ConfirmTime,

		// 收货信息
		ReceiverFirstName: order.ReceiverFirstName,
		ReceiverLastName:  order.ReceiverLastName,
		ReceiverPhone:     order.ReceiverPhone,
		ReceiverAddress:   order.ReceiverAddress,
		ReceiverCountry:   order.ReceiverCountry,
		ReceiverZipCode:   order.ReceiverZipCode,

		// 其他信息
		Remark:      order.Remark,
		LogisticsNo: order.LogisticsNo,

		// 关联数据
		OrderItems: orderItems,
		StatusLogs: statusLogs,
	}

	return detail, nil
}

// 获取订单状态名称
func getOrderStatusName(status int) string {
	switch status {
	case consts.CREATED:
		return "已创建"
	case consts.PAYED:
		return "已支付"
	case consts.SHIPPED:
		return "已发货"
	case consts.DELIVERED:
		return "已送达"
	case consts.CANCELED:
		return "已取消"
	default:
		return "未知状态"
	}
}

func (o *OrderServiceImpl) CustomerGetOrderDetail(ctx context.Context, orderNo string, userId int) (detail *types.OrderDetail, err error) {
	orderInfo, err := o.GetOrderDetail(ctx, orderNo)
	if err != nil {
		log.Logger.Errorf("CustomerGetOrderDetail: get order detail failed, err %s", err.Error())
		return nil, err
	}
	if orderInfo.UserID != userId {
		wrongUserErr := errors.New("Invalid user ID")
		log.Logger.Errorf("CustomerGetOrderDetail: Invalid userID, err %s", wrongUserErr.Error())
		return nil, wrongUserErr
	}
	return orderInfo, nil
}
