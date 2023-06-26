package service

import "time"

type Order struct {
	OrderUID    string `json:"order_uid" faker:"len=15"`
	TrackNumber string `json:"track_number" faker:"len=10"`
	Entry       string `json:"entry" faker:"len=10"`
	Delivery    struct {
		Name    string `json:"name" faker:"name"`
		Phone   string `json:"phone" faker:"e_164_phone_number"`
		Zip     string `json:"zip"`
		City    string `json:"city"`
		Address string `json:"address"`
		Region  string `json:"region"`
		Email   string `json:"email" faker:"email"`
	} `json:"delivery"`
	Payment struct {
		Transaction  string `json:"transaction"`
		RequestID    string `json:"request_id" faker:"len=1"`
		Currency     string `json:"currency" faker:"currency"`
		Provider     string `json:"provider"`
		Amount       int    `json:"amount"`
		PaymentDt    int    `json:"payment_dt"`
		Bank         string `json:"bank" faker:"oneof: alpha, sber, tinkoff"`
		DeliveryCost int    `json:"delivery_cost" faker:"boundary_start=0, boundary_end=1000"`
		GoodsTotal   int    `json:"goods_total" faker:"boundary_start=1, boundary_end=100"`
		CustomFee    int    `json:"custom_fee" faker:"boundary_start=0, boundary_end=20"`
	} `json:"payment"`
	Items             []Item    `json:"items"`
	Locale            string    `json:"locale" faker:"oneof: ru, en"`
	InternalSignature string    `json:"internal_signature" faker:"len=0"`
	CustomerID        string    `json:"customer_id" faker:"oneof: test"`
	DeliveryService   string    `json:"delivery_service" faker:"oneof: DHL, TNT, UPS, FedEx"`
	Shardkey          string    `json:"shardkey" faker:"len=1"`
	SmID              int       `json:"sm_id" faker:"boundary_start=1, boundary_end=99"`
	DateCreated       time.Time `json:"date_created"`
	OofShard          string    `json:"oof_shard" faker:"len=2"`
}

type Item struct {
	ChrtID      int    `json:"chrt_id" faker:"boundary_start=100000, boundary_end=999999"`
	TrackNumber string `json:"track_number" faker:"len=10"`
	Price       int    `json:"price" faker:"boundary_start=10, boundary_end=10000"`
	Rid         string `json:"rid" faker:"len=15"`
	Name        string `json:"name"`
	Sale        int    `json:"sale" faker:"boundary_start=3, boundary_end=30"`
	Size        string `json:"size"`
	TotalPrice  int    `json:"total_price"`
	NmID        int    `json:"nm_id"`
	Brand       string `json:"brand" faker:"len=10"`
	Status      int    `json:"status" faker:"oneof: 202, 500, 404"`
}

type OrderUID string
