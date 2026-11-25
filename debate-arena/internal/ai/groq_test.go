package ai

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroqRetryOn429(t *testing.T) {
	var calls int32
	var requestedModel string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		var payload struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		requestedModel = payload.Model
		if n <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"rate limit"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"final response"}}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`))
	}))
	defer ts.Close()

	p := NewGroqProvider("test", slog.New(slog.NewTextHandler(io.Discard, nil)))
	p.endpoint = ts.URL

	out, err := p.Generate(context.Background(), GenerateRequest{
		SystemPrompt: "system",
		UserMessage:  "user",
		MaxTokens:    50,
		Temperature:  0.8,
	})
	require.NoError(t, err)
	require.Equal(t, "final response", out)
	require.EqualValues(t, 3, atomic.LoadInt32(&calls))
	require.Equal(t, groqChatModel, requestedModel)
}

func TestGroqRejectsEmptyCompletion(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"  "}}]}`))
	}))
	defer ts.Close()

	p := NewGroqProvider("test", slog.New(slog.NewTextHandler(io.Discard, nil)))
	p.endpoint = ts.URL

	out, err := p.Generate(context.Background(), GenerateRequest{
		SystemPrompt: "system",
		UserMessage:  "user",
		MaxTokens:    50,
	})
	require.ErrorContains(t, err, "empty response content")
	require.Empty(t, out)
}
