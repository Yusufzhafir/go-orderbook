package model

type MarketDepthLevel struct {
	Price      Price    `json:"price"`
	Volume     Quantity `json:"volume"`
	OrderCount int      `json:"orderCount"`
}

// MarketDepth represents the full order book depth
type MarketDepth struct {
	Bids      []MarketDepthLevel `json:"bids"` // Highest to lowest price
	Asks      []MarketDepthLevel `json:"asks"` // Lowest to highest price
	Timestamp int64              `json:"timestamp"`
}

// TopOfBook represents best bid/ask
type TopOfBook struct {
	BestBid *MarketDepthLevel `json:"bestBid"`
	BestAsk *MarketDepthLevel `json:"bestAsk"`
	Spread  Price             `json:"spread"`
}
