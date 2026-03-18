package requestid

import (
	"context"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/google/uuid"
)

const HeaderName = "X-Request-Id"

type contextKey struct{}

func Server() middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			requestID := FromContext(ctx)
			if requestID == "" {
				requestID = fromTransport(ctx)
			}
			if requestID == "" {
				requestID = uuid.NewString()
			}

			if tr, ok := transport.FromServerContext(ctx); ok {
				tr.ReplyHeader().Set(HeaderName, requestID)
			}

			ctx = context.WithValue(ctx, contextKey{}, requestID)
			return next(ctx, req)
		}
	}
}

func FromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	requestID, _ := ctx.Value(contextKey{}).(string)
	return requestID
}

func fromTransport(ctx context.Context) string {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return ""
	}

	if requestID := tr.RequestHeader().Get(HeaderName); requestID != "" {
		return requestID
	}

	return tr.RequestHeader().Get("X-Request-ID")
}
