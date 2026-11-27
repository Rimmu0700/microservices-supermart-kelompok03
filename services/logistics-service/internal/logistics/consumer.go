package logistics

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/nusantara-supermart/backend/internal/messaging"
)

const shipmentConsumer = "logistics-shipment-consumer"

// courierServiceFor memetakan nama kurir ke id courier_services (aturan yang dulu ada di Order).
func courierServiceFor(courierName string) string {
	c := strings.ToLower(courierName)
	switch {
	case strings.Contains(c, "jne"):
		return "cs000001-0000-0000-0000-000000000001"
	case strings.Contains(c, "internal"), strings.Contains(c, "express"), strings.Contains(c, "nusantara"):
		return "cs000001-0000-0000-0000-000000000003"
	default:
		return "cs000001-0000-0000-0000-000000000002" // GoSend
	}
}

// OrderPlacedHandler membuat shipping_order (READY_FOR_PICKUP) begitu Order menerbitkan OrderPlaced.
// Dulu baris ini ditulis langsung oleh Order ke tabel shipping_orders (lintas domain).
func OrderPlacedHandler(db *sqlx.DB) messaging.Handler {
	return func(ctx context.Context, ev messaging.Event) error {
		if ev.EventType != messaging.EventOrderPlaced {
			return fmt.Errorf("%w: event_type tidak dikenal: %s", messaging.ErrPoison, ev.EventType)
		}
		var d messaging.OrderPlacedData
		if err := json.Unmarshal(ev.Data, &d); err != nil || d.OrderID == "" {
			return fmt.Errorf("%w: payload OrderPlaced tidak valid", messaging.ErrPoison)
		}

		weight := 0.0
		for _, it := range d.Items {
			weight += float64(it.Quantity) * 0.5
		}
		if weight <= 0 {
			weight = 1.0
		}
		tracking := d.TrackingNumber
		if tracking == "" {
			tracking = fmt.Sprintf("GOSEND-%s-%04d", time.Now().Format("20060102"), time.Now().Unix()%10000)
		}

		tx, err := db.BeginTxx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		inserted, err := messaging.MarkProcessed(ctx, tx, shipmentConsumer, ev.EventID, ev.EventType)
		if err != nil {
			return err
		}
		if !inserted {
			log.Printf("[Dedup Guard] Event %s terdeteksi DUPLIKAT. ACK tanpa membuat pengiriman lagi.", ev.EventID)
			return nil
		}

		soID := uuid.NewString()
		// INSERT IGNORE: aman bila shipping_order untuk order ini sudah dibuat lebih dulu (kunci unik order_id)
		res, err := tx.ExecContext(ctx, `
			INSERT IGNORE INTO shipping_orders (id, order_id, courier_service_id, tracking_number, weight_kg, current_status)
			VALUES (?, ?, ?, ?, ?, 'READY_FOR_PICKUP')`,
			soID, d.OrderID, courierServiceFor(d.CourierName), tracking, weight)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 1 {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO shipping_tracking_logs (id, shipping_order_id, status_description, location)
				VALUES (?, ?, 'Pesanan telah diproses dan siap dijemput oleh armada kurir', 'Hub Gudang Pengiriman')`,
				uuid.NewString(), soID); err != nil {
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		log.Printf("[Logistics] Pengiriman untuk order %s dibuat (resi %s)", d.OrderNumber, tracking)
		return nil
	}
}
