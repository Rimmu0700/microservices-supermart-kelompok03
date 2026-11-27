package inventory

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository interface {
	GetStocks(warehouseID string) ([]InventoryStock, error)
	GetStockByID(id string) (*InventoryStock, error)
	GetStockByProductAndWarehouse(productID, warehouseID string) (*InventoryStock, error)
	UpdateStockQuantity(id string, qtyOnHand int) error
	ReserveStock(productID, warehouseID, orderID string, qty int) error
	ReleaseStock(orderID string) error
	CreateStockMutation(m *StockMutation) error
	GetStockMutations() ([]StockMutation, error)
	GetWarehouses() ([]Warehouse, error)
	CreateWarehouse(w *Warehouse) error
	GetWarehouseZones(warehouseID string) ([]WarehouseZone, error)
	GetLowStockAlerts() ([]LowStockAlert, error)
	GetStockOpnames(warehouseID string) ([]StockOpname, error)
	CreateStockOpname(op *StockOpname) error
}

type mysqlRepository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &mysqlRepository{db: db}
}

func (r *mysqlRepository) GetStocks(warehouseID string) ([]InventoryStock, error) {
	query := `
		SELECT s.id, s.product_id, s.warehouse_id, s.batch_id, s.quantity_on_hand, s.quantity_reserved, s.updated_at,
		       p.title AS product_title, p.sku AS product_sku, p.base_price AS product_price,
		       w.warehouse_name, w.city AS warehouse_city
		FROM inventory_stocks s
		JOIN products p ON s.product_id = p.id
		JOIN warehouses w ON s.warehouse_id = w.id
	`
	args := []interface{}{}
	if warehouseID != "" {
		query += " WHERE s.warehouse_id = ?"
		args = append(args, warehouseID)
	}
	query += " ORDER BY w.city ASC, p.title ASC"

	var stocks []InventoryStock
	err := r.db.Select(&stocks, query, args...)
	return stocks, err
}

func (r *mysqlRepository) GetStockByID(id string) (*InventoryStock, error) {
	query := `
		SELECT s.id, s.product_id, s.warehouse_id, s.batch_id, s.quantity_on_hand, s.quantity_reserved, s.updated_at,
		       p.title AS product_title, p.sku AS product_sku, p.base_price AS product_price,
		       w.warehouse_name, w.city AS warehouse_city
		FROM inventory_stocks s
		JOIN products p ON s.product_id = p.id
		JOIN warehouses w ON s.warehouse_id = w.id
		WHERE s.id = ? LIMIT 1
	`
	var s InventoryStock
	err := r.db.Get(&s, query, id)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *mysqlRepository) GetStockByProductAndWarehouse(productID, warehouseID string) (*InventoryStock, error) {
	query := `
		SELECT s.id, s.product_id, s.warehouse_id, s.batch_id, s.quantity_on_hand, s.quantity_reserved, s.updated_at,
		       p.title AS product_title, p.sku AS product_sku, p.base_price AS product_price,
		       w.warehouse_name, w.city AS warehouse_city
		FROM inventory_stocks s
		JOIN products p ON s.product_id = p.id
		JOIN warehouses w ON s.warehouse_id = w.id
		WHERE s.product_id = ? AND s.warehouse_id = ? LIMIT 1
	`
	var s InventoryStock
	err := r.db.Get(&s, query, productID, warehouseID)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *mysqlRepository) UpdateStockQuantity(id string, qtyOnHand int) error {
	_, err := r.db.Exec("UPDATE inventory_stocks SET quantity_on_hand = ?, updated_at = NOW() WHERE id = ?", qtyOnHand, id)
	return err
}

