package engine

import (
	"fmt"
	"log"
	"time"

	orderbookModel "github.com/Yusufzhafir/go-orderbook/backend/internal/engine/model"
	"github.com/Yusufzhafir/go-orderbook/backend/pkg/model"
	"github.com/google/btree"
)

type OrderBookEngine interface {
	AddOrder(order model.Order) ([]*model.Trade, error)
	CancelOrder(orderID model.OrderId) error
	ModifyOrder(modify model.OrderModify, orderType model.OrderType) ([]*model.Trade, error)
	Initialize()
	OrderSize() int
	GetTopOfBook() *model.TopOfBook
	GetOrderInfos() *model.MarketDepth
}

type OrderBookEngineImpl struct {
	bids, asks *btree.BTree                   // price-level trees
	orders     map[model.OrderId]*model.Order // lookup by ID
}

// Pop from front of slice (queue behavior - FIFO)
func popFront[T any](slice []T) (T, []T) {
	if len(slice) == 0 {
		var zero T
		return zero, slice
	}
	return slice[0], slice[1:]
}

// Pop from back of slice (stack behavior - LIFO)
func popBack[T any](slice []T) (T, []T) {
	if len(slice) == 0 {
		var zero T
		return zero, slice
	}
	last := len(slice) - 1
	return slice[last], slice[:last]
}

// Or modify slice in-place and return the popped element
func popFrontInPlace[T any](slice *[]T) T {
	if len(*slice) == 0 {
		var zero T
		return zero
	}
	item := (*slice)[0]
	*slice = (*slice)[1:]
	return item
}

func (o *OrderBookEngineImpl) canMatch(side model.Side, price model.Price) bool {
	if side == model.BID {
		if o.asks.Len() == 0 {
			return false
		}
		bestAsks := o.asks.Min().(*orderbookModel.AskPriceLevel).Price
		return price >= bestAsks
	}

	if o.bids.Len() == 0 {
		return false
	}

	bestBids := o.bids.Min().(*orderbookModel.BidPriceLevel).Price
	return price <= bestBids
}

func (o *OrderBookEngineImpl) matchOrder() []*model.Trade {
	trades := make([]*model.Trade, 0)
	for {
		if o.bids.Len() == 0 || o.asks.Len() == 0 {
			break
		}

		asksPriceLevel := o.asks.Min().(*orderbookModel.AskPriceLevel)
		bidsPriceLevel := o.bids.Min().(*orderbookModel.BidPriceLevel)

		if bidsPriceLevel.Price < asksPriceLevel.Price {
			break
		}

		for len(asksPriceLevel.Orders) > 0 && len(bidsPriceLevel.Orders) > 0 {

			askOrder := asksPriceLevel.Orders[0]
			bidOrder := bidsPriceLevel.Orders[0]

			bestQuantity := min(askOrder.RemainingQuantity, bidOrder.RemainingQuantity)
			bestPrice := min(askOrder.Price, bidOrder.Price)
			askOrder.Fill(bestQuantity)
			bidOrder.Fill(bestQuantity)

			trades = append(trades, &model.Trade{
				MakerID:  askOrder.ID,
				TakerID:  bidOrder.ID,
				Price:    bestPrice,
				Quantity: bestQuantity,
			})

			if askOrder.IsFilled() {
				delete(o.orders, askOrder.ID)
				asksPriceLevel.Orders = asksPriceLevel.Orders[1:] // Pop front
				asksPriceLevel.TotalVolume -= bestQuantity
			}
			if bidOrder.IsFilled() {
				delete(o.orders, bidOrder.ID)
				bidsPriceLevel.Orders = bidsPriceLevel.Orders[1:]
				bidsPriceLevel.TotalVolume -= bestQuantity

			}

			if len(asksPriceLevel.Orders) == 0 {
				o.asks.Delete(asksPriceLevel)
			}
			if len(bidsPriceLevel.Orders) == 0 {
				o.bids.Delete(bidsPriceLevel)
			}
		}

	}

	//asuming all order that can be matched is already matched 3finished all orders that can be matched
	if o.asks.Len() > 0 {
		asksPriceLevel := o.asks.Min().(*orderbookModel.AskPriceLevel)
		askOrder := asksPriceLevel.Orders[0]
		if askOrder.Type == model.ORDER_FILL_AND_KILL {
			//cancel order
		}
	}
	if o.bids.Len() > 0 {
		bidsPriceLevel := o.bids.Min().(*orderbookModel.BidPriceLevel)

		bidOrder := bidsPriceLevel.Orders[0]
		if bidOrder.Type == model.ORDER_FILL_AND_KILL {
			//cancel order
		}
	}

	return trades
}

func (o *OrderBookEngineImpl) AddOrder(order model.Order) ([]*model.Trade, error) {
	_, ok := o.orders[order.ID]
	if ok {
		return []*model.Trade{}, fmt.Errorf("order already exist for id %d", order.ID)
	}

	if order.Type == model.ORDER_FILL_AND_KILL && !o.canMatch(order.Side, order.Price) {
		return []*model.Trade{}, fmt.Errorf("cannot fill and kill at that price for order id %d", order.ID)
	}

	o.orders[order.ID] = &order

	switch order.Side {
	case model.ASK:
		priceLevel := &orderbookModel.AskPriceLevel{Price: order.Price}

		if !o.asks.Has(priceLevel) {
			priceLevel.TotalVolume = 0
			priceLevel.Orders = make([]*model.Order, 0)
			o.asks.ReplaceOrInsert(priceLevel)
		}

		currentPriceLevel := o.asks.Get(priceLevel).(*orderbookModel.AskPriceLevel)

		if currentPriceLevel == nil {
			panic("WHAT THE HELL HAPPENED?!?!?")
		}

		currentPriceLevel.Orders = append(currentPriceLevel.Orders, &order)
		currentPriceLevel.TotalVolume += order.InitialQuantity

	case model.BID:
		priceLevel := &orderbookModel.BidPriceLevel{Price: order.Price}

		if !o.bids.Has(priceLevel) {
			priceLevel.TotalVolume = 0
			priceLevel.Orders = make([]*model.Order, 0)
			o.bids.ReplaceOrInsert(priceLevel)
		}

		currentPriceLevel := o.bids.Get(priceLevel).(*orderbookModel.BidPriceLevel)

		if currentPriceLevel == nil {
			panic("WHAT THE HELL HAPPENED?!?!?")
		}

		currentPriceLevel.Orders = append(currentPriceLevel.Orders, &order)
		currentPriceLevel.TotalVolume += order.InitialQuantity
	}

	return o.matchOrder(), nil
}

