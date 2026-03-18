package platform

import (
	"github.com/google/wire"

	"project/internal/platform/authn"
	"project/internal/platform/database"
)

var ProviderSet = wire.NewSet(
	authn.NewTokenManager,
	database.NewDB,
	database.NewTransactor,
	database.NewUnitOfWork,
)
