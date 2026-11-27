package inventory

type AdjustStockRequest struct {
	QuantityOnHand int `json:"quantity_on_hand"`
}

type StockMutationRequest struct {
	ProductID              string `json:"product_id"`
	SourceWarehouseID      string `json:"source_warehouse_id"`
	DestinationWarehouseID string `json:"destination_warehouse_id"`
	Quantity               int    `json:"quantity"`
}

type ReserveStockRequest struct {
	ProductID        string `json:"product_id"`
	WarehouseID      string `json:"warehouse_id"`
	ReferenceOrderID string `json:"reference_order_id"`
	ReservedQuantity int    `json:"reserved_quantity"`
}

type CreateOpnameRequest struct {
	WarehouseID string `json:"warehouse_id"`
	OpnameDate  string `json:"opname_date"`
}

type WarehouseRequest struct {
	WarehouseCode string `json:"warehouse_code"`
	WarehouseName string `json:"warehouse_name"`
	Address       string `json:"address"`
	City          string `json:"city"`
}
