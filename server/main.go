package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/clients"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/config"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/grpc"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/http"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/log"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/pkg/utils"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/repository"
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
	go grpc.Init(sigCh)
	go http.Init(sigCh)
	// listen terminage signal
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh // Block until signal is received
	log.Logger.Infof("Received signal: %v, shutting down...", sig)
	utils.CloseKafka()
}
