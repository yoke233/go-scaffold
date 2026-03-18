package main

import (
	"context"
	nethttp "net/http"

	validate "github.com/go-kratos/kratos/contrib/middleware/validate/v2"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport"
	kratosgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
	"gorm.io/gorm"

	"project/internal/conf"
	platformauth "project/internal/platform/authn"
	"project/internal/platform/httpx"
	authmiddleware "project/internal/platform/middleware/authn"
	"project/internal/platform/middleware/requestid"
)

var publicOperations = map[string]struct{}{
	"/user.v1.UserService/CreateUser": {},
}

func newHTTPServer(c *conf.Bootstrap, db *gorm.DB, tokenManager platformauth.TokenManager, services *featureServices) *kratoshttp.Server {
	srv := kratoshttp.NewServer(
		kratoshttp.Address(c.Server.HTTP.Addr),
		kratoshttp.Middleware(serverMiddlewares(tokenManager)...),
	)
	srv.HandleFunc("/healthz", httpx.HealthzHandler())
	srv.Handle("/readyz", httpx.ReadyzHandler(httpx.DBReadinessCheck(db)))
	srv.HandleFunc("/", func(w nethttp.ResponseWriter, _ *nethttp.Request) {
		w.WriteHeader(nethttp.StatusNotFound)
	})
	registerHTTPServices(srv, services)
	return srv
}

func newGRPCServer(c *conf.Bootstrap, tokenManager platformauth.TokenManager, services *featureServices) *kratosgrpc.Server {
	srv := kratosgrpc.NewServer(
		kratosgrpc.Address(c.Server.GRPC.Addr),
		kratosgrpc.Middleware(serverMiddlewares(tokenManager)...),
	)
	registerGRPCServices(srv, services)
	return srv
}

func newApp(hs *kratoshttp.Server, gs *kratosgrpc.Server) *kratos.App {
	return kratos.New(
		kratos.Server(hs, gs),
	)
}

func serverMiddlewares(tokenManager platformauth.TokenManager) []middleware.Middleware {
	return []middleware.Middleware{
		recovery.Recovery(),
		requestid.Server(),
		selector.Server(authmiddleware.Server(tokenManager)).Match(requiresAuthentication).Build(),
		validate.ProtoValidate(),
	}
}

func requiresAuthentication(ctx context.Context, operation string) bool {
	return !isPublicOperation(ctx, operation)
}

func isPublicOperation(ctx context.Context, operation string) bool {
	if _, ok := publicOperations[operation]; ok {
		return true
	}

	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return false
	}

	httpTr, ok := tr.(kratoshttp.Transporter)
	if !ok {
		return false
	}

	if path := httpTr.PathTemplate(); path == "/healthz" || path == "/readyz" {
		return true
	}

	req := httpTr.Request()
	if req == nil || req.URL == nil {
		return false
	}

	return req.URL.Path == "/healthz" || req.URL.Path == "/readyz"
}
