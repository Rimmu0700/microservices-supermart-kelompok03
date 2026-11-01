package messaging

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-sql-driver/mysql"
)

// execer dipenuhi oleh *sqlx.DB maupun *sqlx.Tx (sehingga ledger bisa ikut transaksi bisnis).
type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func IsDuplicateKey(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

// MarkProcessed mencatat (consumer, event_id) ke processed_events.
// inserted=false berarti event ini SUDAH pernah diproses consumer tersebut (duplikat).
func MarkProcessed(ctx context.Context, ex execer, consumer, eventID, eventType string) (bool, error) {
	_, err := ex.ExecContext(ctx,
		`INSERT INTO processed_events (consumer_name, event_id, event_type) VALUES (?, ?, ?)`,
		consumer, eventID, eventType)
	if err != nil {
		if IsDuplicateKey(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// UnmarkProcessed menghapus catatan (dipakai bila pemrosesan gagal setelah klaim).
func UnmarkProcessed(ctx context.Context, ex execer, consumer, eventID string) error {
	_, err := ex.ExecContext(ctx,
		`DELETE FROM processed_events WHERE consumer_name = ? AND event_id = ?`, consumer, eventID)
	return err
}
