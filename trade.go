package tome

import (
	"github.com/cockroachdb/apd"
	"github.com/google/uuid"
	"time"
)

// Trade represents two opposed matched orders.
type Trade struct {
	ID            uint64
	Buyer, Seller uuid.UUID
	Instrument    string
	Qty           int64
	Price         apd.Decimal
	Total         apd.Decimal
	Timestamp     time.Time

	BidOrderID uint64
	AskOrderID uint64
}
