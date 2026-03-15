package middleware

import (
	"context"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"
)

const firebaseUIDKey contextKey = "firebase_uid"

func FirebaseUIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(firebaseUIDKey).(string)
	return v
}

// WithFirebaseUID sets the Firebase UID in context. Used for testing.
func WithFirebaseUID(ctx context.Context, uid string) context.Context {
	return context.WithValue(ctx, firebaseUIDKey, uid)
}

// FirebaseAuth verifies Firebase ID tokens. If authClient is nil, runs in dev mode
// where the Bearer token value is used directly as the Firebase UID.
func FirebaseAuth(authClient *auth.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")

			var uid string
			if authClient == nil {
				// Dev mode: treat bearer value as Firebase UID directly
				uid = token
			} else {
				// Production: verify with Firebase
				decoded, err := authClient.VerifyIDToken(r.Context(), token)
				if err != nil {
					writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid firebase token"})
					return
				}
				uid = decoded.UID
			}

			ctx := context.WithValue(r.Context(), firebaseUIDKey, uid)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
