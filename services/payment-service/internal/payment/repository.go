package payment

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository interface {
	GetPaymentMethods() ([]PaymentMethod, error)
	GetPaymentMethodByID(id string) (*PaymentMethod, error)
	GetInvoices(customerID string) ([]PaymentInvoice, error)
	GetInvoiceByID(id string) (*PaymentInvoice, error)
	GetInvoiceByOrderID(orderID string) (*PaymentInvoice, error)
	CreateInvoice(inv *PaymentInvoice) error
	RecordTransaction(tx *PaymentTransaction) error
	GetTransactionsByInvoiceID(invoiceID string) ([]PaymentTransaction, error)
	GetOrCreateStoreCredit(userID string) (*StoreCredit, error)
	AddCreditTransaction(userID string, amount float64, txType, desc string) error
	GetCreditTransactions(userID string) ([]CreditTransaction, error)
	CreateRefund(r *Refund) error
	GetRefunds(userID string) ([]Refund, error)
	UpdateRefundStatus(id, status string) error
}

type mysqlRepository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &mysqlRepository{db: db}
}

func (r *mysqlRepository) GetPaymentMethods() ([]PaymentMethod, error) {
	var methods []PaymentMethod
	err := r.db.Select(&methods, "SELECT * FROM payment_methods ORDER BY method_name ASC")
	return methods, err
}

func (r *mysqlRepository) GetPaymentMethodByID(id string) (*PaymentMethod, error) {
	var method PaymentMethod
	err := r.db.Get(&method, "SELECT * FROM payment_methods WHERE id = ? LIMIT 1", id)
	if err != nil {
		return nil, err
	}
	return &method, nil
}

func (r *mysqlRepository) GetInvoices(customerID string) ([]PaymentInvoice, error) {
	query := `
		SELECT pi.id, pi.invoice_number, pi.order_id, pi.customer_id, pi.amount, pi.payment_status, pi.due_date, pi.paid_at,
		       o.order_number, u.full_name AS customer_name
		FROM payment_invoices pi
		JOIN orders o ON pi.order_id = o.id
		JOIN users u ON pi.customer_id = u.id
	`
	args := []interface{}{}
	if customerID != "" {
		query += " WHERE pi.customer_id = ?"
		args = append(args, customerID)
	}
	query += " ORDER BY pi.due_date DESC"

	var invoices []PaymentInvoice
	err := r.db.Select(&invoices, query, args...)
	return invoices, err
}

