package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jasonmiller-cc/parameters-core/pkg/config"
)

func TestServerConfig_Addr_Defaults(t *testing.T) {
	var c config.ServerConfig
	if got, want := c.Addr(), "0.0.0.0:8080"; got != want {
		t.Errorf("Addr() = %q, want %q", got, want)
	}
}

func TestServerConfig_Addr_Custom(t *testing.T) {
	c := config.ServerConfig{Host: "127.0.0.1", Port: 9999}
	if got, want := c.Addr(), "127.0.0.1:9999"; got != want {
		t.Errorf("Addr() = %q, want %q", got, want)
	}
}

func TestLoad_MissingFile_NoError(t *testing.T) {
	var cfg config.BaseConfig
	path := filepath.Join(t.TempDir(), "does-not-exist.yaml")
	if err := config.Load(path, "PARAMS_TEST", &cfg); err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
}

func TestLoad_YAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	yaml := "server:\n  host: 10.0.0.1\n  port: 9090\n"
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var cfg config.BaseConfig
	if err := config.Load(path, "PARAMS_TEST", &cfg); err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Server.Host != "10.0.0.1" || cfg.Server.Port != 9090 {
		t.Errorf("Server = %+v, want host=10.0.0.1 port=9090", cfg.Server)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("not: [valid: yaml"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var cfg config.BaseConfig
	if err := config.Load(path, "PARAMS_TEST", &cfg); err == nil {
		t.Fatal("Load() expected error for invalid YAML, got nil")
	}
}

func TestLoad_EnvOverridesYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	yaml := "server:\n  port: 9090\n"
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("PARAMS_TEST_SERVER_PORT", "7777")

	var cfg config.BaseConfig
	if err := config.Load(path, "PARAMS_TEST", &cfg); err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Server.Port != 7777 {
		t.Errorf("Server.Port = %d, want 7777 (env should override YAML)", cfg.Server.Port)
	}
}
