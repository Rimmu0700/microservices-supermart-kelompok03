package messaging

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	// Topic Exchange utama untuk seluruh event domain
	ExchangeEvents = "supermart.events"
	// Dead-Letter Exchange (fanout) dan Dead-Letter Queue
	ExchangeDeadLetter = "supermart.deadletter.exchange"
	QueueDLQ           = "supermart.dlq"

	// Queue consumer (konvensi: <domain-pemilik>.<tujuan>.q)
	QueueInventoryReservation = "inventory.stock-reservation.q"
	QueueOrderCompensation    = "order.reservation-failed.q"
	QueueNotification         = "notification.order-events.q"

	EventOrderPlaced            = "OrderPlaced"
	EventStockReservationFailed = "StockReservationFailed"
)

// Event adalah amplop (envelope) baku untuk semua domain event.
type Event struct {
	EventID   string          `json:"event_id"`   // UUID unik (pelacakan dan idempotensi)
	EventType string          `json:"event_type"` // nama peristiwa bentuk lampau
	Timestamp string          `json:"timestamp"`  // ISO-8601 (UTC)
	Version   string          `json:"version"`
	Producer  string          `json:"producer"`
	Data      json.RawMessage `json:"data"`
}

type OrderItemData struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type OrderPlacedData struct {
	OrderID        string          `json:"order_id"`
	OrderNumber    string          `json:"order_number"`
	CustomerID     string          `json:"customer_id"`
	WarehouseID    string          `json:"warehouse_id"`
	Region         string          `json:"region"`
	TotalNetAmount int64           `json:"total_net_amount"` // rupiah, integer
	Currency       string          `json:"currency"`
	Items          []OrderItemData `json:"items"`
}

type ReservationFailedData struct {
	OrderID string `json:"order_id"`
	Region  string `json:"region"`
	Reason  string `json:"reason"`
}

// NewEvent membungkus data menjadi Event dengan metadata enterprise.
func NewEvent(eventType, producer string, data any) (Event, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return Event{}, err
	}
	return Event{
		EventID:   uuid.NewString(),
		EventType: eventType,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   "1.0",
		Producer:  producer,
		Data:      raw,
	}, nil
}

// AMQPURI membaca alamat broker dari env AMQP_URI (default: lokal).
// vhost "/supermart" ditulis %2Fsupermart pada URI.
func AMQPURI() string {
	if v := os.Getenv("AMQP_URI"); v != "" {
		return v
	}
	return "amqp://nusantara_mq:nusantara_mq_secret@localhost:5672/%2Fsupermart"
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// Slug mengubah "Jakarta Barat" menjadi "jakarta-barat" (tanpa titik, karena titik = pemisah kata routing key).
func Slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.Trim(nonAlnum.ReplaceAllString(s, "-"), "-")
}
