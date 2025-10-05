package dao

import (
	"context"
	"sync"

	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/repository"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/repository/model"
	"gorm.io/gorm"
)

type OrderLogDao interface {
	Create(ctx context.Context, orderLog *model.OrderStatusLog) (id int, err error)
	GetByOrderNo(ctx context.Context, orderNo int) (orderLogList []*model.OrderStatusLog, err error)
}

var (
	orderLogOnce sync.Once
	orderLogDao  *OrderLogDaoImpl
)

type OrderLogDaoImpl struct {
	db *gorm.DB
}

func GetOrderLogDao() *OrderLogDaoImpl {
	orderLogOnce.Do(func() {
		if orderLogDao == nil {
			orderLogDao = &OrderLogDaoImpl{repository.DB}
		}
	})
	return orderLogDao
}

func (d *OrderLogDaoImpl) Create(ctx context.Context, orderLog *model.OrderStatusLog) (id int, err error) {
	result := d.db.WithContext(ctx).Create(orderLog)
	return orderLog.ID, result.Error
}

func (d *OrderLogDaoImpl) GetByOrderNo(ctx context.Context, orderNo int) (orderLogList []*model.OrderStatusLog, err error) {
	err = d.db.WithContext(ctx).Where("order_no = ?", orderNo).Find(&orderLogList).Error
	return
}
