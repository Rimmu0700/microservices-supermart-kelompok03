package messaging

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type QueueSpec struct {
	Name        string
	BindingKeys []string
}

// Binding key memakai wildcard topic: * = tepat 1 kata, # = nol atau lebih kata.
var Queues = []QueueSpec{
	{QueueInventoryReservation, []string{"order.*.placed"}},
	{QueueOrderCompensation, []string{"inventory.*.reservation_failed"}},
	{QueueNotification, []string{"order.#", "inventory.#"}},
}

// DeclareTopology idempoten. Dipanggil oleh SEMUA proses (API dan worker) agar semua queue
// selalu ada, sehingga pesan tetap tertampung walau consumer sedang mati.
func DeclareTopology(ch *amqp.Channel) error {
	// durable = true untuk semua exchange dan queue
	if err := ch.ExchangeDeclare(ExchangeEvents, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("exchange %s: %w", ExchangeEvents, err)
	}

	// Topologi karantina: DLX + DLQ
	if err := ch.ExchangeDeclare(ExchangeDeadLetter, "fanout", true, false, false, false, nil); err != nil {
		return fmt.Errorf("exchange %s: %w", ExchangeDeadLetter, err)
	}
	if _, err := ch.QueueDeclare(QueueDLQ, true, false, false, false, nil); err != nil {
		return fmt.Errorf("queue %s: %w", QueueDLQ, err)
	}
	if err := ch.QueueBind(QueueDLQ, "", ExchangeDeadLetter, false, nil); err != nil {
		return fmt.Errorf("bind %s: %w", QueueDLQ, err)
	}

	// Queue kerja dengan argumen x-dead-letter-exchange
	args := amqp.Table{"x-dead-letter-exchange": ExchangeDeadLetter}
	for _, q := range Queues {
		if _, err := ch.QueueDeclare(q.Name, true, false, false, false, args); err != nil {
			return fmt.Errorf("queue %s: %w", q.Name, err)
		}
		for _, key := range q.BindingKeys {
			if err := ch.QueueBind(q.Name, key, ExchangeEvents, false, nil); err != nil {
				return fmt.Errorf("bind %s (%s): %w", q.Name, key, err)
			}
		}
	}
	return nil
}
