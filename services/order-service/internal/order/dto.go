package order

type AddToCartRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type UpdateCartItemRequest struct {
	Quantity int `json:"quantity"`
}

type CheckoutRequest struct {
	ShippingAddressID string `json:"shipping_address_id"`
	WarehouseID       string `json:"warehouse_id"`
	CourierName       string `json:"courier_name"`
	VoucherCode       string `json:"voucher_code,omitempty"`
}

type UpdateOrderStatusRequest struct {
	StatusCode string `json:"status_code"` // AWAITING_PAYMENT, PROCESSING, SHIPPED, COMPLETED, CANCELLED
	Notes      string `json:"notes"`
}

type CancelOrderRequest struct {
	Reason string `json:"reason"`
}

type AddOrderNoteRequest struct {
	NoteContent string `json:"note_content"`
}
