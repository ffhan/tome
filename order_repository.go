package tome

type OrderRepository interface {
	Save(order Order) error
	GetByID(id uint64) (Order, error)
}

var NOPOrderRepository = &nopOrderRepository{}

type nopOrderRepository struct {
}

func (n *nopOrderRepository) Save(order Order) error {
	return nil
}

func (n *nopOrderRepository) GetByID(id uint64) (Order, error) {
	return Order{}, nil
}
