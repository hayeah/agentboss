package conf

import (
	"os"
	"path/filepath"

	"github.com/google/wire"
)

// StateDir is the root directory for agentboss state (~/.agentboss).
type StateDir string

// Config holds all configuration for agentboss.
type Config struct {
	StateDir StateDir
}

// NewConfig creates a Config with default values.
func NewConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return &Config{
		StateDir: StateDir(filepath.Join(home, ".agentboss")),
	}, nil
}

var ProviderSet = wire.NewSet(
	NewConfig,
	wire.FieldsOf(new(*Config), "StateDir"),
)
