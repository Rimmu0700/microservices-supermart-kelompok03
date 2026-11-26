package procurement

type CreateSupplierRequest struct {
	SupplierCode            string `json:"supplier_code"`
	CompanyName             string `json:"company_name"`
	TaxIdentificationNumber string `json:"tax_identification_number"`
	Address                 string `json:"address"`
	ContactName             string `json:"contact_name,omitempty"`
	ContactEmail            string `json:"contact_email,omitempty"`
	ContactPhone            string `json:"contact_phone,omitempty"`
}

type POItemInput struct {
	ProductID  string  `json:"product_id"`
	OrderedQty int     `json:"ordered_qty"`
	UnitCost   float64 `json:"unit_cost"`
}

type CreatePORequest struct {
	SupplierID  string        `json:"supplier_id"`
	WarehouseID string        `json:"warehouse_id"`
	Items       []POItemInput `json:"items"`
}

type GRNItemInput struct {
	ProductID   string `json:"product_id"`
	ReceivedQty int    `json:"received_qty"`
	Notes       string `json:"notes,omitempty"`
}

type CreateGRNRequest struct {
	PurchaseOrderID string         `json:"purchase_order_id"`
	Items           []GRNItemInput `json:"items"`
}
