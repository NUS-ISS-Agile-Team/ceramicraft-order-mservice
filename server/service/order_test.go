package service

import (
	"context"
	"errors"
	"time"

	// "errors"
	"testing"

	"github.com/NUS-ISS-Agile-Team/ceramicraft-commodity-mservice/common/productpb"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/clients/mocks"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/log"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/pkg/consts"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/pkg/types"

	// "github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/pkg/utils"
	utilMocks "github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/pkg/utils/mocks"
	daoMocks "github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/repository/dao/mocks"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/repository/model"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-payment-mservice/common/paymentpb"
	"github.com/golang/mock/gomock"
	"go.uber.org/zap"
	// "github.com/stretchr/testify/assert"
)

func init() {
	// 初始化测试用logger
	logger, _ := zap.NewDevelopment()
	log.Logger = logger.Sugar()
}

func TestOrderServiceImpl_CreateOrder_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create all mocks
	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockOrderProductDao := daoMocks.NewMockOrderProductDao(ctrl)
	mockProductClient := mocks.NewMockProductServiceClient(ctrl)
	mockPaymentClient := mocks.NewMockPaymentServiceClient(ctrl)
	mockKafkaWriter := utilMocks.NewMockWriter(ctrl)

	// Setup test data
	ctx := context.TODO()
	orderInfo := types.OrderInfo{
		ReceiverFirstName: "John",
		ReceiverLastName:  "Doe",
		ReceiverPhone:     "1234567890",
		ReceiverAddress:   "123 Test St",
		ReceiverCountry:   "USA",
		ReceiverZipCode:   12345,
		Remark:            "Test order",
		OrderItemList: []*types.OrderItemInfo{
			{
				ProductID:   1,
				ProductName: "Test Product",
				Quantity:    2,
				Price:       1000,
			},
		},
	}

	// Mock product service - return sufficient stock
	mockProductClient.EXPECT().
		GetProductList(ctx, &productpb.GetProductListRequest{
			Ids: []int64{1},
		}).
		Return(&productpb.GetProductListResponse{
			Products: []*productpb.Product{
				{
					Id:    1,
					Stock: 10, // Sufficient stock
				},
			},
		}, nil).
		Times(1)

	// Mock order DAO - successful creation
	mockOrderDao.EXPECT().
		Create(ctx, gomock.Any()).
		Return("test-order-123", nil).
		Times(1)

	// Mock order product DAO - successful batch creation
	mockOrderProductDao.EXPECT().
		CreateBatch(ctx, gomock.Any()).
		Return(1, nil).
		Times(1)

	// Mock Kafka messages
	mockKafkaWriter.EXPECT().
		SendMsg(ctx, "order_created", gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	mockKafkaWriter.EXPECT().
		SendMsg(ctx, "order_status_changed", gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	// Mock product stock update
	mockProductClient.EXPECT().
		UpdateStockWithCAS(ctx, &productpb.UpdateStockWithCASRequest{
			Id:   1,
			Deta: -2,
		}).
		Return(&productpb.UpdateStockWithCASResponse{}, nil).
		Times(1)

	// Mock payment service - successful payment
	mockPaymentClient.EXPECT().
		PayOrder(ctx, gomock.Any()).
		Return(&paymentpb.PayOrderResponse{
			Code: 0, // Success
		}, nil).
		Times(1)

	// Mock order status update after payment
	mockOrderDao.EXPECT().
		UpdateStatusAndPayment(ctx, gomock.Any(), consts.PAYED, gomock.Any()).
		Return(nil).
		Times(1)

	mockKafkaWriter.EXPECT().
		SendMsg(ctx, "order_status_changed", gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	// Create service instance with all mocks
	service := &OrderServiceImpl{
		orderDao:             mockOrderDao,
		orderProductDao:      mockOrderProductDao,
		productServiceClient: mockProductClient,
		paymentServiceClient: mockPaymentClient,
		messageWriter:        mockKafkaWriter,
		syncMode:             true,
	}

	// Now we can actually test the CreateOrder method
	orderNo, err := service.CreateOrder(ctx, orderInfo, 123)
	if err != nil {
		t.Errorf("Expected no error, got: %s", err.Error())
	}
	if orderNo == "" {
		t.Errorf("Expected orderNo to be not empty, got: %s", orderNo)
	}
}

func TestOrderServiceImpl_CreateOrder_GetProductListError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockOrderProductDao := daoMocks.NewMockOrderProductDao(ctrl)
	mockProductClient := mocks.NewMockProductServiceClient(ctrl)
	mockPaymentClient := mocks.NewMockPaymentServiceClient(ctrl)
	mockKafkaWriter := utilMocks.NewMockWriter(ctrl)

	// Setup test data
	ctx := context.TODO()
	orderInfo := types.OrderInfo{
		ReceiverFirstName: "John",
		ReceiverLastName:  "Doe",
		ReceiverPhone:     "1234567890",
		ReceiverAddress:   "123 Test St",
		ReceiverCountry:   "USA",
		ReceiverZipCode:   12345,
		Remark:            "Test order",
		OrderItemList: []*types.OrderItemInfo{
			{
				ProductID:   1,
				ProductName: "Test Product",
				Quantity:    2,
				Price:       1000,
			},
		},
	}

	// Mock product service - return error
	mockProductClient.EXPECT().
		GetProductList(ctx, &productpb.GetProductListRequest{
			Ids: []int64{1},
		}).
		Return(nil, errors.New("product service unavailable")).
		Times(1)

	// Create service instance with mocks
	service := &OrderServiceImpl{
		orderDao:             mockOrderDao,
		orderProductDao:      mockOrderProductDao,
		productServiceClient: mockProductClient,
		paymentServiceClient: mockPaymentClient,
		messageWriter:        mockKafkaWriter,
		syncMode:             true,
	}

	// Test the CreateOrder method
	orderNo, err := service.CreateOrder(ctx, orderInfo, 123)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if orderNo != "" {
		t.Errorf("Expected empty orderNo, got: %s", orderNo)
	}
}

func TestOrderServiceImpl_CreateOrder_InsufficientStock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockOrderProductDao := daoMocks.NewMockOrderProductDao(ctrl)
	mockProductClient := mocks.NewMockProductServiceClient(ctrl)
	mockPaymentClient := mocks.NewMockPaymentServiceClient(ctrl)
	mockKafkaWriter := utilMocks.NewMockWriter(ctrl)

	// Setup test data with high quantity
	ctx := context.TODO()
	orderInfo := types.OrderInfo{
		ReceiverFirstName: "John",
		ReceiverLastName:  "Doe",
		ReceiverPhone:     "1234567890",
		ReceiverAddress:   "123 Test St",
		ReceiverCountry:   "USA",
		ReceiverZipCode:   12345,
		Remark:            "Test order",
		OrderItemList: []*types.OrderItemInfo{
			{
				ProductID:   1,
				ProductName: "Test Product",
				Quantity:    10, // High quantity
				Price:       1000,
			},
		},
	}

	// Mock product service - return insufficient stock
	mockProductClient.EXPECT().
		GetProductList(ctx, &productpb.GetProductListRequest{
			Ids: []int64{1},
		}).
		Return(&productpb.GetProductListResponse{
			Products: []*productpb.Product{
				{
					Id:    1,
					Stock: 5, // Insufficient stock
				},
			},
		}, nil).
		Times(1)

	// Create service instance with mocks
	service := &OrderServiceImpl{
		orderDao:             mockOrderDao,
		orderProductDao:      mockOrderProductDao,
		productServiceClient: mockProductClient,
		paymentServiceClient: mockPaymentClient,
		messageWriter:        mockKafkaWriter,
		syncMode:             true,
	}

	// Test the CreateOrder method
	orderNo, err := service.CreateOrder(ctx, orderInfo, 123)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if orderNo != "" {
		t.Errorf("Expected empty orderNo, got: %s", orderNo)
	}
}

