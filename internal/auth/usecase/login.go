package usecase

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/noggrj/hacktown-fase-5-auth-service/internal/auth/domain"
	"github.com/noggrj/hacktown-fase-5-auth-service/internal/platform/jwt"
	"github.com/noggrj/hacktown-fase-5-auth-service/internal/platform/metrics"
)

const tokenTTL = 24 * time.Hour

type LoginUseCase struct {
	users  domain.UserRepository
	issuer *jwt.Issuer
	log    *slog.Logger
}

func NewLogin(users domain.UserRepository, issuer *jwt.Issuer, log *slog.Logger) *LoginUseCase {
	return &LoginUseCase{users: users, issuer: issuer, log: log}
}

// Execute verifies credentials and returns a signed JWT. Deliberately
// returns the same ErrInvalidCredentials whether the email doesn't exist
// or the password doesn't match, so the API never leaks which emails are
// registered.
func (uc *LoginUseCase) Execute(ctx context.Context, email, password string) (string, error) {
	u, err := uc.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			metrics.Logins.WithLabelValues("invalid_credentials").Inc()
			return "", domain.ErrInvalidCredentials
		}
		return "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		metrics.Logins.WithLabelValues("invalid_credentials").Inc()
		return "", domain.ErrInvalidCredentials
	}
	token, err := uc.issuer.Issue(u.ID.String(), u.Email, tokenTTL)
	if err != nil {
		return "", err
	}
	metrics.Logins.WithLabelValues("success").Inc()
	uc.log.Info("user logged in", slog.String("userId", u.ID.String()))
	return token, nil
}
