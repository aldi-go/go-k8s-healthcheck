package config

import (
	"log/slog"
	"os"
	"strconv"
)

// Config holds all application configuration sourced from environment variables.
type Config struct {
	Port        string
	AppEnv      string
	LogLevel    slog.Level
	Version     string
	AppName     string
	ShutdownSec int
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	cfg := &Config{
		Port:        getEnv("PORT", "8080"),
		AppEnv:      getEnv("APP_ENV", "development"),
		Version:     getEnv("VERSION", "dev"),
		AppName:     getEnv("APP_NAME", "go-k8s-healthcheck"),
		ShutdownSec: getEnvInt("SHUTDOWN_TIMEOUT_SEC", 30),
		LogLevel:    parseLogLevel(getEnv("LOG_LEVEL", "info")),
	}
	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