func TestOrderServiceImpl_CreateOrder_OrderDaoCreateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockOrderProductDao := daoMocks.NewMockOrderProductDao(ctrl)
	mockProductClient := mocks.NewMockProductServiceClient(ctrl)
	mockPaymentClient := mocks.NewMockPaymentServiceClient(ctrl)
	mockKafkaWriter := utilMocks.NewMockWriter(ctrl)

	// Setup test data
	ctx := context.TODO()
	orderInfo := types.OrderInfo{
		ReceiverFirstName: "John",
		ReceiverLastName:  "Doe",
		ReceiverPhone:     "1234567890",
		ReceiverAddress:   "123 Test St",
		ReceiverCountry:   "USA",
		ReceiverZipCode:   12345,
		Remark:            "Test order",
		OrderItemList: []*types.OrderItemInfo{
			{
				ProductID:   1,
				ProductName: "Test Product",
				Quantity:    2,
				Price:       1000,
			},
		},
	}

	// Mock product service - return sufficient stock
	mockProductClient.EXPECT().
		GetProductList(ctx, &productpb.GetProductListRequest{
			Ids: []int64{1},
		}).
		Return(&productpb.GetProductListResponse{
			Products: []*productpb.Product{
				{
					Id:    1,
					Stock: 10,
				},
			},
		}, nil).
		Times(1)

	// Mock order DAO - return error
	mockOrderDao.EXPECT().
		Create(ctx, gomock.Any()).
		Return("", errors.New("database connection failed")).
		Times(1)

	// Create service instance with mocks
	service := &OrderServiceImpl{
		orderDao:             mockOrderDao,
		orderProductDao:      mockOrderProductDao,
		productServiceClient: mockProductClient,
		paymentServiceClient: mockPaymentClient,
		messageWriter:        mockKafkaWriter,
		syncMode:             true,
	}

	// Test the CreateOrder method
	orderNo, err := service.CreateOrder(ctx, orderInfo, 123)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if orderNo != "" {
		t.Errorf("Expected empty orderNo, got: %s", orderNo)
	}
}

func TestOrderServiceImpl_ListOrders_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockOrderProductDao := daoMocks.NewMockOrderProductDao(ctrl)
	mockOrderLogDao := daoMocks.NewMockOrderLogDao(ctrl)

	ctx := context.Background()
	req := types.ListOrderRequest{
		UserID: 123,
		Limit:  2,
		Offset: 0,
	}
	orders := []*model.Order{
		{
			OrderNo:           "order1",
			ReceiverFirstName: "A",
			ReceiverLastName:  "B",
			ReceiverPhone:     "123",
			CreateTime:        time.Now(),
			TotalAmount:       100,
			Status:            consts.CREATED,
		},
		{
			OrderNo:           "order2",
			ReceiverFirstName: "C",
			ReceiverLastName:  "D",
			ReceiverPhone:     "456",
			CreateTime:        time.Now(),
			TotalAmount:       200,
			Status:            consts.PAYED,
		},
	}
	mockOrderDao.EXPECT().GetByOrderQuery(ctx, gomock.Any()).Return(orders, nil)

	service := &OrderServiceImpl{
		orderDao:        mockOrderDao,
		orderProductDao: mockOrderProductDao,
		orderLogDao:     mockOrderLogDao,
		syncMode:        true,
	}
	resp, err := service.ListOrders(ctx, req)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if len(resp.Orders) != 2 {
		t.Errorf("Expected 2 orders, got: %d", len(resp.Orders))
	}
	if resp.Orders[0].OrderNo != "order1" || resp.Orders[1].OrderNo != "order2" {
		t.Errorf("OrderNo mismatch: %v", resp.Orders)
	}
}

func TestOrderServiceImpl_ListOrders_DaoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockOrderProductDao := daoMocks.NewMockOrderProductDao(ctrl)
	mockOrderLogDao := daoMocks.NewMockOrderLogDao(ctrl)

	ctx := context.Background()
	req := types.ListOrderRequest{UserID: 123}
	mockOrderDao.EXPECT().GetByOrderQuery(ctx, gomock.Any()).Return(nil, errors.New("db error"))

	service := &OrderServiceImpl{
		orderDao:        mockOrderDao,
		orderProductDao: mockOrderProductDao,
		orderLogDao:     mockOrderLogDao,
		syncMode:        true,
	}
	resp, err := service.ListOrders(ctx, req)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if resp != nil {
		t.Errorf("Expected nil resp, got: %v", resp)
	}
}

