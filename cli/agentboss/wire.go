//go:build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/hayeah/agentboss"
	"github.com/hayeah/agentboss/conf"
)

func InitializeApp() (*App, func(), error) {
	wire.Build(
		conf.ProviderSet,
		agentboss.ProviderSet,
		NewBoss,
		NewApp,
	)
	return nil, nil, nil
}
