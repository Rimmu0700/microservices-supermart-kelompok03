package inventory

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/nusantara-supermart/backend/internal/messaging"
)

var (
	ErrDuplicateEvent    = errors.New("event sudah pernah diproses")
	ErrInsufficientStock = errors.New("stok tersedia tidak mencukupi")
)

type ReservationItem struct {
	ProductID string
	Quantity  int
}

// ReserveOrderStockOnce memproses event OrderPlaced secara idempoten:
//  1. klaim event_id di processed_events (kunci unik),
//  2. reservasi tiap item,
//  3. commit semuanya dalam SATU transaksi.
//
// Bila stok kurang, transaksi di-rollback (termasuk catatan ledger) dan ErrInsufficientStock dikembalikan.
func ReserveOrderStockOnce(ctx context.Context, db *sqlx.DB, consumer, eventID, eventType, orderID, warehouseID string, items []ReservationItem) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	inserted, err := messaging.MarkProcessed(ctx, tx, consumer, eventID, eventType)
	if err != nil {
		return err
	}
	if !inserted {
		return ErrDuplicateEvent
	}

	for _, it := range items {
		res, err := tx.ExecContext(ctx, `
			UPDATE inventory_stocks
			SET quantity_reserved = quantity_reserved + ?, updated_at = NOW()
			WHERE product_id = ? AND warehouse_id = ? AND (quantity_on_hand - quantity_reserved) >= ?`,
			it.Quantity, it.ProductID, warehouseID, it.Quantity)
		if err != nil {
			return err
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("%w: produk %s di gudang %s", ErrInsufficientStock, it.ProductID, warehouseID)
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO stock_reservations (id, product_id, reference_order_id, reserved_quantity, status, created_at)
			VALUES (?, ?, ?, ?, 'HOLD', NOW())`,
			uuid.NewString(), it.ProductID, orderID, it.Quantity); err != nil {
			return err
		}
	}
	return tx.Commit()
}
