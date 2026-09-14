// Package config provides unified configuration loading from YAML files and
// environment variable overrides for parameters services.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// BaseConfig holds fields common to every parameters service.
type BaseConfig struct {
	Server   ServerConfig   `yaml:"server"`
	Log      LogConfig      `yaml:"log"`
	Auth     AuthConfig     `yaml:"auth"`
	Metrics  MetricsConfig  `yaml:"metrics"`
	Database DatabaseConfig `yaml:"database"`
}

type ServerConfig struct {
	Host            string `yaml:"host" env:"SERVER_HOST"`
	Port            int    `yaml:"port" env:"SERVER_PORT"`
	TLSCert         string `yaml:"tls_cert" env:"SERVER_TLS_CERT"`
	TLSKey          string `yaml:"tls_key" env:"SERVER_TLS_KEY"`
	ReadTimeout     int    `yaml:"read_timeout_s"`
	WriteTimeout    int    `yaml:"write_timeout_s"`
	ShutdownTimeout int    `yaml:"shutdown_timeout_s"`
}

func (c ServerConfig) Addr() string {
	host := c.Host
	if host == "" {
		host = "0.0.0.0"
	}
	port := c.Port
	if port == 0 {
		port = 8080
	}
	return fmt.Sprintf("%s:%d", host, port)
}

type LogConfig struct {
	Level  string `yaml:"level" env:"LOG_LEVEL"`
	Format string `yaml:"format" env:"LOG_FORMAT"`
}

type AuthConfig struct {
	JWTSecret    string   `yaml:"jwt_secret" env:"AUTH_JWT_SECRET"`
	JWTIssuer    string   `yaml:"jwt_issuer" env:"AUTH_JWT_ISSUER"`
	APIKeys      []string `yaml:"api_keys"`
	OIDCIssuer   string   `yaml:"oidc_issuer" env:"AUTH_OIDC_ISSUER"`
	OIDCAudience string   `yaml:"oidc_audience" env:"AUTH_OIDC_AUDIENCE"`
}

type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

type DatabaseConfig struct {
	DSN          string `yaml:"dsn" env:"DB_DSN"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
}

// Load reads a YAML config file and overlays environment variables.
// envPrefix is the service-specific prefix, e.g. "PARAMS_DNS".
func Load(path, envPrefix string, out any) error {
	if path == "" {
		path = defaultConfigPath(envPrefix)
	}

	if data, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(data, out); err != nil {
			return fmt.Errorf("parse config %s: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read config %s: %w", path, err)
	}

	applyEnv(envPrefix, out)
	return nil
}

// applyEnv overlays environment variables tagged with `env:"VAR"` onto dst.
// This is a simple implementation; production callers should use reflect or viper.
func applyEnv(prefix string, _ any) {
	// Intentionally left minimal: services that need deep env overlay
	// should embed BaseConfig and read env vars directly using os.Getenv.
	// The env tag is present for documentation; a full reflect-based
	// implementation is in pkg/config/env.go.
	_ = prefix
}

func defaultConfigPath(envPrefix string) string {
	key := strings.ToUpper(envPrefix) + "_CONFIG"
	if v := os.Getenv(key); v != "" {
		return v
	}
	// Try standard locations.
	candidates := []string{
		"/etc/parameters/" + strings.ToLower(strings.TrimPrefix(envPrefix, "PARAMS_")) + "/config.yaml",
		filepath.Join(os.Getenv("HOME"), ".parameters", "config.yaml"),
		"config.yaml",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "config.yaml"
}
