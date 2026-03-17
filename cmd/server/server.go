package main

import (
	"github.com/go-kratos/kratos/v2"
	kratosgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"

	userv1 "project/gen/user/v1"
	walletv1 "project/gen/wallet/v1"
	"project/internal/conf"
	"project/internal/feature/user"
	"project/internal/feature/wallet"
)

func newHTTPServer(c *conf.Bootstrap, userSvc *user.Service, walletSvc *wallet.Service) *kratoshttp.Server {
	srv := kratoshttp.NewServer(
		kratoshttp.Address(c.Server.HTTP.Addr),
	)
	userv1.RegisterUserServiceHTTPServer(srv, userSvc)
	walletv1.RegisterWalletServiceHTTPServer(srv, walletSvc)
	return srv
}

func newGRPCServer(c *conf.Bootstrap, userSvc *user.Service, walletSvc *wallet.Service) *kratosgrpc.Server {
	srv := kratosgrpc.NewServer(
		kratosgrpc.Address(c.Server.GRPC.Addr),
	)
	userv1.RegisterUserServiceServer(srv, userSvc)
	walletv1.RegisterWalletServiceServer(srv, walletSvc)
	return srv
}

func newApp(hs *kratoshttp.Server, gs *kratosgrpc.Server) *kratos.App {
	return kratos.New(
		kratos.Server(hs, gs),
	)
}
