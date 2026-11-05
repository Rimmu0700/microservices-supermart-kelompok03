package order

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"

	"github.com/nusantara-supermart/backend/internal/messaging"
)

const compensationConsumer = "order-compensation-consumer"

// CompensationHandler membatalkan order bila Inventory melaporkan stok tidak cukup (Saga: kompensasi).
// Idempoten lewat tabel processed_events di database order.
func CompensationHandler(db *sqlx.DB, svc Service) messaging.Handler {
	// order_cancellations mencatat siapa pembatal; untuk pembatalan otomatis dipakai user admin seed
	systemUser := os.Getenv("SYSTEM_USER_ID")
	if systemUser == "" {
		systemUser = "u0000001-0000-0000-0000-000000000001"
	}

	return func(ctx context.Context, ev messaging.Event) error {
		if ev.EventType != messaging.EventStockReservationFailed {
			return fmt.Errorf("%w: event_type tidak dikenal: %s", messaging.ErrPoison, ev.EventType)
		}
		var data messaging.ReservationFailedData
		if err := json.Unmarshal(ev.Data, &data); err != nil || data.OrderID == "" {
			return fmt.Errorf("%w: payload StockReservationFailed tidak valid", messaging.ErrPoison)
		}

		inserted, err := messaging.MarkProcessed(ctx, db, compensationConsumer, ev.EventID, ev.EventType)
		if err != nil {
			return err
		}
		if !inserted {
			log.Printf("[Dedup Guard] Event %s terdeteksi DUPLIKAT. ACK tanpa membatalkan lagi.", ev.EventID)
			return nil
		}

		reason := "Stok tidak mencukupi (dibatalkan otomatis oleh sistem): " + data.Reason
		if err := svc.CancelOrder(data.OrderID, reason, systemUser); err != nil {
			if messaging.IsDuplicateKey(err) {
				log.Printf("[Order] Order %s sudah pernah dibatalkan. ACK.", data.OrderID)
				return nil
			}
			_ = messaging.UnmarkProcessed(ctx, db, compensationConsumer, ev.EventID) // lepas klaim agar bisa diulang
			return err
		}
		log.Printf("[Order] Order %s DIBATALKAN otomatis (kompensasi stok habis)", data.OrderID)
		return nil
	}
}