func (o *OrderBookEngineImpl) CancelOrder(orderID model.OrderId) error {
	order, exists := o.orders[orderID]
	if !exists {
		return fmt.Errorf("order not found: %d", orderID)
	}

	// Find and remove from price level
	if order.Side == model.ASK {
		// removeOrderByID
		priceLevel := &orderbookModel.AskPriceLevel{Price: order.Price}
		if item := o.asks.Get(priceLevel); item != nil {
			item.(*orderbookModel.AskPriceLevel).RemoveOrderByID(order.ID)
		}
		priceLevel.TotalVolume -= order.RemainingQuantity
		if len(priceLevel.Orders) == 0 {
			o.asks.Delete(priceLevel)
		}
	} else {
		priceLevel := &orderbookModel.BidPriceLevel{Price: order.Price}
		if item := o.bids.Get(priceLevel); item != nil {
			item.(*orderbookModel.BidPriceLevel).RemoveOrderByID(order.ID)
		}
		priceLevel.TotalVolume -= order.RemainingQuantity
		if len(priceLevel.Orders) == 0 {
			o.bids.Delete(priceLevel)
		}
	}

	delete(o.orders, orderID)
	return nil
}

func (o *OrderBookEngineImpl) ModifyOrder(modify model.OrderModify, orderType model.OrderType) ([]*model.Trade, error) {
	existing, ok := o.orders[modify.ID]
	if !ok {
		return nil, fmt.Errorf("cannot find order with id %v", modify.ID)
	}
	err := o.CancelOrder(existing.ID)
	if err != nil {
		return nil, err
	}

	addOrder, err := o.AddOrder(modify.ToOrder(orderType))
	return addOrder, err
}

func (o *OrderBookEngineImpl) OrderSize() int {
	return o.asks.Len() + o.bids.Len()
}

func (o *OrderBookEngineImpl) getMarketDepth(levels int) *model.MarketDepth {
	depth := &model.MarketDepth{
		Bids:      make([]model.MarketDepthLevel, 0, levels),
		Asks:      make([]model.MarketDepthLevel, 0, levels),
		Timestamp: time.Now().UnixMilli(),
	}

	// Collect bid levels (highest price first)
	bidCount := 0
	o.bids.Ascend(func(item btree.Item) bool {
		if bidCount >= levels {
			return false // Stop iteration
		}

		bidLevel := item.(*orderbookModel.BidPriceLevel)
		depth.Bids = append(depth.Bids, model.MarketDepthLevel{
			Price:      bidLevel.Price,
			Volume:     bidLevel.TotalVolume,
			OrderCount: len(bidLevel.Orders),
		})

		bidCount++
		return true // Continue iteration
	})

	// Collect ask levels (lowest price first)
	askCount := 0
	o.asks.Ascend(func(item btree.Item) bool {
		if askCount >= levels {
			return false
		}

		askLevel := item.(*orderbookModel.AskPriceLevel)
		depth.Asks = append(depth.Asks, model.MarketDepthLevel{
			Price:      askLevel.Price,
			Volume:     askLevel.TotalVolume,
			OrderCount: len(askLevel.Orders),
		})

		askCount++
		return true
	})

	return depth
}

// GetTopOfBook returns best bid and ask
func (o *OrderBookEngineImpl) GetTopOfBook() *model.TopOfBook {
	tob := &model.TopOfBook{}

	// Get best bid (highest price)
	if o.bids.Len() > 0 {
		bestBidItem := o.bids.Min().(*orderbookModel.BidPriceLevel)
		tob.BestBid = &model.MarketDepthLevel{
			Price:      bestBidItem.Price,
			Volume:     bestBidItem.TotalVolume,
			OrderCount: len(bestBidItem.Orders),
		}
	}

	// Get best ask (lowest price)
	if o.asks.Len() > 0 {
		bestAskItem := o.asks.Min().(*orderbookModel.AskPriceLevel)
		tob.BestAsk = &model.MarketDepthLevel{
			Price:      bestAskItem.Price,
			Volume:     bestAskItem.TotalVolume,
			OrderCount: len(bestAskItem.Orders),
		}
	}

	// Calculate spread
	if tob.BestBid != nil && tob.BestAsk != nil {
		tob.Spread = tob.BestAsk.Price - tob.BestBid.Price
	}

	return tob
}

// GetOrderInfos - your original method, now implemented
func (o *OrderBookEngineImpl) GetOrderInfos() *model.MarketDepth {
	return o.getMarketDepth(10) // Default to top 10 levels
}

func (o *OrderBookEngineImpl) Initialize() {
	o.bids = btree.New(32) // degree tuned for performance
	o.asks = btree.New(32)
	o.orders = make(map[model.OrderId]*model.Order)
	log.Println("order book is initialized!!")
}

func NewOrderBookEngine() OrderBookEngine {
	return &OrderBookEngineImpl{}
}
