package authn

import (
	"context"
	"testing"
	"time"

	kratoserrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/transport"

	platformauth "project/internal/platform/authn"
)

type testHeader map[string]string

func (h testHeader) Get(key string) string {
	return h[key]
}

func (h testHeader) Set(key string, value string) {
	h[key] = value
}

func (h testHeader) Add(key string, value string) {
	h[key] = value
}

func (h testHeader) Keys() []string {
	keys := make([]string, 0, len(h))
	for key := range h {
		keys = append(keys, key)
	}
	return keys
}

func (h testHeader) Values(key string) []string {
	if value, ok := h[key]; ok {
		return []string{value}
	}
	return nil
}

type testTransport struct {
	request   transport.Header
	reply     transport.Header
	operation string
}

func (t *testTransport) Kind() transport.Kind {
	return transport.KindHTTP
}

func (t *testTransport) Endpoint() string {
	return "http://127.0.0.1:8080"
}

func (t *testTransport) Operation() string {
	return t.operation
}

func (t *testTransport) RequestHeader() transport.Header {
	return t.request
}

func (t *testTransport) ReplyHeader() transport.Header {
	return t.reply
}

type stubTokenManager struct {
	principal *platformauth.Principal
	err       error
}

func (s stubTokenManager) IssueAccessToken(principal platformauth.Principal) (string, error) {
	return "", nil
}

func (s stubTokenManager) ParseAccessToken(token string) (*platformauth.Principal, error) {
	if s.err != nil {
		return nil, s.err
	}
	if token == "" {
		return nil, context.DeadlineExceeded
	}
	return s.principal, nil
}

func TestServerRejectsMissingBearerToken(t *testing.T) {
	tr := &testTransport{
		request:   testHeader{},
		reply:     testHeader{},
		operation: "/user.v1.UserService/GetUser",
	}
	ctx := transport.NewServerContext(context.Background(), tr)

	handler := Server(stubTokenManager{})(func(ctx context.Context, req any) (any, error) {
		t.Fatal("expected middleware to stop request")
		return nil, nil
	})

	_, err := handler(ctx, nil)
	if err == nil {
		t.Fatal("expected unauthorized error")
	}

	kratosErr := new(kratoserrors.Error)
	if !kratoserrors.As(err, &kratosErr) || kratosErr.Code != 401 {
		t.Fatalf("expected 401 unauthorized, got %v", err)
	}
}

func TestServerRejectsInvalidBearerTokenFormat(t *testing.T) {
	tr := &testTransport{
		request:   testHeader{authorizationHeader: "Token abc"},
		reply:     testHeader{},
		operation: "/user.v1.UserService/GetUser",
	}
	ctx := transport.NewServerContext(context.Background(), tr)

	handler := Server(stubTokenManager{})(func(ctx context.Context, req any) (any, error) {
		t.Fatal("expected middleware to stop request")
		return nil, nil
	})

	_, err := handler(ctx, nil)
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
}

func TestServerInjectsPrincipalIntoContext(t *testing.T) {
	tr := &testTransport{
		request:   testHeader{authorizationHeader: "Bearer valid-token"},
		reply:     testHeader{},
		operation: "/user.v1.UserService/GetUser",
	}
	ctx := transport.NewServerContext(context.Background(), tr)

	expected := &platformauth.Principal{
		UserID:    99,
		Subject:   "user:99",
		IssuedAt:  time.Unix(100, 0),
		ExpiresAt: time.Unix(200, 0),
	}

	handler := Server(stubTokenManager{principal: expected})(func(ctx context.Context, req any) (any, error) {
		principal, ok := platformauth.FromContext(ctx)
		if !ok {
			t.Fatal("expected principal in context")
		}
		if principal.UserID != expected.UserID {
			t.Fatalf("expected user id %d, got %d", expected.UserID, principal.UserID)
		}
		return "ok", nil
	})

	if _, err := handler(ctx, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServerRejectsInvalidAccessToken(t *testing.T) {
	tr := &testTransport{
		request:   testHeader{"authorization": "Bearer expired-token"},
		reply:     testHeader{},
		operation: "/user.v1.UserService/GetUser",
	}
	ctx := transport.NewServerContext(context.Background(), tr)

	handler := Server(stubTokenManager{err: context.DeadlineExceeded})(func(ctx context.Context, req any) (any, error) {
		t.Fatal("expected middleware to stop request")
		return nil, nil
	})

	_, err := handler(ctx, nil)
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
}

func TestParseBearerToken(t *testing.T) {
	token, ok := parseBearerToken("Bearer abc.def")
	if !ok || token != "abc.def" {
		t.Fatalf("expected token abc.def, got %q %v", token, ok)
	}

	if _, ok := parseBearerToken("Bearer"); ok {
		t.Fatal("expected invalid token format")
	}
}
