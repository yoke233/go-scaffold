package authn

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"

	platformauth "project/internal/platform/authn"
)

const authorizationHeader = "Authorization"

func Server(tokenManager platformauth.TokenManager) middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			token, err := bearerTokenFromContext(ctx)
			if err != nil {
				return nil, err
			}

			principal, err := tokenManager.ParseAccessToken(token)
			if err != nil {
				return nil, errors.Unauthorized("UNAUTHORIZED", "invalid access token")
			}

			ctx = platformauth.NewContext(ctx, principal)
			return next(ctx, req)
		}
	}
}

func bearerTokenFromContext(ctx context.Context) (string, error) {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return "", errors.Unauthorized("UNAUTHORIZED", "missing bearer token")
	}

	raw := firstNonEmpty(
		tr.RequestHeader().Get(authorizationHeader),
		tr.RequestHeader().Get(strings.ToLower(authorizationHeader)),
	)
	if strings.TrimSpace(raw) == "" {
		return "", errors.Unauthorized("UNAUTHORIZED", "missing bearer token")
	}

	token, ok := parseBearerToken(raw)
	if !ok {
		return "", errors.Unauthorized("UNAUTHORIZED", "invalid bearer token")
	}
	return token, nil
}

func parseBearerToken(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}

	parts := strings.Fields(value)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	if strings.TrimSpace(parts[1]) == "" {
		return "", false
	}

	return parts[1], true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
