package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userIDKey contextKey = "user_id"

// Auth returns middleware that validates JWT bearer tokens.
func Auth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				http.Error(w, "invalid authorization header", http.StatusUnauthorized)
				return
			}

			tkn, err := jwt.Parse(parts[1], func(token *jwt.Token) (any, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method")
				}
				return []byte(secret), nil
			})
			if err != nil || !tkn.Valid {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			claims, ok := tkn.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, "invalid token claims", http.StatusUnauthorized)
				return
			}
			uid, _ := claims["sub"].(string)
			ctx := context.WithValue(r.Context(), userIDKey, uid)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext extracts the authenticated user ID if present.
func UserIDFromContext(ctx context.Context) (string, bool) {
	uid, ok := ctx.Value(userIDKey).(string)
	return uid, ok && uid != ""
}
