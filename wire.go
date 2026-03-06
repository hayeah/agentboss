package agentboss

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewProcessStore,
	NewTmux,
	NewDetectorRunner,
	NewBoss,
)
