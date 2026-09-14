// Package auth provides JWT and API-key authentication for parameters services.
package auth

import (
	"context"
	"net/http"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	apierrors "github.com/jasonmiller-cc/parameters-core/pkg/errors"
)

// Claims is the JWT claim set used across parameters services.
type Claims struct {
	jwt.RegisteredClaims
	Subject  string   `json:"sub"`
	Email    string   `json:"email,omitempty"`
	Roles    []string `json:"roles,omitempty"`
	Service  string   `json:"svc,omitempty"`
}

type contextKey string

const claimsKey contextKey = "claims"

// JWTConfig holds JWT signing configuration.
type JWTConfig struct {
	Secret []byte
	Issuer string
	TTL    time.Duration
}

// Sign creates a signed JWT token for the given claims.
func (c *JWTConfig) Sign(claims Claims) (string, error) {
	claims.Issuer = c.Issuer
	if claims.ExpiresAt == nil {
		ttl := c.TTL
		if ttl == 0 {
			ttl = 24 * time.Hour
		}
		claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(ttl))
	}
	claims.IssuedAt = jwt.NewNumericDate(time.Now())
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(c.Secret)
}

// Verify parses and validates a JWT token string.
func (c *JWTConfig) Verify(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, apierrors.Unauthorized("unexpected signing method")
		}
		return c.Secret, nil
	})
	if err != nil {
		return nil, apierrors.Unauthorized("invalid token: " + err.Error())
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, apierrors.Unauthorized("invalid token claims")
	}
	return claims, nil
}

// ClaimsFromContext retrieves claims stored by JWTMiddleware.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(claimsKey).(*Claims)
	return c, ok
}

// JWTMiddleware extracts and validates the Bearer token from Authorization header.
// On success the *Claims are stored in the request context.
func JWTMiddleware(cfg *JWTConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token == "" {
				writeUnauthorized(w, "missing authorization header")
				return
			}
			claims, err := cfg.Verify(token)
			if err != nil {
				writeUnauthorized(w, err.Error())
				return
			}
			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// APIKeyMiddleware validates requests against a static set of API keys.
// The key must be supplied as "Authorization: Bearer <key>".
func APIKeyMiddleware(keys []string) func(http.Handler) http.Handler {
	set := make(map[string]bool, len(keys))
	for _, k := range keys {
		set[k] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := bearerToken(r)
			if !set[key] {
				writeUnauthorized(w, "invalid api key")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole is middleware that checks the claims contain at least one of roles.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				writeUnauthorized(w, "unauthenticated")
				return
			}
			if !hasRole(claims.Roles, roles) {
				http.Error(w, `{"error":{"code":"forbidden","message":"insufficient role"}}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func bearerToken(r *http.Request) string {
	hdr := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(hdr, "Bearer "); ok {
		return strings.TrimSpace(after)
	}
	return ""
}

func hasRole(have, need []string) bool {
	set := make(map[string]bool, len(have))
	for _, r := range have {
		set[r] = true
	}
	for _, n := range need {
		if set[n] {
			return true
		}
	}
	return false
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", `Bearer realm="parameters"`)
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"` + msg + `"}}`))
}
