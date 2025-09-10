package model

import (
	"github.com/Yusufzhafir/go-orderbook/backend/pkg/model"
	"github.com/google/btree"
)

// AskPriceLevel ascending
type AskPriceLevel struct {
	Price       model.Price
	Orders      []*model.Order
	TotalVolume model.Quantity
}

func (pl *AskPriceLevel) Less(than btree.Item) bool {
	other := than.(*AskPriceLevel)
	return pl.Price < other.Price
}

// Fast order removal with index tracking
func (pl *AskPriceLevel) RemoveOrderByID(orderID model.OrderId) bool {
	for i, order := range pl.Orders {
		if order.GetId() == orderID {
			// Remove from slice
			pl.Orders = append(pl.Orders[:i], pl.Orders[i+1:]...)
			pl.TotalVolume -= order.GetInitialQuantity()
			return true
		}
	}
	return false
}

// BidPriceLevel descending
type BidPriceLevel struct {
	Price       model.Price
	Orders      []*model.Order
	TotalVolume model.Quantity
}

func (bpl *BidPriceLevel) Less(than btree.Item) bool {
	other := than.(*BidPriceLevel)
	return bpl.Price > other.Price // Reverse
}

// Fast order removal with index tracking
func (pl *BidPriceLevel) RemoveOrderByID(orderID model.OrderId) bool {
	for i, order := range pl.Orders {
		if order.GetId() == orderID {
			// Remove from slice
			pl.Orders = append(pl.Orders[:i], pl.Orders[i+1:]...)
			pl.TotalVolume -= order.GetInitialQuantity()
			return true
		}
	}
	return false
}
