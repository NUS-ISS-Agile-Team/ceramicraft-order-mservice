package service

import (
	"context"
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
)

type OrderService interface {
	CreateOrder(ctx context.Context, orderInfo types.OrderInfo) (orderNo string, err error)
}

type OrderServiceImpl struct {
	lock            sync.Mutex
	orderDao        dao.OrderDao
	orderProductDao dao.OrderProductDao
}

func GetOrderServiceInstance() *OrderServiceImpl {
	return &OrderServiceImpl{
		orderDao:        dao.GetOrderDao(),
		orderProductDao: dao.GetOrderProductDao(),
	}
}

func (o *OrderServiceImpl) CreateOrder(ctx context.Context, orderInfo types.OrderInfo) (orderNo string, err error) {
	o.lock.Lock()
	defer o.lock.Unlock()

	psClient := clients.GetProductClient()

	orderItemIds := make([]int64, len(orderInfo.OrderItemList))
	for idx, item := range orderInfo.OrderItemList {
		orderItemIds[idx] = int64(item.ProductID)
	}

	log.Logger.Infof("CreateOrder: orderItemIds = %v", orderItemIds)

	// 1. rpc: call product service and check if all the related product's stock is enough
	productList, err := psClient.GetProductList(ctx, &productpb.GetProductListRequest{
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

	shippingFee := 1500
	tax := itemTotalAmount * 9 / 100

	// 2. local func: gen order ID
	orderId := utils.GenerateOrderID()

	// 3. save order Info to database
	_, err = dao.GetOrderDao().Create(ctx, &model.Order{
		OrderNo:           orderId,
		UserID:            orderInfo.UserID,
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

	// 4. message queue: send msg -- order ID
	// TODO

	// 5. rpc: call product service and decrease stock
	for _, orderItem := range orderInfo.OrderItemList {
		_, _ = psClient.UpdateStockWithCAS(ctx, &productpb.UpdateStockWithCASRequest{
			Id:   int64(orderItem.ProductID),
			Deta: int64(-1 * orderItem.Quantity),
		})
	}

	// 6. rpc: call payment service and pay
	// TODO

	// 6.1 payment success

	// 6.2 payment failed

	// 7. update order status
	return orderId, nil
}
