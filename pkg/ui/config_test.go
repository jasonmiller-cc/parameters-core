package ui_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jasonmiller-cc/parameters-core/pkg/ui"
)

func TestConfig_Addr_Defaults(t *testing.T) {
	cfg := &ui.Config{}
	if got, want := cfg.Addr(), "0.0.0.0:9090"; got != want {
		t.Errorf("Addr() = %q, want %q", got, want)
	}
}

func TestConfig_PollInterval_Default(t *testing.T) {
	cfg := &ui.Config{}
	if got, want := cfg.PollInterval(), 30*time.Second; got != want {
		t.Errorf("PollInterval() = %v, want %v", got, want)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := ui.DefaultConfig()
	if len(cfg.Services) != 9 {
		t.Errorf("len(Services) = %d, want 9", len(cfg.Services))
	}
	if got, want := cfg.Addr(), "0.0.0.0:9090"; got != want {
		t.Errorf("Addr() = %q, want %q", got, want)
	}
}

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dashboard.yaml")
	yaml := `
server:
  host: 127.0.0.1
  port: 9191
poll:
  interval_s: 15
services:
  - name: dns
    display_name: DNS
    url: http://localhost:8081
    group: network
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := ui.LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if got, want := cfg.Addr(), "127.0.0.1:9191"; got != want {
		t.Errorf("Addr() = %q, want %q", got, want)
	}
	if got, want := cfg.PollInterval(), 15*time.Second; got != want {
		t.Errorf("PollInterval() = %v, want %v", got, want)
	}
	if len(cfg.Services) != 1 || cfg.Services[0].Name != "dns" {
		t.Errorf("Services = %+v, want one entry named dns", cfg.Services)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	if _, err := ui.LoadConfig(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatal("expected error for missing config file, got nil")
	}
}
