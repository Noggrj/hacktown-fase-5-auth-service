package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	Env         string
	DBURL       string
	JWTSecret   string
	ServiceName string
}

func Load() *Config {
	_ = godotenv.Load()
	return &Config{
		Port:        getenv("PORT", "8080"),
		Env:         getenv("APP_ENV", "development"),
		DBURL:       os.Getenv("DB_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		ServiceName: getenv("SERVICE_NAME", "fiapx-auth-service"),
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
