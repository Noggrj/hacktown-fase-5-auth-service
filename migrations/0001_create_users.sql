-- fiapx-auth-service: users table
--
-- Run manually or via a Kubernetes Job at deploy time (see fiapx-infra).
-- Not wrapped in a migration framework for this hackathon's scope — a
-- single additive script is enough for the schema's lifetime.

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);
