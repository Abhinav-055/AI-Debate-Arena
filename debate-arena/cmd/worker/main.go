package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"debate-arena/internal/ai"
	"debate-arena/internal/config"
	"debate-arena/internal/db/queries"
	"debate-arena/internal/models"
	"debate-arena/internal/queue"
	"debate-arena/internal/workers"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

// main runs worker consumers for turn, AI, and judge queues.
func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		panic(err)
	}
	rdb := redis.NewClient(redisOpts)
	defer rdb.Close()

	conn, err := amqp091.Dial(cfg.RabbitMQURL)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		panic(err)
	}
	defer ch.Close()

	if err := queue.DeclareTopology(ch); err != nil {
		panic(err)
	}

	store := queries.NewStore(pool)
	pub := queue.NewPublisher(ch)
	consumer := queue.NewConsumer(ch, logger)

	groqProvider := ai.NewGroqProvider(cfg.GroqAPIKey, logger)

	providerMap := map[models.AgentProvider]ai.Provider{
		models.AgentProviderGroq: groqProvider,
	}

	debateWorker := workers.NewDebateWorker(store, rdb, pub, logger)
	aiWorker := workers.NewAIWorker(store, rdb, pub, providerMap, logger)
	judgeWorker := workers.NewJudgeWorker(store, rdb, groqProvider, logger)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		if err := consumer.Consume(ctx, queue.QueueDebateTurns, debateWorker.HandleTurn); err != nil {
			logger.Error("debate worker failed", "error", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := consumer.Consume(ctx, queue.QueueDebateAITasks, aiWorker.HandleTask); err != nil {
			logger.Error("ai worker failed", "error", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := consumer.Consume(ctx, queue.QueueDebateJudgeTasks, judgeWorker.HandleTask); err != nil {
			logger.Error("judge worker failed", "error", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	cancel()
	wg.Wait()
}
