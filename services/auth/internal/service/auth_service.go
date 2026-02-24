package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/SARVESHVARADKAR123/RealChat/services/auth/internal/config"
	"github.com/SARVESHVARADKAR123/RealChat/services/auth/internal/domain"
	"github.com/SARVESHVARADKAR123/RealChat/services/auth/internal/repository"
	"github.com/SARVESHVARADKAR123/RealChat/services/auth/internal/security"
	"github.com/SARVESHVARADKAR123/RealChat/services/auth/internal/tx"
)

// Publisher sends events to a message broker.
type Publisher interface {
	Publish(ctx context.Context, topic string, key, value []byte) error
}

// AuthService handles authentication business logic.
type AuthService struct {
	repo *repository.AuthRepository
	cfg  config.Config
	pub  Publisher
	tx   *tx.Manager
}

// NewAuthService creates a new AuthService.
func NewAuthService(r *repository.AuthRepository, c config.Config, p Publisher, tx *tx.Manager) *AuthService {
	return &AuthService{repo: r, cfg: c, pub: p, tx: tx}
}

// Register creates a new user with a hashed password and publishes a
// user-created event so downstream services can react.
func (a *AuthService) Register(ctx context.Context, email, password string) (string, error) {
	hash, err := security.HashPassword(password)
	if err != nil {
		return "", err
	}

	userID := uuid.NewString()

	err = a.tx.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		if err := a.repo.CreateUser(ctx, tx, userID, email, hash); err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		// Publish auth.user.created event for downstream services (e.g. profile).
		payload, err := json.Marshal(map[string]string{"user_id": userID})
		if err != nil {
			return fmt.Errorf("failed to marshal user created event: %w", err)
		}

		if err := a.repo.InsertOutbox(ctx, tx, "auth", userID, "USER_CREATED", payload); err != nil {
			return fmt.Errorf("failed to save outbox event: %w", err)
		}
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("registration failed: %w", err)
	}

	return userID, nil
}

// Login authenticates a user by email/password and returns access + refresh tokens.
func (a *AuthService) Login(ctx context.Context, email, password string) (string, string, error) {
	uid, hash, err := a.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", "", domain.ErrInvalidCredentials
		}
		return "", "", err
	}

	if err := security.ComparePassword(hash, password); err != nil {
		return "", "", domain.ErrInvalidCredentials
	}

	slog.Info("user_login_success", "email", email, "user_id", uid)

	access, err := security.GenerateAccess(
		a.cfg.JWTSecret, uid, a.cfg.JWTIssuer, a.cfg.JWTAudience, a.cfg.AccessTokenTTL,
	)
	if err != nil {
		return "", "", err
	}

	refresh, err := security.RandomToken(32)
	if err != nil {
		return "", "", err
	}

	err = a.repo.SaveRefresh(
		ctx, uuid.NewString(), uid,
		security.SHA256(refresh),
		time.Now().Add(a.cfg.RefreshTokenTTL),
	)
	if err != nil {
		return "", "", err
	}

	return access, refresh, nil
}

// Refresh validates a refresh token, rotates it, and returns new access + refresh tokens.
func (a *AuthService) Refresh(ctx context.Context, refresh string) (string, string, error) {
	h := security.SHA256(refresh)

	uid, exp, err := a.repo.GetRefresh(ctx, h)
	if err != nil {
		return "", "", err // Already domain.ErrInvalidToken from repo
	}

	if time.Now().After(exp) {
		return "", "", domain.ErrInvalidToken
	}

	_ = a.repo.RevokeRefresh(ctx, h)

	access, err := security.GenerateAccess(
		a.cfg.JWTSecret, uid, a.cfg.JWTIssuer, a.cfg.JWTAudience, a.cfg.AccessTokenTTL,
	)
	if err != nil {
		return "", "", err
	}

	newRefresh, err := security.RandomToken(32)
	if err != nil {
		return "", "", err
	}

	err = a.repo.SaveRefresh(
		ctx, uuid.NewString(), uid,
		security.SHA256(newRefresh),
		time.Now().Add(a.cfg.RefreshTokenTTL),
	)
	if err != nil {
		return "", "", err
	}

	return access, newRefresh, nil
}

// Logout revokes the given refresh token.
func (a *AuthService) Logout(ctx context.Context, refresh string) error {
	return a.repo.RevokeRefresh(ctx, security.SHA256(refresh))
}
