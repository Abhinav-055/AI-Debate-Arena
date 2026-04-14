package queue

import (
	"context"

	"github.com/rabbitmq/amqp091-go"
)

// Channel abstracts required RabbitMQ channel methods for testing.
type Channel interface {
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp091.Publishing) error
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp091.Table) (<-chan amqp091.Delivery, error)
}
