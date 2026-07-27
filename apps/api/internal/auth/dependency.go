package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/artumont/dotslashstream/internal/httpx"
)

type contextKey struct{}

// AuthRequired is an HTTP middleware that extracts a Bearer token,
// verifies it, loads the user, and stores it in the request context.
// Downstream handlers retrieve it with UserFromContext.
func AuthRequired(svc *Service, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			httpx.WriteError(w, http.StatusUnauthorized, "missing or malformed authorization header")
			return
		}

		user, err := svc.GetUserFromToken(r.Context(), token)
		if err != nil {
			if errors.Is(err, ErrTokenExpired) {
				httpx.WriteError(w, http.StatusUnauthorized, "token expired")
			} else {
				httpx.WriteError(w, http.StatusUnauthorized, "invalid token")
			}
			return
		}

		ctx := context.WithValue(r.Context(), contextKey{}, &userResponse{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			IsAdmin:  user.IsAdmin,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AdminRequired is an HTTP middleware that checks if the authenticated user
// has admin privileges. Must be stacked after AuthRequired.
func AdminRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r)
		if user == nil {
			httpx.WriteError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		if !user.IsAdmin {
			httpx.WriteError(w, http.StatusForbidden, "admin access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// UserFromContext retrieves the authenticated user from the request context.
// Returns nil if no user is present (i.e. AuthRequired was not called).
func UserFromContext(r *http.Request) *userResponse {
	u, _ := r.Context().Value(contextKey{}).(*userResponse)
	return u
}

// extractBearerToken parses the Authorization header and returns the token value.
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(auth, "Bearer ")
}
