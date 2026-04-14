package queue

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
)

type fakeAcknowledger struct{}

func (a *fakeAcknowledger) Ack(tag uint64, multiple bool) error { return nil }
func (a *fakeAcknowledger) Nack(tag uint64, multiple bool, requeue bool) error {
	return nil
}
func (a *fakeAcknowledger) Reject(tag uint64, requeue bool) error { return nil }

type fakeChannel struct {
	published    []amqp091.Publishing
	publishedKey []string
	consumeCh    chan amqp091.Delivery
}

func (f *fakeChannel) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp091.Publishing) error {
	f.published = append(f.published, msg)
	f.publishedKey = append(f.publishedKey, key)
	return nil
}

func (f *fakeChannel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp091.Table) (<-chan amqp091.Delivery, error) {
	return f.consumeCh, nil
}

func TestPublishConsumeRoundTrip(t *testing.T) {
	fc := &fakeChannel{consumeCh: make(chan amqp091.Delivery, 1)}
	pub := NewPublisher(fc)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	consumer := NewConsumer(fc, logger)

	env := Envelope{
		EventType:  "turn.start",
		DebateID:   "debate-1",
		Payload:    json.RawMessage(`{"round":1}`),
		RetryCount: 0,
		Timestamp:  time.Now().UTC(),
	}

	err := pub.Publish(context.Background(), RoutingDebateTurnRequested, env)
	require.NoError(t, err)
	require.Len(t, fc.published, 1)
	require.Equal(t, RoutingDebateTurnRequested, fc.publishedKey[0])

	d := amqp091.Delivery{Acknowledger: &fakeAcknowledger{}, Body: fc.published[0].Body}
	fc.consumeCh <- d
	close(fc.consumeCh)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handled := false
	err = consumer.Consume(ctx, QueueDebateTurns, func(ctx context.Context, got Envelope) error {
		handled = true
		require.Equal(t, env.EventType, got.EventType)
		require.Equal(t, env.DebateID, got.DebateID)
		return nil
	})
	require.NoError(t, err)
	require.True(t, handled)
}
