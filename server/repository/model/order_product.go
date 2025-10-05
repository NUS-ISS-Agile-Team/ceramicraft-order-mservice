package model

import "time"

type OrderProduct struct {
	ID          int       `gorm:"primaryKey;autoIncrement"`
	OrderNo     int       `gorm:"not null;index"`             // 订单号
	ProductID   int       `gorm:"not null"`                   // 商品ID
	ProductName string    `gorm:"type:varchar(128);not null"` // 商品名称
	Price       int       `gorm:"type:int;not null"`          // 商品单价
	Quantity    int       `gorm:"not null"`                   // 商品数量
	TotalPrice  int       `gorm:"type:int;not null"`          // 商品总价
	CreateTime  time.Time `gorm:"autoCreateTime"`             // 创建时间
	UpdateTime  time.Time `gorm:"autoUpdateTime"`             // 更新时间
}

// TableName sets the insert table name for this struct type
func (OrderProduct) TableName() string {
	return "order_products"
}
