package tome

import (
	"errors"
	"fmt"
	"github.com/cockroachdb/apd"
	"github.com/google/uuid"
	"log"
	"sync"
	"time"
)

const (
	MinQty = 1
)

var (
	ErrInvalidQty         = errors.New("invalid quantity provided")
	ErrInvalidMarketPrice = errors.New("price has to be zero for market orders")
	ErrInvalidLimitPrice  = errors.New("price has to be set for limit orders")
	ErrInvalidStopPrice   = errors.New("stop price has to be set for a stop order")

	BaseContext = apd.Context{
		Precision:   0,               // no rounding
		MaxExponent: apd.MaxExponent, // up to 10^5 exponent
		MinExponent: apd.MinExponent, // support only 4 decimal places
		Traps:       apd.DefaultTraps,
	}
)

// Order book contains all active orders for an instrument, handles matching and storage of orders and subsequent trades.
type OrderBook struct {
	Instrument string // instrument name

	marketPrice      apd.Decimal // current market price
	marketPriceMutex sync.RWMutex

	tradeBook *TradeBook // trade book ptr

	orderRepo    OrderRepository  // persistent order storage
	activeOrders map[uint64]Order // quick order retrieval by ID

	orders     *orderContainer // contains all orders sorted by our preferences
	stopOrders *orderContainer // contains all stop orders sorted by our preferences

	orderMutex sync.RWMutex
	matchMutex sync.Mutex // mutex that ensures that matching is always sequential
}

// function that compares two OrderTrackers and returns true if a is less or equal than b
type LessFunc func(a, b OrderTracker) bool

// FIFO - https://corporatefinanceinstitute.com/resources/knowledge/trading-investing/matching-orders/
func makeComparator(priceDescending bool) LessFunc {
	const (
		ascending  bool = true
		descending bool = false
	)
	sort := ascending
	if priceDescending {
		sort = descending
	}
	return func(a, b OrderTracker) bool {
		if a.Type == TypeMarket && b.Type != TypeMarket { // market orders first
			return true
		} else if a.Type != TypeMarket && b.Type == TypeMarket {
			return false
		} else if a.Type == TypeMarket && b.Type == TypeMarket {
			return a.Timestamp < b.Timestamp // if both market order by time
		}
		priceCmp := a.Price - b.Price // compare prices
		if priceCmp == 0 {            // if prices are equal, compare timestamps
			return a.Timestamp < b.Timestamp
		}
		if priceCmp < 0 { // if a price is less than b return true if ascending, false if descending
			return sort
		}
		return !sort // if a price is bigger than b return false if ascending, true if descending
	}
}

func makeStopComparator(priceDescending bool) LessFunc {
	const (
		ascending  bool = true
		descending bool = false
	)
	sort := ascending
	if priceDescending {
		sort = descending
	}
	return func(a, b OrderTracker) bool { // ignores order types because we're always comparing stop prices
		priceCmp := a.Price - b.Price // compare prices
		if priceCmp == 0 {            // if prices are equal, compare timestamps
			return a.Timestamp < b.Timestamp
		}
		if priceCmp < 0 { // if a price is less than b return true if ascending, false if descending
			return sort
		}
		return !sort // if a price is bigger than b return false if ascending, true if descending
	}
}

// Create a new order book.
func NewOrderBook(instrument string, marketPrice apd.Decimal, tradeBook *TradeBook, orderRepo OrderRepository) *OrderBook {
	bidLess := makeComparator(true)
	askLess := makeComparator(false)
	/*
		Note for stop ordering: it makes much more sense to order bids ascending (in reverse of normal bid price ordering)
		because lower prices will more likely be passed by a market price. Higher stop prices are less likely to be
		passed by a market price, therefore we would have to skip a lot of orders to get to stop orders which are likely to be activated.

		Therefore, when calling GetBidsBelow it makes much more sense to start from the lowest stop price, compare it with the market price
		(is the stop price lower than the current market price) and if it's not - break (other prices are going to be even higher than the market price).

		The same logic applies to asks, just in reverse.
	*/
	stopBidLess := makeStopComparator(false)
	stopAskLess := makeStopComparator(true)
	return &OrderBook{
		Instrument:   instrument,
		marketPrice:  marketPrice,
		tradeBook:    tradeBook,
		orderRepo:    orderRepo,
		activeOrders: make(map[uint64]Order),
		orders:       NewOrderContainer(bidLess, askLess),
		stopOrders:   NewOrderContainer(stopBidLess, stopAskLess),
	}
}

