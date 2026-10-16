package messaging

import (
	"context"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

type OutboxMessage struct {
	ID            string // = event_id
	AggregateType string
	AggregateID   string
	EventType     string
	RoutingKey    string
	Payload       []byte // JSON lengkap (Event)
}

// InsertOutbox menulis event ke tabel outbox. Dipanggil di dalam transaksi bisnis yang SAMA.
func InsertOutbox(ctx context.Context, ex execer, m OutboxMessage) error {
	// payload dikirim sebagai string (bukan []byte) agar diterima kolom JSON MySQL
	_, err := ex.ExecContext(ctx, `
		INSERT INTO outbox_events (id, aggregate_type, aggregate_id, event_type, routing_key, payload)
		VALUES (?, ?, ?, ?, ?, ?)`,
		m.ID, m.AggregateType, m.AggregateID, m.EventType, m.RoutingKey, string(m.Payload))
	return err
}

type outboxRow struct {
	ID         string `db:"id"`
	EventType  string `db:"event_type"`
	RoutingKey string `db:"routing_key"`
	Payload    []byte `db:"payload"`
}

// OutboxRelay membaca event yang belum terbit lalu menerbitkannya ke RabbitMQ.
// Bila broker mati, event tetap aman di database dan diterbitkan nanti (at-least-once).
type OutboxRelay struct {
	DB        *sqlx.DB
	URI       string
	Interval  time.Duration
	BatchSize int
}

func (r *OutboxRelay) Run(ctx context.Context) {
	interval := r.Interval
	if interval <= 0 {
		interval = time.Second
	}
	batch := r.BatchSize
	if batch <= 0 {
		batch = 50
	}

	var pub *EventPublisher
	defer func() {
		if pub != nil {
			pub.Close()
		}
	}()

	log.Printf("[Outbox Relay] Aktif (interval %v, batch %d)", interval, batch)
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}

		var rows []outboxRow
		if err := r.DB.SelectContext(ctx, &rows, `
			SELECT id, event_type, routing_key, payload
			FROM outbox_events
			WHERE published_at IS NULL
			ORDER BY created_at ASC
			LIMIT ?`, batch); err != nil {
			log.Printf("[Outbox Relay] Gagal membaca outbox: %v", err)
			continue
		}
		if len(rows) == 0 {
			continue
		}

		if pub == nil {
			p, err := NewEventPublisher(r.URI)
			if err != nil {
				log.Printf("[Outbox Relay] RabbitMQ tidak tersedia, %d event menunggu di outbox: %v", len(rows), err)
				continue
			}
			pub = p
		}

		for _, row := range rows {
			if err := pub.PublishRaw(ctx, ExchangeEvents, row.RoutingKey, row.ID, row.EventType, row.Payload); err != nil {
				log.Printf("[Outbox Relay] Gagal menerbitkan event %s: %v", row.ID, err)
				pub.Close()
				pub = nil
				break
			}
			// Jika proses mati TEPAT di sini, event akan diterbitkan ulang: itu sebabnya consumer wajib idempoten.
			if _, err := r.DB.ExecContext(ctx, `UPDATE outbox_events SET published_at = NOW(3) WHERE id = ?`, row.ID); err != nil {
				log.Printf("[Outbox Relay] Gagal menandai event %s terbit: %v", row.ID, err)
				continue
			}
			log.Printf("[Outbox Relay] Event %s (%s) diterbitkan ke %s", row.ID, row.EventType, row.RoutingKey)
		}
	}
}
