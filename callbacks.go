package tome

type OrderCallbackFunc func(order Order)
type TradeCallbackFunc func(trade Trade)

type OrderCallback interface {
	Execute(order Order)
}

type TradeCallback interface {
	Execute(trade Trade)
}
