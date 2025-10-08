package dao

import (
	"context"
	"sync"
	"time"

	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/repository"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/repository/model"
	"gorm.io/gorm"
)

type OrderDao interface {
	Create(ctx context.Context, o *model.Order) (orderNo string, err error)
	UpdateStatusAndPayment(ctx context.Context, orderNo string, status int, payTime time.Time) error
	GetByOrderNo(ctx context.Context, orderNo string) (o *model.Order, err error)
	GetByOrderQuery(ctx context.Context, query OrderQuery) (oList []*model.Order, err error)
}

var (
	orderOnce            sync.Once
	orderDaoImplInstance *OrderDaoImpl
)

type OrderDaoImpl struct {
	db *gorm.DB
}

func GetOrderDao() *OrderDaoImpl {
	orderOnce.Do(func() {
		if orderDaoImplInstance == nil {
			orderDaoImplInstance = &OrderDaoImpl{repository.DB}
		}
	})
	return orderDaoImplInstance
}

func (d *OrderDaoImpl) Create(ctx context.Context, o *model.Order) (orderNo string, err error) {
	result := d.db.WithContext(ctx).Create(o)
	return o.OrderNo, result.Error
}

func (d *OrderDaoImpl) UpdateStatusAndPayment(ctx context.Context, orderNo string, status int, payTime time.Time) error {
	return d.db.WithContext(ctx).
		Model(&model.Order{}).
		Where("order_no = ?", orderNo).
		Updates(map[string]interface{}{
			"status":   status,
			"pay_time": payTime,
		}).Error
}

func (d *OrderDaoImpl) GetByOrderNo(ctx context.Context, orderNo string) (o *model.Order, err error) {
	o = &model.Order{}
	err = d.db.WithContext(ctx).Where("order_no = ?", orderNo).First(o).Error
	return
}

func (d *OrderDaoImpl) GetByOrderQuery(ctx context.Context, query OrderQuery) (oList []*model.Order, err error) {
	db := d.db.WithContext(ctx).Model(&model.Order{})
	// 这里可以根据 query 字段动态拼接条件
	if query.OrderStatus != 0 {
		db = db.Where("status = ?", query.OrderStatus)
	}
	if query.UserID != 0 {
		db = db.Where("user_id = ?", query.UserID)
	}
	err = db.Find(&oList).Error
	return
}
