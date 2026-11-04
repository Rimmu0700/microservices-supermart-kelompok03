package procurement

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nusantara-supermart/backend/internal/messaging"
)

type Repository interface {
	GetSuppliers() ([]Supplier, error)
	CreateSupplier(s *Supplier, contact *SupplierContact) error
	GetPurchaseOrders() ([]PurchaseOrder, error)
	GetPurchaseOrderByID(id string) (*PurchaseOrder, []PurchaseOrderItem, error)
	CreatePurchaseOrder(po *PurchaseOrder, items []PurchaseOrderItem) error
	UpdatePOStatus(poID, status string) error
	CreateGRN(grn *GoodsReceiptNote, items []GoodsReceiptItem) error
	GetGRNs(poID string) ([]GoodsReceiptNote, error)
	GetPurchaseInvoices() ([]PurchaseInvoice, error)
}

type mysqlRepository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &mysqlRepository{db: db}
}

func (r *mysqlRepository) GetSuppliers() ([]Supplier, error) {
	var suppliers []Supplier
	err := r.db.Select(&suppliers, "SELECT * FROM suppliers ORDER BY company_name ASC")
	return suppliers, err
}

func (r *mysqlRepository) CreateSupplier(s *Supplier, contact *SupplierContact) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	query := `INSERT INTO suppliers (id, supplier_code, company_name, tax_identification_number, address) VALUES (?, ?, ?, ?, ?)`
	_, err = tx.Exec(query, s.ID, s.SupplierCode, s.CompanyName, s.TaxIdentificationNumber, s.Address)
	if err != nil {
		return err
	}

	if contact != nil && contact.ContactName != "" {
		contact.ID = uuid.NewString()
		contact.SupplierID = s.ID
		_, err = tx.Exec("INSERT INTO supplier_contacts (id, supplier_id, contact_name, email, phone) VALUES (?, ?, ?, ?, ?)",
			contact.ID, contact.SupplierID, contact.ContactName, contact.Email, contact.Phone)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *mysqlRepository) GetPurchaseOrders() ([]PurchaseOrder, error) {
	query := `
        SELECT po.id, po.po_number, po.supplier_id, po.warehouse_id, po.po_date, po.total_po_amount, po.status,
               s.company_name AS supplier_name, w.warehouse_name
        FROM purchase_orders po
        JOIN suppliers s ON po.supplier_id = s.id
        JOIN warehouses w ON po.warehouse_id = w.id
        ORDER BY po.po_date DESC
    `
	var pos []PurchaseOrder
	err := r.db.Select(&pos, query)
	return pos, err
}

func (r *mysqlRepository) GetPurchaseOrderByID(id string) (*PurchaseOrder, []PurchaseOrderItem, error) {
	query := `
        SELECT po.id, po.po_number, po.supplier_id, po.warehouse_id, po.po_date, po.total_po_amount, po.status,
               s.company_name AS supplier_name, w.warehouse_name
        FROM purchase_orders po
        JOIN suppliers s ON po.supplier_id = s.id
        JOIN warehouses w ON po.warehouse_id = w.id
        WHERE po.id = ? LIMIT 1
    `
	var po PurchaseOrder
	err := r.db.Get(&po, query, id)
	if err != nil {
		return nil, nil, err
	}

	itemsQuery := `
        SELECT poi.id, poi.purchase_order_id, poi.product_id, poi.ordered_qty, poi.unit_cost,
               p.title AS product_title, p.sku AS product_sku
        FROM purchase_order_items poi
        JOIN products p ON poi.product_id = p.id
        WHERE poi.purchase_order_id = ?
    `
	var items []PurchaseOrderItem
	err = r.db.Select(&items, itemsQuery, id)
	return &po, items, err
}

func (r *mysqlRepository) CreatePurchaseOrder(po *PurchaseOrder, items []PurchaseOrderItem) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if po.ID == "" {
		po.ID = uuid.NewString()
	}
	insertPO := `
        INSERT INTO purchase_orders (id, po_number, supplier_id, warehouse_id, po_date, total_po_amount, status)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `
	_, err = tx.Exec(insertPO, po.ID, po.PONumber, po.SupplierID, po.WarehouseID, po.PODate, po.TotalPOAmount, po.Status)
	if err != nil {
		return fmt.Errorf("insert PO: %w", err)
	}

	for _, item := range items {
		itemID := item.ID
		if itemID == "" {
			itemID = uuid.NewString()
		}
		_, err = tx.Exec(`
            INSERT INTO purchase_order_items (id, purchase_order_id, product_id, ordered_qty, unit_cost)
            VALUES (?, ?, ?, ?, ?)
        `, itemID, po.ID, item.ProductID, item.OrderedQty, item.UnitCost)
		if err != nil {
			return fmt.Errorf("insert PO item: %w", err)
		}
	}

	return tx.Commit()
}

