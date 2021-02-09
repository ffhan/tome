package tome

import "sync"

type TradeBook struct {
	Instrument string

	trades      []Trade
	todayTrades map[uint64]*Trade
	tradeMutex  sync.RWMutex
}

func NewTradeBook(instrument string) *TradeBook {
	return &TradeBook{
		Instrument:  instrument,
		trades:      make([]Trade, 0, 1024),
		todayTrades: make(map[uint64]*Trade),
	}
}

func (t *TradeBook) Enter(trade Trade) {
	t.tradeMutex.Lock()
	defer t.tradeMutex.Unlock()

	t.trades = append(t.trades, trade)
	t.todayTrades[trade.ID] = &t.trades[len(t.trades)-1]
}

func (t *TradeBook) Reject(tradeID uint64) {
	t.tradeMutex.Lock()
	defer t.tradeMutex.Unlock()

	if trade, ok := t.todayTrades[tradeID]; ok {
		trade.Rejected = true
		t.todayTrades[tradeID] = trade
	}
}

func (t *TradeBook) DailyTrades() []Trade {
	t.tradeMutex.RLock()
	defer t.tradeMutex.RUnlock()

	tradesCopy := make([]Trade, len(t.trades))
	copy(tradesCopy, t.trades)
	return tradesCopy
}
