package model

import (
	"fmt"
)

type Order struct {
	ID                OrderId
	Side              Side // BUY or SELL
	Price             Price
	InitialQuantity   Quantity
	RemainingQuantity Quantity
	Type              OrderType
}

func (o *Order) GetFilledQuantity() Quantity {
	return o.InitialQuantity - o.RemainingQuantity
}

func (o *Order) Fill(quantity Quantity) error {
	if quantity > o.RemainingQuantity {
		return fmt.Errorf("order cannot be filled for more than its remaining quantity %d", o.ID)
	}
	o.RemainingQuantity -= quantity
	return nil
}

func (o *Order) IsFilled() bool {
	return o.RemainingQuantity == 0
}

type OrderModify struct {
	ID       OrderId
	Price    Price
	Quantity Quantity
	Side     Side
}

func (om *OrderModify) ToOrder(orderType OrderType) Order {
	return Order{
		ID:                om.ID,
		Side:              om.Side,
		Price:             om.Price,
		InitialQuantity:   om.Quantity,
		RemainingQuantity: om.Quantity,
		Type:              orderType,
	}
}

type Price int64
type Quantity int64
type OrderId int64
type OrderType int8

const (
	ORDER_FILL_AND_KILL OrderType = iota
	ORDER_GOOD_TILL_CANCEL
)