func TestOrderServiceImpl_GetOrderDetail_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockOrderProductDao := daoMocks.NewMockOrderProductDao(ctrl)
	mockOrderLogDao := daoMocks.NewMockOrderLogDao(ctrl)

	ctx := context.Background()
	orderNo := "order1"
	order := &model.Order{
		OrderNo:           orderNo,
		UserID:            123,
		Status:            consts.CREATED,
		TotalAmount:       100,
		PayAmount:         100,
		ShippingFee:       10,
		Tax:               5,
		ReceiverFirstName: "A",
		ReceiverLastName:  "B",
		ReceiverPhone:     "123",
		ReceiverAddress:   "addr",
		ReceiverCountry:   "CN",
		ReceiverZipCode:   10000,
		Remark:            "remark",
		LogisticsNo:       "LN123",
		CreateTime:        time.Now(),
		UpdateTime:        time.Now(),
	}
	products := []*model.OrderProduct{
		{
			ID: 1, ProductID: 2, ProductName: "P1", Price: 10, Quantity: 1, TotalPrice: 10, CreateTime: time.Now(), UpdateTime: time.Now(),
		},
	}
	logs := []*model.OrderStatusLog{
		{
			ID: 1, CurrentStatus: consts.CREATED, Remark: "created", CreateTime: time.Now(),
		},
	}
	mockOrderDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(order, nil)
	mockOrderProductDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(products, nil)
	mockOrderLogDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(logs, nil)

	service := &OrderServiceImpl{
		orderDao:        mockOrderDao,
		orderProductDao: mockOrderProductDao,
		orderLogDao:     mockOrderLogDao,
		syncMode:        true,
	}
	detail, err := service.GetOrderDetail(ctx, orderNo)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if detail.OrderNo != orderNo {
		t.Errorf("OrderNo mismatch: %v", detail.OrderNo)
	}
	if len(detail.OrderItems) != 1 || detail.OrderItems[0].ProductName != "P1" {
		t.Errorf("OrderItems mismatch: %v", detail.OrderItems)
	}
	if len(detail.StatusLogs) != 1 || detail.StatusLogs[0].Remark != "created" {
		t.Errorf("StatusLogs mismatch: %v", detail.StatusLogs)
	}
}

func TestOrderServiceImpl_GetOrderDetail_OrderNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockOrderProductDao := daoMocks.NewMockOrderProductDao(ctrl)
	mockOrderLogDao := daoMocks.NewMockOrderLogDao(ctrl)

	ctx := context.Background()
	orderNo := "order1"
	mockOrderDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(nil, errors.New("not found"))

	service := &OrderServiceImpl{
		orderDao:        mockOrderDao,
		orderProductDao: mockOrderProductDao,
		orderLogDao:     mockOrderLogDao,
		syncMode:        true,
	}
	detail, err := service.GetOrderDetail(ctx, orderNo)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if detail != nil {
		t.Errorf("Expected nil detail, got: %v", detail)
	}
}

func TestOrderServiceImpl_GetOrderDetail_ProductError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockOrderProductDao := daoMocks.NewMockOrderProductDao(ctrl)
	mockOrderLogDao := daoMocks.NewMockOrderLogDao(ctrl)

	ctx := context.Background()
	orderNo := "order1"
	order := &model.Order{OrderNo: orderNo}
	mockOrderDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(order, nil)
	mockOrderProductDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(nil, errors.New("product error"))

	service := &OrderServiceImpl{
		orderDao:        mockOrderDao,
		orderProductDao: mockOrderProductDao,
		orderLogDao:     mockOrderLogDao,
		syncMode:        true,
	}
	detail, err := service.GetOrderDetail(ctx, orderNo)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if detail != nil {
		t.Errorf("Expected nil detail, got: %v", detail)
	}
}

func TestOrderServiceImpl_GetOrderDetail_LogError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockOrderProductDao := daoMocks.NewMockOrderProductDao(ctrl)
	mockOrderLogDao := daoMocks.NewMockOrderLogDao(ctrl)

	ctx := context.Background()
	orderNo := "order1"
	order := &model.Order{OrderNo: orderNo}
	products := []*model.OrderProduct{{ID: 1}}
	mockOrderDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(order, nil)
	mockOrderProductDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(products, nil)
	mockOrderLogDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(nil, errors.New("log error"))

	service := &OrderServiceImpl{
		orderDao:        mockOrderDao,
		orderProductDao: mockOrderProductDao,
		orderLogDao:     mockOrderLogDao,
		syncMode:        true,
	}
	detail, err := service.GetOrderDetail(ctx, orderNo)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if detail != nil {
		t.Errorf("Expected nil detail, got: %v", detail)
	}
}

func TestOrderServiceImpl_CreateOrder_OrderProductDaoCreateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockOrderProductDao := daoMocks.NewMockOrderProductDao(ctrl)
	mockProductClient := mocks.NewMockProductServiceClient(ctrl)
	mockPaymentClient := mocks.NewMockPaymentServiceClient(ctrl)
	mockKafkaWriter := utilMocks.NewMockWriter(ctrl)

	// Setup test data
	ctx := context.TODO()
	orderInfo := types.OrderInfo{
		ReceiverFirstName: "John",
		ReceiverLastName:  "Doe",
		ReceiverPhone:     "1234567890",
		ReceiverAddress:   "123 Test St",
		ReceiverCountry:   "USA",
		ReceiverZipCode:   12345,
		Remark:            "Test order",
		OrderItemList: []*types.OrderItemInfo{
			{
				ProductID:   1,
				ProductName: "Test Product",
				Quantity:    2,
				Price:       1000,
			},
		},
	}

	// Mock product service - return sufficient stock
	mockProductClient.EXPECT().
		GetProductList(ctx, &productpb.GetProductListRequest{
			Ids: []int64{1},
		}).
		Return(&productpb.GetProductListResponse{
			Products: []*productpb.Product{
				{
					Id:    1,
					Stock: 10,
				},
			},
		}, nil).
		Times(1)

	// Mock order DAO - successful creation
	mockOrderDao.EXPECT().
		Create(ctx, gomock.Any()).
		Return("test-order-123", nil).
		Times(1)

	// Mock order product DAO - batch creation returns error
	mockOrderProductDao.EXPECT().
		CreateBatch(ctx, gomock.Any()).
		Return(0, errors.New("failed to create order product")).
		Times(1)

	// Create service instance with mocks
	service := &OrderServiceImpl{
		orderDao:             mockOrderDao,
		orderProductDao:      mockOrderProductDao,
		productServiceClient: mockProductClient,
		paymentServiceClient: mockPaymentClient,
		messageWriter:        mockKafkaWriter,
		syncMode:             true,
	}

	// Test the CreateOrder method
	orderNo, err := service.CreateOrder(ctx, orderInfo, 123)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if orderNo != "" {
		t.Errorf("Expected empty orderNo, got: %s", orderNo)
	}
}

