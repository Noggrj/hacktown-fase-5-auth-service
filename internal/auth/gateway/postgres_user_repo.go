// Package gateway has the Postgres-backed implementation of
// domain.UserRepository.
package gateway

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/noggrj/hacktown-fase-5-auth-service/internal/auth/domain"
)

const uniqueViolation = "23505"

type PostgresUserRepository struct{ pool *pgxpool.Pool }

func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

func (r *PostgresUserRepository) Create(ctx context.Context, u *domain.User) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, created_at) VALUES ($1,$2,$3,$4)`,
		u.ID, u.Email, u.PasswordHash, u.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return domain.ErrEmailAlreadyRegistered
		}
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE email=$1`, email,
	)
	return scanUser(row)
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE id=$1`, id,
	)
	return scanUser(row)
}

func scanUser(row pgx.Row) (*domain.User, error) {
	u := &domain.User{}
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return u, nil
}
