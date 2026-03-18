package requestid

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/transport"
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
	request transport.Header
	reply   transport.Header
}

func (t *testTransport) Kind() transport.Kind {
	return transport.KindHTTP
}

func (t *testTransport) Endpoint() string {
	return "http://127.0.0.1:8080"
}

func (t *testTransport) Operation() string {
	return "/test.Service/Test"
}

func (t *testTransport) RequestHeader() transport.Header {
	return t.request
}

func (t *testTransport) ReplyHeader() transport.Header {
	return t.reply
}

func TestServerMiddlewarePreservesIncomingRequestID(t *testing.T) {
	tr := &testTransport{
		request: testHeader{HeaderName: "req-123"},
		reply:   testHeader{},
	}
	ctx := transport.NewServerContext(context.Background(), tr)

	handler := Server()(func(ctx context.Context, req any) (any, error) {
		if got := FromContext(ctx); got != "req-123" {
			t.Fatalf("expected request id %q, got %q", "req-123", got)
		}
		return "ok", nil
	})

	if _, err := handler(ctx, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := tr.reply.Get(HeaderName); got != "req-123" {
		t.Fatalf("expected reply header %q, got %q", "req-123", got)
	}
}

func TestServerMiddlewareGeneratesRequestIDWhenMissing(t *testing.T) {
	tr := &testTransport{
		request: testHeader{},
		reply:   testHeader{},
	}
	ctx := transport.NewServerContext(context.Background(), tr)

	handler := Server()(func(ctx context.Context, req any) (any, error) {
		if got := FromContext(ctx); got == "" {
			t.Fatal("expected generated request id")
		}
		return "ok", nil
	})

	if _, err := handler(ctx, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := tr.reply.Get(HeaderName); got == "" {
		t.Fatal("expected reply header request id")
	}
}
