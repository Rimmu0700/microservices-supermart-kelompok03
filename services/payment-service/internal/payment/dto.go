package payment

type PayInvoiceRequest struct {
	InvoiceID       string `json:"invoice_id"`
	OrderID         string `json:"order_id"`
	PaymentMethodID string `json:"payment_method_id"`
}

type CreateInvoiceRequest struct {
	OrderID    string  `json:"order_id"`
	CustomerID string  `json:"customer_id"`
	Amount     float64 `json:"amount"`
}

type TopupCreditRequest struct {
	Amount float64 `json:"amount"`
}

type RefundRequest struct {
	OrderID string  `json:"order_id"`
	Amount  float64 `json:"amount"`
	Reason  string  `json:"reason"`
}

type ApproveRefundRequest struct {
	Status string `json:"status"` // APPROVED, REJECTED
}