func TestOrderServiceImpl_CreateOrder_KafkaOrderCreatedError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockOrderProductDao := daoMocks.NewMockOrderProductDao(ctrl)
	mockProductClient := mocks.NewMockProductServiceClient(ctrl)
	mockPaymentClient := mocks.NewMockPaymentServiceClient(ctrl)
	mockKafkaWriter := utilMocks.NewMockWriter(ctrl)

	// Setup test data
	ctx := context.TODO()
	orderInfo := types.OrderInfo{
		ReceiverFirstName: "John",
		ReceiverLastName:  "Doe",
		ReceiverPhone:     "1234567890",
		ReceiverAddress:   "123 Test St",
		ReceiverCountry:   "USA",
		ReceiverZipCode:   12345,
		Remark:            "Test order",
		OrderItemList: []*types.OrderItemInfo{
			{
				ProductID:   1,
				ProductName: "Test Product",
				Quantity:    2,
				Price:       1000,
			},
		},
	}

	// Mock product service - return sufficient stock
	mockProductClient.EXPECT().
		GetProductList(ctx, &productpb.GetProductListRequest{
			Ids: []int64{1},
		}).
		Return(&productpb.GetProductListResponse{
			Products: []*productpb.Product{
				{
					Id:    1,
					Stock: 10,
				},
			},
		}, nil).
		Times(1)

	// Mock order DAO - successful creation
	mockOrderDao.EXPECT().
		Create(ctx, gomock.Any()).
		Return("test-order-123", nil).
		Times(1)

	// Mock order product DAO - successful batch creation
	mockOrderProductDao.EXPECT().
		CreateBatch(ctx, gomock.Any()).
		Return(1, nil).
		Times(1)

	// Mock Kafka messages - return error for order_created
	mockKafkaWriter.EXPECT().
		SendMsg(ctx, "order_created", gomock.Any(), gomock.Any()).
		Return(errors.New("kafka connection failed")).
		Times(1)

	// Create service instance with mocks
	service := &OrderServiceImpl{
		orderDao:             mockOrderDao,
		orderProductDao:      mockOrderProductDao,
		productServiceClient: mockProductClient,
		paymentServiceClient: mockPaymentClient,
		messageWriter:        mockKafkaWriter,
		syncMode:             true,
	}

	// Test the CreateOrder method
	orderNo, err := service.CreateOrder(ctx, orderInfo, 123)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if orderNo != "" {
		t.Errorf("Expected empty orderNo, got: %s", orderNo)
	}
}

