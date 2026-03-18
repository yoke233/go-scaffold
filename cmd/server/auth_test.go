package main

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/go-kratos/kratos/v2/transport"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
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

type testHTTPTransport struct {
	request      transport.Header
	reply        transport.Header
	operation    string
	pathTemplate string
	requestObj   *http.Request
}

func (t *testHTTPTransport) Kind() transport.Kind {
	return transport.KindHTTP
}

func (t *testHTTPTransport) Endpoint() string {
	return "http://127.0.0.1:8080"
}

func (t *testHTTPTransport) Operation() string {
	return t.operation
}

func (t *testHTTPTransport) RequestHeader() transport.Header {
	return t.request
}

func (t *testHTTPTransport) ReplyHeader() transport.Header {
	return t.reply
}

func (t *testHTTPTransport) Request() *http.Request {
	return t.requestObj
}

func (t *testHTTPTransport) PathTemplate() string {
	return t.pathTemplate
}

var _ kratoshttp.Transporter = (*testHTTPTransport)(nil)

func TestIsPublicOperationRecognizesCreateUser(t *testing.T) {
	if !isPublicOperation(context.Background(), "/user.v1.UserService/CreateUser") {
		t.Fatal("expected CreateUser to be public")
	}
	if isPublicOperation(context.Background(), "/user.v1.UserService/GetUser") {
		t.Fatal("expected GetUser to require auth")
	}
	if !requiresAuthentication(context.Background(), "/user.v1.UserService/GetUser") {
		t.Fatal("expected GetUser to require authentication")
	}
	if requiresAuthentication(context.Background(), "/user.v1.UserService/CreateUser") {
		t.Fatal("expected CreateUser to skip authentication")
	}
}

func TestIsPublicOperationRecognizesHealthEndpoints(t *testing.T) {
	req := &http.Request{URL: &url.URL{Path: "/healthz"}}
	tr := &testHTTPTransport{
		request:      testHeader{},
		reply:        testHeader{},
		pathTemplate: "/healthz",
		requestObj:   req,
	}

	ctx := transport.NewServerContext(context.Background(), tr)
	if !isPublicOperation(ctx, "") {
		t.Fatal("expected /healthz to be public")
	}

	tr.pathTemplate = "/readyz"
	tr.requestObj = &http.Request{URL: &url.URL{Path: "/readyz"}}
	ctx = transport.NewServerContext(context.Background(), tr)
	if !isPublicOperation(ctx, "") {
		t.Fatal("expected /readyz to be public")
	}
}
