package main

import "github.com/hayeah/agentboss"

// Boss bundles the dependencies that CLI subcommands need.
type Boss struct {
	store    *agentboss.ProcessStore
	tmux     *agentboss.Tmux
	detector *agentboss.DetectorRunner
	boss     *agentboss.Boss
}

// NewBoss creates the CLI Boss bundle.
func NewBoss(store *agentboss.ProcessStore, tmux *agentboss.Tmux, detector *agentboss.DetectorRunner, boss *agentboss.Boss) *Boss {
	return &Boss{store: store, tmux: tmux, detector: detector, boss: boss}
}