func TestOrderServiceImpl_CreateOrder_PaymentFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockOrderProductDao := daoMocks.NewMockOrderProductDao(ctrl)
	mockProductClient := mocks.NewMockProductServiceClient(ctrl)
	mockPaymentClient := mocks.NewMockPaymentServiceClient(ctrl)
	mockKafkaWriter := utilMocks.NewMockWriter(ctrl)

	// Setup test data
	ctx := context.TODO()
	orderInfo := types.OrderInfo{
		ReceiverFirstName: "John",
		ReceiverLastName:  "Doe",
		ReceiverPhone:     "1234567890",
		ReceiverAddress:   "123 Test St",
		ReceiverCountry:   "USA",
		ReceiverZipCode:   12345,
		Remark:            "Test order",
		OrderItemList: []*types.OrderItemInfo{
			{
				ProductID:   1,
				ProductName: "Test Product",
				Quantity:    2,
				Price:       1000,
			},
		},
	}

	// Mock product service - return sufficient stock
	mockProductClient.EXPECT().
		GetProductList(ctx, gomock.Any()).
		Return(&productpb.GetProductListResponse{
			Products: []*productpb.Product{
				{
					Id:    1,
					Stock: 10,
				},
			},
		}, nil).
		Times(1)

	// Mock successful order creation
	mockOrderDao.EXPECT().
		Create(ctx, gomock.Any()).
		Return("test-order-123", nil).
		Times(1)

	mockOrderProductDao.EXPECT().
		CreateBatch(ctx, gomock.Any()).
		Return(1, nil).
		Times(1)

	// Mock Kafka messages
	mockKafkaWriter.EXPECT().
		SendMsg(ctx, "order_created", gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	mockKafkaWriter.EXPECT().
		SendMsg(ctx, "order_status_changed", gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	// Mock product stock update
	mockProductClient.EXPECT().
		UpdateStockWithCAS(ctx, gomock.Any()).
		Return(&productpb.UpdateStockWithCASResponse{}, nil).
		Times(1)

	// Mock payment service - payment failed with error message
	errorMsg := "Insufficient balance"
	mockPaymentClient.EXPECT().
		PayOrder(ctx, gomock.Any()).
		Return(&paymentpb.PayOrderResponse{
			Code:     1, // Failed
			ErrorMsg: &errorMsg,
		}, nil).
		Times(1)

	// Mock cancel message
	mockKafkaWriter.EXPECT().
		SendMsg(ctx, "order_canceled", gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	// Create service instance with mocks
	service := &OrderServiceImpl{
		orderDao:             mockOrderDao,
		orderProductDao:      mockOrderProductDao,
		productServiceClient: mockProductClient,
		paymentServiceClient: mockPaymentClient,
		messageWriter:        mockKafkaWriter,
		syncMode:             true,
	}

	// Test the CreateOrder method
	orderNo, err := service.CreateOrder(ctx, orderInfo, 123)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if orderNo != "" {
		t.Errorf("Expected empty orderNo, got: %s", orderNo)
	}
}

func TestOrderServiceImpl_CustomerGetOrderDetail_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockOrderProductDao := daoMocks.NewMockOrderProductDao(ctrl)
	mockOrderLogDao := daoMocks.NewMockOrderLogDao(ctrl)

	ctx := context.Background()
	orderNo := "order1"
	userID := 123
	order := &model.Order{
		OrderNo:           orderNo,
		UserID:            userID,
		Status:            consts.CREATED,
		TotalAmount:       100,
		PayAmount:         100,
		ShippingFee:       10,
		Tax:               5,
		ReceiverFirstName: "A",
		ReceiverLastName:  "B",
		ReceiverPhone:     "123",
		ReceiverAddress:   "addr",
		ReceiverCountry:   "CN",
		ReceiverZipCode:   10000,
		Remark:            "remark",
		LogisticsNo:       "LN123",
		CreateTime:        time.Now(),
		UpdateTime:        time.Now(),
	}
	products := []*model.OrderProduct{
		{
			ID:          1,
			ProductID:   2,
			ProductName: "P1",
			Price:       10,
			Quantity:    1,
			TotalPrice:  10,
			CreateTime:  time.Now(),
			UpdateTime:  time.Now(),
		},
	}
	logs := []*model.OrderStatusLog{
		{
			ID:            1,
			CurrentStatus: consts.CREATED,
			Remark:        "created",
			CreateTime:    time.Now(),
		},
	}
	mockOrderDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(order, nil)
	mockOrderProductDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(products, nil)
	mockOrderLogDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(logs, nil)

	service := &OrderServiceImpl{
		orderDao:        mockOrderDao,
		orderProductDao: mockOrderProductDao,
		orderLogDao:     mockOrderLogDao,
		syncMode:        true,
	}
	detail, err := service.CustomerGetOrderDetail(ctx, orderNo, userID)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if detail.OrderNo != orderNo {
		t.Errorf("OrderNo mismatch: %v", detail.OrderNo)
	}
	if detail.UserID != userID {
		t.Errorf("UserID mismatch: expected %d, got %d", userID, detail.UserID)
	}
	if len(detail.OrderItems) != 1 || detail.OrderItems[0].ProductName != "P1" {
		t.Errorf("OrderItems mismatch: %v", detail.OrderItems)
	}
	if len(detail.StatusLogs) != 1 || detail.StatusLogs[0].Remark != "created" {
		t.Errorf("StatusLogs mismatch: %v", detail.StatusLogs)
	}
}

func TestOrderServiceImpl_CustomerGetOrderDetail_WrongUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockOrderProductDao := daoMocks.NewMockOrderProductDao(ctrl)
	mockOrderLogDao := daoMocks.NewMockOrderLogDao(ctrl)

	ctx := context.Background()
	orderNo := "order1"
	orderOwnerID := 123
	wrongUserID := 456
	order := &model.Order{
		OrderNo:           orderNo,
		UserID:            orderOwnerID, // 订单属于用户123
		Status:            consts.CREATED,
		TotalAmount:       100,
		PayAmount:         100,
		ShippingFee:       10,
		Tax:               5,
		ReceiverFirstName: "A",
		ReceiverLastName:  "B",
		ReceiverPhone:     "123",
		ReceiverAddress:   "addr",
		ReceiverCountry:   "CN",
		ReceiverZipCode:   10000,
		Remark:            "remark",
		LogisticsNo:       "LN123",
		CreateTime:        time.Now(),
		UpdateTime:        time.Now(),
	}
	products := []*model.OrderProduct{
		{
			ID:          1,
			ProductID:   2,
			ProductName: "P1",
			Price:       10,
			Quantity:    1,
			TotalPrice:  10,
			CreateTime:  time.Now(),
			UpdateTime:  time.Now(),
		},
	}
	logs := []*model.OrderStatusLog{
		{
			ID:            1,
			CurrentStatus: consts.CREATED,
			Remark:        "created",
			CreateTime:    time.Now(),
		},
	}

	// Mock完整的GetOrderDetail调用链
	mockOrderDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(order, nil)
	mockOrderProductDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(products, nil)
	mockOrderLogDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(logs, nil)

	service := &OrderServiceImpl{
		orderDao:        mockOrderDao,
		orderProductDao: mockOrderProductDao,
		orderLogDao:     mockOrderLogDao,
		syncMode:        true,
	}

	// 用户456尝试访问用户123的订单
	detail, err := service.CustomerGetOrderDetail(ctx, orderNo, wrongUserID)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if err.Error() != "invalid user ID" {
		t.Errorf("Expected 'invalid user ID' error, got: %v", err)
	}
	if detail != nil {
		t.Errorf("Expected nil detail, got: %v", detail)
	}
}

func TestOrderServiceImpl_CustomerGetOrderDetail_OrderNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockOrderProductDao := daoMocks.NewMockOrderProductDao(ctrl)
	mockOrderLogDao := daoMocks.NewMockOrderLogDao(ctrl)

	ctx := context.Background()
	orderNo := "order1"
	userID := 123
	mockOrderDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(nil, errors.New("not found"))

	service := &OrderServiceImpl{
		orderDao:        mockOrderDao,
		orderProductDao: mockOrderProductDao,
		orderLogDao:     mockOrderLogDao,
		syncMode:        true,
	}
	detail, err := service.CustomerGetOrderDetail(ctx, orderNo, userID)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if detail != nil {
		t.Errorf("Expected nil detail, got: %v", detail)
	}
}

