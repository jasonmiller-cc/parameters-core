package ui

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the top-level dashboard configuration.
type Config struct {
	Server struct {
		Host            string `yaml:"host"`
		Port            int    `yaml:"port"`
		TLSCert         string `yaml:"tls_cert"`
		TLSKey          string `yaml:"tls_key"`
		ReadTimeoutS    int    `yaml:"read_timeout_s"`
		WriteTimeoutS   int    `yaml:"write_timeout_s"`
		ShutdownTimeout int    `yaml:"shutdown_timeout_s"`
	} `yaml:"server"`

	Auth struct {
		JWTSecret string `yaml:"jwt_secret"`
	} `yaml:"auth"`

	Poll struct {
		IntervalS int `yaml:"interval_s"`
	} `yaml:"poll"`

	Services []ServiceEntry `yaml:"services"`
}

func (c *Config) Addr() string {
	host := c.Server.Host
	if host == "" {
		host = "0.0.0.0"
	}
	port := c.Server.Port
	if port == 0 {
		port = 9090
	}
	return fmt.Sprintf("%s:%d", host, port)
}

func (c *Config) PollInterval() time.Duration {
	s := c.Poll.IntervalS
	if s <= 0 {
		s = 30
	}
	return time.Duration(s) * time.Second
}

// LoadConfig reads a YAML config file.
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		path = "dashboard.yaml"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// DefaultConfig returns a Config pre-populated with all known parameters services
// running on localhost with default ports.
func DefaultConfig() *Config {
	cfg := &Config{}
	cfg.Server.Port = 9090
	cfg.Poll.IntervalS = 30
	cfg.Services = []ServiceEntry{
		{Name: "dns", DisplayName: "DNS", URL: "http://localhost:8081", Description: "DNS zone and record management", Group: "network"},
		{Name: "dhcp", DisplayName: "DHCP", URL: "http://localhost:8082", Description: "DHCP server management", Group: "network"},
		{Name: "network", DisplayName: "Network", URL: "http://localhost:8083", Description: "Network interface and routing", Group: "network"},
		{Name: "ntp", DisplayName: "NTP", URL: "http://localhost:8084", Description: "NTP service management", Group: "network"},
		{Name: "ldap", DisplayName: "LDAP", URL: "http://localhost:8085", Description: "LDAP directory management", Group: "identity"},
		{Name: "kerberos", DisplayName: "Kerberos", URL: "http://localhost:8086", Description: "Kerberos KDC management", Group: "identity"},
		{Name: "iam", DisplayName: "IAM", URL: "http://localhost:8087", Description: "Identity and access management", Group: "identity"},
		{Name: "ca", DisplayName: "CA", URL: "http://localhost:8088", Description: "Certificate authority and ACME", Group: "security"},
		{Name: "fs", DisplayName: "Filesystem", URL: "http://localhost:8089", Description: "Filesystem and quota management", Group: "storage"},
	}
	return cfg
}
