package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nusantara-supermart/backend/internal/catalog"
	"github.com/nusantara-supermart/backend/internal/inventory"
	"github.com/nusantara-supermart/backend/internal/messaging"
	"github.com/nusantara-supermart/backend/internal/order"
	"github.com/nusantara-supermart/backend/pkg/config"
	"github.com/nusantara-supermart/backend/pkg/database"
)

const consumerName = "order-compensation-worker"

func main() {
	cfg := config.LoadConfig()
	db, err := database.ConnectMySQL(cfg)
	if err != nil {
		log.Fatalf("[Order Worker] Gagal konek MySQL: %v", err)
	}
	defer db.Close()

	catalogSvc := catalog.NewService(catalog.NewRepository(db))
	inventorySvc := inventory.NewService(inventory.NewRepository(db))
	orderSvc := order.NewService(order.NewRepository(db), catalogSvc, inventorySvc)

	// order_cancellations punya FK ke users, jadi pembatal otomatis memakai user admin seed
	systemUser := os.Getenv("SYSTEM_USER_ID")
	if systemUser == "" {
		systemUser = "u0000001-0000-0000-0000-000000000001"
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	handler := func(ctx context.Context, ev messaging.Event) error {
		if ev.EventType != messaging.EventStockReservationFailed {
			return fmt.Errorf("%w: event_type tidak dikenal: %s", messaging.ErrPoison, ev.EventType)
		}
		var data messaging.ReservationFailedData
		if err := json.Unmarshal(ev.Data, &data); err != nil || data.OrderID == "" {
			return fmt.Errorf("%w: payload StockReservationFailed tidak valid", messaging.ErrPoison)
		}

		// Klaim event di ledger (idempotensi)
		inserted, err := messaging.MarkProcessed(ctx, db, consumerName, ev.EventID, ev.EventType)
		if err != nil {
			return err
		}
		if !inserted {
			log.Printf("[Dedup Guard] Event %s terdeteksi DUPLIKAT. ACK tanpa membatalkan lagi.", ev.EventID)
			return nil
		}

		reason := "Stok tidak mencukupi (dibatalkan otomatis oleh sistem): " + data.Reason
		if err := orderSvc.CancelOrder(data.OrderID, reason, systemUser); err != nil {
			if messaging.IsDuplicateKey(err) {
				log.Printf("[Order Worker] Order %s sudah pernah dibatalkan. ACK.", data.OrderID)
				return nil
			}
			_ = messaging.UnmarkProcessed(ctx, db, consumerName, ev.EventID) // lepas klaim agar bisa diulang
			return err
		}
		log.Printf("[Order Worker] Order %s DIBATALKAN otomatis (kompensasi stok habis)", data.OrderID)
		return nil
	}

	uri := messaging.AMQPURI()
	for ctx.Err() == nil {
		if err := messaging.Consume(ctx, uri, messaging.QueueOrderCompensation, consumerName, 10, handler); err != nil {
			log.Printf("[Order Worker] %v. Mencoba lagi dalam 3 detik...", err)
			time.Sleep(3 * time.Second)
		}
	}
	log.Println("[Order Worker] Berhenti.")
}
