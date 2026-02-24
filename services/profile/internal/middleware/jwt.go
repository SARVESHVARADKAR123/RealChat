package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const UserKey ctxKey = "uid"

// JWT returns middleware that validates HS256 JWTs using the given shared secret.
func JWT(secret []byte, iss, aud string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				w.WriteHeader(401)
				return
			}

			tok := strings.TrimPrefix(h, "Bearer ")

			parsed, err := jwt.Parse(tok, func(t *jwt.Token) (interface{}, error) {
				// Prevent algorithm confusion — only accept HMAC.
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return secret, nil
			}, jwt.WithIssuer(iss), jwt.WithAudience(aud))

			if err != nil || !parsed.Valid {
				w.WriteHeader(401)
				return
			}

			claims := parsed.Claims.(jwt.MapClaims)
			uid := claims["sub"].(string)

			ctx := context.WithValue(r.Context(), UserKey, uid)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
