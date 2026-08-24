// Package domain has the User entity and validation rules. No knowledge
// of HTTP, SQL or password hashing — those are usecase/gateway concerns.
package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound               = errors.New("not found")
	ErrEmailAlreadyRegistered = errors.New("email already registered")
	ErrInvalidCredentials     = errors.New("invalid credentials")
)

var emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// User is the aggregate root for the Auth domain.
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

// NewUser validates and normalizes the email, returning a User with a
// fresh ID. PasswordHash is left empty — hashing is the usecase's job.
func NewUser(email string) (*User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if !emailRe.MatchString(email) {
		return nil, errors.New("invalid email")
	}
	return &User{
		ID:        uuid.New(),
		Email:     email,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// ValidatePassword enforces the minimum strength rule for registration.
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}
