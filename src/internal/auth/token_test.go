package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var testTokenSecret = []byte("test-token-secret-with-at-least-32-bytes")

func newTestTokenManager(t *testing.T) *TokenManager {
	t.Helper()
	manager, err := NewTokenManager(TokenConfig{
		Secret:   testTokenSecret,
		Issuer:   "test-registry",
		Audience: "registry.example.com",
		TTL:      2 * time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	return manager
}

func TestTokenManagerIssueAndValidate(t *testing.T) {
	manager := newTestTokenManager(t)
	now := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	manager.now = func() time.Time { return now }
	access := []ResourceAction{
		{
			Type:    "repository",
			Name:    "dev/image",
			Actions: []string{"pull", "push"},
		},
	}

	issued, err := manager.Issue("pavel", access)
	if err != nil {
		t.Fatal(err)
	}
	if issued.ExpiresAt.Sub(issued.IssuedAt) != 2*time.Hour {
		t.Fatalf("token lifetime = %s", issued.ExpiresAt.Sub(issued.IssuedAt))
	}

	claims, err := manager.Validate(issued.Value)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "pavel" {
		t.Fatalf("subject = %q", claims.Subject)
	}
	if len(claims.Access) != 1 || claims.Access[0].Name != "dev/image" {
		t.Fatalf("access = %#v", claims.Access)
	}
}

func TestTokenManagerRejectsExpiredToken(t *testing.T) {
	manager := newTestTokenManager(t)
	now := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	manager.now = func() time.Time { return now }
	issued, err := manager.Issue("pavel", nil)
	if err != nil {
		t.Fatal(err)
	}

	manager.now = func() time.Time { return now.Add(3 * time.Hour) }
	if _, err := manager.Validate(issued.Value); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expired token error = %v", err)
	}
}

func TestTokenManagerRejectsUnexpectedClaimsAndAlgorithm(t *testing.T) {
	manager := newTestTokenManager(t)
	now := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	manager.now = func() time.Time { return now }

	tests := []struct {
		name   string
		claims Claims
		method jwt.SigningMethod
	}{
		{
			name: "missing expiration",
			claims: Claims{RegisteredClaims: jwt.RegisteredClaims{
				Issuer:   "test-registry",
				Subject:  "pavel",
				Audience: jwt.ClaimStrings{"registry.example.com"},
				IssuedAt: jwt.NewNumericDate(now),
			}},
			method: jwt.SigningMethodHS256,
		},
		{
			name: "wrong issuer",
			claims: Claims{RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    "another-registry",
				Subject:   "pavel",
				Audience:  jwt.ClaimStrings{"registry.example.com"},
				ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(now),
			}},
			method: jwt.SigningMethodHS256,
		},
		{
			name: "missing issued at",
			claims: Claims{RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    "test-registry",
				Subject:   "pavel",
				Audience:  jwt.ClaimStrings{"registry.example.com"},
				ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			}},
			method: jwt.SigningMethodHS256,
		},
		{
			name: "wrong audience",
			claims: Claims{RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    "test-registry",
				Subject:   "pavel",
				Audience:  jwt.ClaimStrings{"another-service"},
				ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(now),
			}},
			method: jwt.SigningMethodHS256,
		},
		{
			name: "unexpected algorithm",
			claims: Claims{RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    "test-registry",
				Subject:   "pavel",
				Audience:  jwt.ClaimStrings{"registry.example.com"},
				ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(now),
			}},
			method: jwt.SigningMethodHS384,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := jwt.NewWithClaims(tt.method, tt.claims).
				SignedString(testTokenSecret)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := manager.Validate(raw); !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("token error = %v", err)
			}
		})
	}
}

func TestTokenManagerRejectsWrongSignatureAndMalformedToken(t *testing.T) {
	manager := newTestTokenManager(t)
	now := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	manager.now = func() time.Time { return now }

	claims := Claims{RegisteredClaims: jwt.RegisteredClaims{
		Issuer:    "test-registry",
		Subject:   "pavel",
		Audience:  jwt.ClaimStrings{"registry.example.com"},
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		IssuedAt:  jwt.NewNumericDate(now),
	}}
	wrongSignature, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte("another-token-secret-with-at-least-32-bytes"))
	if err != nil {
		t.Fatal(err)
	}

	for _, rawToken := range []string{wrongSignature, "not-a-jwt"} {
		if _, err := manager.Validate(rawToken); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("token %q error = %v", rawToken, err)
		}
	}
}

func TestNewTokenManagerValidatesConfig(t *testing.T) {
	tests := []TokenConfig{
		{
			Secret:   []byte("short"),
			Issuer:   "issuer",
			Audience: "audience",
			TTL:      time.Hour,
		},
		{
			Secret:   testTokenSecret,
			Audience: "audience",
			TTL:      time.Hour,
		},
		{
			Secret: testTokenSecret,
			Issuer: "issuer",
			TTL:    time.Hour,
		},
		{
			Secret:   testTokenSecret,
			Issuer:   "issuer",
			Audience: "audience",
		},
	}

	for _, config := range tests {
		if _, err := NewTokenManager(config); !errors.Is(err, ErrInvalidTokenConfig) {
			t.Fatalf("config %#v error = %v", config, err)
		}
	}
}
