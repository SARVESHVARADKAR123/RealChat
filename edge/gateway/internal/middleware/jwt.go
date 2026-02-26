package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

func JWT(secret, issuer, audience string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString, err := extractToken(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			claims, err := verifyToken(tokenString, secret, issuer, audience)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			sub, ok := claims["sub"].(string)
			if !ok {
				http.Error(w, "invalid token claims", http.StatusUnauthorized)
				return
			}

			slog.Info("jwt_parsed", "sub", sub, "token_prefix", tokenString[:10])

			ctx := InjectUserID(r.Context(), sub)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractToken(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", fmt.Errorf("missing token")
	}

	parts := strings.Split(header, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid token format")
	}

	return parts[1], nil
}

func verifyToken(tokenString, secret, issuer, audience string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	if issuer != "" {
		if iss, ok := claims["iss"].(string); !ok || iss != issuer {
			return nil, fmt.Errorf("invalid token issuer")
		}
	}

	if audience != "" {
		if aud, ok := claims["aud"].(string); !ok || aud != audience {
			return nil, fmt.Errorf("invalid token audience")
		}
	}

	return claims, nil
}