func (r *mysqlRepository) UpdatePOStatus(poID, status string) error {
	_, err := r.db.Exec("UPDATE purchase_orders SET status = ? WHERE id = ?", status, poID)
	return err
}

func (r *mysqlRepository) CreateGRN(grn *GoodsReceiptNote, items []GoodsReceiptItem) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if grn.ID == "" {
		grn.ID = uuid.NewString()
	}
	grn.ReceivedDate = time.Now()

	query := `INSERT INTO goods_receipt_notes (id, grn_number, purchase_order_id, received_date) VALUES (?, ?, ?, ?)`
	_, err = tx.Exec(query, grn.ID, grn.GRNNumber, grn.PurchaseOrderID, grn.ReceivedDate)
	if err != nil {
		return err
	}

	// Fetch warehouse ID from purchase order to update stock
	var warehouseID string
	err = tx.Get(&warehouseID, "SELECT warehouse_id FROM purchase_orders WHERE id = ?", grn.PurchaseOrderID)
	if err != nil {
		return fmt.Errorf("lookup PO warehouse: %w", err)
	}

	eventItems := make([]messaging.GoodsReceivedItem, 0, len(items))
	for _, item := range items {
		itemID := item.ID
		if itemID == "" {
			itemID = uuid.NewString()
		}
		_, err = tx.Exec("INSERT INTO goods_receipt_items (id, grn_id, product_id, received_qty) VALUES (?, ?, ?, ?)",
			itemID, grn.ID, item.ProductID, item.ReceivedQty)
		if err != nil {
			return fmt.Errorf("insert GRN item: %w", err)
		}

		eventItems = append(eventItems, messaging.GoodsReceivedItem{ProductID: item.ProductID, Quantity: item.ReceivedQty})
	}

	// Stok TIDAK lagi diubah langsung (tabel inventory_stocks milik Inventory).
	// Procurement menerbitkan event GoodsReceived lewat transactional outbox (satu transaksi dengan GRN).
	var city string
	if err := tx.Get(&city, "SELECT city FROM warehouses WHERE id = ?", warehouseID); err != nil {
		city = "" // replika belum sinkron: pakai wilayah "unknown"
	}
	region := messaging.Slug(city)
	if region == "" {
		region = "unknown"
	}

	ev, err := messaging.NewEvent(messaging.EventGoodsReceived, "procurement-service", messaging.GoodsReceivedData{
		GRNID:           grn.ID,
		GRNNumber:       grn.GRNNumber,
		PurchaseOrderID: grn.PurchaseOrderID,
		WarehouseID:     warehouseID,
		Items:           eventItems,
	})
	if err != nil {
		return fmt.Errorf("build GoodsReceived event: %w", err)
	}

	payload, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("encode GoodsReceived event: %w", err)
	}

	if err := messaging.InsertOutbox(context.Background(), tx, messaging.OutboxMessage{
		ID:            ev.EventID,
		AggregateType: "goods_receipt",
		AggregateID:   grn.ID,
		EventType:     ev.EventType,
		RoutingKey:    fmt.Sprintf("procurement.%s.goods_received", region),
		Payload:       payload,
	}); err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}

	// Mark PO as RECEIVED
	_, err = tx.Exec("UPDATE purchase_orders SET status = 'RECEIVED' WHERE id = ?", grn.PurchaseOrderID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *mysqlRepository) GetGRNs(poID string) ([]GoodsReceiptNote, error) {
	query := `
        SELECT grn.id, grn.grn_number, grn.purchase_order_id, grn.received_date,
               po.po_number
        FROM goods_receipt_notes grn
        JOIN purchase_orders po ON grn.purchase_order_id = po.id
    `
	args := []interface{}{}
	if poID != "" {
		query += " WHERE grn.purchase_order_id = ?"
		args = append(args, poID)
	}
	query += " ORDER BY grn.received_date DESC"

	var grns []GoodsReceiptNote
	err := r.db.Select(&grns, query, args...)
	return grns, err
}

func (r *mysqlRepository) GetPurchaseInvoices() ([]PurchaseInvoice, error) {
	query := `
        SELECT pi.id, pi.invoice_number, pi.purchase_order_id, pi.invoice_amount, pi.due_date,
               po.po_number, s.company_name AS supplier_name
        FROM purchase_invoices pi
        JOIN purchase_orders po ON pi.purchase_order_id = po.id
        JOIN suppliers s ON po.supplier_id = s.id
        ORDER BY pi.due_date DESC
    `
	var invoices []PurchaseInvoice
	err := r.db.Select(&invoices, query)
	return invoices, err
}
