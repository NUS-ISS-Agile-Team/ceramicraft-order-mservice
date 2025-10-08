package model

import "time"

type OrderStatusLog struct {
	ID            int       `gorm:"primaryKey;autoIncrement"`
	OrderNo       int       `gorm:"not null;index"`    // 订单号
	UserID        int       `gorm:"int;not null"`      // 关联用户ID
	CurrentStatus int       `gorm:"type:int;not null"` // 当前状态
	Remark        string    `gorm:"type:varchar(256)"` // 备注
	CreateTime    time.Time `gorm:"autoCreateTime"`    // 变更时间
}

// TableName sets the insert table name for this struct type
func (OrderStatusLog) TableName() string {
	return "order_status_logs"
}
