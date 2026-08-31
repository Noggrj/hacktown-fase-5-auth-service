package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	Env         string
	DBURL       string
	JWTSecret   string
	ServiceName string

	// CORSAllowedOrigins lists origins allowed to call this API from a
	// browser (the frontend SPA). "*" (the default) is fine here: auth
	// is a bearer token, not a cookie, so there's no credentialed
	// request for a wildcard origin to expose.
	CORSAllowedOrigins []string
}

func Load() *Config {
	_ = godotenv.Load()
	return &Config{
		Port:               getenv("PORT", "8080"),
		Env:                getenv("APP_ENV", "development"),
		DBURL:              os.Getenv("DB_URL"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		ServiceName:        getenv("SERVICE_NAME", "fiapx-auth-service"),
		CORSAllowedOrigins: splitCSV(getenv("CORS_ALLOWED_ORIGINS", "*")),
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
