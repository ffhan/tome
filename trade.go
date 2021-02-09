package tome

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"time"
)

type Trade struct {
	ID            uint64
	Buyer, Seller uuid.UUID
	Instrument    string
	Qty           int64
	Price         decimal.Decimal
	Total         decimal.Decimal
	Timestamp     time.Time
	Rejected      bool // trade rejection (e.g. because of IOC)

	BidOrderID uint64
	AskOrderID uint64
}
