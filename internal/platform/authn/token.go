package authn

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"project/internal/conf"
)

type Claims struct {
	UID int64 `json:"uid"`
	jwt.RegisteredClaims
}

type TokenManager interface {
	IssueAccessToken(principal Principal) (string, error)
	ParseAccessToken(token string) (*Principal, error)
}

type JWTTokenManager struct {
	issuer         string
	signingKey     []byte
	accessTokenTTL time.Duration
	now            func() time.Time
}

func NewTokenManager(c *conf.Bootstrap) (TokenManager, error) {
	ttl, err := time.ParseDuration(c.Auth.JWT.AccessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("parse auth.jwt.access_token_ttl: %w", err)
	}

	return &JWTTokenManager{
		issuer:         c.Auth.JWT.Issuer,
		signingKey:     []byte(c.Auth.JWT.SigningKey),
		accessTokenTTL: ttl,
		now:            time.Now,
	}, nil
}

func (m *JWTTokenManager) IssueAccessToken(principal Principal) (string, error) {
	if principal.UserID <= 0 {
		return "", errors.New("authn: principal user id must be greater than zero")
	}
	if strings.TrimSpace(principal.Subject) == "" {
		return "", errors.New("authn: principal subject is required")
	}

	issuedAt := principal.IssuedAt
	if issuedAt.IsZero() {
		issuedAt = m.now().UTC()
	}
	expiresAt := principal.ExpiresAt
	if expiresAt.IsZero() {
		expiresAt = issuedAt.Add(m.accessTokenTTL)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UID: principal.UserID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   principal.Subject,
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	})

	return token.SignedString(m.signingKey)
}

func (m *JWTTokenManager) ParseAccessToken(token string) (*Principal, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("authn: token is required")
	}

	claims := &Claims{}
	parsedToken, err := jwt.ParseWithClaims(
		token,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("authn: unexpected signing method %s", token.Method.Alg())
			}
			return m.signingKey, nil
		},
		jwt.WithIssuer(m.issuer),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(m.now),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		return nil, err
	}
	if !parsedToken.Valid {
		return nil, errors.New("authn: token is invalid")
	}
	if claims.UID <= 0 {
		return nil, errors.New("authn: uid claim is required")
	}
	if strings.TrimSpace(claims.Subject) == "" {
		return nil, errors.New("authn: subject claim is required")
	}
	if claims.IssuedAt == nil {
		return nil, errors.New("authn: iat claim is required")
	}
	if claims.ExpiresAt == nil {
		return nil, errors.New("authn: exp claim is required")
	}

	return &Principal{
		UserID:    claims.UID,
		Subject:   claims.Subject,
		IssuedAt:  claims.IssuedAt.Time,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}
