package inventory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/nusantara-supermart/backend/internal/messaging"
)

const (
	reservationConsumer = "inventory-reservation-consumer"
	goodsInConsumer     = "inventory-goods-received-consumer"
)

// ReservationHandler memproses OrderPlaced: reservasi stok secara idempoten.
// Jika stok kurang, Inventory menerbitkan StockReservationFailed (kompensasi untuk Order).
func ReservationHandler(db *sqlx.DB, amqpURI string) messaging.Handler {
	var pub *messaging.EventPublisher

	publishFailure := func(ctx context.Context, srcEventID string, data messaging.OrderPlacedData, reason string) error {
		if pub == nil {
			p, err := messaging.NewEventPublisher(amqpURI)
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

	return func(ctx context.Context, ev messaging.Event) error {
		// Validasi: kegagalan di sini = poison pill (tidak akan pernah berhasil)
		if ev.EventType != messaging.EventOrderPlaced {
			return fmt.Errorf("%w: event_type tidak dikenal: %s", messaging.ErrPoison, ev.EventType)
		}
		var data messaging.OrderPlacedData
		if err := json.Unmarshal(ev.Data, &data); err != nil || data.OrderID == "" || data.WarehouseID == "" || len(data.Items) == 0 {
			return fmt.Errorf("%w: payload OrderPlaced tidak valid", messaging.ErrPoison)
		}
		items := make([]ReservationItem, 0, len(data.Items))
		for _, it := range data.Items {
			if it.ProductID == "" || it.Quantity <= 0 {
				return fmt.Errorf("%w: item tidak valid (product=%q qty=%d)", messaging.ErrPoison, it.ProductID, it.Quantity)
			}
			items = append(items, ReservationItem{ProductID: it.ProductID, Quantity: it.Quantity})
		}

		// Mutasi bisnis idempoten (ledger + reservasi dalam satu transaksi)
		err := ReserveOrderStockOnce(ctx, db, reservationConsumer, ev.EventID, ev.EventType, data.OrderID, data.WarehouseID, items)
		switch {
		case err == nil:
			log.Printf("[Inventory] Stok order %s BERHASIL direservasi (event %s)", data.OrderID, ev.EventID)
			return nil
		case errors.Is(err, ErrDuplicateEvent):
			log.Printf("[Dedup Guard] Event %s terdeteksi DUPLIKAT. Melewati logika bisnis dan langsung ACK.", ev.EventID)
			return nil
		case errors.Is(err, ErrInsufficientStock):
			log.Printf("[Inventory] Stok TIDAK CUKUP untuk order %s: %v. Menerbitkan event kompensasi.", data.OrderID, err)
			return publishFailure(ctx, ev.EventID, data, err.Error())
		default:
			return err // galat sementara (mis. DB down): NACK + requeue
		}
	}
}

// GoodsReceivedHandler memproses GoodsReceived dari Procurement: menambah stok di gudang tujuan.
// Dulu Procurement menulis langsung ke tabel inventory_stocks (lintas domain).
func GoodsReceivedHandler(db *sqlx.DB) messaging.Handler {
	return func(ctx context.Context, ev messaging.Event) error {
		if ev.EventType != messaging.EventGoodsReceived {
			return fmt.Errorf("%w: event_type tidak dikenal: %s", messaging.ErrPoison, ev.EventType)
		}
		var d messaging.GoodsReceivedData
		if err := json.Unmarshal(ev.Data, &d); err != nil || d.WarehouseID == "" || len(d.Items) == 0 {
			return fmt.Errorf("%w: payload GoodsReceived tidak valid", messaging.ErrPoison)
		}

		tx, err := db.BeginTxx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		inserted, err := messaging.MarkProcessed(ctx, tx, goodsInConsumer, ev.EventID, ev.EventType)
		if err != nil {
			return err
		}
		if !inserted {
			log.Printf("[Dedup Guard] Event %s terdeteksi DUPLIKAT. ACK tanpa menambah stok lagi.", ev.EventID)
			return nil
		}

		for _, it := range d.Items {
			if it.ProductID == "" || it.Quantity <= 0 {
				return fmt.Errorf("%w: item GRN tidak valid (product=%q qty=%d)", messaging.ErrPoison, it.ProductID, it.Quantity)
			}
			res, err := tx.ExecContext(ctx, `
				UPDATE inventory_stocks
				SET quantity_on_hand = quantity_on_hand + ?, updated_at = NOW()
				WHERE product_id = ? AND warehouse_id = ?`,
				it.Quantity, it.ProductID, d.WarehouseID)
			if err != nil {
				return err
			}
			if n, _ := res.RowsAffected(); n == 0 {
				if _, err := tx.ExecContext(ctx, `
					INSERT INTO inventory_stocks (id, product_id, warehouse_id, quantity_on_hand, quantity_reserved)
					VALUES (?, ?, ?, ?, 0)`,
					uuid.NewString(), it.ProductID, d.WarehouseID, it.Quantity); err != nil {
					return err
				}
			}
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		log.Printf("[Inventory] Stok bertambah dari GRN %s (%d item)", d.GRNNumber, len(d.Items))
		return nil
	}
}
