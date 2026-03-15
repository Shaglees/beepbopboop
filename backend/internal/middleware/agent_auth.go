package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type contextKey string

const agentIDKey contextKey = "agent_id"

func AgentIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(agentIDKey).(string)
	return v
}

// WithAgentID sets the agent ID in context. Used for testing.
func WithAgentID(ctx context.Context, agentID string) context.Context {
	return context.WithValue(ctx, agentIDKey, agentID)
}

func AgentAuth(tokenRepo *repository.TokenRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid authorization header"})
				return
			}

			rawToken := strings.TrimPrefix(auth, "Bearer ")
			agentID, err := tokenRepo.ValidateToken(rawToken)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or revoked token"})
				return
			}

			ctx := context.WithValue(r.Context(), agentIDKey, agentID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
