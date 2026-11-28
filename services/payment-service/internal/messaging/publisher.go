package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var ErrUnroutable = errors.New("pesan tidak terrute ke queue mana pun")

type EventPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	returns chan amqp.Return
}

func NewEventPublisher(uri string) (*EventPublisher, error) {
	conn, err := amqp.Dial(uri)
	if err != nil {
		return nil, fmt.Errorf("gagal koneksi RabbitMQ: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("gagal membuka channel: %w", err)
	}
	if err := DeclareTopology(ch); err != nil {
		conn.Close()
		return nil, fmt.Errorf("gagal deklarasi topologi: %w", err)
	}
	// Publisher Confirms: broker mengonfirmasi pesan sudah diterima dan ditulis
	if err := ch.Confirm(false); err != nil {
		conn.Close()
		return nil, fmt.Errorf("gagal mengaktifkan publisher confirms: %w", err)
	}
	returns := ch.NotifyReturn(make(chan amqp.Return, 16))
	return &EventPublisher{conn: conn, channel: ch, returns: returns}, nil
}

// PublishRaw menerbitkan body JSON yang sudah jadi (dipakai relay outbox).
func (p *EventPublisher) PublishRaw(ctx context.Context, exchange, routingKey, messageID, eventType string, body []byte) error {
	dc, err := p.channel.PublishWithDeferredConfirmWithContext(ctx,
		exchange,
		routingKey,
		true,  // mandatory: dikembalikan broker bila tidak ada queue yang cocok
		false, // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent, // WAJIB: pesan ditulis ke disk (nilai 2)
			ContentType:  "application/json",
			MessageId:    messageID,
			Type:         eventType,
			Timestamp:    time.Now(),
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("gagal menerbitkan pesan: %w", err)
	}

	acked, err := dc.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("gagal menunggu publisher confirm: %w", err)
	}
	if !acked {
		return fmt.Errorf("broker menolak (nack) pesan %s", messageID)
	}

	select {
	case r := <-p.returns:
		return fmt.Errorf("%w: routing key=%s", ErrUnroutable, r.RoutingKey)
	default:
	}
	return nil
}

// PublishDomainEvent: padanan dari acuan tugas (payload any -> JSON -> publish).
func (p *EventPublisher) PublishDomainEvent(ctx context.Context, exchange, routingKey string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	var id, typ string
	if ev, ok := payload.(Event); ok {
		id, typ = ev.EventID, ev.EventType
	}
	return p.PublishRaw(ctx, exchange, routingKey, id, typ, body)
}

func (p *EventPublisher) Close() {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
}
