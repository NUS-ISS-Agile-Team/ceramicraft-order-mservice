package clients

import (
	"sync"

	"github.com/NUS-ISS-Agile-Team/ceramicraft-commodity-mservice/client"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-commodity-mservice/common/productpb"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/config"
)

var (
	productClientInstance productpb.ProductServiceClient
	productClientOnce   sync.Once
)

func InitProductClient(cfg *config.CommodityClient) productpb.ProductServiceClient {
	productClientOnce.Do(func() {
		productClientInstance, _ = client.GetProductServiceClient(&client.GRpcClientConfig{
			Host: cfg.Host,
			Port: cfg.Port,
		})
	})
	return productClientInstance
}

func GetProductClient() productpb.ProductServiceClient {
	return productClientInstance
}
