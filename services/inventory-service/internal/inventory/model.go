package inventory

import (
	"time"
)

type Warehouse struct {
	ID            string `db:"id" json:"id"`
	WarehouseCode string `db:"warehouse_code" json:"warehouse_code"`
	WarehouseName string `db:"warehouse_name" json:"warehouse_name"`
	Address       string `db:"address" json:"address"`
	City          string `db:"city" json:"city"`
}

type WarehouseZone struct {
	ID          string `db:"id" json:"id"`
	WarehouseID string `db:"warehouse_id" json:"warehouse_id"`
	ZoneCode    string `db:"zone_code" json:"zone_code"`
	ZoneType    string `db:"zone_type" json:"zone_type"`
}

type WarehouseShelf struct {
	ID                 string   `db:"id" json:"id"`
	ZoneID             string   `db:"zone_id" json:"zone_id"`
	ShelfCode          string   `db:"shelf_code" json:"shelf_code"`
	CapacityCubicMeter *float64 `db:"capacity_cubic_meter" json:"capacity_cubic_meter"`
}

type InventoryStock struct {
	ID               string    `db:"id" json:"id"`
	ProductID        string    `db:"product_id" json:"product_id"`
	WarehouseID      string    `db:"warehouse_id" json:"warehouse_id"`
	BatchID          *string   `db:"batch_id" json:"batch_id"`
	QuantityOnHand   int       `db:"quantity_on_hand" json:"quantity_on_hand"`
	QuantityReserved int       `db:"quantity_reserved" json:"quantity_reserved"`
	UpdatedAt        time.Time `db:"updated_at" json:"updated_at"`

	// Joined fields
	ProductTitle  string  `db:"product_title" json:"product_title"`
	ProductSKU    string  `db:"product_sku" json:"product_sku"`
	ProductPrice  float64 `db:"product_price" json:"product_price"`
	WarehouseName string  `db:"warehouse_name" json:"warehouse_name"`
	WarehouseCity string  `db:"warehouse_city" json:"warehouse_city"`
}

type StockMutation struct {
	ID                     string    `db:"id" json:"id"`
	ProductID              string    `db:"product_id" json:"product_id"`
	SourceWarehouseID      string    `db:"source_warehouse_id" json:"source_warehouse_id"`
	DestinationWarehouseID string    `db:"destination_warehouse_id" json:"destination_warehouse_id"`
	Quantity               int       `db:"quantity" json:"quantity"`
	MutationDate           time.Time `db:"mutation_date" json:"mutation_date"`

	// Joined
	ProductTitle    string `db:"product_title" json:"product_title,omitempty"`
	SourceWhName    string `db:"source_wh_name" json:"source_warehouse_name,omitempty"`
	DestWhName      string `db:"dest_wh_name" json:"destination_warehouse_name,omitempty"`
}

type StockReservation struct {
	ID               string    `db:"id" json:"id"`
	ProductID        string    `db:"product_id" json:"product_id"`
	ReferenceOrderID string    `db:"reference_order_id" json:"reference_order_id"`
	ReservedQuantity int       `db:"reserved_quantity" json:"reserved_quantity"`
	Status           string    `db:"status" json:"status"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}

type StockOpname struct {
	ID                string    `db:"id" json:"id"`
	WarehouseID       string    `db:"warehouse_id" json:"warehouse_id"`
	OpnameNumber      string    `db:"opname_number" json:"opname_number"`
	OpnameDate        string    `db:"opname_date" json:"opname_date"`
	ConductedByUserID string    `db:"conducted_by_user_id" json:"conducted_by_user_id"`
	WarehouseName     string    `db:"warehouse_name" json:"warehouse_name,omitempty"`
	ConductedByName   string    `db:"conducted_by_name" json:"conducted_by_name,omitempty"`
}

type LowStockAlert struct {
	ID            string    `db:"id" json:"id"`
	ProductID     string    `db:"product_id" json:"product_id"`
	WarehouseID   string    `db:"warehouse_id" json:"warehouse_id"`
	CurrentStock  int       `db:"current_stock" json:"current_stock"`
	Threshold     int       `db:"threshold" json:"threshold"`
	IsResolved    bool      `db:"is_resolved" json:"is_resolved"`
	AlertTime     time.Time `db:"alert_time" json:"alert_time"`
	ProductTitle  string    `db:"product_title" json:"product_title,omitempty"`
	ProductSKU    string    `db:"product_sku" json:"product_sku,omitempty"`
	WarehouseName string    `db:"warehouse_name" json:"warehouse_name,omitempty"`
}