// Get all bids ordered the same way they are matched.
func (o *OrderBook) GetBids() []Order {
	o.orderMutex.RLock()
	defer o.orderMutex.RUnlock()
	orders := make([]Order, 0, o.orders.Len(SideBuy))
	for iter := o.orders.Iterator(SideBuy); iter.Valid(); iter.Next() {
		orders = append(orders, o.activeOrders[iter.Key().OrderID])
	}
	return orders
}

// Get all asks ordered the same way they are matched.
func (o *OrderBook) GetAsks() []Order {
	o.orderMutex.RLock()
	defer o.orderMutex.RUnlock()
	orders := make([]Order, 0, o.orders.Len(SideSell))
	for iter := o.orders.Iterator(SideSell); iter.Valid(); iter.Next() {
		orders = append(orders, o.activeOrders[iter.Key().OrderID])
	}
	return orders
}

// Get all stop bids.
func (o *OrderBook) GetStopBids() []Order {
	o.orderMutex.RLock()
	defer o.orderMutex.RUnlock()
	orders := make([]Order, 0, o.stopOrders.Len(SideBuy))
	for iter := o.stopOrders.Iterator(SideBuy); iter.Valid(); iter.Next() {
		orders = append(orders, o.activeOrders[iter.Key().OrderID])
	}
	return orders
}

// Get all stop asks.
func (o *OrderBook) GetStopAsks() []Order {
	o.orderMutex.RLock()
	defer o.orderMutex.RUnlock()
	orders := make([]Order, 0, o.stopOrders.Len(SideSell))
	for iter := o.stopOrders.Iterator(SideSell); iter.Valid(); iter.Next() {
		orders = append(orders, o.activeOrders[iter.Key().OrderID])
	}
	return orders
}

// Get a market price.
func (o *OrderBook) MarketPrice() apd.Decimal {
	o.marketPriceMutex.RLock()
	defer o.marketPriceMutex.RUnlock()
	return o.marketPrice
}

// Set a market price.
func (o *OrderBook) SetMarketPrice(price apd.Decimal, fPrice float64) {
	o.marketPriceMutex.Lock()
	o.marketPrice = price
	o.marketPriceMutex.Unlock()

	bids := o.stopOrders.GetBidsBelow(fPrice)
	o.addOrders(bids)
	asks := o.stopOrders.GetAsksAbove(fPrice)
	o.addOrders(asks)
}

func (o *OrderBook) addOrders(trackers []OrderTracker) {
	for _, bid := range trackers {
		order, ok := o.getActiveOrder(bid.OrderID)
		o.stopOrders.Remove(bid.OrderID)
		if !ok {
			panic(fmt.Errorf("order with ID %d not found", bid.OrderID))
		}
		if _, err := o.submit(order, bid); err != nil {
			log.Println(err) // todo: better handling of these events
		}
	}
}

// Get an order from activeOrders map.
func (o *OrderBook) getActiveOrder(id uint64) (Order, bool) {
	o.orderMutex.RLock()
	defer o.orderMutex.RUnlock()
	order, ok := o.activeOrders[id]
	return order, ok
}

// Insert an order in activeOrders map.
func (o *OrderBook) setActiveOrder(order Order) error {
	o.orderMutex.Lock()
	defer o.orderMutex.Unlock()
	if _, ok := o.activeOrders[order.ID]; ok {
		return fmt.Errorf("order with ID %d already exists", order.ID)
	}
	o.activeOrders[order.ID] = order
	return nil
}

// Add an order to books - make it matchable against other orders.
func (o *OrderBook) addToBooks(tracker OrderTracker) {
	o.orderMutex.Lock()
	o.orders.Add(tracker) // enter pointer to the tree
	o.orderMutex.Unlock()
}

func (o *OrderBook) storeOrder(order Order) error {
	if err := o.setActiveOrder(order); err != nil {
		o.orders.Remove(order.ID)
		return err
	}
	return o.orderRepo.Save(order)
}

// Update an active order.
func (o *OrderBook) updateActiveOrder(order Order) error {
	o.orderMutex.Lock()
	defer o.orderMutex.Unlock()
	if _, ok := o.activeOrders[order.ID]; !ok {
		return fmt.Errorf("order with ID %d hasn't yet been saved", order.ID)
	}
	o.activeOrders[order.ID] = order
	return o.orderRepo.Save(order)
}

