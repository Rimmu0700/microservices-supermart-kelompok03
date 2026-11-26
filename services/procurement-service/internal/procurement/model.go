package procurement

import (
	"time"
)

type Supplier struct {
	ID                      string `db:"id" json:"id"`
	SupplierCode            string `db:"supplier_code" json:"supplier_code"`
	CompanyName             string `db:"company_name" json:"company_name"`
	TaxIdentificationNumber string `db:"tax_identification_number" json:"tax_identification_number"`
	Address                 string `db:"address" json:"address"`
}

type SupplierContact struct {
	ID          string `db:"id" json:"id"`
	SupplierID  string `db:"supplier_id" json:"supplier_id"`
	ContactName string `db:"contact_name" json:"contact_name"`
	Email       string `db:"email" json:"email"`
	Phone       string `db:"phone" json:"phone"`
}

type PurchaseOrder struct {
	ID            string    `db:"id" json:"id"`
	PONumber      string    `db:"po_number" json:"po_number"`
	SupplierID    string    `db:"supplier_id" json:"supplier_id"`
	WarehouseID   string    `db:"warehouse_id" json:"warehouse_id"`
	PODate        string    `db:"po_date" json:"po_date"`
	TotalPOAmount float64   `db:"total_po_amount" json:"total_po_amount"`
	Status        string    `db:"status" json:"status"` // PENDING, APPROVED, RECEIVED, CANCELLED

	// Joined
	SupplierName  string `db:"supplier_name" json:"supplier_name,omitempty"`
	WarehouseName string `db:"warehouse_name" json:"warehouse_name,omitempty"`
}

type PurchaseOrderItem struct {
	ID              string  `db:"id" json:"id"`
	PurchaseOrderID string  `db:"purchase_order_id" json:"purchase_order_id"`
	ProductID       string  `db:"product_id" json:"product_id"`
	OrderedQty      int     `db:"ordered_qty" json:"ordered_qty"`
	UnitCost        float64 `db:"unit_cost" json:"unit_cost"`

	// Joined
	ProductTitle string `db:"product_title" json:"product_title,omitempty"`
	ProductSKU   string `db:"product_sku" json:"product_sku,omitempty"`
}

type GoodsReceiptNote struct {
	ID              string    `db:"id" json:"id"`
	GRNNumber       string    `db:"grn_number" json:"grn_number"`
	PurchaseOrderID string    `db:"purchase_order_id" json:"purchase_order_id"`
	ReceivedDate    time.Time `db:"received_date" json:"received_date"`

	// Joined
	PONumber string `db:"po_number" json:"po_number,omitempty"`
}

type GoodsReceiptItem struct {
	ID          string  `db:"id" json:"id"`
	GRNID       string  `db:"grn_id" json:"grn_id"`
	ProductID   string  `db:"product_id" json:"product_id"`
	ReceivedQty int     `db:"received_qty" json:"received_qty"`
	Notes       *string `db:"notes" json:"notes"`
}

type PurchaseInvoice struct {
	ID              string  `db:"id" json:"id"`
	InvoiceNumber   string  `db:"invoice_number" json:"invoice_number"`
	PurchaseOrderID string  `db:"purchase_order_id" json:"purchase_order_id"`
	InvoiceAmount   float64 `db:"invoice_amount" json:"invoice_amount"`
	DueDate         string  `db:"due_date" json:"due_date"`

	// Joined
	PONumber     string `db:"po_number" json:"po_number,omitempty"`
	SupplierName string `db:"supplier_name" json:"supplier_name,omitempty"`
}
