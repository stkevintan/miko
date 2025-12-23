package config

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/spf13/viper"
	"github.com/stkevintan/miko/pkg/cookiecloud"
	"github.com/stkevintan/miko/pkg/log"
	"github.com/stkevintan/miko/pkg/registry"
)

var (
	//go:embed config.toml
	defaultConfigToml []byte
)

// Config holds all configuration for our service
type Config struct {
	Version     string              `json:"version" mapstructure:"version"`
	Server      *ServerConfig       `json:"server" mapstructure:"server"`
	Log         *log.Config         `json:"log" mapstructure:"log"`
	CookieCloud *cookiecloud.Config `json:"cookiecloud" mapstructure:"cookiecloud"`
	Registry    *registry.Config    `json:"registry" mapstructure:"registry"`
	Database    *DatabaseConfig     `json:"database" mapstructure:"database"`
}

func (c *Config) Validate() error {
	if c.Server == nil {
		return errors.New("server config is required")
	}
	if c.Log == nil {
		return errors.New("log config is required")
	}
	if c.CookieCloud == nil {
		return errors.New("cookiecloud config is required")
	}
	if c.Registry == nil {
		return errors.New("registry config is required")
	}
	if c.Database == nil {
		return errors.New("database config is required")
	}
	return nil
}

type ServerConfig struct {
	Port int `json:"port" mapstructure:"port"`
}

type DatabaseConfig struct {
	Driver string `json:"driver" mapstructure:"driver"`
	DSN    string `json:"dsn" mapstructure:"dsn"`
}

// Load loads configuration from config files + environment variables with sensible defaults.
//
// Supported sources (highest precedence last):
// - defaults
// - config file (optional): ./config.{yaml,yml,json,toml}, ./config/config.{...}, $HOME/.miko/config.{...}
// - environment variables: MIKO_* (e.g. MIKO_PORT, MIKO_NMAPI_COOKIE_FILEPATH)
// - legacy environment variables: PORT, ENVIRONMENT, LOG_LEVEL (kept for backward compatibility)
func Load() (*Config, error) {
	v := viper.New()
	v.SetTypeByDefaultValue(true)
	v.AllowEmptyEnv(true)
	// Load embedded defaults first, then merge user config on top.
	// This makes missing keys automatically fall back to defaults.
	v.SetConfigType("toml")
	if err := v.ReadConfig(bytes.NewReader(defaultConfigToml)); err != nil {
		return nil, fmt.Errorf("read embedded default config: %w", err)
	}

	// Optional config file
	if configFile := os.Getenv("MIKO_CONFIG"); configFile != "" {
		v.SetConfigFile(configFile)
	} else {
		v.SetConfigName("config")
		v.AddConfigPath(".")
		home, err := os.UserHomeDir()
		if err != nil {
			panic(fmt.Sprintf("os.UserHomeDir: %s", err))
		}
		if home != "" {
			v.AddConfigPath(path.Join(home, ".miko"))
		}
	}

	// Environment variables: MIKO_PORT, MIKO_LOG_LEVEL, MIKO_NMAPI_COOKIE_FILEPATH, etc.
	v.SetEnvPrefix("MIKO")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Merge optional config file (if present) over defaults.
	if err := v.MergeInConfig(); err != nil {
		// Ignore “not found”; error out on parse/permission/etc.
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, err
		}
	}

	var cfg Config
	// Unmarshal into our pre-initialized struct so pointer fields stay non-nil.
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("v.Unmarshal: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("cfg.Validate: %w", err)
	}

	log.Debug("Configuration loaded successfully, %+v", &cfg)

	return &cfg, nil
}
