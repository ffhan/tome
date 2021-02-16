package tome

import (
	"log"
	"sort"
)

type orderContainer struct {
	Bids, Asks *orderMap
	trackers   map[uint64]OrderTracker
}

func NewOrderContainer(bidLess, askLess LessFunc) *orderContainer {
	return &orderContainer{
		Bids:     newOrderMap(bidLess),
		Asks:     newOrderMap(askLess),
		trackers: make(map[uint64]OrderTracker),
	}
}

func (o *orderContainer) Add(tracker OrderTracker) {
	if tracker.Side == SideBuy {
		o.Bids.Set(tracker, true)
	} else {
		o.Asks.Set(tracker, true)
	}
	o.trackers[tracker.OrderID] = tracker
}

func (o *orderContainer) Remove(id uint64) {
	tracker, ok := o.trackers[id]
	if !ok {
		log.Printf("cannot remove order: no tracker for id %d", id)
		return
	}
	delete(o.trackers, id)
	if tracker.Side == SideBuy {
		o.Bids.Del(tracker)
	} else {
		o.Asks.Del(tracker)
	}
}

func (o *orderContainer) Get(id uint64) (OrderTracker, bool) {
	t, ok := o.trackers[id]
	return t, ok
}

func (o *orderContainer) Iterator(side OrderSide) forwardIteratorOrderMap {
	if side == SideBuy {
		return o.Bids.Iterator()
	}
	return o.Asks.Iterator()
}

func (o *orderContainer) Len(side OrderSide) int {
	if side == SideBuy {
		return o.Bids.Len()
	}
	return o.Asks.Len()
}

// Get ask trackers below or equal the price. Sorted by time ascending.
func (o *orderContainer) GetAsksBelow(price float64) []OrderTracker {
	trackers := make([]OrderTracker, 0)
	for iter := o.Asks.Iterator(); iter.Valid(); iter.Next() {
		if iter.Key().Price <= price {
			trackers = append(trackers, iter.Key())
		} else {
			break // iterator returns a sorted array, if price is bigger we don't have to look any further
		}
	}
	sort.Slice(trackers, func(i, j int) bool {
		return trackers[i].Timestamp < trackers[j].Timestamp
	})
	return trackers
}

// Get bid trackers above or equal the price. Sorted by time ascending.
func (o *orderContainer) GetBidsAbove(price float64) []OrderTracker {
	trackers := make([]OrderTracker, 0)
	for iter := o.Bids.Iterator(); iter.Valid(); iter.Next() {
		if iter.Key().Price >= price {
			trackers = append(trackers, iter.Key())
		} else {
			break // iterator returns a sorted array, if price is bigger we don't have to look any further
		}
	}
	sort.Slice(trackers, func(i, j int) bool {
		return trackers[i].Timestamp < trackers[j].Timestamp
	})
	return trackers
}
