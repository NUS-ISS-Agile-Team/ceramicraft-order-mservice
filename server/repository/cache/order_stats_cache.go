package cache

import (
	"sync"
	"time"

	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/log"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/pkg/types"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/repository/dao"
)

type IOrderStatsCache interface {
	GetOrderStats() (types.OrderStats, error)
}

type orderStatsCache struct {
	data     *types.OrderStats
	orderDao dao.OrderDao
}

var (
	orderStatsCacheInstance IOrderStatsCache
	orderStatsCacheSyncOnce sync.Once
)

func GetOrderStatsCache() IOrderStatsCache {
	orderStatsCacheSyncOnce.Do(func() {
		orderStatsCache := &orderStatsCache{
			orderDao: dao.GetOrderDao(),
		}
		err := orderStatsCache.loadOrderStats()
		if err != nil {
			panic(err)
		}
		go orderStatsCache.startLoadDataSchedule()
		orderStatsCacheInstance = orderStatsCache
	})
	return orderStatsCacheInstance
}

// GetOrderStats implements IOrderStatsCache.
func (o *orderStatsCache) GetOrderStats() (types.OrderStats, error) {
	if o.data == nil {
		return o.orderDao.GetOrderStats()
	}
	return *o.data, nil
}

func (o *orderStatsCache) startLoadDataSchedule() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		err := o.loadOrderStats()
		if err != nil {
			log.Logger.Errorf("failed to reload order stats: %v", err)
		}
	}
}

func (o *orderStatsCache) loadOrderStats() error {
	stats, err := o.orderDao.GetOrderStats()
	if err != nil {
		log.Logger.Errorf("failed to load order stats: %v", err)
		return err
	}
	if stats.TotalOrders > 0 {
		stats.AvgSalesPerOrder = stats.TotalSales / stats.TotalOrders
	}
	o.data = &stats
	log.Logger.Infof("successfully loaded order stats: %+v", stats)
	return nil
}
