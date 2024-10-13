package main

type Customer struct {
	ID        int     `json:"id"`
	FirstName string  `json:"firstName"`
	LastName  string  `json:"lastName"`
	Email     *string `json:"email"`
}

type Merchant struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Money int

func (m Money) toFloat() float64 {
	return float64(m) / 100.0
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type Product struct {
	ID       int      `json:"id"`
	OrderID  int      `json:"-"`
	Merchant Merchant `json:"merchant"`
	Name     string   `json:"name"`
	Price    Money    `json:"price"`
	Amount   int      `json:"amount"`
}

func (p *Product) toProductV() ProductV {
	return ProductV{
		ID:       p.ID,
		OrderID:  p.OrderID,
		Merchant: p.Merchant,
		Name:     p.Name,
		Price:    p.Price.toFloat(),
		Amount:   p.Amount,
	}
}

// view для цены float64
type ProductV struct {
	ID       int      `json:"id"`
	OrderID  int      `json:"-"`
	Merchant Merchant `json:"merchant"`
	Name     string   `json:"name"`
	Price    float64  `json:"price"`
	Amount   int      `json:"amount"`
}

type ProductRestock struct {
	ProductID int `json:"productId"`
	Amount    int `json:"amount"`
}

type StockAmount struct {
	Amount *int `json:"amount" validate:"required,gte=0"`
}

type Order struct {
	ID       int       `json:"id"`
	Customer Customer  `json:"customer"`
	Products []Product `json:"products"`
	Checkout Money     `json:"checkout"`
}

func (o *Order) calcCheckout() {
	var checkout Money
	for _, p := range o.Products {
		checkout += p.Price * Money(p.Amount)
	}
	o.Checkout = checkout
}

func (o *Order) toOrderV() OrderV {
	psV := make([]ProductV, 0, len(o.Products))
	for _, p := range o.Products {
		pV := p.toProductV()
		psV = append(psV, pV)
	}

	return OrderV{
		ID:       o.ID,
		Customer: o.Customer,
		Products: psV,
		Checkout: o.Checkout.toFloat(),
	}
}

// view для цены float64
type OrderV struct {
	ID       int        `json:"id"`
	Customer Customer   `json:"customer"`
	Products []ProductV `json:"products"`
	Checkout float64    `json:"checkout"`
}
