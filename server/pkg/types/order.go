package types

// OrderInfo service layer input
type OrderInfo struct {
	UserID            int              `json:"user_id"`             // 下单用户
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
