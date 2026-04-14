package subscriptions

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
)

// Event represents a pub/sub payload forwarded to GraphQL subscribers.
type Event struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// Manager routes Redis pub/sub events to debate subscribers.
type Manager struct {
	redis  *redis.Client
	logger *slog.Logger

	mu          sync.RWMutex
	subscribers map[string][]chan Event
}

// NewManager constructs a subscription manager.
func NewManager(redisClient *redis.Client, logger *slog.Logger) *Manager {
	return &Manager{
		redis:       redisClient,
		logger:      logger,
		subscribers: make(map[string][]chan Event),
	}
}

// Start subscribes to Redis pattern debate:* and dispatches events.
func (m *Manager) Start(ctx context.Context) error {
	ps := m.redis.PSubscribe(ctx, "debate:*")
	defer ps.Close()

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-ps.Channel():
			if msg == nil {
				continue
			}
			debateID := strings.TrimPrefix(msg.Channel, "debate:")
			var event Event
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				m.logger.Error("decode subscription event", "error", err)
				continue
			}
			m.forward(debateID, event)
		}
	}
}

// Subscribe registers a buffered channel for a debate stream.
func (m *Manager) Subscribe(debateID string) (<-chan Event, func(), error) {
	if debateID == "" {
		return nil, nil, fmt.Errorf("subscribe: empty debate id")
	}
	ch := make(chan Event, 50)

	m.mu.Lock()
	m.subscribers[debateID] = append(m.subscribers[debateID], ch)
	m.mu.Unlock()

	unsubscribe := func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		arr := m.subscribers[debateID]
		next := make([]chan Event, 0, len(arr))
		for _, c := range arr {
			if c != ch {
				next = append(next, c)
			}
		}
		if len(next) == 0 {
			delete(m.subscribers, debateID)
		} else {
			m.subscribers[debateID] = next
		}
		close(ch)
	}

	return ch, unsubscribe, nil
}

func (m *Manager) forward(debateID string, event Event) {
	m.mu.RLock()
	listeners := append([]chan Event(nil), m.subscribers[debateID]...)
	m.mu.RUnlock()

	for _, ch := range listeners {
		select {
		case ch <- event:
		default:
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- event:
			default:
				m.logger.Warn("dropping subscription event due to backpressure", "debate_id", debateID, "event_type", event.Type)
			}
		}
	}
}
