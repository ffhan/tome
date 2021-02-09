package tome

type TradeRepository interface {
	Store(trade Trade) error
	GetByID(id uint64) (Trade, error)
}
