package router

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	_ "github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/docs"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/http/api"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/metrics"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-user-mservice/common/middleware"
	swaggerFiles "github.com/swaggo/files"
	gs "github.com/swaggo/gin-swagger"
)

const (
	serviceURIPrefix = "/order-ms/v1"
)

func NewRouter() *gin.Engine {
	r := gin.Default()

	basicGroup := r.Group(serviceURIPrefix)
	{
		basicGroup.Use(metrics.MetricsMiddleware())
		basicGroup.GET("/metrics", gin.WrapH(promhttp.Handler()))

		basicGroup.GET("/swagger/*any", gs.WrapHandler(
			swaggerFiles.Handler,
			gs.URL("/order-ms/v1/swagger/doc.json"),
		))
		basicGroup.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong",
			})
		})

		merchantGroup := basicGroup.Group("/merchant")
		{
			merchantGroup.Use(middleware.AuthMiddleware())
			merchantGroup.POST("/orders/list", api.ListOrders)
			merchantGroup.GET("/orders/:order_no", api.GetOrderDetail) // get order detail
			merchantGroup.PATCH("/orders/:order_no/ship", api.ShipOrder) // ship order
		}

		customerGroup := basicGroup.Group("/customer")
		{
			customerGroup.Use(middleware.AuthMiddleware())
			customerGroup.POST("/orders", api.CreateOrder) // create order
			customerGroup.POST("/orders/list", api.CustomerListOrders)
			customerGroup.GET("/orders/:order_no", api.CustomerGetOrderDetail) // get order detail
			customerGroup.PATCH("/orders/:order_no/confirm", api.ConfirmOrder) // confirm order
		}
	}
	return r
}
