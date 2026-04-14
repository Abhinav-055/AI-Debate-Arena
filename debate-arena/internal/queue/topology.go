package queue

import (
	"fmt"

	"github.com/rabbitmq/amqp091-go"
)

const (
	// ExchangeDebateEvents is the main topic exchange for debate events.
	ExchangeDebateEvents = "debate.events"
	// ExchangeDebateDLX is the dead-letter exchange for retries.
	ExchangeDebateDLX = "debate.dlx"

	// QueueDebateTurns receives turn orchestration requests.
	QueueDebateTurns = "debate.turns"
	// QueueDebateAITasks receives AI generation tasks.
	QueueDebateAITasks = "debate.ai.tasks"
	// QueueDebateJudgeTasks receives final judging tasks.
	QueueDebateJudgeTasks = "debate.judge.tasks"

	// RoutingDebateTurnRequested routes turn events.
	RoutingDebateTurnRequested = "debate.turn.requested"
	// RoutingDebateAIGenerate routes AI generation events.
	RoutingDebateAIGenerate = "debate.ai.generate"
	// RoutingDebateJudgeEvaluate routes judge evaluation events.
	RoutingDebateJudgeEvaluate = "debate.judge.evaluate"
)

// DeclareTopology ensures exchanges, queues, and bindings exist.
func DeclareTopology(ch *amqp091.Channel) error {
	if err := ch.ExchangeDeclare(ExchangeDebateEvents, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare events exchange: %w", err)
	}
	if err := ch.ExchangeDeclare(ExchangeDebateDLX, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dlx exchange: %w", err)
	}

	args := amqp091.Table{
		"x-dead-letter-exchange":    ExchangeDebateDLX,
		"x-message-ttl":             int32(30000),
		"x-max-delivery-attempts":   int32(3),
		"x-dead-letter-routing-key": "dead.letter",
	}

	if _, err := ch.QueueDeclare(QueueDebateTurns, true, false, false, false, args); err != nil {
		return fmt.Errorf("declare turns queue: %w", err)
	}
	if _, err := ch.QueueDeclare(QueueDebateAITasks, true, false, false, false, args); err != nil {
		return fmt.Errorf("declare ai queue: %w", err)
	}
	if _, err := ch.QueueDeclare(QueueDebateJudgeTasks, true, false, false, false, args); err != nil {
		return fmt.Errorf("declare judge queue: %w", err)
	}

	if _, err := ch.QueueDeclare("debate.dlx", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dlx queue: %w", err)
	}

	if err := ch.QueueBind(QueueDebateTurns, RoutingDebateTurnRequested, ExchangeDebateEvents, false, nil); err != nil {
		return fmt.Errorf("bind turns queue: %w", err)
	}
	if err := ch.QueueBind(QueueDebateAITasks, RoutingDebateAIGenerate, ExchangeDebateEvents, false, nil); err != nil {
		return fmt.Errorf("bind ai queue: %w", err)
	}
	if err := ch.QueueBind(QueueDebateJudgeTasks, RoutingDebateJudgeEvaluate, ExchangeDebateEvents, false, nil); err != nil {
		return fmt.Errorf("bind judge queue: %w", err)
	}
	if err := ch.QueueBind("debate.dlx", "dead.letter", ExchangeDebateDLX, false, nil); err != nil {
		return fmt.Errorf("bind dlx queue: %w", err)
	}

	return nil
}
