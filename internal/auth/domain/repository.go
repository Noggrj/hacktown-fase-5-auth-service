package domain

import (
	"context"

	"github.com/google/uuid"
)

// UserRepository persists User aggregates.
type UserRepository interface {
	Create(ctx context.Context, u *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
}
