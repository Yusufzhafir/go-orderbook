package model

import "time"

type Trade struct {
	Side      Side
	MakerID   OrderId
	TakerID   OrderId
	Price     Price
	Quantity  Quantity
	Timestamp time.Time
}
