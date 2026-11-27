package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/nusantara-supermart/backend/internal/messaging"
)

const invoiceConsumer = "payment-invoice-consumer"

// OrderPlacedHandler membuat faktur UNPAID begitu Order menerbitkan OrderPlaced.
// Dulu faktur ditulis langsung oleh Order ke tabel payment_invoices (lintas domain).
// Idempoten: (consumer, event_id) dicatat di processed_events dalam transaksi yang sama.
func OrderPlacedHandler(db *sqlx.DB) messaging.Handler {
	return func(ctx context.Context, ev messaging.Event) error {
		if ev.EventType != messaging.EventOrderPlaced {
			return fmt.Errorf("%w: event_type tidak dikenal: %s", messaging.ErrPoison, ev.EventType)
		}
		var d messaging.OrderPlacedData
		if err := json.Unmarshal(ev.Data, &d); err != nil || d.OrderID == "" || d.CustomerID == "" {
			return fmt.Errorf("%w: payload OrderPlaced tidak valid", messaging.ErrPoison)
		}
		amount := d.TotalNetExact
		if amount <= 0 {
			amount = float64(d.TotalNetAmount)
		}

		tx, err := db.BeginTxx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		inserted, err := messaging.MarkProcessed(ctx, tx, invoiceConsumer, ev.EventID, ev.EventType)
		if err != nil {
			return err
		}
		if !inserted {
			log.Printf("[Dedup Guard] Event %s terdeteksi DUPLIKAT. ACK tanpa membuat faktur lagi.", ev.EventID)
			return nil
		}

		now := time.Now()
		invNumber := fmt.Sprintf("INV-%s-%04d", now.Format("20060102"), now.Unix()%10000)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO payment_invoices (id, invoice_number, order_id, customer_id, amount, payment_status, due_date)
			VALUES (?, ?, ?, ?, ?, 'UNPAID', ?)`,
			uuid.NewString(), invNumber, d.OrderID, d.CustomerID, amount, now.Add(24*time.Hour)); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		log.Printf("[Payment] Faktur %s dibuat untuk order %s (Rp %.0f)", invNumber, d.OrderNumber, amount)
		return nil
	}
}
