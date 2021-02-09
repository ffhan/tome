package tome

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"math/rand"
	"runtime"
	"testing"
	"time"
)

const instrument = "TEST"

func createOrder(id uint64, oType OrderType, params OrderParams, qty int64, price, stopPrice decimal.Decimal, side OrderSide) Order {
	return Order{
		ID:         id,
		Instrument: instrument,
		CustomerID: uuid.UUID{},
		Timestamp:  time.Now(),
		Type:       oType,
		Params:     params,
		Qty:        qty,
		FilledQty:  0,
		Price:      price,
		StopPrice:  stopPrice,
		Side:       side,
	}
}

func setup(price float64) (*TradeBook, *OrderBook) {
	tb := NewTradeBook(instrument)
	ob := NewOrderBook(instrument, decimal.NewFromFloat(price), tb, NOPOrderRepository)
	return tb, ob
}

func TestOrderBook_MarketReject(t *testing.T) {
	_, ob := setup(20.25)

	matched, err := ob.Add(createOrder(1, TypeMarket, 0, 5, decimal.Decimal{}, decimal.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for market order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeMarket, 0, 2, decimal.Decimal{}, decimal.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for market order, got a match")
	}
}

func TestOrderBook_MarketToLimit(t *testing.T) {
	tb, ob := setup(20.25)

	matched, err := ob.Add(createOrder(1, TypeLimit, 0, 5, decimal.NewFromFloat(20.12), decimal.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for market order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeMarket, 0, 2, decimal.Decimal{}, decimal.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("expected match for market order, got no match")
	}
	if len(tb.trades) == 0 {
		t.Fatal("expected one trade, got none")
	}
	t.Logf("trade: %+v", tb.trades[0])
}

func TestOrderBook_LimitToMarket(t *testing.T) {
	tb, ob := setup(20.25)

	matched, err := ob.Add(createOrder(1, TypeMarket, 0, 2, decimal.Decimal{}, decimal.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for market order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, 0, 5, decimal.NewFromFloat(20.12), decimal.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("expected match for market order, got no match")
	}
	if len(tb.trades) == 0 {
		t.Fatal("expected one trade, got none")
	}
	t.Logf("trade: %+v", tb.trades[0])
	if ob.asks.Len() != 0 {
		t.Errorf("expected 0 asks, got %d", ob.asks.Len())
	}
	if ob.bids.Len() != 1 {
		t.Errorf("expected 1 bid, got %d", ob.bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_No_Match(t *testing.T) {
	tb, ob := setup(20.25)

	matched, err := ob.Add(createOrder(1, TypeLimit, 0, 2, decimal.NewFromFloat(20.25), decimal.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for market order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, 0, 5, decimal.NewFromFloat(20.12), decimal.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	if len(tb.trades) != 0 {
		t.Errorf("expected no trades, got %d trades", len(tb.trades))
	}
	if ob.asks.Len() != 1 {
		t.Errorf("expected 1 ask, got %d", ob.asks.Len())
	}
	if ob.bids.Len() != 1 {
		t.Errorf("expected 1 bid, got %d", ob.bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_Match(t *testing.T) {
	tb, ob := setup(20.25)

	matched, err := ob.Add(createOrder(1, TypeLimit, 0, 2, decimal.NewFromFloat(20.10), decimal.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for market order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, 0, 5, decimal.NewFromFloat(20.12), decimal.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("expected a match for this order, got a match")
	}
	if len(tb.trades) != 1 {
		t.Errorf("expected a trade, got %d trades", len(tb.trades))
	}
	if ob.asks.Len() != 0 {
		t.Errorf("expected 0 asks, got %d", ob.asks.Len())
	}
	if ob.bids.Len() != 1 {
		t.Errorf("expected 1 bid, got %d", ob.bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_Match_FullQty(t *testing.T) {
	tb, ob := setup(20.25)

	matched, err := ob.Add(createOrder(1, TypeLimit, 0, 5, decimal.NewFromFloat(20.12), decimal.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for market order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, 0, 5, decimal.NewFromFloat(20.12), decimal.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("expected a match for this order, got a match")
	}
	if len(tb.trades) != 1 {
		t.Errorf("expected a trade, got %d trades", len(tb.trades))
	}
	if ob.asks.Len() != 0 {
		t.Errorf("expected 0 asks, got %d", ob.asks.Len())
	}
	if ob.bids.Len() != 0 {
		t.Errorf("expected 0 bids, got %d", ob.bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_First_AON_Reject(t *testing.T) {
	tb, ob := setup(20.25)

	matched, err := ob.Add(createOrder(1, TypeLimit, ParamAON, 5, decimal.NewFromFloat(20.10), decimal.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, 0, 2, decimal.NewFromFloat(20.12), decimal.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for  order, got a match")
	}
	if len(tb.trades) != 0 {
		t.Errorf("expected no trades, got %d trades", len(tb.trades))
	}
	if ob.asks.Len() != 1 {
		t.Errorf("expected 1 ask, got %d", ob.asks.Len())
	}
	if ob.bids.Len() != 1 {
		t.Errorf("expected 1 bid, got %d", ob.bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_Second_AON_Reject(t *testing.T) {
	tb, ob := setup(20.25)

	matched, err := ob.Add(createOrder(1, TypeLimit, 0, 2, decimal.NewFromFloat(20.10), decimal.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, ParamAON, 5, decimal.NewFromFloat(20.12), decimal.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for  order, got a match")
	}
	if len(tb.trades) != 0 {
		t.Errorf("expected no trades, got %d trades", len(tb.trades))
	}
	if ob.asks.Len() != 1 {
		t.Errorf("expected 1 ask, got %d", ob.asks.Len())
	}
	if ob.bids.Len() != 1 {
		t.Errorf("expected 1 bid, got %d", ob.bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_Both_AON_Reject(t *testing.T) {
	tb, ob := setup(20.25)

	matched, err := ob.Add(createOrder(1, TypeLimit, ParamAON, 2, decimal.NewFromFloat(20.10), decimal.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, ParamAON, 5, decimal.NewFromFloat(20.12), decimal.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for  order, got a match")
	}
	if len(tb.trades) != 0 {
		t.Errorf("expected no trades, got %d trades", len(tb.trades))
	}
	if ob.asks.Len() != 1 {
		t.Errorf("expected 1 ask, got %d", ob.asks.Len())
	}
	if ob.bids.Len() != 1 {
		t.Errorf("expected 1 bid, got %d", ob.bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_Both_AON(t *testing.T) {
	tb, ob := setup(20.25)

	matched, err := ob.Add(createOrder(1, TypeLimit, ParamAON, 5, decimal.NewFromFloat(20.10), decimal.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, ParamAON, 5, decimal.NewFromFloat(20.12), decimal.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("expected a match for this order, got no match")
	}
	if len(tb.trades) != 1 {
		t.Errorf("expected a trade, got %d trades", len(tb.trades))
	}
	if ob.asks.Len() != 0 {
		t.Errorf("expected 0 asks, got %d", ob.asks.Len())
	}
	if ob.bids.Len() != 0 {
		t.Errorf("expected 0 bids, got %d", ob.bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_First_AON(t *testing.T) {
	tb, ob := setup(20.25)

	matched, err := ob.Add(createOrder(1, TypeLimit, ParamAON, 3, decimal.NewFromFloat(20.10), decimal.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, 0, 5, decimal.NewFromFloat(20.12), decimal.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("expected a match for this order, got no match")
	}
	if len(tb.trades) != 1 {
		t.Errorf("expected a trade, got %d trades", len(tb.trades))
	} else {
		t.Logf("trade: %+v", tb.trades[0])
	}
	if ob.asks.Len() != 0 {
		t.Errorf("expected 0 asks, got %d", ob.asks.Len())
	}
	if ob.bids.Len() != 1 {
		t.Errorf("expected 1 bid, got %d", ob.bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_Second_AON(t *testing.T) {
	tb, ob := setup(20.25)

	matched, err := ob.Add(createOrder(1, TypeLimit, 0, 3, decimal.NewFromFloat(20.10), decimal.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, ParamAON, 2, decimal.NewFromFloat(20.12), decimal.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("expected a match for this order, got no match")
	}
	if len(tb.trades) != 1 {
		t.Errorf("expected a trade, got %d trades", len(tb.trades))
	} else {
		t.Logf("trade: %+v", tb.trades[0])
	}
	if ob.asks.Len() != 1 {
		t.Errorf("expected 1 ask, got %d", ob.asks.Len())
	}
	if ob.bids.Len() != 0 {
		t.Errorf("expected 0 bids, got %d", ob.bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_First_IOC_Reject(t *testing.T) {
	tb, ob := setup(20.25)

	matched, err := ob.Add(createOrder(1, TypeLimit, ParamIOC, 3, decimal.NewFromFloat(20.10), decimal.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	if ob.asks.Len() != 0 {
		t.Fatalf("expected no asks, got %d", ob.asks.Len())
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, 0, 2, decimal.NewFromFloat(20.12), decimal.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	if len(tb.trades) != 0 {
		t.Errorf("expected no trades, got %d trades", len(tb.trades))
	}
	if ob.asks.Len() != 0 {
		t.Errorf("expected 0 asks, got %d", ob.asks.Len())
	}
	if ob.bids.Len() != 1 {
		t.Errorf("expected 1 bid, got %d", ob.bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_Second_IOC(t *testing.T) {
	tb, ob := setup(20.25)

	matched, err := ob.Add(createOrder(1, TypeLimit, 0, 3, decimal.NewFromFloat(20.10), decimal.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, ParamIOC, 2, decimal.NewFromFloat(20.12), decimal.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("expected a match for this order, got no matches")
	}
	if len(tb.trades) != 1 {
		t.Errorf("expected no trades, got %d trades", len(tb.trades))
	}
	if ob.asks.Len() != 1 {
		t.Errorf("expected 1 ask, got %d", ob.asks.Len())
	}
	if ob.bids.Len() != 0 {
		t.Errorf("expected 0 bids, got %d", ob.bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_Second_IOC_CancelCheck(t *testing.T) {
	tb, ob := setup(20.25)

	matched, err := ob.Add(createOrder(1, TypeLimit, 0, 3, decimal.NewFromFloat(20.10), decimal.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, ParamIOC, 5, decimal.NewFromFloat(20.12), decimal.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("expected a match for this order, got no matches")
	}
	if len(tb.trades) != 1 {
		t.Errorf("expected no trades, got %d trades", len(tb.trades))
	}
	if ob.asks.Len() != 0 {
		t.Errorf("expected 0 asks, got %d", ob.asks.Len())
	}
	if ob.bids.Len() != 0 {
		t.Errorf("expected 0 bids, got %d", ob.bids.Len())
	}
	order := ob.activeOrders[1]
	if !order.IsCancelled() {
		t.Log("IOC order should be cancelled after partial fill")
	}
	if order.FilledQty != 3 {
		t.Logf("expected filled qty for IOC order %d, got %d", 3, order.FilledQty)
	}
	t.Logf("%+v", order)
}

func TestOrderBook_Add_Bids(t *testing.T) {
	// test order sorting
	_, ob := setup(20.25)

	type orderData struct {
		Type      OrderType
		Params    OrderParams
		Qty       int64
		Price     decimal.Decimal
		StopPrice decimal.Decimal
		Side      OrderSide
	}

	data := [...]orderData{
		{TypeLimit, 0, 5, decimal.NewFromFloat(20.10), decimal.Decimal{}, SideBuy},
		{TypeMarket, ParamAON, 11, decimal.Decimal{}, decimal.Decimal{}, SideBuy},
		{TypeLimit, 0, 2, decimal.NewFromFloat(20.10), decimal.Decimal{}, SideBuy},
		{TypeLimit, 0, 2, decimal.NewFromFloat(20.65), decimal.Decimal{}, SideBuy},
		{TypeMarket, 0, 4, decimal.Decimal{}, decimal.Decimal{}, SideBuy},
	}

	for i, d := range data {
		_, _ = ob.Add(createOrder(uint64(i+1), d.Type, d.Params, d.Qty, d.Price, d.StopPrice, d.Side))
	}

	sorted := []int{1, 4, 3, 0, 2}

	i := 0
	for iter := ob.bids.Iterator(); iter.Valid(); iter.Next() {
		order := ob.activeOrders[iter.Key().OrderID]

		expectedData := data[sorted[i]]
		equals := uint64(sorted[i]+1) == order.ID && expectedData.Type == order.Type && expectedData.Params == order.Params && expectedData.Qty == order.Qty && expectedData.Price == order.Price && expectedData.StopPrice == order.StopPrice && expectedData.Side == order.Side
		if !equals {
			t.Errorf("expected order ID %d to be in place %d, got a different order", sorted[i]+1, i)
		}

		i += 1
		t.Logf("%+v", order)
	}
}

func TestOrderBook_Add_Asks(t *testing.T) {
	// test order sorting
	_, ob := setup(20.25)

	type orderData struct {
		Type      OrderType
		Params    OrderParams
		Qty       int64
		Price     decimal.Decimal
		StopPrice decimal.Decimal
		Side      OrderSide
	}

	data := [...]orderData{
		{TypeLimit, 0, 7, decimal.NewFromFloat(20.00), decimal.Decimal{}, SideSell},
		{TypeLimit, 0, 2, decimal.NewFromFloat(20.13), decimal.Decimal{}, SideSell},
		{TypeLimit, 0, 8, decimal.NewFromFloat(20.00), decimal.Decimal{}, SideSell},
		{TypeMarket, 0, 9, decimal.Decimal{}, decimal.Decimal{}, SideSell},
		{TypeLimit, 0, 3, decimal.NewFromFloat(20.55), decimal.Decimal{}, SideSell},
	}

	for i, d := range data {
		_, _ = ob.Add(createOrder(uint64(i+1), d.Type, d.Params, d.Qty, d.Price, d.StopPrice, d.Side))
	}

	sorted := []int{3, 0, 2, 1, 4}

	i := 0
	for iter := ob.asks.Iterator(); iter.Valid(); iter.Next() {
		order := ob.activeOrders[iter.Key().OrderID]

		expectedData := data[sorted[i]]
		equals := uint64(sorted[i]+1) == order.ID && expectedData.Type == order.Type && expectedData.Params == order.Params && expectedData.Qty == order.Qty && expectedData.Price == order.Price && expectedData.StopPrice == order.StopPrice && expectedData.Side == order.Side
		if !equals {
			t.Errorf("expected order ID %d to be in place %d, got a different order", sorted[i]+1, i)
		}

		i += 1
		t.Logf("%+v", order)
	}
}

func TestOrderBook_Add_MarketPrice_Change(t *testing.T) {
	_, ob := setup(20.25)

	matched, err := ob.Add(createOrder(1, TypeLimit, 0, 2, decimal.NewFromFloat(20.10), decimal.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for market order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, 0, 5, decimal.NewFromFloat(20.12), decimal.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("expected a match for this order, got a match")
	}
	if !ob.marketPrice.Equal(decimal.NewFromFloat(20.12)) {
		t.Errorf("expected market price to be %f, got %s", 20.12, ob.marketPrice.String())
	}
}

func BenchmarkOrderBook_Add(b *testing.B) {
	b.ReportAllocs()
	var match bool
	var err error
	_, ob := setup(20.25)

	orders := make([]Order, b.N)
	for i := range orders {
		order := createRandomOrder(i + 1)
		orders[i] = order
	}
	b.Logf("b.N: %d bids: %d asks: %d orders: %d ", b.N, ob.bids.Len(), ob.asks.Len(), len(ob.activeOrders))
	runtime.GC()

	measureMemory(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		match, err = ob.Add(orders[i])
	}
	b.StopTimer()

	_ = match
	_ = err
	b.Logf("orders len: %d bids len: %d asks len: %d", len(ob.activeOrders), ob.bids.Len(), ob.asks.Len())

	measureMemory(b)
}

func measureMemory(b *testing.B) {
	var endMem runtime.MemStats
	runtime.ReadMemStats(&endMem)
	b.Logf("total: %dB stack: %dB GCCPUFraction: %f total heap alloc: %dB", endMem.TotalAlloc,
		endMem.StackInuse, endMem.GCCPUFraction, endMem.HeapAlloc)
	b.Logf("alloc: %dB heap inuse: %dB", endMem.Alloc, endMem.HeapInuse)
}

func createRandomOrder(i int) Order {
	isMarket := rand.Int()%8 == 0
	isBuy := rand.Int()%2 == 0
	isAON := rand.Int()%20 == 0
	isIOC := rand.Int()%25 == 0

	qty := int64(rand.Int()%190) + 10
	price := 20.25 + (rand.Float64()-0.5)*4

	oType := TypeLimit
	if isMarket {
		price = 0
		oType = TypeMarket
	}
	var params OrderParams
	if isAON {
		//params |= ParamAON
	}
	if isIOC {
		//params |= ParamIOC
	}
	oSide := SideSell
	if isBuy {
		oSide = SideBuy
	}

	order := Order{
		ID:         uint64(i + 1),
		Instrument: instrument,
		CustomerID: uuid.UUID{},
		Timestamp:  time.Now(),
		Type:       oType,
		Params:     params,
		Qty:        qty,
		FilledQty:  0,
		Price:      decimal.NewFromFloat(price),
		StopPrice:  decimal.Decimal{},
		Side:       oSide,
		Cancelled:  false,
	}
	return order
}
