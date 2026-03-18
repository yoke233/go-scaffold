package authn

import (
	"strings"
	"testing"
	"time"

	"project/internal/conf"
)

func TestJWTTokenManagerIssueAndParse(t *testing.T) {
	manager, err := NewTokenManager(&conf.Bootstrap{
		Auth: conf.Auth{
			JWT: conf.JWTAuth{
				Issuer:         "test-issuer",
				SigningKey:     "test-secret",
				AccessTokenTTL: "1h",
			},
		},
	})
	if err != nil {
		t.Fatalf("NewTokenManager returned error: %v", err)
	}

	jwtManager := manager.(*JWTTokenManager)
	jwtManager.now = func() time.Time {
		return time.Unix(1700000000, 0).UTC()
	}

	token, err := manager.IssueAccessToken(Principal{
		UserID:  7,
		Subject: "user:7",
	})
	if err != nil {
		t.Fatalf("IssueAccessToken returned error: %v", err)
	}

	principal, err := manager.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("ParseAccessToken returned error: %v", err)
	}

	if principal.UserID != 7 {
		t.Fatalf("expected user id 7, got %d", principal.UserID)
	}
	if principal.Subject != "user:7" {
		t.Fatalf("expected subject user:7, got %s", principal.Subject)
	}
}

func TestJWTTokenManagerRejectsWrongIssuer(t *testing.T) {
	source, err := NewTokenManager(&conf.Bootstrap{
		Auth: conf.Auth{
			JWT: conf.JWTAuth{
				Issuer:         "issuer-a",
				SigningKey:     "shared-secret",
				AccessTokenTTL: "1h",
			},
		},
	})
	if err != nil {
		t.Fatalf("NewTokenManager returned error: %v", err)
	}

	token, err := source.IssueAccessToken(Principal{
		UserID:  1,
		Subject: "user:1",
	})
	if err != nil {
		t.Fatalf("IssueAccessToken returned error: %v", err)
	}

	target, err := NewTokenManager(&conf.Bootstrap{
		Auth: conf.Auth{
			JWT: conf.JWTAuth{
				Issuer:         "issuer-b",
				SigningKey:     "shared-secret",
				AccessTokenTTL: "1h",
			},
		},
	})
	if err != nil {
		t.Fatalf("NewTokenManager returned error: %v", err)
	}

	if _, err := target.ParseAccessToken(token); err == nil {
		t.Fatal("expected issuer validation error")
	}
}

func TestJWTTokenManagerRejectsInvalidSigningKey(t *testing.T) {
	source, err := NewTokenManager(&conf.Bootstrap{
		Auth: conf.Auth{
			JWT: conf.JWTAuth{
				Issuer:         "issuer",
				SigningKey:     "secret-a",
				AccessTokenTTL: "1h",
			},
		},
	})
	if err != nil {
		t.Fatalf("NewTokenManager returned error: %v", err)
	}

	token, err := source.IssueAccessToken(Principal{
		UserID:  1,
		Subject: "user:1",
	})
	if err != nil {
		t.Fatalf("IssueAccessToken returned error: %v", err)
	}

	target, err := NewTokenManager(&conf.Bootstrap{
		Auth: conf.Auth{
			JWT: conf.JWTAuth{
				Issuer:         "issuer",
				SigningKey:     "secret-b",
				AccessTokenTTL: "1h",
			},
		},
	})
	if err != nil {
		t.Fatalf("NewTokenManager returned error: %v", err)
	}

	if _, err := target.ParseAccessToken(token); err == nil {
		t.Fatal("expected signature validation error")
	}
}

func TestJWTTokenManagerRejectsExpiredToken(t *testing.T) {
	manager, err := NewTokenManager(&conf.Bootstrap{
		Auth: conf.Auth{
			JWT: conf.JWTAuth{
				Issuer:         "issuer",
				SigningKey:     "secret",
				AccessTokenTTL: "1h",
			},
		},
	})
	if err != nil {
		t.Fatalf("NewTokenManager returned error: %v", err)
	}

	jwtManager := manager.(*JWTTokenManager)
	jwtManager.now = func() time.Time {
		return time.Unix(2000, 0).UTC()
	}

	token, err := manager.IssueAccessToken(Principal{
		UserID:    1,
		Subject:   "user:1",
		IssuedAt:  time.Unix(1000, 0).UTC(),
		ExpiresAt: time.Unix(1500, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("IssueAccessToken returned error: %v", err)
	}

	if _, err := manager.ParseAccessToken(token); err == nil {
		t.Fatal("expected expiration validation error")
	}
}

func TestJWTTokenManagerRequiresPrincipalFields(t *testing.T) {
	manager, err := NewTokenManager(&conf.Bootstrap{
		Auth: conf.Auth{
			JWT: conf.JWTAuth{
				Issuer:         "issuer",
				SigningKey:     "secret",
				AccessTokenTTL: "1h",
			},
		},
	})
	if err != nil {
		t.Fatalf("NewTokenManager returned error: %v", err)
	}

	if _, err := manager.IssueAccessToken(Principal{Subject: "user:1"}); err == nil {
		t.Fatal("expected missing user id error")
	}
	if _, err := manager.IssueAccessToken(Principal{UserID: 1}); err == nil {
		t.Fatal("expected missing subject error")
	}
}

func TestNewTokenManagerRejectsInvalidDuration(t *testing.T) {
	_, err := NewTokenManager(&conf.Bootstrap{
		Auth: conf.Auth{
			JWT: conf.JWTAuth{
				Issuer:         "issuer",
				SigningKey:     "secret",
				AccessTokenTTL: "bad-duration",
			},
		},
	})
	if err == nil {
		t.Fatal("expected parse duration error")
	}
	if !strings.Contains(err.Error(), "parse auth.jwt.access_token_ttl") {
		t.Fatalf("unexpected error: %v", err)
	}
}
