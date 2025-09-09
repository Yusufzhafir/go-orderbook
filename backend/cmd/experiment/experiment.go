package main

import (
	"log"

	"github.com/Yusufzhafir/go-orderbook/backend/internal/engine"
	"github.com/Yusufzhafir/go-orderbook/backend/pkg/model"
)

func main() {
	newOb := engine.NewOrderBookEngine()
	newOb.Initialize()

	result, err := newOb.AddOrder(model.NewOrder(
		1,
		model.ASK,
		10_000,
		10,
		model.ORDER_GOOD_TILL_CANCEL,
	))

	log.Printf("the trade result : %+v \n err: %v", result, err)
	log.Println(newOb.GetOrderInfos())

	result, err = newOb.AddOrder(model.NewOrder(
		1,
		model.ASK,
		10_000,
		10,
		model.ORDER_GOOD_TILL_CANCEL,
	))

	//should still be one
	log.Printf("the trade result : %v \n err: %v", result, err)
	log.Println(newOb.GetOrderInfos())

	cancelOrderId := model.OrderId(2)
	result, err = newOb.AddOrder(model.NewOrder(
		cancelOrderId,
		model.BID,
		00,
		10,
		model.ORDER_GOOD_TILL_CANCEL,
	))

	//should have both bid and asks
	log.Printf("the trade result : %+v \n err: %v", result, err)
	log.Println(newOb.GetOrderInfos())

	result, err = newOb.AddOrder(model.NewOrder(
		3,
		model.BID,
		10_000,
		10,
		model.ORDER_GOOD_TILL_CANCEL,
	))

	//should have only bids
	log.Printf("the trade result : %+v \n err: %v", result, err)
	log.Println(newOb.GetOrderInfos())

	result, err = newOb.ModifyOrder(model.OrderModify{
		ID:       cancelOrderId,
		Side:     model.BID,
		Price:    10_000,
		Quantity: 10,
	}, model.ORDER_GOOD_TILL_CANCEL)
	log.Printf("the trade result : %+v \n err: %v", result, err)
	log.Println(newOb.GetOrderInfos())

	err = newOb.CancelOrder(cancelOrderId)

	//should have only bids
	log.Printf("cancel the result err: %v", err)
	log.Println(newOb.GetOrderInfos())

}
