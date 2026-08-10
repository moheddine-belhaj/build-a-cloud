// Package config reads process configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Addr        string
	DatabaseURL string
	JWTSecret   string
	JWTTTL      time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Addr:   ":" + getEnv("PORT", "8080"),
		JWTTTL: 24 * time.Hour,
	}

	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	cfg.JWTSecret = os.Getenv("JWT_SECRET")
	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}

	if ttl := os.Getenv("JWT_TTL"); ttl != "" {
		d, err := time.ParseDuration(ttl)
		if err != nil {
			return Config{}, fmt.Errorf("invalid JWT_TTL: %w", err)
		}
		cfg.JWTTTL = d
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
