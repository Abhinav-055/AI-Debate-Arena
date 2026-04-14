package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

// Envelope is the standard message wrapper used across all queues.
type Envelope struct {
	EventType  string          `json:"event_type"`
	DebateID   string          `json:"debate_id"`
	Payload    json.RawMessage `json:"payload"`
	RetryCount int             `json:"retry_count"`
	Timestamp  time.Time       `json:"timestamp"`
}

// Publisher emits queue events.
type Publisher struct {
	ch Channel
}

// NewPublisher constructs a queue publisher.
func NewPublisher(ch Channel) *Publisher {
	return &Publisher{ch: ch}
}

// Publish serializes and sends one envelope to RabbitMQ.
func (p *Publisher) Publish(ctx context.Context, routingKey string, msg Envelope) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}

	if err := p.ch.PublishWithContext(ctx, ExchangeDebateEvents, routingKey, false, false, amqp091.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp091.Persistent,
		Timestamp:    time.Now().UTC(),
		Body:         body,
	}); err != nil {
		return fmt.Errorf("publish envelope: %w", err)
	}

	return nil
}
