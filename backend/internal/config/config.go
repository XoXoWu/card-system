package config

import (
	"os"
	"time"
)

type Config struct {
	Listen        string
	DBPath        string
	AdminUser     string
	AdminPassword string
	JWTSecret     string
	TokenTTL      time.Duration
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func Load() *Config {
	return &Config{
		Listen:        getenv("CARD_LISTEN", ":8080"),
		DBPath:        getenv("CARD_DB_PATH", "data/data.db"),
		AdminUser:     getenv("CARD_ADMIN_USER", "admin"),
		AdminPassword: getenv("CARD_ADMIN_PASSWORD", "admin123"),
		JWTSecret:     getenv("CARD_JWT_SECRET", "change-me-in-production"),
		TokenTTL:      12 * time.Hour,
	}
}
