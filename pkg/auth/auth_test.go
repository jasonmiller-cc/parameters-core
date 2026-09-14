package auth_test

import (
	"testing"
	"time"

	"github.com/jasonmiller-cc/parameters-core/pkg/auth"
)

func TestJWT_RoundTrip(t *testing.T) {
	cfg := &auth.JWTConfig{
		Secret: []byte("test-secret-32-bytes-padding-xxx"),
		Issuer: "parameters-test",
		TTL:    time.Hour,
	}

	claims := auth.Claims{Subject: "user123", Email: "test@example.com", Roles: []string{"admin"}}
	token, err := cfg.Sign(claims)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	got, err := cfg.Verify(token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if got.Subject != "user123" {
		t.Errorf("subject: want user123, got %s", got.Subject)
	}
	if got.Email != "test@example.com" {
		t.Errorf("email: want test@example.com, got %s", got.Email)
	}
}

func TestJWT_InvalidToken(t *testing.T) {
	cfg := &auth.JWTConfig{Secret: []byte("secret")}
	_, err := cfg.Verify("not.a.token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}