func TestOrderServiceImpl_CustomerGetOrderDetail_ProductError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockOrderProductDao := daoMocks.NewMockOrderProductDao(ctrl)
	mockOrderLogDao := daoMocks.NewMockOrderLogDao(ctrl)

	ctx := context.Background()
	orderNo := "order1"
	userID := 123
	order := &model.Order{OrderNo: orderNo, UserID: userID}
	mockOrderDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(order, nil)
	mockOrderProductDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(nil, errors.New("product error"))

	service := &OrderServiceImpl{
		orderDao:        mockOrderDao,
		orderProductDao: mockOrderProductDao,
		orderLogDao:     mockOrderLogDao,
		syncMode:        true,
	}
	detail, err := service.CustomerGetOrderDetail(ctx, orderNo, userID)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if detail != nil {
		t.Errorf("Expected nil detail, got: %v", detail)
	}
}

func TestOrderServiceImpl_CustomerGetOrderDetail_LogError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockOrderProductDao := daoMocks.NewMockOrderProductDao(ctrl)
	mockOrderLogDao := daoMocks.NewMockOrderLogDao(ctrl)

	ctx := context.Background()
	orderNo := "order1"
	userID := 123
	order := &model.Order{OrderNo: orderNo, UserID: userID}
	products := []*model.OrderProduct{{ID: 1}}
	mockOrderDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(order, nil)
	mockOrderProductDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(products, nil)
	mockOrderLogDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(nil, errors.New("log error"))

	service := &OrderServiceImpl{
		orderDao:        mockOrderDao,
		orderProductDao: mockOrderProductDao,
		orderLogDao:     mockOrderLogDao,
		syncMode:        true,
	}
	detail, err := service.CustomerGetOrderDetail(ctx, orderNo, userID)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if detail != nil {
		t.Errorf("Expected nil detail, got: %v", detail)
	}
}

// TestOrderServiceImpl_UpdateOrderStatus_ToShipped_Success tests successful update to shipped status
func TestOrderServiceImpl_UpdateOrderStatus_ToShipped_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockMessageWriter := utilMocks.NewMockWriter(ctrl)

	ctx := context.Background()
	orderNo := "TEST001"
	newStatus := int(consts.SHIPPED)
	logisticsInfo := "SF12345"

	// Mock successful DAO operations
	mockOrderDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(&model.Order{
		OrderNo: orderNo,
		Status:  int(consts.PAYED),
	}, nil)
	mockOrderDao.EXPECT().UpdateStatusWithDeliveryInfo(ctx, orderNo, newStatus, gomock.Any(), logisticsInfo).Return(nil)

	// Mock successful Kafka message
	mockMessageWriter.EXPECT().SendMsg(ctx, "order_status_changed", gomock.Any(), gomock.Any()).Return(nil)

	service := &OrderServiceImpl{
		orderDao:      mockOrderDao,
		messageWriter: mockMessageWriter,
		syncMode:      true,
	}

	err := service.UpdateOrderStatus(ctx, orderNo, newStatus, logisticsInfo)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// TestOrderServiceImpl_UpdateOrderStatus_ToDelivered_Success tests successful update to delivered status
func TestOrderServiceImpl_UpdateOrderStatus_ToDelivered_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockMessageWriter := utilMocks.NewMockWriter(ctrl)

	ctx := context.Background()
	orderNo := "TEST002"
	newStatus := int(consts.DELIVERED)

	// Mock successful DAO operations
	mockOrderDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(&model.Order{
		OrderNo: orderNo,
		Status:  int(consts.SHIPPED),
	}, nil)
	mockOrderDao.EXPECT().UpdateStatusAndConfirmTime(ctx, orderNo, newStatus, gomock.Any()).Return(nil)

	// Mock successful Kafka message
	mockMessageWriter.EXPECT().SendMsg(ctx, "order_status_changed", gomock.Any(), gomock.Any()).Return(nil)

	service := &OrderServiceImpl{
		orderDao:      mockOrderDao,
		messageWriter: mockMessageWriter,
		syncMode:      true,
	}

	err := service.UpdateOrderStatus(ctx, orderNo, newStatus, "")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// TestOrderServiceImpl_UpdateOrderStatus_OrderNotFound tests order not found scenario
func TestOrderServiceImpl_UpdateOrderStatus_OrderNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)

	ctx := context.Background()
	orderNo := "NOTFOUND"
	newStatus := int(consts.SHIPPED)

	mockOrderDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(nil, errors.New("order not found"))

	service := &OrderServiceImpl{
		orderDao: mockOrderDao,
		syncMode: true,
	}

	err := service.UpdateOrderStatus(ctx, orderNo, newStatus, "")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

// TestOrderServiceImpl_UpdateOrderStatus_InvalidStatusTransition tests invalid status transition
func TestOrderServiceImpl_UpdateOrderStatus_InvalidStatusTransition(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)

	ctx := context.Background()
	orderNo := "TEST003"
	newStatus := int(consts.DELIVERED)

	// Mock order with status that cannot transition directly to delivered
	mockOrderDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(&model.Order{
		OrderNo: orderNo,
		Status:  int(consts.PAYED), // Cannot go from PAYED(2) to DELIVERED(4)
	}, nil)

	service := &OrderServiceImpl{
		orderDao: mockOrderDao,
		syncMode: true,
	}

	err := service.UpdateOrderStatus(ctx, orderNo, newStatus, "")
	if err == nil {
		t.Errorf("Expected error for invalid status transition, got nil")
	}
}

// TestOrderServiceImpl_UpdateOrderStatus_UnsupportedStatus tests unsupported status update
func TestOrderServiceImpl_UpdateOrderStatus_UnsupportedStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)

	ctx := context.Background()
	orderNo := "TEST004"
	newStatus := int(consts.PAYED) // Unsupported status for this method

	mockOrderDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(&model.Order{
		OrderNo: orderNo,
		Status:  int(consts.CREATED),
	}, nil)

	service := &OrderServiceImpl{
		orderDao: mockOrderDao,
		syncMode: true,
	}

	err := service.UpdateOrderStatus(ctx, orderNo, newStatus, "")
	if err == nil {
		t.Errorf("Expected error for unsupported status, got nil")
	}
}

