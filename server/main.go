package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/clients"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/config"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/grpc"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/http"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/log"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/metrics"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/pkg/utils"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/repository"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/service"
	userUtils "github.com/NUS-ISS-Agile-Team/ceramicraft-user-mservice/common/utils"
)

var (
	sigCh = make(chan os.Signal, 1)
)

// @title       订单服务 API
// @version     1.0
// @description 订单微服务相关接口
// @BasePath    /order-ms/v1
func main() {
	config.Init()
	log.InitLogger()
	repository.Init()
	userUtils.InitJwtSecret()
	utils.InitKafka()
	clients.InitAllClients(config.Config)
	metrics.RegisterMetrics()
	go grpc.Init(sigCh)
	go http.Init(sigCh)
	go utils.GetReader().ConsumeMessage(context.Background())
	startAutoConfirmJob(context.Background(), service.GetOrderServiceInstance())
	// listen terminage signal
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh // Block until signal is received
	log.Logger.Infof("Received signal: %v, shutting down...", sig)
	utils.CloseKafka()
}

func startAutoConfirmJob(ctx context.Context, orderService *service.OrderServiceImpl) {
    timer := utils.NewMyTimer(30 * time.Second)
    
    // 创建一个包装函数
    task := func() {
        orderService.OrderAutoConfirm(ctx)
    }
    
    go timer.Start(ctx, task)
    
    log.Logger.Info("Auto confirm job started")
}
