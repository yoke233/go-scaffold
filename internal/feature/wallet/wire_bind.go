package wallet

import (
	"github.com/google/wire"

	"project/internal/domain/ports"
)

// WireBind binds Facade to the ports.WalletQuery interface for cross-domain DI.
// Separate from wire.go so codegen won't overwrite it.
var WireBind = wire.NewSet(
	NewFacade,
	wire.Bind(new(ports.WalletQuery), new(*Facade)),
	wire.Bind(new(ports.WalletWriter), new(*Facade)),
)
