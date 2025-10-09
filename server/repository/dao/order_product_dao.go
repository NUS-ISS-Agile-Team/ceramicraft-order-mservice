package dao

import (
	"context"
	"sync"

	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/repository"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/repository/model"
	"gorm.io/gorm"
)

type OrderProductDao interface {
	Create(ctx context.Context, orderProduct *model.OrderProduct) (id int, err error)
	GetByOrderNo(ctx context.Context, orderNo string) (orderProductList []*model.OrderProduct, err error)
}

var (
	orderProductOnce            sync.Once
	orderProductDaoImplInstance *OrderProductDaoImpl
)

type OrderProductDaoImpl struct {
	db *gorm.DB
}

func GetOrderProductDao() *OrderProductDaoImpl {
	orderProductOnce.Do(func() {
		if orderProductDaoImplInstance == nil {
			orderProductDaoImplInstance = &OrderProductDaoImpl{repository.DB}
		}
	})
	return orderProductDaoImplInstance
}

func (d *OrderProductDaoImpl) Create(ctx context.Context, orderProduct *model.OrderProduct) (id int, err error) {
	result := d.db.WithContext(ctx).Create(orderProduct)
	return orderProduct.ID, result.Error
}

func (d *OrderProductDaoImpl) GetByOrderNo(ctx context.Context, orderNo string) (orderProductList []*model.OrderProduct, err error) {
	err = d.db.WithContext(ctx).Where("order_no = ?", orderNo).Find(&orderProductList).Error
	return
}
