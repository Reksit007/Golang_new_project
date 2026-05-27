package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPAddr    string
	DatabaseURL string
}

func Load() Config {
	_ = godotenv.Load()

	return Config{
		HTTPAddr:    getEnv("HTTP_ADDR", ":28080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://admin:admin@localhost:5433/subscriptions?sslmode=disable"),
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
