package model

import "time"

type Trade struct {
	MakerID   OrderId
	TakerID   OrderId
	Price     Price
	Quantity  Quantity
	Timestamp time.Time
}
