package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"debate-arena/internal/graph"
	"debate-arena/internal/graph/generated"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestGraphQLWebsocketAllowsConfiguredOrigin(t *testing.T) {
	const origin = "http://localhost:5173"
	srv := httptest.NewServer(newGraphQLServer(
		generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{}}),
		origin,
	))
	defer srv.Close()

	dialer := websocket.Dialer{Subprotocols: []string{"graphql-transport-ws"}}
	conn, response, err := dialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), http.Header{
		"Origin": []string{origin},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, response.StatusCode)
	require.NoError(t, conn.Close())
}

func TestGraphQLWebsocketRejectsUnconfiguredOrigin(t *testing.T) {
	srv := httptest.NewServer(newGraphQLServer(
		generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{}}),
		"http://localhost:5173",
	))
	defer srv.Close()

	dialer := websocket.Dialer{Subprotocols: []string{"graphql-transport-ws"}}
	conn, response, err := dialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), http.Header{
		"Origin": []string{"http://attacker.example"},
	})
	require.Error(t, err)
	require.Equal(t, http.StatusForbidden, response.StatusCode)
	if conn != nil {
		require.NoError(t, conn.Close())
	}
}
