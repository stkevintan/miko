package config

import (
	"os"
	"path"
	"strconv"

	"github.com/chaunsin/netease-cloud-music/api"
	"github.com/chaunsin/netease-cloud-music/pkg/cookie"
)

// Config holds all configuration for our service
type Config struct {
	Port        int         `json:"port"`
	Environment string      `json:"environment"`
	LogLevel    string      `json:"log_level"`
	NmApi       *api.Config `json:"nmapi"`
}

// Load loads configuration from environment variables with sensible defaults
func Load() (*Config, error) {
	home, _ := os.UserHomeDir()
	cfg := &Config{
		Port:        8082,
		Environment: "development",
		LogLevel:    "info",
		NmApi: &api.Config{
			Debug:   false,
			Timeout: 0,
			Retry:   0,
			Cookie: cookie.Config{
				Options:  nil,
				Filepath: path.Join(home, ".miko", "cookie.json"),
				Interval: 0,
			},
		},
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
