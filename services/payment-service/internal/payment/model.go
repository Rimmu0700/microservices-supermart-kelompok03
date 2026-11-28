package payment

import (
	"time"
)

type PaymentMethod struct {
	ID         string `db:"id" json:"id"`
	MethodCode string `db:"method_code" json:"method_code"`
	MethodName string `db:"method_name" json:"method_name"`
}

type PaymentGateway struct {
	ID          string `db:"id" json:"id"`
	GatewayName string `db:"gateway_name" json:"gateway_name"`
	APIEndpoint string `db:"api_endpoint" json:"api_endpoint"`
	IsActive    bool   `db:"is_active" json:"is_active"`
}

type PaymentInvoice struct {
	ID            string     `db:"id" json:"id"`
	InvoiceNumber string     `db:"invoice_number" json:"invoice_number"`
	OrderID       string     `db:"order_id" json:"order_id"`
	CustomerID    string     `db:"customer_id" json:"customer_id"`
	Amount        float64    `db:"amount" json:"amount"`
	PaymentStatus string     `db:"payment_status" json:"payment_status"` // UNPAID, PAID, EXPIRED, CANCELLED
	DueDate       time.Time  `db:"due_date" json:"due_date"`
	PaidAt        *time.Time `db:"paid_at" json:"paid_at"`

	// Joined
	OrderNumber  string `db:"order_number" json:"order_number,omitempty"`
	CustomerName string `db:"customer_name" json:"customer_name,omitempty"`
}

type PaymentTransaction struct {
	ID                   string    `db:"id" json:"id"`
	InvoiceID            string    `db:"invoice_id" json:"invoice_id"`
	PaymentMethodID      string    `db:"payment_method_id" json:"payment_method_id"`
	GatewayID            *string   `db:"gateway_id" json:"gateway_id"`
	TransactionReference string    `db:"transaction_reference" json:"transaction_reference"`
	AmountPaid           float64   `db:"amount_paid" json:"amount_paid"`
	Status               string    `db:"status" json:"status"` // PENDING, SUCCESS, FAILED
	TransactionTime      time.Time `db:"transaction_time" json:"transaction_time"`

	// Joined
	MethodName string `db:"method_name" json:"method_name,omitempty"`
}

type StoreCredit struct {
	ID             string    `db:"id" json:"id"`
	UserID         string    `db:"user_id" json:"user_id"`
	CurrentBalance float64   `db:"balance" json:"balance"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at,omitempty"`
}

type CreditTransaction struct {
	ID              string    `db:"id" json:"id"`
	StoreCreditID   string    `db:"store_credit_id" json:"store_credit_id"`
	Amount          float64   `db:"amount" json:"amount"`
	TransactionType string    `db:"transaction_type" json:"transaction_type"` // CREDIT, DEBIT
	Description     *string   `db:"description" json:"description"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
}

type Refund struct {
	ID                string     `db:"id" json:"id"`
	RefundNumber      string     `db:"refund_number" json:"refund_number"`
	OrderID           string     `db:"order_id" json:"order_id"`
	RequestedByUserID string     `db:"requested_by_user_id" json:"requested_by_user_id"`
	TotalRefundAmount float64    `db:"total_refund_amount" json:"total_refund_amount"`
	RefundStatus      string     `db:"refund_status" json:"refund_status"` // PENDING, APPROVED, REJECTED, COMPLETED
	CreatedAt         time.Time  `db:"created_at" json:"created_at"`
	OrderNumber       string     `db:"order_number" json:"order_number,omitempty"`
}
