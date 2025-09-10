package model

import (
	"fmt"
)

type Order struct {
	id                OrderId
	side              Side // BUY or SELL
	price             Price
	initialQuantity   Quantity
	remainingQuantity Quantity
	orderType         OrderType
}

func NewOrder(id OrderId, side Side, price Price, quantity Quantity, orderType OrderType) Order {
	return Order{
		id:                id,
		side:              side,
		price:             price,
		initialQuantity:   quantity,
		remainingQuantity: quantity,
		orderType:         orderType,
	}
}
func NewEmptyOrder(id OrderId, side Side, price Price, quantity Quantity, orderType OrderType) Order {
	return Order{}
}

func (o *Order) GetFilledQuantity() Quantity {
	return o.initialQuantity - o.remainingQuantity
}

func (o *Order) Fill(quantity Quantity) error {
	if quantity > o.remainingQuantity {
		return fmt.Errorf("order cannot be filled for more than its remaining quantity %d", o.id)
	}
	o.remainingQuantity -= quantity
	return nil
}

func (o *Order) IsFilled() bool {
	return o.remainingQuantity == 0
}

func (o *Order) GetRemainingQuantity() Quantity {
	return o.remainingQuantity
}

func (o *Order) GetPrice() Price {
	return o.price
}

func (o *Order) GetId() OrderId {
	return o.id
}

func (o *Order) GetType() OrderType {
	return o.orderType
}

func (o *Order) GetSide() Side {
	return o.side
}

func (o *Order) GetInitialQuantity() Quantity {
	return o.initialQuantity
}

type OrderModify struct {
	ID       OrderId
	Price    Price
	Quantity Quantity
	Side     Side
}

func (om *OrderModify) ToOrder(orderType OrderType) Order {
	return NewOrder(
		om.ID,
		om.Side,
		om.Price,
		om.Quantity,
		orderType,
	)
}

type Price uint64
type Quantity uint64
type OrderId uint64
type OrderType uint8

const (
	ORDER_FILL_AND_KILL OrderType = iota
	ORDER_GOOD_TILL_CANCEL
)
