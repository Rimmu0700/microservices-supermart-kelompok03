package order

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nusantara-supermart/backend/internal/messaging"
)

type Repository interface {
	GetOrCreateCart(customerID string) (*Cart, error)
	GetCartItems(cartID string) ([]CartItem, error)
	AddToCart(cartID, productID string, quantity int) error
	UpdateCartItem(itemID string, quantity int) error
	RemoveCartItem(itemID string) error
	ClearCart(cartID string) error
	CreateOrder(o *Order, items []OrderItem, shipping *OrderShippingDetail, outbox *messaging.OutboxMessage) error
	GetOrders(customerID, statusID string) ([]Order, error)
	GetOrderByID(id string) (*Order, error)
	GetOrderItems(orderID string) ([]OrderItem, error)
	GetOrderShippingDetail(orderID string) (*OrderShippingDetail, error)
	GetOrderStatusHistories(orderID string) ([]OrderStatusHistory, error)
	GetOrderNotes(orderID string) ([]OrderNote, error)
	AddOrderNote(note *OrderNote) error
	UpdateOrderStatus(orderID, statusID, notes string) error
	CancelOrder(orderID, reason, userID, cancelledStatusID string) error
	GetOrderStatuses() ([]OrderStatus, error)
	GetOrderStatusByCode(code string) (*OrderStatus, error)
}

type mysqlRepository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &mysqlRepository{db: db}
}

