package config_test

import (
	"testing"

	"github.com/jasonmiller-cc/parameters-core/pkg/config"
)

type nested struct {
	Enabled bool   `env:"ENABLED"`
	Name    string `env:"NAME"`
}

type testStruct struct {
	Host   string `env:"HOST"`
	Port   int    `env:"PORT"`
	Nested nested
}

func TestApplyEnv_SetsTaggedFields(t *testing.T) {
	t.Setenv("PARAMS_TEST_HOST", "example.com")
	t.Setenv("PARAMS_TEST_PORT", "4242")
	t.Setenv("PARAMS_TEST_ENABLED", "true")
	t.Setenv("PARAMS_TEST_NAME", "svc")

	s := testStruct{}
	config.ApplyEnv("PARAMS_TEST", &s)

	if s.Host != "example.com" {
		t.Errorf("Host = %q, want example.com", s.Host)
	}
	if s.Port != 4242 {
		t.Errorf("Port = %d, want 4242", s.Port)
	}
	if !s.Nested.Enabled {
		t.Error("Nested.Enabled = false, want true")
	}
	if s.Nested.Name != "svc" {
		t.Errorf("Nested.Name = %q, want svc", s.Nested.Name)
	}
}

func TestApplyEnv_LeavesUnsetFieldsAlone(t *testing.T) {
	s := testStruct{Host: "unchanged"}
	config.ApplyEnv("PARAMS_TEST_UNSET_PREFIX", &s)

	if s.Host != "unchanged" {
		t.Errorf("Host = %q, want unchanged (no env var set)", s.Host)
	}
}

func TestApplyEnv_FallsBackWithoutPrefix(t *testing.T) {
	t.Setenv("HOST", "fallback.example.com")

	s := testStruct{}
	config.ApplyEnv("PARAMS_NOPREFIXMATCH", &s)

	if s.Host != "fallback.example.com" {
		t.Errorf("Host = %q, want fallback.example.com", s.Host)
	}
}

func TestApplyEnv_NonPointer_NoPanic(t *testing.T) {
	config.ApplyEnv("PARAMS_TEST", testStruct{})
}

func TestApplyEnv_NilPointer_NoPanic(t *testing.T) {
	var s *testStruct
	config.ApplyEnv("PARAMS_TEST", s)
}
