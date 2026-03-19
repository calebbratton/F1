package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DBHost          string
	DBPort          string
	DBUser          string
	DBPassword      string
	DBName          string
	ServerPort      string
	APIKey          string // if set, X-API-Key header is required
	RateLimitRPS    int    // requests per second per IP (0 = disabled)
	RateLimitBurst  int
	EnableIngestor  bool // connect to F1 live timing stream
}

func Load() *Config {
	return &Config{
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "5432"),
		DBUser:         getEnv("DB_USER", "postgres"),
		DBPassword:     getEnv("DB_PASSWORD", ""),
		DBName:         getEnv("DB_NAME", "f1"),
		ServerPort:     getEnv("SERVER_PORT", "8080"),
		APIKey:         getEnv("API_KEY", ""),
		RateLimitRPS:   getEnvInt("RATE_LIMIT_RPS", 30),
		RateLimitBurst: getEnvInt("RATE_LIMIT_BURST", 60),
		EnableIngestor: getEnv("ENABLE_INGESTOR", "false") == "true",
	}
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName,
	)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
