//go:build wireinject

package main

import (
	"log/slog"

	"github.com/go-kratos/kratos/v2"
	"github.com/google/wire"

	"project/internal/conf"
	"project/internal/platform"
)

func wireApp(*conf.Bootstrap, *slog.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(
		platform.ProviderSet,
		featureProviderSet,
		newHTTPServer,
		newGRPCServer,
		newApp,
	))
}
