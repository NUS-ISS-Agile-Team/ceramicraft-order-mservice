package api

import (
	"errors"
	"net/http"

	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/pkg/consts"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/pkg/types"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/service"
	"github.com/gin-gonic/gin"
)

// CreateOrder godoc
// @Summary 创建订单
// @Description 创建一个新订单
// @Tags Order
// @Accept json
// @Produce json
// @Param order body types.OrderInfo true "订单信息"
// @Success 200 {object} Response
// @Failure 500 {object} Response
// @Router /customer/orders [post]
func CreateOrder(ctx *gin.Context) {
	var req types.OrderInfo
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, RespError(ctx, err))
		return
	}

	orderNo, err := service.GetOrderServiceInstance().CreateOrder(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, RespError(ctx, err))
		return
	}

	ctx.JSON(http.StatusOK, RespSuccess(ctx, orderNo))
}

// ListOrders godoc
// @Summary 查询订单列表
// @Description 根据条件查询订单列表，支持分页
// @Tags Order
// @Accept json
// @Produce json
// @Param request body types.ListOrderRequest true "查询条件"
// @Success 200 {object} Response{data=types.ListOrderResponse}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /merchant/orders/list [post]
func ListOrders(ctx *gin.Context) {
	var req types.ListOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, RespError(ctx, err))
		return
	}

	// 设置默认分页参数
	if req.Limit <= 0 {
		req.Limit = 20 // 默认每页20条
	}
	if req.Limit > 100 {
		req.Limit = 100 // 最大每页100条
	}

	resp, err := service.GetOrderServiceInstance().ListOrders(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, RespError(ctx, err))
		return
	}

	ctx.JSON(http.StatusOK, RespSuccess(ctx, resp))
}

// GetOrderDetail godoc
// @Summary 查询订单详情
// @Description 根据订单号查询订单详情，包括订单基本信息、商品列表和状态日志
// @Tags Order
// @Accept json
// @Produce json
// @Param order_no path string true "订单号"
// @Success 200 {object} Response{data=types.OrderDetail}
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /merchant/orders/{order_no} [get]
func GetOrderDetail(ctx *gin.Context) {
	orderNo := ctx.Param("order_no")
	if orderNo == "" {
		ctx.JSON(http.StatusBadRequest, RespError(ctx, errors.New("订单号不能为空")))
		return
	}

	detail, err := service.GetOrderServiceInstance().GetOrderDetail(ctx, orderNo)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, RespError(ctx, err))
		return
	}

	ctx.JSON(http.StatusOK, RespSuccess(ctx, detail))
}

// CustomerListOrders godoc
// @Summary 用户侧查询订单列表
// @Description 根据userID查询订单列表，支持分页，支持根据时间筛选
// @Tags Order
// @Accept json
// @Produce json
// @Param request body types.CustomerListOrderRequest true "查询条件"
// @Success 200 {object} Response{data=types.ListOrderResponse}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /customer/orders/list [post]
func CustomerListOrders(ctx *gin.Context) {
	var req types.CustomerListOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, RespError(ctx, err))
		return
	}

	// 设置默认分页参数
	if req.Limit <= 0 {
		req.Limit = 20 // 默认每页20条
	}
	if req.Limit > 100 {
		req.Limit = 100 // 最大每页100条
	}

	userID := ctx.Value("userID").(int)
	resp, err := service.GetOrderServiceInstance().ListOrders(ctx, types.ListOrderRequest{
		UserID:    userID,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Limit:     req.Limit,
		Offset:    req.Offset,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, RespError(ctx, err))
		return
	}

	ctx.JSON(http.StatusOK, RespSuccess(ctx, resp))
}

// CustomerGetOrderDetail godoc
// @Summary 用户侧查询订单详情
// @Description 根据订单号查询订单详情，包括订单基本信息、商品列表和状态日志
// @Tags Order
// @Accept json
// @Produce json
// @Param order_no path string true "订单号"
// @Success 200 {object} Response{data=types.OrderDetail}
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /customer/orders/{order_no} [get]
func CustomerGetOrderDetail(ctx *gin.Context) {
	orderNo := ctx.Param("order_no")
	if orderNo == "" {
		ctx.JSON(http.StatusBadRequest, RespError(ctx, errors.New("订单号不能为空")))
		return
	}

	userID := ctx.Value("userID").(int)
	detail, err := service.GetOrderServiceInstance().CustomerGetOrderDetail(ctx, orderNo, userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, RespError(ctx, err))
		return
	}

	ctx.JSON(http.StatusOK, RespSuccess(ctx, detail))
}

// ShipOrder godoc
// @Summary 商家发货
// @Description 商家标记订单为已发货状态，并添加物流单号
// @Tags Order
// @Accept json
// @Produce json
// @Param request body types.ShipOrderRequest true "发货信息"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /merchant/orders/{order_no}/ship [patch]
func ShipOrder(ctx *gin.Context) {
	orderNo := ctx.Param("order_no")
	if orderNo == "" {
		ctx.JSON(http.StatusBadRequest, RespError(ctx, errors.New("订单号不能为空")))
		return
	}

	var req types.ShipOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, RespError(ctx, err))
		return
	}

	if req.TrackingNo == "" {
		ctx.JSON(http.StatusBadRequest, RespError(ctx, errors.New("物流单号不能为空")))
		return
	}

	// 调用 service 层更新订单状态为已发货
	err := service.GetOrderServiceInstance().UpdateOrderStatus(ctx, orderNo, consts.SHIPPED, req.TrackingNo) // 3 表示 SHIPPED
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, RespError(ctx, err))
		return
	}

	ctx.JSON(http.StatusOK, RespSuccess(ctx, "订单发货成功"))
}

// ConfirmOrder godoc
// @Summary 用户确认收货
// @Description 用户确认收到商品，订单状态变更为已收货
// @Tags Order
// @Accept json
// @Produce json
// @Param request body types.ConfirmOrderRequest true "确认收货信息"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /customer/orders/{order_no}/confirm [patch]
func ConfirmOrder(ctx *gin.Context) {
	orderNo := ctx.Param("order_no")
	if orderNo == "" {
		ctx.JSON(http.StatusBadRequest, RespError(ctx, errors.New("订单号不能为空")))
		return
	}

	// 调用 service 层更新订单状态为已收货
	err := service.GetOrderServiceInstance().UpdateOrderStatus(ctx, orderNo, consts.DELIVERED, "")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, RespError(ctx, err))
		return
	}

	ctx.JSON(http.StatusOK, RespSuccess(ctx, "确认收货成功"))
}
