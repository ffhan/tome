package tome

import (
	"github.com/cockroachdb/apd"
	"github.com/google/uuid"
	"math/rand"
	"runtime"
	"testing"
	"time"
)

const instrument = "TEST"

func createOrder(id uint64, oType OrderType, params OrderParams, qty int64, price, stopPrice apd.Decimal, side OrderSide) Order {
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

func setup(coeff int64, exp int32) (*TradeBook, *OrderBook) {
	tb := NewTradeBook(instrument)

	ob := NewOrderBook(instrument, *apd.New(coeff, exp), tb, NOPOrderRepository)
	return tb, ob
}

func TestOrderBook_MarketReject(t *testing.T) {
	_, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeMarket, 0, 5, apd.Decimal{}, apd.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for market order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeMarket, 0, 2, apd.Decimal{}, apd.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for market order, got a match")
	}
}

func TestOrderBook_MarketToLimit(t *testing.T) {
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, 0, 5, *apd.New(2012, -2), apd.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for market order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeMarket, 0, 2, apd.Decimal{}, apd.Decimal{}, SideSell))
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
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeMarket, 0, 2, apd.Decimal{}, apd.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for market order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, 0, 5, *apd.New(2012, -2), apd.Decimal{}, SideBuy))
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
	if ob.orders.Asks.Len() != 0 {
		t.Errorf("expected 0 asks, got %d", ob.orders.Asks.Len())
	}
	if ob.orders.Bids.Len() != 1 {
		t.Errorf("expected 1 bid, got %d", ob.orders.Bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_No_Match(t *testing.T) {
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, 0, 2, *apd.New(2025, -2), apd.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for market order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, 0, 5, *apd.New(2012, -2), apd.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	if len(tb.trades) != 0 {
		t.Errorf("expected no trades, got %d trades", len(tb.trades))
	}
	if ob.orders.Asks.Len() != 1 {
		t.Errorf("expected 1 ask, got %d", ob.orders.Asks.Len())
	}
	if ob.orders.Bids.Len() != 1 {
		t.Errorf("expected 1 bid, got %d", ob.orders.Bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_Match(t *testing.T) {
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, 0, 2, *apd.New(2010, -2), apd.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for market order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, 0, 5, *apd.New(2012, -2), apd.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("expected a match for this order, got a match")
	}
	if len(tb.trades) != 1 {
		t.Errorf("expected a trade, got %d trades", len(tb.trades))
	}
	if ob.orders.Asks.Len() != 0 {
		t.Errorf("expected 0 asks, got %d", ob.orders.Asks.Len())
	}
	if ob.orders.Bids.Len() != 1 {
		t.Errorf("expected 1 bid, got %d", ob.orders.Bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_Match_FullQty(t *testing.T) {
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, 0, 5, *apd.New(2012, -2), apd.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for market order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, 0, 5, *apd.New(2012, -2), apd.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("expected a match for this order, got a match")
	}
	if len(tb.trades) != 1 {
		t.Errorf("expected a trade, got %d trades", len(tb.trades))
	}
	if ob.orders.Asks.Len() != 0 {
		t.Errorf("expected 0 asks, got %d", ob.orders.Asks.Len())
	}
	if ob.orders.Bids.Len() != 0 {
		t.Errorf("expected 0 bids, got %d", ob.orders.Bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_First_AON_Reject(t *testing.T) {
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, ParamAON, 5, *apd.New(2010, -2), apd.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, 0, 2, *apd.New(2012, -2), apd.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for  order, got a match")
	}
	if len(tb.trades) != 0 {
		t.Errorf("expected no trades, got %d trades", len(tb.trades))
	}
	if ob.orders.Asks.Len() != 1 {
		t.Errorf("expected 1 ask, got %d", ob.orders.Asks.Len())
	}
	if ob.orders.Bids.Len() != 1 {
		t.Errorf("expected 1 bid, got %d", ob.orders.Bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_Second_AON_Reject(t *testing.T) {
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, 0, 2, *apd.New(2010, -2), apd.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, ParamAON, 5, *apd.New(2012, -2), apd.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for  order, got a match")
	}
	if len(tb.trades) != 0 {
		t.Errorf("expected no trades, got %d trades", len(tb.trades))
	}
	if ob.orders.Asks.Len() != 1 {
		t.Errorf("expected 1 ask, got %d", ob.orders.Asks.Len())
	}
	if ob.orders.Bids.Len() != 1 {
		t.Errorf("expected 1 bid, got %d", ob.orders.Bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_Both_AON_Reject(t *testing.T) {
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, ParamAON, 2, *apd.New(2010, -2), apd.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, ParamAON, 5, *apd.New(2012, -2), apd.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for  order, got a match")
	}
	if len(tb.trades) != 0 {
		t.Errorf("expected no trades, got %d trades", len(tb.trades))
	}
	if ob.orders.Asks.Len() != 1 {
		t.Errorf("expected 1 ask, got %d", ob.orders.Asks.Len())
	}
	if ob.orders.Bids.Len() != 1 {
		t.Errorf("expected 1 bid, got %d", ob.orders.Bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_Both_AON(t *testing.T) {
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, ParamAON, 5, *apd.New(2010, -2), apd.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, ParamAON, 5, *apd.New(2012, -2), apd.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("expected a match for this order, got no match")
	}
	if len(tb.trades) != 1 {
		t.Errorf("expected a trade, got %d trades", len(tb.trades))
	}
	if ob.orders.Asks.Len() != 0 {
		t.Errorf("expected 0 asks, got %d", ob.orders.Asks.Len())
	}
	if ob.orders.Bids.Len() != 0 {
		t.Errorf("expected 0 bids, got %d", ob.orders.Bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_First_AON(t *testing.T) {
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, ParamAON, 3, *apd.New(2010, -2), apd.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, 0, 5, *apd.New(2012, -2), apd.Decimal{}, SideBuy))
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
	if ob.orders.Asks.Len() != 0 {
		t.Errorf("expected 0 asks, got %d", ob.orders.Asks.Len())
	}
	if ob.orders.Bids.Len() != 1 {
		t.Errorf("expected 1 bid, got %d", ob.orders.Bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_Second_AON(t *testing.T) {
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, 0, 3, *apd.New(2010, -2), apd.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, ParamAON, 2, *apd.New(2012, -2), apd.Decimal{}, SideBuy))
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
	if ob.orders.Asks.Len() != 1 {
		t.Errorf("expected 1 ask, got %d", ob.orders.Asks.Len())
	}
	if ob.orders.Bids.Len() != 0 {
		t.Errorf("expected 0 bids, got %d", ob.orders.Bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_First_IOC_Reject(t *testing.T) {
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, ParamIOC, 3, *apd.New(2010, -2), apd.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	if ob.orders.Asks.Len() != 0 {
		t.Fatalf("expected no asks, got %d", ob.orders.Asks.Len())
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, 0, 2, *apd.New(2012, -2), apd.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	if len(tb.trades) != 0 {
		t.Errorf("expected no trades, got %d trades", len(tb.trades))
	}
	if ob.orders.Asks.Len() != 0 {
		t.Errorf("expected 0 asks, got %d", ob.orders.Asks.Len())
	}
	if ob.orders.Bids.Len() != 1 {
		t.Errorf("expected 1 bid, got %d", ob.orders.Bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_Second_IOC(t *testing.T) {
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, 0, 3, *apd.New(2010, -2), apd.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, ParamIOC, 2, *apd.New(2012, -2), apd.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("expected a match for this order, got no matches")
	}
	if len(tb.trades) != 1 {
		t.Errorf("expected no trades, got %d trades", len(tb.trades))
	}
	if ob.orders.Asks.Len() != 1 {
		t.Errorf("expected 1 ask, got %d", ob.orders.Asks.Len())
	}
	if ob.orders.Bids.Len() != 0 {
		t.Errorf("expected 0 bids, got %d", ob.orders.Bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_Second_IOC_CancelCheck(t *testing.T) {
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, 0, 3, *apd.New(2010, -2), apd.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, ParamIOC, 5, *apd.New(2012, -2), apd.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("expected a match for this order, got no matches")
	}
	if len(tb.trades) != 1 {
		t.Errorf("expected no trades, got %d trades", len(tb.trades))
	}
	if ob.orders.Asks.Len() != 0 {
		t.Errorf("expected 0 asks, got %d", ob.orders.Asks.Len())
	}
	if ob.orders.Bids.Len() != 0 {
		t.Errorf("expected 0 bids, got %d", ob.orders.Bids.Len())
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
	_, ob := setup(2025, -2)

	type orderData struct {
		Type      OrderType
		Params    OrderParams
		Qty       int64
		Price     apd.Decimal
		StopPrice apd.Decimal
		Side      OrderSide
	}

	data := [...]orderData{
		{TypeLimit, 0, 5, *apd.New(2010, -2), apd.Decimal{}, SideBuy},
		{TypeMarket, ParamAON, 11, apd.Decimal{}, apd.Decimal{}, SideBuy},
		{TypeLimit, 0, 2, *apd.New(2010, -2), apd.Decimal{}, SideBuy},
		{TypeLimit, 0, 2, *apd.New(2065, -2), apd.Decimal{}, SideBuy},
		{TypeMarket, 0, 4, apd.Decimal{}, apd.Decimal{}, SideBuy},
	}

	for i, d := range data {
		_, _ = ob.Add(createOrder(uint64(i+1), d.Type, d.Params, d.Qty, d.Price, d.StopPrice, d.Side))
	}

	sorted := []int{1, 4, 3, 0, 2}

	i := 0
	for iter := ob.orders.Bids.Iterator(); iter.Valid(); iter.Next() {
		order := ob.activeOrders[iter.Key().OrderID]

		expectedData := data[sorted[i]]

		var priceEq, stopPriceEq apd.Decimal
		if _, err := BaseContext.Cmp(&priceEq, &expectedData.Price, &order.Price); err != nil {
			t.Fatal(err)
		}
		if _, err := BaseContext.Cmp(&stopPriceEq, &expectedData.StopPrice, &order.StopPrice); err != nil {
			t.Fatal(err)
		}

		equals := uint64(sorted[i]+1) == order.ID && expectedData.Type == order.Type && expectedData.Params == order.Params && expectedData.Qty == order.Qty && priceEq.IsZero() && stopPriceEq.IsZero() && expectedData.Side == order.Side
		if !equals {
			t.Errorf("expected order ID %d to be in place %d, got a different order", sorted[i]+1, i)
		}

		i += 1
		t.Logf("%+v", order)
	}
}

func TestOrderBook_Add_Asks(t *testing.T) {
	// test order sorting
	_, ob := setup(2025, -2)

	type orderData struct {
		Type      OrderType
		Params    OrderParams
		Qty       int64
		Price     apd.Decimal
		StopPrice apd.Decimal
		Side      OrderSide
	}

	data := [...]orderData{
		{TypeLimit, 0, 7, *apd.New(2000, -2), apd.Decimal{}, SideSell},
		{TypeLimit, 0, 2, *apd.New(2013, -2), apd.Decimal{}, SideSell},
		{TypeLimit, 0, 8, *apd.New(2000, -2), apd.Decimal{}, SideSell},
		{TypeMarket, 0, 9, apd.Decimal{}, apd.Decimal{}, SideSell},
		{TypeLimit, 0, 3, *apd.New(2055, -2), apd.Decimal{}, SideSell},
	}

	for i, d := range data {
		_, _ = ob.Add(createOrder(uint64(i+1), d.Type, d.Params, d.Qty, d.Price, d.StopPrice, d.Side))
	}

	sorted := []int{3, 0, 2, 1, 4}

	i := 0
	for iter := ob.orders.Asks.Iterator(); iter.Valid(); iter.Next() {
		order := ob.activeOrders[iter.Key().OrderID]

		expectedData := data[sorted[i]]

		var priceEq, stopPriceEq apd.Decimal
		if _, err := BaseContext.Cmp(&priceEq, &expectedData.Price, &order.Price); err != nil {
			t.Fatal(err)
		}
		if _, err := BaseContext.Cmp(&stopPriceEq, &expectedData.StopPrice, &order.StopPrice); err != nil {
			t.Fatal(err)
		}

		equals := uint64(sorted[i]+1) == order.ID && expectedData.Type == order.Type && expectedData.Params == order.Params && expectedData.Qty == order.Qty && priceEq.IsZero() && stopPriceEq.IsZero() && expectedData.Side == order.Side
		if !equals {
			t.Errorf("expected order ID %d to be in place %d, got a different order", sorted[i]+1, i)
		}

		i += 1
		t.Logf("%+v", order)
	}
}

func TestOrderBook_Add_MarketPrice_Change(t *testing.T) {
	_, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, 0, 2, *apd.New(2010, -2), apd.Decimal{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for market order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, 0, 5, *apd.New(2012, -2), apd.Decimal{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("expected a match for this order, got a match")
	}
	var eq apd.Decimal
	if _, err := BaseContext.Cmp(&eq, &ob.marketPrice, apd.New(2012, -2)); err != nil {
		t.Fatal(err)
	}
	if !eq.IsZero() {
		t.Errorf("expected market price to be %f, got %s", 20.12, ob.marketPrice.String())
	}
}

func BenchmarkOrderBook_Add(b *testing.B) {
	//ballast := make([]byte, 1<<30) // 1GB of memory ballast, to reduce round trips to the kernel
	//_ = ballast

	var match bool
	var err error
	_, ob := setup(2025, -2)

	orders := make([]Order, b.N)
	for i := range orders {
		order := createRandomOrder(i + 1)
		orders[i] = order
	}
	b.Logf("b.N: %d bids: %d asks: %d orders: %d ", b.N, ob.orders.Bids.Len(), ob.orders.Asks.Len(), len(ob.activeOrders))

	measureMemory(b)
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		match, err = ob.Add(orders[i])
	}
	b.StopTimer()

	_ = match
	_ = err
	b.Logf("orders len: %d bids len: %d asks len: %d", len(ob.activeOrders), ob.orders.Bids.Len(), ob.orders.Asks.Len())

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
	isMarket := rand.Int()%20 == 0
	isBuy := rand.Int()%2 == 0
	isAON := rand.Int()%20 == 0
	isIOC := rand.Int()%25 == 0

	qty := int64(rand.Int()%190) + 10
	price := apd.New(int64(2025+rand.Intn(200)-100), -2)

	oType := TypeLimit
	if isMarket {
		price = apd.New(0, 0)
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
		Price:      *price,
		StopPrice:  apd.Decimal{},
		Side:       oSide,
		Cancelled:  false,
	}
	return order
}
