package router

import (
	"github.com/gin-gonic/gin"

	_ "github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/docs"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/http/api"
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
		}

		customerGroup := basicGroup.Group("/customer")
		{
			customerGroup.Use(middleware.AuthMiddleware())
			customerGroup.POST("/create", api.CreateOrder)
		}
	}
	return r
}