func (r *mysqlRepository) GetOrCreateCart(customerID string) (*Cart, error) {
	var c Cart
	err := r.db.Get(&c, "SELECT id, user_id, created_at FROM carts WHERE user_id = ? LIMIT 1", customerID)
	if err == nil {
		c.CustomerID = c.UserID
		return &c, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	newCart := &Cart{
		ID:         uuid.NewString(),
		UserID:     customerID,
		CustomerID: customerID,
		CreatedAt:  time.Now(),
	}
	_, err = r.db.Exec("INSERT INTO carts (id, user_id, created_at) VALUES (?, ?, ?)",
		newCart.ID, newCart.UserID, newCart.CreatedAt)
	if err != nil {
		return nil, err
	}
	return newCart, nil
}

func (r *mysqlRepository) GetCartItems(cartID string) ([]CartItem, error) {
	query := `
        SELECT ci.id, ci.cart_id, ci.product_id, ci.quantity,
               p.title AS product_title, p.sku AS product_sku, p.base_price AS product_price,
               COALESCE(pi.image_url, 'https://images.unsplash.com/photo-1586201375761-83865001e31c?w=500') AS image_url
        FROM cart_items ci
        JOIN products p ON ci.product_id = p.id
        LEFT JOIN product_images pi ON p.id = pi.product_id AND pi.is_primary = 1
        WHERE ci.cart_id = ?
        ORDER BY ci.id ASC
    `
	var items []CartItem
	err := r.db.Select(&items, query, cartID)
	if err != nil {
		return nil, err
	}

	for i := range items {
		items[i].Subtotal = float64(items[i].Quantity) * items[i].ProductPrice
	}
	return items, nil
}

func (r *mysqlRepository) AddToCart(cartID, productID string, quantity int) error {
	var existingItem CartItem
	err := r.db.Get(&existingItem, "SELECT id, quantity FROM cart_items WHERE cart_id = ? AND product_id = ? LIMIT 1", cartID, productID)
	if err == nil {
		newQty := existingItem.Quantity + quantity
		_, err = r.db.Exec("UPDATE cart_items SET quantity = ? WHERE id = ?", newQty, existingItem.ID)
		return err
	}

	itemID := uuid.NewString()
	_, err = r.db.Exec("INSERT INTO cart_items (id, cart_id, product_id, quantity) VALUES (?, ?, ?, ?)", itemID, cartID, productID, quantity)
	return err
}

func (r *mysqlRepository) UpdateCartItem(itemID string, quantity int) error {
	if quantity <= 0 {
		return r.RemoveCartItem(itemID)
	}
	_, err := r.db.Exec("UPDATE cart_items SET quantity = ? WHERE id = ?", quantity, itemID)
	return err
}

func (r *mysqlRepository) RemoveCartItem(itemID string) error {
	_, err := r.db.Exec("DELETE FROM cart_items WHERE id = ?", itemID)
	return err
}

func (r *mysqlRepository) ClearCart(cartID string) error {
	_, err := r.db.Exec("DELETE FROM cart_items WHERE cart_id = ?", cartID)
	return err
}

func (r *mysqlRepository) CreateOrder(o *Order, items []OrderItem, shipping *OrderShippingDetail, outbox *messaging.OutboxMessage) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if o.ID == "" {
		o.ID = uuid.NewString()
	}
	now := time.Now()
	o.CreatedAt = now

	insertOrder := `
        INSERT INTO orders (id, order_number, customer_id, shipping_address_id, order_status_id, warehouse_id,
                            total_gross_amount, discount_amount, tax_amount, shipping_fee, total_net_amount, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `
	_, err = tx.Exec(insertOrder, o.ID, o.OrderNumber, o.CustomerID, o.ShippingAddressID, o.OrderStatusID, o.WarehouseID,
		o.TotalGrossAmount, o.DiscountAmount, o.TaxAmount, o.ShippingFee, o.TotalNetAmount, o.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	for _, item := range items {
		itemID := item.ID
		if itemID == "" {
			itemID = uuid.NewString()
		}
		_, err = tx.Exec(`
            INSERT INTO order_items (id, order_id, product_id, quantity, unit_price, subtotal)
            VALUES (?, ?, ?, ?, ?, ?)
        `, itemID, o.ID, item.ProductID, item.Quantity, item.UnitPrice, item.Subtotal)
		if err != nil {
			return fmt.Errorf("insert order item: %w", err)
		}
	}

	if shipping != nil {
		shippingID := shipping.ID
		if shippingID == "" {
			shippingID = uuid.NewString()
		}
		_, err = tx.Exec(`
            INSERT INTO order_shipping_details (id, order_id, courier_name, tracking_number, shipping_cost)
            VALUES (?, ?, ?, ?, ?)
        `, shippingID, o.ID, shipping.CourierName, shipping.TrackingNumber, shipping.ShippingCost)
		if err != nil {
			return fmt.Errorf("insert order shipping: %w", err)
		}
	}

	// Status history
	histID := uuid.NewString()
	_, err = tx.Exec(`
        INSERT INTO order_status_histories (id, order_id, order_status_id, notes, changed_at)
        VALUES (?, ?, ?, 'Pesanan baru dibuat', NOW())
    `, histID, o.ID, o.OrderStatusID)
	if err != nil {
		return fmt.Errorf("insert status history: %w", err)
	}

	// Transactional Outbox: event ditulis di transaksi YANG SAMA dengan order.
	// Jika transaksi gagal, event ikut batal. Jika sukses, event pasti tersimpan.
	if outbox != nil {
		if err := messaging.InsertOutbox(context.Background(), tx, *outbox); err != nil {
			return fmt.Errorf("insert outbox event: %w", err)
		}
	}

	// Generate invoice for payment domain (monolithic cross-domain coupling)
	invID := uuid.NewString()
	invNumber := fmt.Sprintf("INV-%s-%04d", now.Format("20060102"), time.Now().Unix()%10000)
	dueDate := now.Add(24 * time.Hour)
	_, err = tx.Exec(`
        INSERT INTO payment_invoices (id, invoice_number, order_id, customer_id, amount, payment_status, due_date)
        VALUES (?, ?, ?, ?, ?, 'UNPAID', ?)
    `, invID, invNumber, o.ID, o.CustomerID, o.TotalNetAmount, dueDate)
	if err != nil {
		return fmt.Errorf("insert payment invoice: %w", err)
	}

	// Generate shipping order for logistics domain (monolithic cross-domain coupling)
	courierServiceID := "cs000001-0000-0000-0000-000000000002" // default GoSend
	trackingNumber := fmt.Sprintf("GOSEND-%s-%04d", now.Format("20060102"), time.Now().Unix()%10000)

	if shipping != nil {
		cName := strings.ToLower(shipping.CourierName)
		if strings.Contains(cName, "jne") {
			courierServiceID = "cs000001-0000-0000-0000-000000000001"
			trackingNumber = fmt.Sprintf("JNE-%s-%04d", now.Format("20060102"), time.Now().Unix()%10000)
		} else if strings.Contains(cName, "internal") || strings.Contains(cName, "express") || strings.Contains(cName, "nusantara") {
			courierServiceID = "cs000001-0000-0000-0000-000000000003"
			trackingNumber = fmt.Sprintf("EXP-%s-%04d", now.Format("20060102"), time.Now().Unix()%10000)
		}

		shipping.TrackingNumber = &trackingNumber
		_, _ = tx.Exec(`UPDATE order_shipping_details SET tracking_number = ? WHERE id = ?`, trackingNumber, shipping.ID)
	}

	soID := uuid.NewString()
	var totalWeightKg float64 = 0.0
	for _, it := range items {
		totalWeightKg += float64(it.Quantity) * 0.5
	}
	if totalWeightKg <= 0 {
		totalWeightKg = 1.0
	}

	_, err = tx.Exec(`
        INSERT INTO shipping_orders (id, order_id, courier_service_id, tracking_number, weight_kg, current_status)
        VALUES (?, ?, ?, ?, ?, 'READY_FOR_PICKUP')
    `, soID, o.ID, courierServiceID, trackingNumber, totalWeightKg)
	if err != nil {
		return fmt.Errorf("insert shipping order: %w", err)
	}

	// Insert initial tracking log
	logID := uuid.NewString()
	_, _ = tx.Exec(`
        INSERT INTO shipping_tracking_logs (id, shipping_order_id, status_description, location)
        VALUES (?, ?, 'Pesanan telah diproses dan siap dijemput oleh armada kurir', 'Hub Gudang Pengiriman')
    `, logID, soID)

	return tx.Commit()
}

func (r *mysqlRepository) GetOrders(customerID, statusID string) ([]Order, error) {
	query := `
        SELECT o.id, o.order_number, o.customer_id, o.shipping_address_id, o.order_status_id, o.warehouse_id,
               o.total_gross_amount, o.discount_amount, o.tax_amount, o.shipping_fee, o.total_net_amount, o.created_at,
               os.status_code, os.status_name,
               u.full_name AS customer_name, u.email AS customer_email,
               w.warehouse_name,
               CONCAT(ua.street_address, ', ', ua.city) AS shipping_address
        FROM orders o
        JOIN order_statuses os ON o.order_status_id = os.id
        JOIN users u ON o.customer_id = u.id
        JOIN warehouses w ON o.warehouse_id = w.id
        LEFT JOIN user_addresses ua ON o.shipping_address_id = ua.id
    `
	args := []interface{}{}
	whereClauses := []string{}

	if customerID != "" {
		whereClauses = append(whereClauses, "o.customer_id = ?")
		args = append(args, customerID)
	}
	if statusID != "" {
		whereClauses = append(whereClauses, "o.order_status_id = ?")
		args = append(args, statusID)
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + fmt.Sprintf("%s", joinStrings(whereClauses, " AND "))
	}
	query += " ORDER BY o.created_at DESC"

	var orders []Order
	err := r.db.Select(&orders, query, args...)
	return orders, err
}

func (r *mysqlRepository) GetOrderByID(id string) (*Order, error) {
	query := `
        SELECT o.id, o.order_number, o.customer_id, o.shipping_address_id, o.order_status_id, o.warehouse_id,
               o.total_gross_amount, o.discount_amount, o.tax_amount, o.shipping_fee, o.total_net_amount, o.created_at,
               os.status_code, os.status_name,
               u.full_name AS customer_name, u.email AS customer_email,
               w.warehouse_name,
               CONCAT(ua.street_address, ', ', ua.city) AS shipping_address
        FROM orders o
        JOIN order_statuses os ON o.order_status_id = os.id
        JOIN users u ON o.customer_id = u.id
        JOIN warehouses w ON o.warehouse_id = w.id
        LEFT JOIN user_addresses ua ON o.shipping_address_id = ua.id
        WHERE o.id = ? LIMIT 1
    `
	var o Order
	err := r.db.Get(&o, query, id)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *mysqlRepository) GetOrderItems(orderID string) ([]OrderItem, error) {
	query := `
        SELECT oi.id, oi.order_id, oi.product_id, oi.quantity, oi.unit_price, oi.subtotal,
               p.title AS product_title, p.sku AS product_sku
        FROM order_items oi
        JOIN products p ON oi.product_id = p.id
        WHERE oi.order_id = ?
    `
	var items []OrderItem
	err := r.db.Select(&items, query, orderID)
	return items, err
}

func (r *mysqlRepository) GetOrderShippingDetail(orderID string) (*OrderShippingDetail, error) {
	var detail OrderShippingDetail
	err := r.db.Get(&detail, "SELECT * FROM order_shipping_details WHERE order_id = ? LIMIT 1", orderID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &detail, err
}

func (r *mysqlRepository) GetOrderStatusHistories(orderID string) ([]OrderStatusHistory, error) {
	query := `
        SELECT osh.id, osh.order_id, osh.order_status_id, osh.notes, osh.changed_at,
               os.status_name
        FROM order_status_histories osh
        JOIN order_statuses os ON osh.order_status_id = os.id
        WHERE osh.order_id = ?
        ORDER BY osh.changed_at ASC
    `
	var histories []OrderStatusHistory
	err := r.db.Select(&histories, query, orderID)
	return histories, err
}

func (r *mysqlRepository) GetOrderNotes(orderID string) ([]OrderNote, error) {
	query := `
        SELECT onote.id, onote.order_id, onote.author_user_id, onote.note_content, onote.created_at,
               u.full_name AS author_name
        FROM order_notes onote
        JOIN users u ON onote.author_user_id = u.id
        WHERE onote.order_id = ?
        ORDER BY onote.created_at ASC
    `
	var notes []OrderNote
	err := r.db.Select(&notes, query, orderID)
	return notes, err
}

func (r *mysqlRepository) AddOrderNote(note *OrderNote) error {
	if note.ID == "" {
		note.ID = uuid.NewString()
	}
	query := `INSERT INTO order_notes (id, order_id, author_user_id, note_content, created_at) VALUES (?, ?, ?, ?, NOW())`
	_, err := r.db.Exec(query, note.ID, note.OrderID, note.AuthorUserID, note.NoteContent)
	return err
}

func (r *mysqlRepository) UpdateOrderStatus(orderID, statusID, notes string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("UPDATE orders SET order_status_id = ? WHERE id = ?", statusID, orderID)
	if err != nil {
		return err
	}

	histID := uuid.NewString()
	_, err = tx.Exec(`
        INSERT INTO order_status_histories (id, order_id, order_status_id, notes, changed_at)
        VALUES (?, ?, ?, ?, NOW())
    `, histID, orderID, statusID, notes)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *mysqlRepository) CancelOrder(orderID, reason, userID, cancelledStatusID string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("UPDATE orders SET order_status_id = ? WHERE id = ?", cancelledStatusID, orderID)
	if err != nil {
		return err
	}

	cnlID := uuid.NewString()
	_, err = tx.Exec(`
        INSERT INTO order_cancellations (id, order_id, cancel_reason, cancelled_by_user_id, cancelled_at)
        VALUES (?, ?, ?, ?, NOW())
    `, cnlID, orderID, reason, userID)
	if err != nil {
		return err
	}

	histID := uuid.NewString()
	_, err = tx.Exec(`
        INSERT INTO order_status_histories (id, order_id, order_status_id, notes, changed_at)
        VALUES (?, ?, ?, ?, NOW())
    `, histID, orderID, cancelledStatusID, "Dibatalkan: "+reason)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *mysqlRepository) GetOrderStatuses() ([]OrderStatus, error) {
	var statuses []OrderStatus
	err := r.db.Select(&statuses, "SELECT * FROM order_statuses")
	return statuses, err
}

func (r *mysqlRepository) GetOrderStatusByCode(code string) (*OrderStatus, error) {
	var s OrderStatus
	err := r.db.Get(&s, "SELECT * FROM order_statuses WHERE status_code = ? LIMIT 1", code)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func joinStrings(strs []string, sep string) string {
	res := ""
	for i, s := range strs {
		if i > 0 {
			res += sep
		}
		res += s
	}
	return res
}
