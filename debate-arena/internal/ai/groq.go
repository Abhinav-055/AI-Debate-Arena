package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// GroqProvider implements the Groq API.
type GroqProvider struct {
	apiKey     string
	endpoint   string
	httpClient *http.Client
	logger     *slog.Logger
}

const groqChatModel = "openai/gpt-oss-20b"

// NewGroqProvider creates a Groq provider client.
func NewGroqProvider(apiKey string, logger *slog.Logger) *GroqProvider {
	return &GroqProvider{
		apiKey:     apiKey,
		endpoint:   "https://api.groq.com/openai/v1/chat/completions",
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     logger,
	}
}

// Name returns the provider identifier.
func (p *GroqProvider) Name() string {
	return "groq"
}

// Generate creates a completion with the configured Groq chat model.
func (p *GroqProvider) Generate(ctx context.Context, req GenerateRequest) (string, error) {
	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type groqRequest struct {
		Model       string    `json:"model"`
		Messages    []message `json:"messages"`
		MaxTokens   int       `json:"max_tokens"`
		Temperature float32   `json:"temperature"`
	}
	type groqResponse struct {
		Choices []struct {
			Message message `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}

	payload := groqRequest{
		Model: groqChatModel,
		Messages: []message{
			{Role: "system", Content: req.SystemPrompt},
			{Role: "user", Content: req.UserMessage},
		},
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}

	var out string
	err := RetryWithBackoff(ctx, 3, func() error {
		body, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal groq request: %w", err)
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("new groq request: %w", err)
		}
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := p.httpClient.Do(httpReq)
		if err != nil {
			return &ProviderError{Provider: p.Name(), Message: err.Error(), Temporary: true}
		}
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == http.StatusTooManyRequests {
			return &ProviderError{Provider: p.Name(), Code: resp.StatusCode, Message: string(respBody), Temporary: true}
		}
		if resp.StatusCode >= 500 {
			return &ProviderError{Provider: p.Name(), Code: resp.StatusCode, Message: string(respBody), Temporary: true}
		}
		if resp.StatusCode >= 400 {
			return &ProviderError{Provider: p.Name(), Code: resp.StatusCode, Message: string(respBody), Temporary: false}
		}

		var decoded groqResponse
		if err := json.Unmarshal(respBody, &decoded); err != nil {
			return fmt.Errorf("decode groq response: %w", err)
		}
		if len(decoded.Choices) == 0 {
			return &ProviderError{Provider: p.Name(), Message: "empty choices", Temporary: false}
		}

		out = strings.TrimSpace(decoded.Choices[0].Message.Content)
		if out == "" {
			return &ProviderError{Provider: p.Name(), Message: "empty response content", Temporary: false}
		}
		LogTokenUsage(p.logger, p.Name(), decoded.Usage.PromptTokens, decoded.Usage.CompletionTokens)
		return nil
	})
	if err != nil {
		return "", err
	}

	return out, nil
}