func (r *mysqlRepository) ReserveStock(productID, warehouseID, orderID string, qty int) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		UPDATE inventory_stocks
		SET quantity_reserved = quantity_reserved + ?, updated_at = NOW()
		WHERE product_id = ? AND warehouse_id = ? AND (quantity_on_hand - quantity_reserved) >= ?
	`, qty, productID, warehouseID, qty)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil || rows == 0 {
		return fmt.Errorf("insufficient available stock to reserve")
	}

	resID := uuid.NewString()
	_, err = tx.Exec(`
		INSERT INTO stock_reservations (id, product_id, reference_order_id, reserved_quantity, status, created_at)
		VALUES (?, ?, ?, ?, 'HOLD', NOW())
	`, resID, productID, orderID, qty)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *mysqlRepository) ReleaseStock(orderID string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var reservations []StockReservation
	err = tx.Select(&reservations, "SELECT * FROM stock_reservations WHERE reference_order_id = ? AND status = 'HOLD'", orderID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	for _, res := range reservations {
		_, err = tx.Exec(`
			UPDATE inventory_stocks
			SET quantity_reserved = GREATEST(0, quantity_reserved - ?), updated_at = NOW()
			WHERE product_id = ?
		`, res.ReservedQuantity, res.ProductID)
		if err != nil {
			return err
		}
	}

	_, err = tx.Exec("UPDATE stock_reservations SET status = 'RELEASED' WHERE reference_order_id = ?", orderID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *mysqlRepository) CreateStockMutation(m *StockMutation) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Deduct from source
	res, err := tx.Exec(`
		UPDATE inventory_stocks
		SET quantity_on_hand = quantity_on_hand - ?, updated_at = NOW()
		WHERE product_id = ? AND warehouse_id = ? AND quantity_on_hand >= ?
	`, m.Quantity, m.ProductID, m.SourceWarehouseID, m.Quantity)
	if err != nil || func() bool { rows, _ := res.RowsAffected(); return rows == 0 }() {
		return fmt.Errorf("insufficient stock at source warehouse")
	}

	// Add to destination (or insert if not exists)
	destStockQuery := `
		INSERT INTO inventory_stocks (id, product_id, warehouse_id, quantity_on_hand, quantity_reserved)
		VALUES (?, ?, ?, ?, 0)
		ON DUPLICATE KEY UPDATE quantity_on_hand = quantity_on_hand + VALUES(quantity_on_hand), updated_at = NOW()
	`
	_, err = tx.Exec(destStockQuery, uuid.NewString(), m.ProductID, m.DestinationWarehouseID, m.Quantity)
	if err != nil {
		return err
	}

	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	m.MutationDate = time.Now()
	insertMutation := `
		INSERT INTO stock_mutations (id, product_id, source_warehouse_id, destination_warehouse_id, quantity, mutation_date)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err = tx.Exec(insertMutation, m.ID, m.ProductID, m.SourceWarehouseID, m.DestinationWarehouseID, m.Quantity, m.MutationDate)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *mysqlRepository) GetStockMutations() ([]StockMutation, error) {
	query := `
		SELECT sm.id, sm.product_id, sm.source_warehouse_id, sm.destination_warehouse_id, sm.quantity, sm.mutation_date,
		       p.title AS product_title, sw.warehouse_name AS source_wh_name, dw.warehouse_name AS dest_wh_name
		FROM stock_mutations sm
		JOIN products p ON sm.product_id = p.id
		JOIN warehouses sw ON sm.source_warehouse_id = sw.id
		JOIN warehouses dw ON sm.destination_warehouse_id = dw.id
		ORDER BY sm.mutation_date DESC
		LIMIT 50
	`
	var mutations []StockMutation
	err := r.db.Select(&mutations, query)
	return mutations, err
}

func (r *mysqlRepository) GetWarehouses() ([]Warehouse, error) {
	var warehouses []Warehouse
	err := r.db.Select(&warehouses, "SELECT * FROM warehouses ORDER BY city ASC, warehouse_name ASC")
	return warehouses, err
}

func (r *mysqlRepository) CreateWarehouse(w *Warehouse) error {
	if w.ID == "" {
		w.ID = uuid.NewString()
	}
	query := `INSERT INTO warehouses (id, warehouse_code, warehouse_name, address, city) VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query, w.ID, w.WarehouseCode, w.WarehouseName, w.Address, w.City)
	return err
}

func (r *mysqlRepository) GetWarehouseZones(warehouseID string) ([]WarehouseZone, error) {
	var zones []WarehouseZone
	err := r.db.Select(&zones, "SELECT * FROM warehouse_zones WHERE warehouse_id = ? ORDER BY zone_code ASC", warehouseID)
	return zones, err
}

func (r *mysqlRepository) GetLowStockAlerts() ([]LowStockAlert, error) {
	query := `
		SELECT lsa.id, lsa.product_id, lsa.warehouse_id, lsa.current_stock, lsa.threshold, lsa.is_resolved, lsa.alert_time,
		       p.title AS product_title, p.sku AS product_sku, w.warehouse_name
		FROM low_stock_alerts lsa
		JOIN products p ON lsa.product_id = p.id
		JOIN warehouses w ON lsa.warehouse_id = w.id
		ORDER BY lsa.is_resolved ASC, lsa.alert_time DESC
	`
	var alerts []LowStockAlert
	err := r.db.Select(&alerts, query)
	return alerts, err
}

func (r *mysqlRepository) GetStockOpnames(warehouseID string) ([]StockOpname, error) {
	query := `
		SELECT so.id, so.warehouse_id, so.opname_number, so.opname_date, so.conducted_by_user_id,
		       w.warehouse_name, u.full_name AS conducted_by_name
		FROM stock_opnames so
		JOIN warehouses w ON so.warehouse_id = w.id
		JOIN users u ON so.conducted_by_user_id = u.id
	`
	args := []interface{}{}
	if warehouseID != "" {
		query += " WHERE so.warehouse_id = ?"
		args = append(args, warehouseID)
	}
	query += " ORDER BY so.opname_date DESC"

	var opnames []StockOpname
	err := r.db.Select(&opnames, query, args...)
	return opnames, err
}

func (r *mysqlRepository) CreateStockOpname(op *StockOpname) error {
	if op.ID == "" {
		op.ID = uuid.NewString()
	}
	query := `INSERT INTO stock_opnames (id, warehouse_id, opname_number, opname_date, conducted_by_user_id) VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query, op.ID, op.WarehouseID, op.OpnameNumber, op.OpnameDate, op.ConductedByUserID)
	return err
}
