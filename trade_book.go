package tome

import (
	"sort"
	"sync"
)

// Trade book stores all daily trades in-memory.
// It flushes new trades periodically to persistent storage. (TODO)
type TradeBook struct {
	Instrument string

	trades      map[uint64]Trade
	tradeMutex  sync.RWMutex
	lastTradeID uint64
}

// Create a new trade book.
func NewTradeBook(instrument string) *TradeBook {
	return &TradeBook{
		Instrument: instrument,
		trades:     make(map[uint64]Trade),
	}
}

// Enter a new trade.
func (t *TradeBook) Enter(trade Trade) {
	t.tradeMutex.Lock()
	defer t.tradeMutex.Unlock()

	t.trades[t.lastTradeID] = trade
	t.lastTradeID += 1
}

// Return all daily trades in a trade book.
func (t *TradeBook) DailyTrades() []Trade {
	t.tradeMutex.RLock()
	defer t.tradeMutex.RUnlock()

	tradesCopy := make([]Trade, len(t.trades))
	i := 0
	for _, trade := range t.trades {
		tradesCopy[i] = trade
		i += 1
	}
	sort.Slice(tradesCopy, func(i, j int) bool {
		return tradesCopy[i].Timestamp.Before(tradesCopy[j].Timestamp)
	})
	return tradesCopy
}
