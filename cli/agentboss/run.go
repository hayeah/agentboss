package main

import (
	"os"

	"github.com/hayeah/agentboss"
	"github.com/spf13/cobra"
)

func newRunCmd(b *Boss) *cobra.Command {
	var flagKey string
	var flagCWD string
	var flagDetector string

	cmd := &cobra.Command{
		Use:   "run [flags] -- COMMAND [ARGS...]",
		Short: "Spawn an interactive CLI in a tmux session",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd := flagCWD
			if cwd == "" {
				var err error
				cwd, err = os.Getwd()
				if err != nil {
					return err
				}
			}

			return b.boss.Run(agentboss.RunOpts{
				Key:      flagKey,
				CWD:      cwd,
				CMD:      args,
				Detector: flagDetector,
			})
		},
	}

	cmd.Flags().StringVar(&flagKey, "key", "", "Explicit key for hash identity")
	cmd.Flags().StringVar(&flagCWD, "cwd", "", "Working directory (default: current)")
	cmd.Flags().StringVar(&flagDetector, "detector", "", "Named detector script (e.g. claude, codex)")

	return cmd
}
