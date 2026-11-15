package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey struct{}

func Auth(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if raw == "" {
				if c, err := r.Cookie("token"); err == nil {
					raw = c.Value
				}
			}
			var claims jwt.RegisteredClaims
			_, err := jwt.ParseWithClaims(raw, &claims,
				func(*jwt.Token) (any, error) { return secret, nil },
				jwt.WithValidMethods([]string{"HS256"}))
			if err != nil || claims.Subject == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), ctxKey{}, claims.Subject)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func LoginFrom(ctx context.Context) (string, bool) {
	login, ok := ctx.Value(ctxKey{}).(string)
	return login, ok
}
