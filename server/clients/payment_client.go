package clients

import (
	"sync"

	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/config"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/log"
	paymentClient "github.com/NUS-ISS-Agile-Team/ceramicraft-payment-mservice/client"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-payment-mservice/common/paymentpb"
)

var (
	paymentClientInstance paymentpb.PaymentServiceClient
	paymentClientOnce sync.Once
)

func InitPaymentClient(cfg *config.PaymentClient) {
	paymentClientOnce.Do(func() {
		var err error
		paymentClientInstance, err = paymentClient.GetPaymentClient(&paymentClient.GRpcClientConfig{
			Host: cfg.Host,
			Port: cfg.Port,
		})
		if err != nil {
			log.Logger.Errorf("InitPaymentClient: init failed, err %s", err.Error())
		}
		log.Logger.Infoln("InitPaymentClient: success")
	})
}

func GetPaymentClient() paymentpb.PaymentServiceClient {
	return paymentClientInstance
}