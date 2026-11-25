package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"debate-arena/internal/config"
	"debate-arena/internal/db/queries"
	"debate-arena/internal/graph"
	"debate-arena/internal/graph/generated"
	"debate-arena/internal/middleware"
	"debate-arena/internal/queue"
	"debate-arena/internal/subscriptions"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/vektah/gqlparser/v2/ast"
)

// main runs the GraphQL API server.
func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parseLogLevel(cfg.LogLevel)}))

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

	rabbitCtx, rabbitCancel := context.WithTimeout(ctx, time.Minute)
	conn, err := dialRabbitMQ(rabbitCtx, cfg.RabbitMQURL, logger)
	rabbitCancel()
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
	subs := subscriptions.NewManager(rdb, logger)
	go func() {
		if err := subs.Start(ctx); err != nil {
			logger.Error("subscription manager stopped", "error", err)
		}
	}()

	srv := newGraphQLServer(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{
		Store:         store,
		Publisher:     pub,
		Subscriptions: subs,
		Logger:        logger,
	}}), cfg.FrontendOrigin)

	r := chi.NewRouter()
	r.Use(middleware.CORS())
	r.Use(middleware.Logging(logger))
	r.Use(middleware.Auth(cfg.JWTSecret))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		playground.Handler("GraphQL", "/query").ServeHTTP(w, r)
	})
	r.Handle("/query", srv)
	r.Handle("/graphql", srv)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	httpSrv := &http.Server{Addr: ":" + cfg.Port, Handler: r}
	go func() {
		logger.Info("api server started", "port", cfg.Port)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server failed", "error", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = httpSrv.Shutdown(shutdownCtx)
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func dialRabbitMQ(ctx context.Context, url string, logger *slog.Logger) (*amqp091.Connection, error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		conn, err := amqp091.Dial(url)
		if err == nil {
			return conn, nil
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("connect to RabbitMQ: %w (last attempt: %v)", ctx.Err(), err)
		case <-ticker.C:
			logger.Warn("RabbitMQ unavailable; retrying", "error", err)
		}
	}
}

func newGraphQLServer(schema graphql.ExecutableSchema, frontendOrigin string) *handler.Server {
	srv := handler.New(schema)
	srv.AddTransport(transport.Websocket{
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				return origin == "" || origin == frontendOrigin
			},
		},
		KeepAlivePingInterval: 10 * time.Second,
	})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})
	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))
	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{Cache: lru.New[string](100)})
	return srv
}
