package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// ErrPoison dikembalikan Handler untuk pesan yang TIDAK akan pernah bisa diproses (poison pill).
// Pesan itu ditolak tanpa requeue sehingga masuk ke DLQ lewat DLX.
var ErrPoison = errors.New("poison message")

// Handler: nil = sukses (ACK); ErrPoison = Reject(requeue=false) -> DLQ; error lain = NACK + requeue.
type Handler func(ctx context.Context, ev Event) error

// Consume membaca satu queue sampai ctx selesai atau koneksi terputus.
func Consume(ctx context.Context, uri, queue, tag string, prefetch int, h Handler) error {
	conn, err := amqp.Dial(uri)
	if err != nil {
		return fmt.Errorf("gagal koneksi RabbitMQ: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("gagal membuka channel: %w", err)
	}
	defer ch.Close()

	if err := DeclareTopology(ch); err != nil {
		return fmt.Errorf("gagal deklarasi topologi: %w", err)
	}

	// 1. QoS: maksimal `prefetch` pesan unacknowledged per worker (membatasi memori)
	if err := ch.Qos(prefetch, 0, false); err != nil {
		return fmt.Errorf("gagal menyetel QoS: %w", err)
	}

	// 2. auto_ack = FALSE (manual acknowledgement)
	msgs, err := ch.Consume(queue, tag, false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("gagal mendaftarkan consumer: %w", err)
	}

	closed := conn.NotifyClose(make(chan *amqp.Error, 1))
	delay := processDelay()
	log.Printf("[Consumer %s] Menunggu pesan pada queue %s (prefetch=%d, manual ACK)", tag, queue, prefetch)

	for {
		select {
		case <-ctx.Done():
			return nil
		case e := <-closed:
			return fmt.Errorf("koneksi RabbitMQ terputus: %v", e)
		case d, ok := <-msgs:
			if !ok {
				return errors.New("channel consumer ditutup")
			}
			handleDelivery(ctx, tag, d, h, delay)
		}
	}
}

func handleDelivery(parent context.Context, tag string, d amqp.Delivery, h Handler, delay time.Duration) {
	if delay > 0 {
		time.Sleep(delay) // hanya untuk demo (PROCESS_DELAY_MS)
	}

	var ev Event
	if err := json.Unmarshal(d.Body, &ev); err != nil || ev.EventID == "" || ev.EventType == "" {
		log.Printf("[Consumer %s] POISON PILL (payload rusak / tanpa event_id): %v -> Reject(requeue=false) ke DLQ", tag, err)
		_ = d.Reject(false)
		return
	}

	ctx, cancel := context.WithTimeout(parent, 15*time.Second)
	defer cancel()

	err := h(ctx, ev)
	switch {
	case err == nil:
		_ = d.Ack(false) // ACK hanya SETELAH data berhasil di-commit
	case errors.Is(err, ErrPoison):
		log.Printf("[Consumer %s] POISON PILL event %s: %v -> Reject(requeue=false) ke DLQ", tag, ev.EventID, err)
		_ = d.Reject(false)
	default:
		log.Printf("[Consumer %s] Gagal sementara event %s: %v -> NACK requeue", tag, ev.EventID, err)
		time.Sleep(2 * time.Second)
		_ = d.Nack(false, true)
	}
}

func processDelay() time.Duration {
	if ms, err := strconv.Atoi(os.Getenv("PROCESS_DELAY_MS")); err == nil && ms > 0 {
		return time.Duration(ms) * time.Millisecond
	}
	return 0
}
