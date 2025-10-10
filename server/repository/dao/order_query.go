package dao

import "time"

type OrderQuery struct {
	UserID      int       // 用户ID筛选
	OrderStatus int       // 订单状态筛选
	StartTime   time.Time // 创建时间开始范围
	EndTime     time.Time // 创建时间结束范围
	OrderNo     string    // 订单号筛选
	Limit       int       // 分页限制
	Offset      int       // 分页偏移
}
