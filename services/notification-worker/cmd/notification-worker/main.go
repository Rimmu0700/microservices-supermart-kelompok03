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

	"github.com/nusantara-supermart/backend/internal/messaging"
	"github.com/nusantara-supermart/backend/pkg/config"
	"github.com/nusantara-supermart/backend/pkg/database"
)

const consumerName = "notification-worker"

func main() {
	cfg := config.LoadConfig()
	db, err := database.ConnectMySQL(cfg)
	if err != nil {
		log.Fatalf("[Notification Worker] Gagal konek MySQL: %v", err)
	}
	defer db.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	handler := func(ctx context.Context, ev messaging.Event) error {
		inserted, err := messaging.MarkProcessed(ctx, db, consumerName, ev.EventID, ev.EventType)
		if err != nil {
			return err
		}
		if !inserted {
			log.Printf("[Dedup Guard] Notifikasi untuk event %s sudah pernah dikirim. ACK.", ev.EventID)
			return nil
		}

		switch ev.EventType {
		case messaging.EventOrderPlaced:
			var d messaging.OrderPlacedData
			if err := json.Unmarshal(ev.Data, &d); err != nil {
				_ = messaging.UnmarkProcessed(ctx, db, consumerName, ev.EventID)
				return fmt.Errorf("%w: payload OrderPlaced tidak valid", messaging.ErrPoison)
			}
			log.Printf("[WhatsApp/Email Gateway] Ke pelanggan %s: Pesanan %s diterima. Total Rp %d. Silakan lakukan pembayaran.",
				d.CustomerID, d.OrderNumber, d.TotalNetAmount)
		case messaging.EventStockReservationFailed:
			var d messaging.ReservationFailedData
			if err := json.Unmarshal(ev.Data, &d); err != nil {
				_ = messaging.UnmarkProcessed(ctx, db, consumerName, ev.EventID)
				return fmt.Errorf("%w: payload StockReservationFailed tidak valid", messaging.ErrPoison)
			}
			log.Printf("[WhatsApp/Email Gateway] Mohon maaf, stok untuk pesanan %s tidak mencukupi. Pesanan dibatalkan.", d.OrderID)
		default:
			log.Printf("[Notification Worker] Event %s (%s) diabaikan.", ev.EventID, ev.EventType)
		}
		return nil
	}

	uri := messaging.AMQPURI()
	for ctx.Err() == nil {
		if err := messaging.Consume(ctx, uri, messaging.QueueNotification, consumerName, 10, handler); err != nil {
			log.Printf("[Notification Worker] %v. Mencoba lagi dalam 3 detik...", err)
			time.Sleep(3 * time.Second)
		}
	}
	log.Println("[Notification Worker] Berhenti.")
}
