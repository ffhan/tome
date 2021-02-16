package tome

import (
	"testing"
	"time"
)

func TestOrderContainer_Add(t *testing.T) {
	c := NewOrderContainer(makeComparator(true), makeComparator(false))

	orders := [...]OrderTracker{
		{OrderID: 1, Price: 20.25, Timestamp: time.Now().UnixNano(), Side: SideBuy},
		{OrderID: 2, Price: 20.25, Timestamp: time.Now().UnixNano(), Side: SideSell},
		{OrderID: 3, Price: 20.50, Timestamp: time.Now().UnixNano(), Side: SideBuy},
		{OrderID: 4, Price: 20.45, Timestamp: time.Now().UnixNano(), Side: SideSell},
		{OrderID: 5, Price: 20.10, Timestamp: time.Now().UnixNano(), Side: SideBuy},
		{OrderID: 6, Price: 20.18, Timestamp: time.Now().UnixNano(), Side: SideSell},
		{OrderID: 7, Price: 20.25, Timestamp: time.Now().UnixNano(), Side: SideBuy},
		{OrderID: 8, Price: 20.45, Timestamp: time.Now().UnixNano(), Side: SideSell},
	}

	sortedBids := [...]int{2, 0, 6, 4}
	sortedAsks := [...]int{5, 1, 3, 7}

	for _, o := range orders {
		c.Add(o)
	}

	i := 0
	for iter := c.Bids.Iterator(); iter.Valid(); iter.Next() {
		order := iter.Key()

		expectedID := orders[sortedBids[i]].OrderID
		if order.OrderID != expectedID {
			t.Errorf("expected order ID %d, got %d", expectedID, order.OrderID)
		}

		i += 1
	}

	i = 0
	for iter := c.Asks.Iterator(); iter.Valid(); iter.Next() {
		order := iter.Key()

		expectedID := orders[sortedAsks[i]].OrderID
		if order.OrderID != expectedID {
			t.Errorf("expected order ID %d, got %d", expectedID, order.OrderID)
		}

		i += 1
	}
}

func TestOrderContainer_GetBidsAbove(t *testing.T) {
	c := NewOrderContainer(makeComparator(true), makeComparator(false)) // simulate stop order container

	orders := [...]OrderTracker{
		{OrderID: 1, Price: 20.25, Timestamp: time.Now().UnixNano(), Side: SideBuy},
		{OrderID: 2, Price: 20.25, Timestamp: time.Now().UnixNano(), Side: SideSell},
		{OrderID: 3, Price: 20.50, Timestamp: time.Now().UnixNano(), Side: SideBuy},
		{OrderID: 4, Price: 20.45, Timestamp: time.Now().UnixNano(), Side: SideSell},
		{OrderID: 5, Price: 20.10, Timestamp: time.Now().UnixNano(), Side: SideBuy},
		{OrderID: 6, Price: 20.18, Timestamp: time.Now().UnixNano(), Side: SideSell},
		{OrderID: 7, Price: 20.25, Timestamp: time.Now().UnixNano(), Side: SideBuy},
		{OrderID: 8, Price: 20.45, Timestamp: time.Now().UnixNano(), Side: SideSell},
	}
	results := [...]int{0, 2, 6}

	for _, o := range orders {
		c.Add(o)
	}

	above := c.GetBidsAbove(20.25)

	if len(above) != len(results) {
		t.Fatalf("expected %d results, got %d", len(results), len(above))
	}

	for i, tracker := range above {
		expected := orders[results[i]]

		if tracker.OrderID != expected.OrderID {
			t.Errorf("expected ID %d, got %d", expected.OrderID, tracker.OrderID)
		}
	}
}

func TestOrderContainer_GetAsksBelow(t *testing.T) {
	c := NewOrderContainer(makeComparator(true), makeComparator(false))

	orders := [...]OrderTracker{
		{OrderID: 1, Price: 20.25, Timestamp: time.Now().UnixNano(), Side: SideBuy},
		{OrderID: 2, Price: 20.25, Timestamp: time.Now().UnixNano(), Side: SideSell},
		{OrderID: 3, Price: 20.50, Timestamp: time.Now().UnixNano(), Side: SideBuy},
		{OrderID: 4, Price: 20.45, Timestamp: time.Now().UnixNano(), Side: SideSell},
		{OrderID: 5, Price: 20.10, Timestamp: time.Now().UnixNano(), Side: SideBuy},
		{OrderID: 6, Price: 20.18, Timestamp: time.Now().UnixNano(), Side: SideSell},
		{OrderID: 7, Price: 20.25, Timestamp: time.Now().UnixNano(), Side: SideBuy},
		{OrderID: 8, Price: 20.45, Timestamp: time.Now().UnixNano(), Side: SideSell},
	}

	results := [...]int{1, 5}

	for _, o := range orders {
		c.Add(o)
	}

	above := c.GetAsksBelow(20.25)

	if len(above) != len(results) {
		t.Fatalf("expected %d results, got %d", len(results), len(above))
	}

	for i, tracker := range above {
		expected := orders[results[i]]

		if tracker.OrderID != expected.OrderID {
			t.Errorf("expected ID %d, got %d", expected.OrderID, tracker.OrderID)
		}
	}
}

func BenchmarkOrderContainer_Add(b *testing.B) {
	c := NewOrderContainer(makeComparator(true), makeComparator(false))

	orders := make([]OrderTracker, b.N)
	for i := 0; i < b.N; i++ {
		order := createRandomOrder(i + 1)
		price, err := order.Price.Float64()
		if err != nil {
			b.Fatal(err)
		}
		orders[i] = OrderTracker{
			OrderID:   order.ID,
			Type:      order.Type,
			Price:     price,
			Side:      order.Side,
			Timestamp: order.Timestamp.UnixNano(),
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		c.Add(orders[i])
	}
}

func BenchmarkOrderContainer_GetBidsAbove(b *testing.B) {
	c := NewOrderContainer(makeComparator(true), makeComparator(false))

	orders := make([]OrderTracker, b.N)
	for i := 0; i < b.N; i++ {
		order := createRandomOrder(i + 1)
		price, err := order.Price.Float64()
		if err != nil {
			b.Fatal(err)
		}
		orders[i] = OrderTracker{
			OrderID:   order.ID,
			Type:      order.Type,
			Price:     price,
			Side:      order.Side,
			Timestamp: order.Timestamp.UnixNano(),
		}
	}

	for i := 0; i < b.N; i++ {
		c.Add(orders[i])
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = c.GetBidsAbove(orders[i].Price)
	}
}
