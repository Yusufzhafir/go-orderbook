package main

import (
	"log"

	"github.com/Yusufzhafir/go-orderbook/backend/internal/engine"
	"github.com/Yusufzhafir/go-orderbook/backend/pkg/model"
)

func main() {
	newOb := engine.NewOrderBookEngine()
	newOb.Initialize()

	result, err := newOb.AddOrder(model.Order{
		ID:                1,
		Side:              model.ASK,
		Price:             10_000,
		InitialQuantity:   10,
		RemainingQuantity: 10,
		Type:              model.ORDER_GOOD_TILL_CANCEL,
	})

	log.Printf("the trade result : %+v \n err: %v", result, err)
	log.Println(newOb.GetOrderInfos())

	result, err = newOb.AddOrder(model.Order{
		ID:                1,
		Side:              model.ASK,
		Price:             10_000,
		InitialQuantity:   10,
		RemainingQuantity: 10,
		Type:              model.ORDER_GOOD_TILL_CANCEL,
	})

	//should still be one
	log.Printf("the trade result : %v \n err: %v", result, err)
	log.Println(newOb.GetOrderInfos())

	result, err = newOb.AddOrder(model.Order{
		ID:                2,
		Side:              model.BID,
		Price:             9_000,
		InitialQuantity:   10,
		RemainingQuantity: 10,
		Type:              model.ORDER_GOOD_TILL_CANCEL,
	})

	//should have both bid and asks
	log.Printf("the trade result : %+v \n err: %v", result, err)
	log.Println(newOb.GetOrderInfos())

	result, err = newOb.AddOrder(model.Order{
		ID:                3,
		Side:              model.BID,
		Price:             10_000,
		InitialQuantity:   10,
		RemainingQuantity: 10,
		Type:              model.ORDER_GOOD_TILL_CANCEL,
	})

	//should have only bids
	log.Printf("the trade result : %+v \n err: %v", result, err)
	log.Println(newOb.GetOrderInfos())

}

// func printSliceOfPointers(args []*any) {
// 	res := ""
// 	for i, e := range args {
// 		fmt.Append(res, i)
// 	}
// }
