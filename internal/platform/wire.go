package platform

import (
	"github.com/google/wire"

	"project/internal/platform/database"
)

var ProviderSet = wire.NewSet(database.NewDB)
