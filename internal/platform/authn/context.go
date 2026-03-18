package authn

import (
	"context"
	"time"
)

type Principal struct {
	UserID    int64
	Subject   string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

type principalContextKey struct{}

func NewContext(ctx context.Context, principal *Principal) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func FromContext(ctx context.Context) (*Principal, bool) {
	if ctx == nil {
		return nil, false
	}

	principal, ok := ctx.Value(principalContextKey{}).(*Principal)
	if !ok || principal == nil {
		return nil, false
	}

	return principal, true
}

func MustFromContext(ctx context.Context) *Principal {
	principal, ok := FromContext(ctx)
	if !ok {
		panic("authn: principal missing from context")
	}
	return principal
}

func UserID(ctx context.Context) (int64, bool) {
	principal, ok := FromContext(ctx)
	if !ok || principal.UserID <= 0 {
		return 0, false
	}
	return principal.UserID, true
}
