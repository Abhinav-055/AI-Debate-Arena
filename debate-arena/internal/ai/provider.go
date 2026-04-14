package ai

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"time"
)

// Provider generates content for a debate turn.
type Provider interface {
	Generate(ctx context.Context, req GenerateRequest) (string, error)
	Name() string
}

// GenerateRequest is a normalized generation request for providers.
type GenerateRequest struct {
	SystemPrompt string
	UserMessage  string
	MaxTokens    int
	Temperature  float32
}

// ProviderError classifies provider failures.
type ProviderError struct {
	Provider  string
	Code      int
	Message   string
	Temporary bool
}

// Error returns the formatted provider error string.
func (e *ProviderError) Error() string {
	return fmt.Sprintf("provider=%s code=%d temporary=%t message=%s", e.Provider, e.Code, e.Temporary, e.Message)
}

// ShouldRetry reports whether another attempt should be made.
func (e *ProviderError) ShouldRetry() bool {
	return e != nil && e.Temporary
}

// RetryWithBackoff retries function execution using exponential delays.
func RetryWithBackoff(ctx context.Context, retries int, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt < retries; attempt++ {
		if err := fn(); err != nil {
			lastErr = err
			if pe, ok := err.(*ProviderError); ok {
				if !pe.ShouldRetry() {
					return err
				}
				if pe.Code == http.StatusTooManyRequests {
					delay := time.Duration(math.Pow(2, float64(attempt))) * time.Second
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(delay):
					}
					continue
				}
			}
			continue
		}
		return nil
	}
	return lastErr
}

// LogTokenUsage logs provider token usage metrics.
func LogTokenUsage(logger *slog.Logger, provider string, promptTokens, completionTokens int) {
	logger.Info("provider token usage",
		"provider", provider,
		"prompt_tokens", promptTokens,
		"completion_tokens", completionTokens,
	)
}
