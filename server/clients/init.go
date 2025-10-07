package clients

import "github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/config"

func InitAllClients(cfg *config.Conf) {
	_ = InitProductClient(cfg.CommodityClient)
	InitPaymentClient(cfg.PaymentClient)
}
