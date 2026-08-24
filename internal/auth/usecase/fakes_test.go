package usecase_test

import (
	"context"
	"io"
	"log/slog"
	"sync"

	"github.com/google/uuid"

	"github.com/noggrj/fiapx-auth-service/internal/auth/domain"
)

// fakeUserRepo is a concurrency-safe in-memory domain.UserRepository used
// by usecase tests so they don't need a real Postgres.
type fakeUserRepo struct {
	mu       sync.Mutex
	byEmail  map[string]*domain.User
	byID     map[uuid.UUID]*domain.User
	createFn func(*domain.User) error // optional hook to force errors
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		byEmail: make(map[string]*domain.User),
		byID:    make(map[uuid.UUID]*domain.User),
	}
}

func (f *fakeUserRepo) Create(_ context.Context, u *domain.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createFn != nil {
		if err := f.createFn(u); err != nil {
			return err
		}
	}
	if _, exists := f.byEmail[u.Email]; exists {
		return domain.ErrEmailAlreadyRegistered
	}
	cp := *u
	f.byEmail[u.Email] = &cp
	f.byID[u.ID] = &cp
	return nil
}

func (f *fakeUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byEmail[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (f *fakeUserRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
