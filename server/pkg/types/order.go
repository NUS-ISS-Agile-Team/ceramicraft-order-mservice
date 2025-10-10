package types

import "time"

// OrderInfo service layer input
type OrderInfo struct {
	ReceiverFirstName string           `json:"receiver_first_name"` // 收货人姓名
	ReceiverLastName  string           `json:"receiver_last_name"`  // 收货人姓名
	ReceiverPhone     string           `json:"receiver_phone"`      // 收货人电话
	ReceiverAddress   string           `json:"receiver_address"`    // 收货地址
	ReceiverCountry   string           `json:"receiver_country"`    // 收货人国家
	ReceiverZipCode   int              `json:"receiver_zip_code"`   // 收货人邮政编码
	Remark            string           `json:"remark"`              // 备注
	OrderItemList     []*OrderItemInfo `json:"order_item_list"`     // 订单商品列表
}

type OrderItemInfo struct {
	ProductID   int    `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
	Price       int    `json:"price"`
}

type OrderMessage struct {
	UserID            int              `json:"user_id"`             // 下单用户
	OrderID           string           `json:"order_id"`            // 订单ID
	ReceiverFirstName string           `json:"receiver_first_name"` // 收货人姓名
	ReceiverLastName  string           `json:"receiver_last_name"`  // 收货人姓名
	ReceiverPhone     string           `json:"receiver_phone"`      // 收货人电话
	ReceiverAddress   string           `json:"receiver_address"`    // 收货地址
	ReceiverCountry   string           `json:"receiver_country"`    // 收货人国家
	ReceiverZipCode   int              `json:"receiver_zip_code"`   // 收货人邮政编码
	Remark            string           `json:"remark"`              // 备注
	OrderItemList     []*OrderItemInfo `json:"order_item_list"`
}

type OrderStatusChangedMessage struct {
	OrderNo       string `json:"order_no"`
	UserId        int    `json:"user_id"`
	CurrentStatus int    `json:"current_status"`
	Remark        string `json:"remark"`
}

// list order
type OrderInfoInList struct {
	OrderNo           string    `json:"order_no"`
	ReceiverFirstName string    `json:"receiver_first_name"` // 收货人姓名
	ReceiverLastName  string    `json:"receiver_last_name"`  // 收货人姓名
	ReceiverPhone     string    `json:"receiver_phone"`      // 收货人电话
	CreateTime        time.Time `json:"create_time"`
	TotalAmount       int       `json:"total_amount"`
	Status            string    `json:"status"`
}

type ListOrderRequest struct {
	UserID      int       `json:"user_id"`      // 用户ID筛选
	OrderStatus int       `json:"order_status"` // 订单状态筛选
	StartTime   time.Time `json:"start_time"`   // 创建时间开始范围
	EndTime     time.Time `json:"end_time"`     // 创建时间结束范围
	OrderNo     string    `json:"order_no"`     // 订单号筛选
	Limit       int       `json:"limit"`        // 分页限制
	Offset      int       `json:"offset"`       // 分页偏移
}

type ListOrderResponse struct {
	Orders []*OrderInfoInList `json:"orders"`
	Total  int                `json:"total"`
}

type OrderDetail struct {
    // 基本订单信息
    OrderNo           string     `json:"order_no"`            // 订单编号
    UserID            int        `json:"user_id"`             // 下单用户
    Status            int        `json:"status"`              // 订单状态
    StatusName        string     `json:"status_name"`         // 订单状态名称
    TotalAmount       int        `json:"total_amount"`        // 总金额
    PayAmount         int        `json:"pay_amount"`          // 实际支付金额
    ShippingFee       int        `json:"shipping_fee"`        // 运费
    Tax               int        `json:"tax"`                 // 税费
    PayTime           time.Time `json:"pay_time"`            // 支付时间
    CreateTime        time.Time  `json:"create_time"`         // 创建时间
    UpdateTime        time.Time  `json:"update_time"`         // 更新时间
    DeliveryTime      time.Time `json:"delivery_time"`       // 发货时间
    ConfirmTime       time.Time `json:"confirm_time"`        // 收货确认时间
    
    // 收货信息
    ReceiverFirstName string `json:"receiver_first_name"` // 收货人姓名
    ReceiverLastName  string `json:"receiver_last_name"`  // 收货人姓名
    ReceiverPhone     string `json:"receiver_phone"`      // 收货人电话
    ReceiverAddress   string `json:"receiver_address"`    // 收货地址
    ReceiverCountry   string `json:"receiver_country"`    // 收货人国家
    ReceiverZipCode   int    `json:"receiver_zip_code"`   // 收货人邮政编码
    
    // 其他信息
    Remark      string `json:"remark"`       // 备注
    LogisticsNo string `json:"logistics_no"` // 物流单号
    
    // 订单商品列表
    OrderItems []*OrderItemDetail `json:"order_items"`
    
    // 订单状态变更日志
    StatusLogs []*OrderStatusLogDetail `json:"status_logs"`
}

type OrderItemDetail struct {
    ID          int       `json:"id"`           // 订单商品ID
    ProductID   int       `json:"product_id"`   // 商品ID
    ProductName string    `json:"product_name"` // 商品名称
    Price       int       `json:"price"`        // 商品单价
    Quantity    int       `json:"quantity"`     // 商品数量
    TotalPrice  int       `json:"total_price"`  // 商品总价
    CreateTime  time.Time `json:"create_time"`  // 创建时间
    UpdateTime  time.Time `json:"update_time"`  // 更新时间
}

type OrderStatusLogDetail struct {
    ID            int       `json:"id"`             // 日志ID
    CurrentStatus int       `json:"current_status"` // 当前状态
    StatusName    string    `json:"status_name"`    // 状态名称
    Remark        string    `json:"remark"`         // 备注
    CreateTime    time.Time `json:"create_time"`    // 变更时间
}

type CustomerListOrderRequest struct {
	StartTime   time.Time `json:"start_time"`   // 创建时间开始范围
	EndTime     time.Time `json:"end_time"`     // 创建时间结束范围
	Limit       int       `json:"limit"`        // 分页限制
	Offset      int       `json:"offset"`       // 分页偏移
}