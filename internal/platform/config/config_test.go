package config_test

import (
	"testing"

	"github.com/noggrj/hacktown-fase-5-auth-service/internal/platform/config"
)

func TestLoad_DefaultsWhenEnvUnset(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("SERVICE_NAME", "")
	t.Setenv("DB_URL", "")
	t.Setenv("JWT_SECRET", "")

	cfg := config.Load()
	if cfg.Port != "8080" {
		t.Fatalf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.Env != "development" {
		t.Fatalf("expected default env development, got %s", cfg.Env)
	}
	if cfg.ServiceName != "fiapx-auth-service" {
		t.Fatalf("expected default service name, got %s", cfg.ServiceName)
	}
}

func TestLoad_ReadsOverrides(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("APP_ENV", "production")
	t.Setenv("DB_URL", "postgres://x")
	t.Setenv("JWT_SECRET", "shhh")

	cfg := config.Load()
	if cfg.Port != "9090" || cfg.Env != "production" || cfg.DBURL != "postgres://x" || cfg.JWTSecret != "shhh" {
		t.Fatalf("env overrides not applied: %+v", cfg)
	}
}
