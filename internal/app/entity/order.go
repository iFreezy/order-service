package entity

import (
	"time"

	"github.com/gofrs/uuid"
)

type Order struct {
	ID         int64       `gorm:"column:id;autoIncrement;primaryKey:false"`
	GUID       uuid.UUID   `gorm:"column:guid;type:uuid;primaryKey"`
	UserGUID   *uuid.UUID  `gorm:"column:user_guid;type:uuid"`
	TotalPrice int64       `gorm:"column:total_price"`
	Currency   string      `gorm:"column:currency"`
	Status     string      `gorm:"column:status"`
	CreatedAt  time.Time   `gorm:"column:created_at"`
	UpdatedAt  time.Time   `gorm:"column:updated_at"`
	Items      []OrderItem `gorm:"foreignKey:OrderGUID;references:GUID"`
}

func (Order) TableName() string { return "orders" }

type OrderItem struct {
	ID          int64     `gorm:"column:id;autoIncrement;primaryKey:false"`
	GUID        uuid.UUID `gorm:"column:guid;type:uuid;primaryKey"`
	OrderGUID   uuid.UUID `gorm:"column:order_guid;type:uuid"`
	ProductGUID uuid.UUID `gorm:"column:product_guid;type:uuid"`
	Quantity    int       `gorm:"column:quantity"`
	UnitPrice   int64     `gorm:"column:unit_price"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (OrderItem) TableName() string { return "order_items" }

type RequestOrderCreate struct {
	UserGUID *uuid.UUID               `json:"user_guid"`
	Currency string                   `json:"currency" binding:"required,len=3"`
	Items    []RequestOrderItemCreate `json:"items" binding:"required,min=1,dive"`
}
type RequestOrderItemCreate struct {
	ProductGUID uuid.UUID `json:"product_guid" binding:"required"`
	Quantity    int       `json:"quantity" binding:"required,gt=0,lte=2147483647"`
	UnitPrice   int64     `json:"unit_price" binding:"required,gt=0"`
}
type RequestOrderUpdate struct {
	Status string `json:"status" binding:"required,max=32"`
}
type RequestOrderList struct {
	Status   *string    `json:"status" binding:"omitempty,max=32"`
	UserGUID *uuid.UUID `json:"user_guid" binding:"omitempty"`
}
type ResponseOrderItem struct {
	GUID        uuid.UUID `json:"guid"`
	ProductGUID uuid.UUID `json:"product_guid"`
	Quantity    int       `json:"quantity"`
	UnitPrice   int64     `json:"unit_price"`
}
type ResponseOrderCreate struct {
	GUID       uuid.UUID           `json:"guid"`
	UserGUID   *uuid.UUID          `json:"user_guid"`
	TotalPrice int64               `json:"total_price"`
	Currency   string              `json:"currency"`
	Status     string              `json:"status"`
	Items      []ResponseOrderItem `json:"items"`
	CreatedAt  time.Time           `json:"created_at"`
}
type ResponseOrderGet struct {
	ResponseOrderCreate
	UpdatedAt time.Time `json:"updated_at"`
}
type ResponseOrderUpdate = ResponseOrderGet
type ResponseOrderListItem struct {
	GUID       uuid.UUID  `json:"guid"`
	UserGUID   *uuid.UUID `json:"user_guid"`
	TotalPrice int64      `json:"total_price"`
	Currency   string     `json:"currency"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}
type ResponseOrderList struct {
	Data []ResponseOrderListItem `json:"data"`
}
