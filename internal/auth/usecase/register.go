package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"golang.org/x/crypto/bcrypt"

	"github.com/noggrj/fiapx-auth-service/internal/auth/domain"
)

type RegisterUseCase struct {
	users domain.UserRepository
	log   *slog.Logger
}

func NewRegister(users domain.UserRepository, log *slog.Logger) *RegisterUseCase {
	return &RegisterUseCase{users: users, log: log}
}

func (uc *RegisterUseCase) Execute(ctx context.Context, email, password string) (*domain.User, error) {
	if err := domain.ValidatePassword(password); err != nil {
		return nil, err
	}
	u, err := domain.NewUser(email)
	if err != nil {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	u.PasswordHash = string(hash)
	if err := uc.users.Create(ctx, u); err != nil {
		return nil, err
	}
	uc.log.Info("user registered", slog.String("userId", u.ID.String()))
	return u, nil
}