func (r *mysqlRepository) GetInvoiceByID(id string) (*PaymentInvoice, error) {
	query := `
		SELECT pi.id, pi.invoice_number, pi.order_id, pi.customer_id, pi.amount, pi.payment_status, pi.due_date, pi.paid_at,
		       o.order_number, u.full_name AS customer_name
		FROM payment_invoices pi
		JOIN orders o ON pi.order_id = o.id
		JOIN users u ON pi.customer_id = u.id
		WHERE pi.id = ? LIMIT 1
	`
	var inv PaymentInvoice
	err := r.db.Get(&inv, query, id)
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *mysqlRepository) GetInvoiceByOrderID(orderID string) (*PaymentInvoice, error) {
	query := `
		SELECT pi.id, pi.invoice_number, pi.order_id, pi.customer_id, pi.amount, pi.payment_status, pi.due_date, pi.paid_at,
		       o.order_number, u.full_name AS customer_name
		FROM payment_invoices pi
		JOIN orders o ON pi.order_id = o.id
		JOIN users u ON pi.customer_id = u.id
		WHERE pi.order_id = ? LIMIT 1
	`
	var inv PaymentInvoice
	err := r.db.Get(&inv, query, orderID)
	if err == nil {
		return &inv, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// Auto-create invoice if missing for this existing order
	var o struct {
		ID             string  `db:"id"`
		OrderNumber    string  `db:"order_number"`
		CustomerID     string  `db:"customer_id"`
		TotalNetAmount float64 `db:"total_net_amount"`
		CustomerName   string  `db:"customer_name"`
	}
	orderQuery := `
		SELECT o.id, o.order_number, o.customer_id, o.total_net_amount, u.full_name AS customer_name
		FROM orders o
		JOIN users u ON o.customer_id = u.id
		WHERE o.id = ? LIMIT 1
	`
	err = r.db.Get(&o, orderQuery, orderID)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	invID := uuid.NewString()
	invNumber := fmt.Sprintf("INV-%s-%04d", time.Now().Format("20060102"), time.Now().Unix()%10000)
	dueDate := time.Now().Add(24 * time.Hour)

	insertQuery := `
		INSERT INTO payment_invoices (id, invoice_number, order_id, customer_id, amount, payment_status, due_date)
		VALUES (?, ?, ?, ?, ?, 'UNPAID', ?)
	`
	_, err = r.db.Exec(insertQuery, invID, invNumber, o.ID, o.CustomerID, o.TotalNetAmount, dueDate)
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	return &PaymentInvoice{
		ID:            invID,
		InvoiceNumber: invNumber,
		OrderID:       o.ID,
		CustomerID:    o.CustomerID,
		Amount:        o.TotalNetAmount,
		PaymentStatus: "UNPAID",
		DueDate:       dueDate,
		OrderNumber:   o.OrderNumber,
		CustomerName:  o.CustomerName,
	}, nil
}

func (r *mysqlRepository) CreateInvoice(inv *PaymentInvoice) error {
	if inv.ID == "" {
		inv.ID = uuid.NewString()
	}
	query := `
		INSERT INTO payment_invoices (id, invoice_number, order_id, customer_id, amount, payment_status, due_date)
		VALUES (?, ?, ?, ?, ?, 'UNPAID', ?)
	`
	_, err := r.db.Exec(query, inv.ID, inv.InvoiceNumber, inv.OrderID, inv.CustomerID, inv.Amount, inv.DueDate)
	return err
}

func (r *mysqlRepository) RecordTransaction(tx *PaymentTransaction) error {
	dbTx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer dbTx.Rollback()

	if tx.ID == "" {
		tx.ID = uuid.NewString()
	}
	tx.TransactionTime = time.Now()

	insertTx := `
		INSERT INTO payment_transactions (id, invoice_id, payment_method_id, gateway_id, transaction_reference, amount_paid, status, transaction_time)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = dbTx.Exec(insertTx, tx.ID, tx.InvoiceID, tx.PaymentMethodID, tx.GatewayID, tx.TransactionReference, tx.AmountPaid, tx.Status, tx.TransactionTime)
	if err != nil {
		return fmt.Errorf("insert transaction: %w", err)
	}

	if tx.Status == "SUCCESS" {
		_, err = dbTx.Exec("UPDATE payment_invoices SET payment_status = 'PAID', paid_at = NOW() WHERE id = ?", tx.InvoiceID)
		if err != nil {
			return fmt.Errorf("update invoice: %w", err)
		}
	}

	return dbTx.Commit()
}

func (r *mysqlRepository) GetTransactionsByInvoiceID(invoiceID string) ([]PaymentTransaction, error) {
	query := `
		SELECT pt.id, pt.invoice_id, pt.payment_method_id, pt.gateway_id, pt.transaction_reference, pt.amount_paid, pt.status, pt.transaction_time,
		       pm.method_name
		FROM payment_transactions pt
		JOIN payment_methods pm ON pt.payment_method_id = pm.id
		WHERE pt.invoice_id = ?
		ORDER BY pt.transaction_time DESC
	`
	var txs []PaymentTransaction
	err := r.db.Select(&txs, query, invoiceID)
	return txs, err
}

func (r *mysqlRepository) GetOrCreateStoreCredit(userID string) (*StoreCredit, error) {
	var sc StoreCredit
	err := r.db.Get(&sc, "SELECT id, user_id, balance FROM store_credits WHERE user_id = ? LIMIT 1", userID)
	if err == nil {
		return &sc, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	newID := uuid.NewString()
	_, err = r.db.Exec("INSERT INTO store_credits (id, user_id, balance) VALUES (?, ?, 0.00)", newID, userID)
	if err != nil {
		return nil, err
	}
	return &StoreCredit{ID: newID, UserID: userID, CurrentBalance: 0.00}, nil
}

func (r *mysqlRepository) AddCreditTransaction(userID string, amount float64, txType, desc string) error {
	sc, err := r.GetOrCreateStoreCredit(userID)
	if err != nil {
		return err
	}

	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if txType == "CREDIT" {
		_, err = tx.Exec("UPDATE store_credits SET balance = balance + ? WHERE id = ?", amount, sc.ID)
	} else {
		res, execErr := tx.Exec("UPDATE store_credits SET balance = balance - ? WHERE id = ? AND balance >= ?", amount, sc.ID, amount)
		if execErr != nil {
			return execErr
		}
		rows, _ := res.RowsAffected()
		if rows == 0 {
			return fmt.Errorf("insufficient store credit balance")
		}
	}
	if err != nil {
		return err
	}

	txID := uuid.NewString()
	_, err = tx.Exec(`
		INSERT INTO credit_transactions (id, store_credit_id, amount, transaction_type, description, created_at)
		VALUES (?, ?, ?, ?, ?, NOW())
	`, txID, sc.ID, amount, txType, desc)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *mysqlRepository) GetCreditTransactions(userID string) ([]CreditTransaction, error) {
	sc, err := r.GetOrCreateStoreCredit(userID)
	if err != nil {
		return nil, err
	}
	var txs []CreditTransaction
	err = r.db.Select(&txs, "SELECT * FROM credit_transactions WHERE store_credit_id = ? ORDER BY created_at DESC", sc.ID)
	return txs, err
}

func (r *mysqlRepository) CreateRefund(ref *Refund) error {
	if ref.ID == "" {
		ref.ID = uuid.NewString()
	}
	query := `
		INSERT INTO refunds (id, refund_number, order_id, requested_by_user_id, total_refund_amount, refund_status, created_at)
		VALUES (?, ?, ?, ?, ?, 'PENDING', NOW())
	`
	_, err := r.db.Exec(query, ref.ID, ref.RefundNumber, ref.OrderID, ref.RequestedByUserID, ref.TotalRefundAmount)
	return err
}

func (r *mysqlRepository) GetRefunds(userID string) ([]Refund, error) {
	query := `
		SELECT r.id, r.refund_number, r.order_id, r.requested_by_user_id, r.total_refund_amount, r.refund_status, r.created_at,
		       o.order_number
		FROM refunds r
		JOIN orders o ON r.order_id = o.id
	`
	args := []interface{}{}
	if userID != "" {
		query += " WHERE r.requested_by_user_id = ?"
		args = append(args, userID)
	}
	query += " ORDER BY r.created_at DESC"

	var refunds []Refund
	err := r.db.Select(&refunds, query, args...)
	return refunds, err
}

func (r *mysqlRepository) UpdateRefundStatus(id, status string) error {
	_, err := r.db.Exec("UPDATE refunds SET refund_status = ? WHERE id = ?", status, id)
	return err
}
