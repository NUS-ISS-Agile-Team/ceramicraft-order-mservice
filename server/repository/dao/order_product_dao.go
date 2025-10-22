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
	CreateBatch(ctx context.Context, products []model.OrderProduct) (rows int, err error)
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

// CreateBatch inserts multiple order product records using GORM's batch insert.
// It returns the number of rows affected and any error encountered.
func (d *OrderProductDaoImpl) CreateBatch(ctx context.Context, products []model.OrderProduct) (rows int, err error) {
	if len(products) == 0 {
		return 0, nil
	}
	result := d.db.WithContext(ctx).Create(&products)
	return int(result.RowsAffected), result.Error
}

func (d *OrderProductDaoImpl) GetByOrderNo(ctx context.Context, orderNo string) (orderProductList []*model.OrderProduct, err error) {
	err = d.db.WithContext(ctx).Where("order_no = ?", orderNo).Find(&orderProductList).Error
	return
}
