package dao

import (
	"context"
	"sync"
	"time"

	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/log"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/pkg/consts"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/pkg/types"
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
	AutoConfirmShippedOrders(ctx context.Context, shippedStatus int, deliveredStatus int, daysThreshold int) (orderNos []types.OrderNoAndUserId, err error)
	GetOrderStats() (types.OrderStats, error)
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
			"status":        status,
			"delivery_time": t,
			"logistics_no":  shippingNo,
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

// AutoConfirmShippedOrders 自动确认已发货超过指定天数的订单
// 查询 status = shippedStatus 且 delivery_time 距离当前时间大于 daysThreshold 天的订单
// 将它们的状态更新为 deliveredStatus，并返回更新成功的订单号列表
func (d *OrderDaoImpl) AutoConfirmShippedOrders(ctx context.Context, shippedStatus int, deliveredStatus int, daysThreshold int) (orderNos []types.OrderNoAndUserId, err error) {
	// 1. 计算截止时间：当前时间 - daysThreshold 天
	thresholdTime := time.Now().Add(-time.Duration(daysThreshold) * 24 * time.Hour)

	// 2. 查询符合条件的订单
	var orders []*model.Order
	err = d.db.WithContext(ctx).
		Model(&model.Order{}).
		Where("status = ?", shippedStatus).
		Where("delivery_time IS NOT NULL").
		Where("delivery_time <= ?", thresholdTime).
		Find(&orders).Error
	if err != nil {
		return nil, err
	}

	// 如果没有订单需要确认，直接返回
	if len(orders) == 0 {
		return nil, nil
	}

	// 3. 提取订单号列表
	orderNosAndUserIDs := make([]types.OrderNoAndUserId, 0, len(orders))
	orderNo := make([]string, 0, len(orders))
	for _, order := range orders {
		orderNosAndUserIDs = append(orderNosAndUserIDs, types.OrderNoAndUserId{
			OrderNo: order.OrderNo,
			UserID:  order.UserID,
		})
		orderNo = append(orderNo, order.OrderNo)
	}

	// 4. 批量更新订单状态
	now := time.Now()
	err = d.db.WithContext(ctx).
		Model(&model.Order{}).
		Where("order_no IN ?", orderNo).
		Updates(map[string]interface{}{
			"status":       deliveredStatus,
			"confirm_time": now,
		}).Error
	if err != nil {
		return nil, err
	}

	return orderNosAndUserIDs, nil
}

func (d *OrderDaoImpl) GetOrderStats() (types.OrderStats, error) {
	var stats types.OrderStats
	err := d.db.WithContext(context.Background()).
		Model(&model.Order{}).
		Select([]string{
			"COUNT(order_no) AS total_orders",
			"sum(total_amount) as total_sales",
			"count(distinct user_id) as total_customers",
		}).Where("status in (?)", []int{consts.DELIVERED, consts.PAYED, consts.SHIPPED}).
		Scan(&stats).Error
	if err != nil {
		log.Logger.Errorf("Failed to get order stats: %v", err)
	}
	return stats, err
}
