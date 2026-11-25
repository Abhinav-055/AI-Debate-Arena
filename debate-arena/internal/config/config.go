package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config contains application runtime configuration.
type Config struct {
	DatabaseURL    string
	RedisURL       string
	RabbitMQURL    string
	GroqAPIKey     string
	JWTSecret      string
	FrontendOrigin string
	Port           string
	LogLevel       string
	MaxRounds      int
}

// Load reads environment-backed configuration via Viper.
func Load() (Config, error) {
	v := viper.New()
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	_ = v.ReadInConfig()

	v.SetDefault("PORT", "8080")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("MAX_ROUNDS", 3)
	v.SetDefault("FRONTEND_ORIGIN", "http://localhost:5173")

	cfg := Config{
		DatabaseURL:    v.GetString("DATABASE_URL"),
		RedisURL:       v.GetString("REDIS_URL"),
		RabbitMQURL:    v.GetString("RABBITMQ_URL"),
		GroqAPIKey:     v.GetString("GROQ_API_KEY"),
		JWTSecret:      v.GetString("JWT_SECRET"),
		FrontendOrigin: v.GetString("FRONTEND_ORIGIN"),
		Port:           v.GetString("PORT"),
		LogLevel:       v.GetString("LOG_LEVEL"),
		MaxRounds:      v.GetInt("MAX_ROUNDS"),
	}

	if cfg.DatabaseURL == "" || cfg.RedisURL == "" || cfg.RabbitMQURL == "" || cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("config validation: required env vars are missing")
	}

	return cfg, nil
}
