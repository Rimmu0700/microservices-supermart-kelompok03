package order

import (
	"time"
)

type Cart struct {
	ID         string     `db:"id" json:"id"`
	UserID     string     `db:"user_id" json:"user_id"`
	CustomerID string     `db:"-" json:"customer_id"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
	Items      []CartItem `json:"items,omitempty"`
}

type CartItem struct {
	ID           string    `db:"id" json:"id"`
	CartID       string    `db:"cart_id" json:"cart_id"`
	ProductID    string    `db:"product_id" json:"product_id"`
	Quantity     int       `db:"quantity" json:"quantity"`
	ProductTitle string    `db:"product_title" json:"product_title"`
	ProductSKU   string    `db:"product_sku" json:"product_sku"`
	ProductPrice float64   `db:"product_price" json:"product_price"`
	ImageURL     string    `db:"image_url" json:"image_url"`
	Subtotal     float64   `json:"subtotal"`
}

type OrderStatus struct {
	ID         string `db:"id" json:"id"`
	StatusCode string `db:"status_code" json:"status_code"`
	StatusName string `db:"status_name" json:"status_name"`
}

type Order struct {
	ID                string    `db:"id" json:"id"`
	OrderNumber       string    `db:"order_number" json:"order_number"`
	CustomerID        string    `db:"customer_id" json:"customer_id"`
	ShippingAddressID string    `db:"shipping_address_id" json:"shipping_address_id"`
	OrderStatusID     string    `db:"order_status_id" json:"order_status_id"`
	WarehouseID       string    `db:"warehouse_id" json:"warehouse_id"`
	TotalGrossAmount  float64   `db:"total_gross_amount" json:"total_gross_amount"`
	DiscountAmount    float64   `db:"discount_amount" json:"discount_amount"`
	TaxAmount         float64   `db:"tax_amount" json:"tax_amount"`
	ShippingFee       float64   `db:"shipping_fee" json:"shipping_fee"`
	TotalNetAmount    float64   `db:"total_net_amount" json:"total_net_amount"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time `db:"-" json:"updated_at,omitempty"`

	// Joined fields
	StatusCode      string `db:"status_code" json:"status_code,omitempty"`
	StatusName      string `db:"status_name" json:"status_name,omitempty"`
	CustomerName    string `db:"customer_name" json:"customer_name,omitempty"`
	CustomerEmail   string `db:"customer_email" json:"customer_email,omitempty"`
	WarehouseName   string `db:"warehouse_name" json:"warehouse_name,omitempty"`
	ShippingAddress string `db:"shipping_address" json:"shipping_address,omitempty"`
}

type OrderItem struct {
	ID           string  `db:"id" json:"id"`
	OrderID      string  `db:"order_id" json:"order_id"`
	ProductID    string  `db:"product_id" json:"product_id"`
	Quantity     int     `db:"quantity" json:"quantity"`
	UnitPrice    float64 `db:"unit_price" json:"unit_price"`
	Subtotal     float64 `db:"subtotal" json:"subtotal"`
	ProductTitle string  `db:"product_title" json:"product_title,omitempty"`
	ProductSKU   string  `db:"product_sku" json:"product_sku,omitempty"`
}

type OrderStatusHistory struct {
	ID            string    `db:"id" json:"id"`
	OrderID       string    `db:"order_id" json:"order_id"`
	OrderStatusID string    `db:"order_status_id" json:"order_status_id"`
	Notes         *string   `db:"notes" json:"notes"`
	ChangedAt     time.Time `db:"changed_at" json:"changed_at"`
	CreatedAt     time.Time `db:"-" json:"created_at,omitempty"`
	StatusName    string    `db:"status_name" json:"status_name,omitempty"`
}

type OrderShippingDetail struct {
	ID             string  `db:"id" json:"id"`
	OrderID        string  `db:"order_id" json:"order_id"`
	CourierName    string  `db:"courier_name" json:"courier_name"`
	TrackingNumber *string `db:"tracking_number" json:"tracking_number"`
	ShippingCost   float64 `db:"shipping_cost" json:"shipping_cost"`
}

type OrderNote struct {
	ID           string    `db:"id" json:"id"`
	OrderID      string    `db:"order_id" json:"order_id"`
	AuthorUserID string    `db:"author_user_id" json:"author_user_id"`
	NoteContent  string    `db:"note_content" json:"note_content"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	AuthorName   string    `db:"author_name" json:"author_name,omitempty"`
}

type OrderCancellation struct {
	ID                string    `db:"id" json:"id"`
	OrderID           string    `db:"order_id" json:"order_id"`
	CancelReason      string    `db:"cancel_reason" json:"cancel_reason"`
	Reason            string    `db:"-" json:"reason,omitempty"`
	CancelledByUserID string    `db:"cancelled_by_user_id" json:"cancelled_by_user_id"`
	CancelledAt       time.Time `db:"cancelled_at" json:"cancelled_at"`
}
