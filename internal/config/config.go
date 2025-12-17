package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for our service
type Config struct {
	Port        int    `json:"port"`
	Environment string `json:"environment"`
	LogLevel    string `json:"log_level"`
}

// Load loads configuration from environment variables with sensible defaults
func Load() (*Config, error) {
	cfg := &Config{
		Port:        8080,
		Environment: "development",
		LogLevel:    "info",
	}

	// Override with environment variables if present
	if port := os.Getenv("PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Port = p
		}
	}

	if env := os.Getenv("ENVIRONMENT"); env != "" {
		cfg.Environment = env
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		cfg.LogLevel = logLevel
	}

	return cfg, nil
}