// Removes an order from books - removes it from possible matches.
func (o *OrderBook) removeFromBooks(orderID uint64) {
	order, ok := o.getActiveOrder(orderID)
	if !ok {
		return
	}
	if err := o.orderRepo.Save(order); err != nil { // ensure we store the latest order data
		log.Printf("cannot save the order %+v to the repo - repository data might be inconsistent\n", order.ID)
	}

	o.orderMutex.Lock()
	o.orders.Remove(orderID)
	delete(o.activeOrders, orderID) // remove an active order
	o.orderMutex.Unlock()
}

// Cancel an order.
func (o *OrderBook) Cancel(id uint64) error {
	o.orderMutex.RLock()
	order, ok := o.activeOrders[id]
	o.orderMutex.RUnlock()

	if !ok {
		return nil
	}
	order.Cancel()
	return o.updateActiveOrder(order) // todo: remove from active orders
}

// get an OrderTracker from order ID. Returns false if OrderTracker under that ID doesn't exist.
func (o *OrderBook) getOrderTracker(orderID uint64) (OrderTracker, bool) {
	o.orderMutex.RLock()
	defer o.orderMutex.RUnlock()
	return o.orders.Get(orderID)
}

// Add a new order. Order can be matched immediately or later (or never), depending on order parameters and order type.
// Returns true if order was matched (partially or fully), false otherwise.
func (o *OrderBook) Add(order Order) (bool, error) {
	if order.Qty <= MinQty { // check the qty
		return false, ErrInvalidQty
	}
	if order.Type == TypeMarket && !order.Price.IsZero() {
		return false, ErrInvalidMarketPrice
	}
	if order.Type == TypeLimit && order.Price.IsZero() {
		return false, ErrInvalidLimitPrice
	}
	if order.Params.Is(ParamStop) && order.StopPrice.IsZero() {
		return false, ErrInvalidStopPrice
	}

	orderPrice, err := order.Price.Float64()
	if err != nil {
		return false, err
	}

	tracker := OrderTracker{
		OrderID:   order.ID,
		Type:      order.Type,
		Price:     orderPrice,
		Side:      order.Side,
		Timestamp: order.Timestamp.UnixNano(),
	}

	if order.Params.Is(ParamStop) {
		marketPrice := o.MarketPrice()

		orderStopPrice, err := order.StopPrice.Float64()
		if err != nil {
			return false, err
		}

		tracker := OrderTracker{
			OrderID:   order.ID,
			Type:      order.Type,
			Price:     orderStopPrice,
			Side:      order.Side,
			Timestamp: order.Timestamp.UnixNano(),
		}

		switch order.Side {
		case SideBuy:
			// if market price is lower than the bid stop price add as a stop order
			// otherwise process immediately
			if marketPrice.Cmp(&order.StopPrice) < 0 {
				o.stopOrders.Add(tracker)
				if err := o.storeOrder(order); err != nil {
					return false, err
				}
				return false, nil
			}
		case SideSell:
			// if market price is higher than the ask stop price add as a stop order
			// otherwise proces immediately
			if marketPrice.Cmp(&order.StopPrice) > 0 {
				o.stopOrders.Add(tracker)
				if err := o.storeOrder(order); err != nil {
					return false, err
				}
				return false, nil
			}
		}
	}

	return o.submit(order, tracker)
}

// submit an order for matching and store it. Returns true if matched (partially or fully), false if not.
func (o *OrderBook) submit(order Order, tracker OrderTracker) (bool, error) {
	var matched bool

	if order.IsBid() {
		// order is a bid, match with asks
		matched, _ = o.matchOrder(tracker.Price, &order, o.orders.Asks)
	} else {
		// order is an ask, match with bids
		matched, _ = o.matchOrder(tracker.Price, &order, o.orders.Bids)
	}

	addToBooks := true

	if order.Params.Is(ParamIOC) && !order.IsFilled() {
		order.Cancel()                                  // cancel the rest of the order
		if err := o.orderRepo.Save(order); err != nil { // store the order (not in the books)
			return matched, err
		}
		addToBooks = false // don't add the order to the books (keep it stored but not active)
	}

	if !order.IsFilled() && addToBooks {
		o.addToBooks(tracker)
		if err := o.storeOrder(order); err != nil {
			return matched, err
		}
	}
	return matched, nil
}

// return a minimum of two int64s
func min(q1, q2 int64) int64 {
	if q1 <= q2 {
		return q1
	}
	return q2
}

