package clients

import (
	"sync"

	paymentClient "github.com/NUS-ISS-Agile-Team/ceramicraft-payment-mservice/client"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/config"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-payment-mservice/common/paymentpb"
)

var (
	paymentClientInstance paymentpb.PaymentServiceClient
	paymentClientOnce sync.Once
)

func InitPaymentClient(cfg *config.PaymentClient) {
	paymentClientOnce.Do(func() {
		paymentClientInstance, _ = paymentClient.GetPaymentClient(&paymentClient.GRpcClientConfig{
			Host: cfg.Host,
			Port: cfg.Port,
		})
	})
}

func GetPaymentClient() paymentpb.PaymentServiceClient {
	return paymentClientInstance
}