// TestOrderServiceImpl_UpdateOrderStatus_DaoUpdateError tests DAO update error
func TestOrderServiceImpl_UpdateOrderStatus_DaoUpdateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)

	ctx := context.Background()
	orderNo := "TEST005"
	newStatus := int(consts.DELIVERED)

	mockOrderDao.EXPECT().GetByOrderNo(ctx, orderNo).Return(&model.Order{
		OrderNo: orderNo,
		Status:  int(consts.SHIPPED),
	}, nil)
	mockOrderDao.EXPECT().UpdateStatusAndConfirmTime(ctx, orderNo, newStatus, gomock.Any()).Return(errors.New("database error"))

	service := &OrderServiceImpl{
		orderDao: mockOrderDao,
		syncMode: true,
	}

	err := service.UpdateOrderStatus(ctx, orderNo, newStatus, "")
	if err == nil {
		t.Errorf("Expected error from DAO update, got nil")
	}
}

// TestOrderServiceImpl_OrderAutoConfirm_Success tests successful auto-confirmation
func TestOrderServiceImpl_OrderAutoConfirm_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockLocker := utilMocks.NewMockLocker(ctrl)
	mockKafkaWriter := utilMocks.NewMockWriter(ctrl)

	ctx := context.Background()

	// Mock lock acquisition success
	mockLocker.EXPECT().Lock(ctx).Return(nil).Times(1)
	mockLocker.EXPECT().Unlock(ctx).Return(nil).Times(1)

	// Mock DAO returns 2 orders to auto-confirm
	autoConfirmedOrders := []types.OrderNoAndUserId{
		{OrderNo: "ORDER001", UserID: 101},
		{OrderNo: "ORDER002", UserID: 102},
	}
	mockOrderDao.EXPECT().
		AutoConfirmShippedOrders(ctx, consts.SHIPPED, consts.DELIVERED, AUTO_CONFIRM_AFTER_DAYS).
		Return(autoConfirmedOrders, nil).
		Times(1)

	// Mock Kafka messages for each order
	mockKafkaWriter.EXPECT().
		SendMsg(ctx, "order_status_changed", "ORDER001", gomock.Any()).
		Return(nil).
		Times(1)
	mockKafkaWriter.EXPECT().
		SendMsg(ctx, "order_status_changed", "ORDER002", gomock.Any()).
		Return(nil).
		Times(1)

	service := &OrderServiceImpl{
		orderDao:          mockOrderDao,
		messageWriter:     mockKafkaWriter,
		distributedLocker: mockLocker,
		syncMode:          true,
	}

	// Execute
	service.OrderAutoConfirm(ctx)
}

// TestOrderServiceImpl_OrderAutoConfirm_LockFailed tests lock acquisition failure
func TestOrderServiceImpl_OrderAutoConfirm_LockFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLocker := utilMocks.NewMockLocker(ctrl)

	ctx := context.Background()

	// Mock lock acquisition failure (another instance holds the lock)
	mockLocker.EXPECT().Lock(ctx).Return(errors.New("lock already held")).Times(1)
	// Unlock should NOT be called if lock acquisition fails
	mockLocker.EXPECT().Unlock(ctx).Times(0)

	service := &OrderServiceImpl{
		distributedLocker: mockLocker,
		syncMode:          true,
	}

	// Execute - should return gracefully without error
	service.OrderAutoConfirm(ctx)
}

// TestOrderServiceImpl_OrderAutoConfirm_DaoError tests DAO error during auto-confirm
func TestOrderServiceImpl_OrderAutoConfirm_DaoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockLocker := utilMocks.NewMockLocker(ctrl)

	ctx := context.Background()

	// Mock lock acquisition success
	mockLocker.EXPECT().Lock(ctx).Return(nil).Times(1)
	mockLocker.EXPECT().Unlock(ctx).Return(nil).Times(1)

	// Mock DAO returns error
	mockOrderDao.EXPECT().
		AutoConfirmShippedOrders(ctx, consts.SHIPPED, consts.DELIVERED, AUTO_CONFIRM_AFTER_DAYS).
		Return(nil, errors.New("database connection failed")).
		Times(1)

	service := &OrderServiceImpl{
		orderDao:          mockOrderDao,
		distributedLocker: mockLocker,
		syncMode:          true,
	}

	// Execute - should handle error gracefully
	service.OrderAutoConfirm(ctx)
}

// TestOrderServiceImpl_OrderAutoConfirm_NoOrders tests when no orders need auto-confirmation
func TestOrderServiceImpl_OrderAutoConfirm_NoOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockLocker := utilMocks.NewMockLocker(ctrl)
	mockKafkaWriter := utilMocks.NewMockWriter(ctrl)

	ctx := context.Background()

	// Mock lock acquisition success
	mockLocker.EXPECT().Lock(ctx).Return(nil).Times(1)
	mockLocker.EXPECT().Unlock(ctx).Return(nil).Times(1)

	// Mock DAO returns empty list
	mockOrderDao.EXPECT().
		AutoConfirmShippedOrders(ctx, consts.SHIPPED, consts.DELIVERED, AUTO_CONFIRM_AFTER_DAYS).
		Return([]types.OrderNoAndUserId{}, nil).
		Times(1)

	// Kafka message should NOT be sent when no orders
	mockKafkaWriter.EXPECT().SendMsg(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	service := &OrderServiceImpl{
		orderDao:          mockOrderDao,
		messageWriter:     mockKafkaWriter,
		distributedLocker: mockLocker,
		syncMode:          true,
	}

	// Execute
	service.OrderAutoConfirm(ctx)
}

