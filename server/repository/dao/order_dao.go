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
	UpdateStatusAndConfirmTime(ctx context.Context, orderNo string, status int, t time.Time) (err error)
	UpdateStatusWithDeliveryInfo(ctx context.Context, orderNo string, status int, t time.Time, shippingNo string) (err error)
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

func (d *OrderDaoImpl) UpdateStatusAndConfirmTime(ctx context.Context, orderNo string, status int, t time.Time) (err error) {
	return d.db.WithContext(ctx).
		Model(&model.Order{}).
		Where("order_no = ?", orderNo).
		Updates(map[string]interface{}{
			"status":       status,
			"confirm_time": t,
		}).Error
}

func (d *OrderDaoImpl) UpdateStatusWithDeliveryInfo(ctx context.Context, orderNo string, status int, t time.Time, shippingNo string) (err error) {
	return d.db.WithContext(ctx).
		Model(&model.Order{}).
		Where("order_no = ?", orderNo).
		Updates(map[string]interface{}{
			"status":       status,
			"delivery_time": t,
			"logistics_no": shippingNo,
		}).Error
}

func (d *OrderDaoImpl) GetByOrderNo(ctx context.Context, orderNo string) (o *model.Order, err error) {
	o = &model.Order{}
	err = d.db.WithContext(ctx).Where("order_no = ?", orderNo).First(o).Error
	return
}

func (d *OrderDaoImpl) GetByOrderQuery(ctx context.Context, query OrderQuery) (oList []*model.Order, err error) {
	db := d.db.WithContext(ctx).Model(&model.Order{})

	// 根据 query 字段动态拼接条件
	if query.OrderStatus != 0 {
		db = db.Where("status = ?", query.OrderStatus)
	}
	if query.UserID != 0 {
		db = db.Where("user_id = ?", query.UserID)
	}
	if query.OrderNo != "" {
		db = db.Where("order_no LIKE ?", "%"+query.OrderNo+"%")
	}

	// 根据创建时间范围筛选
	if !query.StartTime.IsZero() {
		db = db.Where("create_time >= ?", query.StartTime)
	}
	if !query.EndTime.IsZero() {
		db = db.Where("create_time <= ?", query.EndTime)
	}

	// 按创建时间倒序排列
	db = db.Order("create_time DESC")

	// 分页支持
	if query.Limit > 0 {
		db = db.Limit(query.Limit)
	}
	if query.Offset > 0 {
		db = db.Offset(query.Offset)
	}

	err = db.Find(&oList).Error
	return
}
