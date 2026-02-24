package security

import (
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateAccess creates a signed HS256 JWT access token with the given claims.
func GenerateAccess(secret, userID, issuer, audience string, ttl time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"iss": issuer,
		"aud": audience,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(ttl).Unix(),
	}

	slog.Debug("generating_jwt", "user_id", userID, "iss", issuer)

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}
