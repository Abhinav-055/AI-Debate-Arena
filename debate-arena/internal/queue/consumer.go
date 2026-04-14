package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

// Handler handles a decoded envelope.
type Handler func(ctx context.Context, env Envelope) error

// Consumer wraps delivery consumption and decoding.
type Consumer struct {
	ch     Channel
	logger *slog.Logger
}

// NewConsumer constructs a generic queue consumer.
func NewConsumer(ch Channel, logger *slog.Logger) *Consumer {
	return &Consumer{ch: ch, logger: logger}
}

// Consume starts consuming one queue and dispatching messages to the handler.
func (c *Consumer) Consume(ctx context.Context, queueName string, handler Handler) error {
	deliveries, err := c.ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume from queue: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-deliveries:
			if !ok {
				return nil
			}

			var env Envelope
			if err := json.Unmarshal(d.Body, &env); err != nil {
				c.logger.Error("invalid envelope", "error", err, "queue", queueName)
				_ = d.Nack(false, false)
				continue
			}

			if err := handler(ctx, env); err != nil {
				c.logger.Error("handler failed", "error", err, "event_type", env.EventType, "debate_id", env.DebateID, "retry_count", env.RetryCount)
				if repErr := c.retryOrDeadLetter(ctx, queueName, env); repErr != nil {
					c.logger.Error("retry publish failed", "error", repErr, "event_type", env.EventType, "debate_id", env.DebateID)
					_ = d.Nack(false, false)
					continue
				}
				_ = d.Ack(false)
				continue
			}

			_ = d.Ack(false)
		}
	}
}

func (c *Consumer) retryOrDeadLetter(ctx context.Context, queueName string, env Envelope) error {
	env.RetryCount++
	env.Timestamp = time.Now().UTC()

	body, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal retry envelope: %w", err)
	}

	exchange := ExchangeDebateEvents
	routingKey := routingKeyForQueue(queueName)
	if env.RetryCount >= 3 {
		exchange = ExchangeDebateDLX
		routingKey = "dead.letter"
	}

	if err := c.ch.PublishWithContext(ctx, exchange, routingKey, false, false, amqp091.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp091.Persistent,
		Timestamp:    time.Now().UTC(),
		Body:         body,
	}); err != nil {
		return fmt.Errorf("publish retry message: %w", err)
	}

	return nil
}

func routingKeyForQueue(queueName string) string {
	switch queueName {
	case QueueDebateTurns:
		return RoutingDebateTurnRequested
	case QueueDebateAITasks:
		return RoutingDebateAIGenerate
	case QueueDebateJudgeTasks:
		return RoutingDebateJudgeEvaluate
	default:
		return ""
	}
}