// match an order against other offers, return if an order was matched (partially or not) and error if it occurs
func (o *OrderBook) matchOrder(orderPrice float64, order *Order, offers *orderMap) (bool, error) {
	//o.matchMutex.Lock()
	//defer o.matchMutex.Unlock()
	// this method shouldn't handle stop orders
	// we only have to take care of AON param (FOK will be handled in submit because of IOC) & market/limit types
	var matched bool

	var buyer, seller uuid.UUID
	var bidOrderID, askOrderID uint64
	buying := order.IsBid()
	if buying {
		buyer = order.CustomerID
		bidOrderID = order.ID
	} else {
		seller = order.CustomerID
		askOrderID = order.ID
	}

	removeOrders := make([]uint64, 0)

	defer func() {
		for _, orderID := range removeOrders {
			o.removeFromBooks(orderID)
		}
	}()

	currentAON := order.Params.Is(ParamAON)
	for iter := offers.Iterator(); iter.Valid(); iter.Next() {
		oppositeTracker := iter.Key()
		oppositeOrder, ok := o.getActiveOrder(oppositeTracker.OrderID)
		if !ok {
			panic("should NEVER happen - tracker exists but active order does not")
		}
		oppositeAON := oppositeOrder.Params.Is(ParamAON)

		if oppositeOrder.IsCancelled() {
			removeOrders = append(removeOrders, oppositeOrder.ID) // mark order for removal
			continue                                              // don't match with this order
		}

		qty := min(order.UnfilledQty(), oppositeOrder.UnfilledQty())
		// ensure AONs are filled completely
		if currentAON && qty != order.UnfilledQty() {
			continue // couldn't find a match - we require AON but couldn't fill the order in one trade
		}
		if oppositeAON && qty != oppositeOrder.UnfilledQty() {
			continue // couldn't find a match - other offer requires AON but our order can't fill it completely
		}

		var price apd.Decimal
		var fPrice float64
		switch order.Type { // look only after the best available price
		case TypeMarket:
			switch oppositeOrder.Type {
			case TypeMarket:
				continue // two opposing market orders are usually forbidden (rejected) - continue matching
			case TypeLimit:
				price = oppositeOrder.Price // crossing the spread
				fPrice = oppositeTracker.Price
			default:
				panicOnOrderType(oppositeOrder)
			}
		case TypeLimit: // if buying buy for less or equal than our price, if selling sell for more or equal to our price
			myPrice := order.Price
			if buying {
				switch oppositeOrder.Type {
				case TypeMarket: // we have a limit, they are selling at our price
					price = myPrice
					fPrice = orderPrice
				case TypeLimit:
					// check if we can cross the spread
					if myPrice.Cmp(&oppositeOrder.Price) < 0 {
						return matched, nil // other prices are going to be even higher than our limit
					} else {
						// our bid is higher or equal to their ask - set price to myPrice
						price = myPrice // e.g. our bid is $20.10, their ask is $20 - trade executes at $20.10
						fPrice = orderPrice
					}
				default:
					panicOnOrderType(oppositeOrder)
				}
			} else { // we're selling
				switch oppositeOrder.Type {
				case TypeMarket: // we have a limit, they are buying at our specified price
					price = myPrice
					fPrice = orderPrice
				case TypeLimit:
					// check if we can cross the spread
					if myPrice.Cmp(&oppositeOrder.Price) > 0 {
						// we can't match since our ask is higher than the best bid
						return matched, nil
					} else {
						// our ask is lower or equal to their bid - match!
						price = oppositeOrder.Price // set price to their bid
						fPrice = oppositeTracker.Price
					}
				default:
					panicOnOrderType(oppositeOrder)
				}
			}
		default:
			panicOnOrderType(*order)
		}
		if buying {
			seller = oppositeOrder.CustomerID
			askOrderID = oppositeOrder.ID
		} else {
			buyer = oppositeOrder.CustomerID
			bidOrderID = oppositeOrder.ID
		}

		order.FilledQty += qty
		oppositeOrder.FilledQty += qty

		matched = true
		if oppositeOrder.UnfilledQty() == 0 { // if the other order is filled completely - remove it from the order book
			removeOrders = append(removeOrders, oppositeOrder.ID)
		} else {
			if err := o.updateActiveOrder(oppositeOrder); err != nil { // otherwise update it
				return matched, err
			}
		}
		o.tradeBook.Enter(Trade{
			Buyer:      buyer,
			Seller:     seller,
			Instrument: o.Instrument,
			Qty:        qty,
			Price:      price,
			Timestamp:  time.Now(),
			BidOrderID: bidOrderID,
			AskOrderID: askOrderID,
		})
		o.SetMarketPrice(price, fPrice)
		if order.IsFilled() {
			return true, nil
		}
	}
	return matched, nil
}

func panicOnOrderType(order Order) {
	panic(fmt.Errorf("order type \"%d\" not implemented", order.Type))
}