// TestOrderServiceImpl_OrderAutoConfirm_MessageSendError tests Kafka message send failure
func TestOrderServiceImpl_OrderAutoConfirm_MessageSendError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockLocker := utilMocks.NewMockLocker(ctrl)
	mockKafkaWriter := utilMocks.NewMockWriter(ctrl)

	ctx := context.Background()

	// Mock lock acquisition success
	mockLocker.EXPECT().Lock(ctx).Return(nil).Times(1)
	mockLocker.EXPECT().Unlock(ctx).Return(nil).Times(1)

	// Mock DAO returns orders
	autoConfirmedOrders := []types.OrderNoAndUserId{
		{OrderNo: "ORDER001", UserID: 101},
	}
	mockOrderDao.EXPECT().
		AutoConfirmShippedOrders(ctx, consts.SHIPPED, consts.DELIVERED, AUTO_CONFIRM_AFTER_DAYS).
		Return(autoConfirmedOrders, nil).
		Times(1)

	// Mock Kafka message send fails
	mockKafkaWriter.EXPECT().
		SendMsg(ctx, "order_status_changed", "ORDER001", gomock.Any()).
		Return(errors.New("kafka connection failed")).
		Times(1)

	service := &OrderServiceImpl{
		orderDao:          mockOrderDao,
		messageWriter:     mockKafkaWriter,
		distributedLocker: mockLocker,
		syncMode:          true,
	}

	// Execute - should log error but continue
	service.OrderAutoConfirm(ctx)
}

// TestOrderServiceImpl_OrderAutoConfirm_UnlockError tests unlock failure
func TestOrderServiceImpl_OrderAutoConfirm_UnlockError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockLocker := utilMocks.NewMockLocker(ctrl)
	mockKafkaWriter := utilMocks.NewMockWriter(ctrl)

	ctx := context.Background()

	// Mock lock acquisition success
	mockLocker.EXPECT().Lock(ctx).Return(nil).Times(1)
	// Mock unlock failure
	mockLocker.EXPECT().Unlock(ctx).Return(errors.New("unlock failed")).Times(1)

	// Mock DAO returns orders
	autoConfirmedOrders := []types.OrderNoAndUserId{
		{OrderNo: "ORDER001", UserID: 101},
	}
	mockOrderDao.EXPECT().
		AutoConfirmShippedOrders(ctx, consts.SHIPPED, consts.DELIVERED, AUTO_CONFIRM_AFTER_DAYS).
		Return(autoConfirmedOrders, nil).
		Times(1)

	// Mock Kafka message send success
	mockKafkaWriter.EXPECT().
		SendMsg(ctx, "order_status_changed", "ORDER001", gomock.Any()).
		Return(nil).
		Times(1)

	service := &OrderServiceImpl{
		orderDao:          mockOrderDao,
		messageWriter:     mockKafkaWriter,
		distributedLocker: mockLocker,
		syncMode:          true,
	}

	// Execute - should log error but not panic
	service.OrderAutoConfirm(ctx)
}

// TestOrderServiceImpl_OrderAutoConfirm_MultipleOrders tests auto-confirming multiple orders
func TestOrderServiceImpl_OrderAutoConfirm_MultipleOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockLocker := utilMocks.NewMockLocker(ctrl)
	mockKafkaWriter := utilMocks.NewMockWriter(ctrl)

	ctx := context.Background()

	// Mock lock acquisition success
	mockLocker.EXPECT().Lock(ctx).Return(nil).Times(1)
	mockLocker.EXPECT().Unlock(ctx).Return(nil).Times(1)

	// Mock DAO returns 5 orders to auto-confirm
	autoConfirmedOrders := []types.OrderNoAndUserId{
		{OrderNo: "ORDER001", UserID: 101},
		{OrderNo: "ORDER002", UserID: 102},
		{OrderNo: "ORDER003", UserID: 103},
		{OrderNo: "ORDER004", UserID: 104},
		{OrderNo: "ORDER005", UserID: 105},
	}
	mockOrderDao.EXPECT().
		AutoConfirmShippedOrders(ctx, consts.SHIPPED, consts.DELIVERED, AUTO_CONFIRM_AFTER_DAYS).
		Return(autoConfirmedOrders, nil).
		Times(1)

	// Mock Kafka messages for each order
	for _, order := range autoConfirmedOrders {
		mockKafkaWriter.EXPECT().
			SendMsg(ctx, "order_status_changed", order.OrderNo, gomock.Any()).
			Return(nil).
			Times(1)
	}

	service := &OrderServiceImpl{
		orderDao:          mockOrderDao,
		messageWriter:     mockKafkaWriter,
		distributedLocker: mockLocker,
		syncMode:          true,
	}

	// Execute
	service.OrderAutoConfirm(ctx)
}

// TestOrderServiceImpl_OrderAutoConfirm_PartialMessageFailure tests when some Kafka messages fail
func TestOrderServiceImpl_OrderAutoConfirm_PartialMessageFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockLocker := utilMocks.NewMockLocker(ctrl)
	mockKafkaWriter := utilMocks.NewMockWriter(ctrl)

	ctx := context.Background()

	// Mock lock acquisition success
	mockLocker.EXPECT().Lock(ctx).Return(nil).Times(1)
	mockLocker.EXPECT().Unlock(ctx).Return(nil).Times(1)

	// Mock DAO returns 3 orders
	autoConfirmedOrders := []types.OrderNoAndUserId{
		{OrderNo: "ORDER001", UserID: 101},
		{OrderNo: "ORDER002", UserID: 102},
		{OrderNo: "ORDER003", UserID: 103},
	}
	mockOrderDao.EXPECT().
		AutoConfirmShippedOrders(ctx, consts.SHIPPED, consts.DELIVERED, AUTO_CONFIRM_AFTER_DAYS).
		Return(autoConfirmedOrders, nil).
		Times(1)

	// First message succeeds
	mockKafkaWriter.EXPECT().
		SendMsg(ctx, "order_status_changed", "ORDER001", gomock.Any()).
		Return(nil).
		Times(1)

	// Second message fails
	mockKafkaWriter.EXPECT().
		SendMsg(ctx, "order_status_changed", "ORDER002", gomock.Any()).
		Return(errors.New("kafka timeout")).
		Times(1)

	// Third message succeeds
	mockKafkaWriter.EXPECT().
		SendMsg(ctx, "order_status_changed", "ORDER003", gomock.Any()).
		Return(nil).
		Times(1)

	service := &OrderServiceImpl{
		orderDao:          mockOrderDao,
		messageWriter:     mockKafkaWriter,
		distributedLocker: mockLocker,
		syncMode:          true,
	}

	// Execute - should continue processing all orders despite one failure
	service.OrderAutoConfirm(ctx)
}
