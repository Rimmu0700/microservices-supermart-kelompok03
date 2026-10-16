package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/nusantara-supermart/backend/internal/inventory"
	"github.com/nusantara-supermart/backend/internal/messaging"
	"github.com/nusantara-supermart/backend/pkg/config"
	"github.com/nusantara-supermart/backend/pkg/database"
)

const consumerName = "inventory-worker"

func main() {
	cfg := config.LoadConfig()
	db, err := database.ConnectMySQL(cfg)
	if err != nil {
		log.Fatalf("[Inventory Worker] Gagal konek MySQL: %v", err)
	}
	defer db.Close()

	uri := messaging.AMQPURI()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Publisher untuk event kompensasi (dibuka saat dibutuhkan)
	var pub *messaging.EventPublisher
	publishFailure := func(ctx context.Context, srcEventID string, data messaging.OrderPlacedData, reason string) error {
		if pub == nil {
			p, err := messaging.NewEventPublisher(uri)
			if err != nil {
				return err
			}
			pub = p
		}
		region := data.Region
		if region == "" {
			region = "unknown"
		}
		fail, err := messaging.NewEvent(messaging.EventStockReservationFailed, "inventory-service",
			messaging.ReservationFailedData{OrderID: data.OrderID, Region: region, Reason: reason})
		if err != nil {
			return err
		}
		// event_id deterministik: redelivery menghasilkan event_id yang sama, jadi sisi Order bisa mendeduplikasi
		fail.EventID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(srcEventID+":reservation_failed")).String()

		key := "inventory." + region + ".reservation_failed"
		if err := pub.PublishDomainEvent(ctx, messaging.ExchangeEvents, key, fail); err != nil {
			pub.Close()
			pub = nil
			return err
		}
		return nil
	}

	handler := func(ctx context.Context, ev messaging.Event) error {
		// ---- Validasi: kegagalan di sini = poison pill (tidak akan pernah berhasil) ----
		if ev.EventType != messaging.EventOrderPlaced {
			return fmt.Errorf("%w: event_type tidak dikenal: %s", messaging.ErrPoison, ev.EventType)
		}
		var data messaging.OrderPlacedData
		if err := json.Unmarshal(ev.Data, &data); err != nil || data.OrderID == "" || data.WarehouseID == "" || len(data.Items) == 0 {
			return fmt.Errorf("%w: payload OrderPlaced tidak valid", messaging.ErrPoison)
		}
		items := make([]inventory.ReservationItem, 0, len(data.Items))
		for _, it := range data.Items {
			if it.ProductID == "" || it.Quantity <= 0 {
				return fmt.Errorf("%w: item tidak valid (product=%q qty=%d)", messaging.ErrPoison, it.ProductID, it.Quantity)
			}
			items = append(items, inventory.ReservationItem{ProductID: it.ProductID, Quantity: it.Quantity})
		}

		// ---- Mutasi bisnis idempoten (ledger + reservasi dalam satu transaksi) ----
		err := inventory.ReserveOrderStockOnce(ctx, db, consumerName, ev.EventID, ev.EventType, data.OrderID, data.WarehouseID, items)
		switch {
		case err == nil:
			log.Printf("[Inventory Worker] Stok order %s BERHASIL direservasi (event %s)", data.OrderID, ev.EventID)
			return nil
		case errors.Is(err, inventory.ErrDuplicateEvent):
			log.Printf("[Dedup Guard] Event %s terdeteksi DUPLIKAT. Melewati logika bisnis dan langsung ACK.", ev.EventID)
			return nil
		case errors.Is(err, inventory.ErrInsufficientStock):
			log.Printf("[Inventory Worker] Stok TIDAK CUKUP untuk order %s: %v. Menerbitkan event kompensasi.", data.OrderID, err)
			return publishFailure(ctx, ev.EventID, data, err.Error())
		default:
			return err // galat sementara (mis. DB down): NACK + requeue
		}
	}

	for ctx.Err() == nil {
		if err := messaging.Consume(ctx, uri, messaging.QueueInventoryReservation, consumerName, 10, handler); err != nil {
			log.Printf("[Inventory Worker] %v. Mencoba lagi dalam 3 detik...", err)
			time.Sleep(3 * time.Second)
		}
	}
	log.Println("[Inventory Worker] Berhenti.")
}
