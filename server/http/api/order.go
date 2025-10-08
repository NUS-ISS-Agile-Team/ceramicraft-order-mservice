package api

import (
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