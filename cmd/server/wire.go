//go:build wireinject

package main

import (
	"log/slog"

	"github.com/go-kratos/kratos/v2"
	"github.com/google/wire"

	"project/internal/conf"
	"project/internal/feature/user"
	"project/internal/feature/wallet"
	"project/internal/platform"
)

func wireApp(*conf.Bootstrap, *slog.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(
		platform.ProviderSet,
		user.ProviderSet,
		wallet.ProviderSet,
		newHTTPServer,
		newGRPCServer,
		newApp,
	))
}
