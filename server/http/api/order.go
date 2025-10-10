package api

import (
	"errors"
	"net/http"

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
// @Router /customer/create [post]
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
// @Router /merchant/list [post]
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
	if req.Limit == 0 {
		req.Limit = 10
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
// @Router /merchant/detail/{order_no} [get]
func GetOrderDetail(ctx *gin.Context) {
	orderNo := ctx.Param("order_no")
	if orderNo == "" {
		ctx.JSON(http.StatusBadRequest, RespError(ctx, errors.New("订单号不能为空")))
		return
	}

	detail, err := service.GetOrderServiceInstance().GetOrderDetail(ctx, orderNo)
	if err != nil {
		// 可以根据具体错误类型返回不同的状态码
		if err.Error() == "record not found" {
			ctx.JSON(http.StatusNotFound, RespError(ctx, errors.New("订单不存在")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, RespError(ctx, err))
		return
	}

	ctx.JSON(http.StatusOK, RespSuccess(ctx, detail))
}
