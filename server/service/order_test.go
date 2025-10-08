package service

import (
	"context"
	"errors"

	// "errors"
	"testing"

	"github.com/NUS-ISS-Agile-Team/ceramicraft-commodity-mservice/common/productpb"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/clients/mocks"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/log"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/pkg/consts"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/pkg/types"
	utilMocks "github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/pkg/utils/mocks"
	daoMocks "github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/repository/dao/mocks"
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
	ctx := context.WithValue(context.Background(), "userID", 123)
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

	// Mock order product DAO - successful creation
	mockOrderProductDao.EXPECT().
		Create(ctx, gomock.Any()).
		Return(1, nil).
		Times(1)

	// Mock Kafka messages
	mockKafkaWriter.EXPECT().
		SendMsg(ctx, "order_created", gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	mockKafkaWriter.EXPECT().
		SendMsg(ctx, "order_status_changed", gomock.Any(), "1").
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
		SendMsg(ctx, "order_status_changed", gomock.Any(), "2").
		Return(nil).
		AnyTimes()

	// Create service instance with all mocks
	service := &OrderServiceImpl{
		orderDao:             mockOrderDao,
		orderProductDao:      mockOrderProductDao,
		productServiceClient: mockProductClient,
		paymentServiceClient: mockPaymentClient,
		messageWriter:        mockKafkaWriter,
	}

	// Now we can actually test the CreateOrder method
	orderNo, err := service.CreateOrder(ctx, orderInfo)
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
	ctx := context.WithValue(context.Background(), "userID", 123)
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
	}

	// Test the CreateOrder method
	orderNo, err := service.CreateOrder(ctx, orderInfo)
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
	ctx := context.WithValue(context.Background(), "userID", 123)
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
	}

	// Test the CreateOrder method
	orderNo, err := service.CreateOrder(ctx, orderInfo)
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
	ctx := context.WithValue(context.Background(), "userID", 123)
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
	}

	// Test the CreateOrder method
	orderNo, err := service.CreateOrder(ctx, orderInfo)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if orderNo != "" {
		t.Errorf("Expected empty orderNo, got: %s", orderNo)
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
	ctx := context.WithValue(context.Background(), "userID", 123)
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

	// Mock order product DAO - return error
	mockOrderProductDao.EXPECT().
		Create(ctx, gomock.Any()).
		Return(0, errors.New("failed to create order product")).
		Times(1)

	// Create service instance with mocks
	service := &OrderServiceImpl{
		orderDao:             mockOrderDao,
		orderProductDao:      mockOrderProductDao,
		productServiceClient: mockProductClient,
		paymentServiceClient: mockPaymentClient,
		messageWriter:        mockKafkaWriter,
	}

	// Test the CreateOrder method
	orderNo, err := service.CreateOrder(ctx, orderInfo)
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
	ctx := context.WithValue(context.Background(), "userID", 123)
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

	// Mock order product DAO - successful creation
	mockOrderProductDao.EXPECT().
		Create(ctx, gomock.Any()).
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
	}

	// Test the CreateOrder method
	orderNo, err := service.CreateOrder(ctx, orderInfo)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if orderNo != "" {
		t.Errorf("Expected empty orderNo, got: %s", orderNo)
	}
}

func TestOrderServiceImpl_CreateOrder_PaymentError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockOrderProductDao := daoMocks.NewMockOrderProductDao(ctrl)
	mockProductClient := mocks.NewMockProductServiceClient(ctrl)
	mockPaymentClient := mocks.NewMockPaymentServiceClient(ctrl)
	mockKafkaWriter := utilMocks.NewMockWriter(ctrl)

	// Setup test data
	ctx := context.WithValue(context.Background(), "userID", 123)
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
		Create(ctx, gomock.Any()).
		Return(1, nil).
		Times(1)

	// Mock Kafka messages
	mockKafkaWriter.EXPECT().
		SendMsg(ctx, "order_created", gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	mockKafkaWriter.EXPECT().
		SendMsg(ctx, "order_status_changed", gomock.Any(), "1").
		Return(nil).
		AnyTimes()

	// Mock product stock update
	mockProductClient.EXPECT().
		UpdateStockWithCAS(ctx, gomock.Any()).
		Return(&productpb.UpdateStockWithCASResponse{}, nil).
		Times(1)

	// Mock payment service - return error
	mockPaymentClient.EXPECT().
		PayOrder(ctx, gomock.Any()).
		Return(nil, errors.New("payment service unavailable")).
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
	}

	// Test the CreateOrder method
	orderNo, err := service.CreateOrder(ctx, orderInfo)
	if err != nil {
		t.Errorf("Expected no error")
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
	ctx := context.WithValue(context.Background(), "userID", 123)
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
		Create(ctx, gomock.Any()).
		Return(1, nil).
		Times(1)

	// Mock Kafka messages
	mockKafkaWriter.EXPECT().
		SendMsg(ctx, "order_created", gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	mockKafkaWriter.EXPECT().
		SendMsg(ctx, "order_status_changed", gomock.Any(), "1").
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
	}

	// Test the CreateOrder method
	orderNo, err := service.CreateOrder(ctx, orderInfo)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if orderNo != "" {
		t.Errorf("Expected empty orderNo, got: %s", orderNo)
	}
}

func TestOrderServiceImpl_CreateOrder_UpdateStatusError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockOrderDao := daoMocks.NewMockOrderDao(ctrl)
	mockOrderProductDao := daoMocks.NewMockOrderProductDao(ctrl)
	mockProductClient := mocks.NewMockProductServiceClient(ctrl)
	mockPaymentClient := mocks.NewMockPaymentServiceClient(ctrl)
	mockKafkaWriter := utilMocks.NewMockWriter(ctrl)

	// Setup test data
	ctx := context.WithValue(context.Background(), "userID", 123)
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
		Create(ctx, gomock.Any()).
		Return(1, nil).
		Times(1)

	// Mock Kafka messages
	mockKafkaWriter.EXPECT().
		SendMsg(ctx, "order_created", gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	mockKafkaWriter.EXPECT().
		SendMsg(ctx, "order_status_changed", gomock.Any(), "1").
		Return(nil).
		AnyTimes()

	// Mock product stock update
	mockProductClient.EXPECT().
		UpdateStockWithCAS(ctx, gomock.Any()).
		Return(&productpb.UpdateStockWithCASResponse{}, nil).
		Times(1)

	// Mock payment service - successful payment
	mockPaymentClient.EXPECT().
		PayOrder(ctx, gomock.Any()).
		Return(&paymentpb.PayOrderResponse{
			Code: 0, // Success
		}, nil).
		Times(1)

	// Mock order status update error
	mockOrderDao.EXPECT().
		UpdateStatusAndPayment(ctx, gomock.Any(), consts.PAYED, gomock.Any()).
		Return(errors.New("failed to update order status")).
		Times(1)

	// Create service instance with mocks
	service := &OrderServiceImpl{
		orderDao:             mockOrderDao,
		orderProductDao:      mockOrderProductDao,
		productServiceClient: mockProductClient,
		paymentServiceClient: mockPaymentClient,
		messageWriter:        mockKafkaWriter,
	}

	// Test the CreateOrder method
	orderNo, err := service.CreateOrder(ctx, orderInfo)
	if err != nil {
		t.Errorf("Expected no error")
	}
	if orderNo != "" {
		t.Errorf("Expected empty orderNo, got: %s", orderNo)
	}
}
