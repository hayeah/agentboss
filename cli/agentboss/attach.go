package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newAttachCmd(b *Boss) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attach HASH",
		Short: "Attach to a supervised CLI's tmux session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			proc, err := b.store.Resolve(args[0])
			if err != nil {
				return err
			}

			if !b.store.IsAlive(proc.Hash) {
				return fmt.Errorf("process %s is not running", proc.HashID)
			}

			return b.tmux.Attach(proc.TmuxSession)
		},
	}

	return cmd
}
