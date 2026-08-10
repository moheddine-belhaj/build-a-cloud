// Package middleware provides HTTP middleware wrapping stdlib handlers,
// matching the plain net/http.ServeMux routing used in cmd/main.go.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/moheddine-belhaj/build-a-cloud/week-3/api/internal/auth"
)

type ctxKey string

const userIDKey ctxKey = "userID"

// RequireAuth returns a middleware that rejects requests without a valid
// "Authorization: Bearer <token>" header, and otherwise stashes the token's
// user id in the request context for downstream handlers.
func RequireAuth(secret string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			token, ok := strings.CutPrefix(header, "Bearer ")
			if !ok || token == "" {
				http.Error(w, `{"message":"missing bearer token"}`, http.StatusUnauthorized)
				return
			}

			userID, err := auth.ParseToken(secret, token)
			if err != nil {
				http.Error(w, `{"message":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			next(w, r.WithContext(WithUserID(r.Context(), userID)))
		}
	}
}

// WithUserID stashes an authenticated user id on the context, as RequireAuth
// does for real requests. Exported so tests can simulate an authenticated
// request without going through a real token.
func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromContext returns the authenticated user id stashed by RequireAuth.
func UserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userIDKey).(int64)
	return userID, ok
}
