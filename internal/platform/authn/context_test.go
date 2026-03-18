package authn

import (
	"context"
	"testing"
	"time"
)

func TestContextHelpers(t *testing.T) {
	principal := &Principal{
		UserID:    42,
		Subject:   "user:42",
		IssuedAt:  time.Unix(100, 0),
		ExpiresAt: time.Unix(200, 0),
	}

	ctx := NewContext(context.Background(), principal)

	got, ok := FromContext(ctx)
	if !ok {
		t.Fatal("expected principal in context")
	}
	if got.UserID != 42 {
		t.Fatalf("expected user id 42, got %d", got.UserID)
	}

	userID, ok := UserID(ctx)
	if !ok || userID != 42 {
		t.Fatalf("expected user id helper to return 42, got %d %v", userID, ok)
	}

	if MustFromContext(ctx).Subject != "user:42" {
		t.Fatal("expected MustFromContext to return stored principal")
	}
}

func TestFromContextHandlesMissingPrincipal(t *testing.T) {
	if _, ok := FromContext(context.Background()); ok {
		t.Fatal("expected missing principal to return false")
	}
	if _, ok := UserID(context.Background()); ok {
		t.Fatal("expected missing user id to return false")
	}
}

func TestMustFromContextPanicsWhenMissing(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()

	_ = MustFromContext(context.Background())
}
