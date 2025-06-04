package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/trevortippery/moving-checklist/db"
	"github.com/trevortippery/moving-checklist/utils"
)

type contextKey string

const userContextKey = contextKey("user")

func SetUser(r *http.Request, user *db.User) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}

func GetUser(r *http.Request) *db.User {
	user, ok := r.Context().Value(userContextKey).(*db.User)
	if !ok {
		return nil
	}
	return user
}

type AuthMiddleware struct {
	UserStore db.UserStore
}

func NewAuthMiddleware(userStore db.UserStore) *AuthMiddleware {
	return &AuthMiddleware{
		UserStore: userStore,
	}
}

func (am *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// No auth header, set anonymous user (nil here)
			r = SetUser(r, nil)
			next.ServeHTTP(w, r)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "invalid authorization header"})
			return
		}

		token := parts[1]

		user, err := am.UserStore.GetUserByToken(r.Context(), token, "auth")
		if err != nil || user == nil {
			utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "invalid or expired token"})
			return
		}

		r = SetUser(r, user)
		next.ServeHTTP(w, r)
	})
}

// RequireUser middleware to enforce that a user is authenticated
func RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUser(r)
		if user == nil {
			utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "you must be logged in to access this route"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
