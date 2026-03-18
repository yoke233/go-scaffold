package main

import (
	nethttp "net/http"

	validate "github.com/go-kratos/kratos/contrib/middleware/validate/v2"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	kratosgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
	"gorm.io/gorm"

	"project/internal/conf"
	"project/internal/platform/httpx"
	"project/internal/platform/middleware/requestid"
)

func newHTTPServer(c *conf.Bootstrap, db *gorm.DB, services *featureServices) *kratoshttp.Server {
	srv := kratoshttp.NewServer(
		kratoshttp.Address(c.Server.HTTP.Addr),
		kratoshttp.Middleware(
			recovery.Recovery(),
			requestid.Server(),
			validate.ProtoValidate(),
		),
	)
	srv.HandleFunc("/healthz", httpx.HealthzHandler())
	srv.Handle("/readyz", httpx.ReadyzHandler(httpx.DBReadinessCheck(db)))
	srv.HandleFunc("/", func(w nethttp.ResponseWriter, _ *nethttp.Request) {
		w.WriteHeader(nethttp.StatusNotFound)
	})
	registerHTTPServices(srv, services)
	return srv
}

func newGRPCServer(c *conf.Bootstrap, services *featureServices) *kratosgrpc.Server {
	srv := kratosgrpc.NewServer(
		kratosgrpc.Address(c.Server.GRPC.Addr),
		kratosgrpc.Middleware(
			recovery.Recovery(),
			requestid.Server(),
			validate.ProtoValidate(),
		),
	)
	registerGRPCServices(srv, services)
	return srv
}

func newApp(hs *kratoshttp.Server, gs *kratosgrpc.Server) *kratos.App {
	return kratos.New(
		kratos.Server(hs, gs),
	)
}
