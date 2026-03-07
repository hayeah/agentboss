package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newOutputCmd(b *Boss) *cobra.Command {
	var lines int
	var escapes bool

	cmd := &cobra.Command{
		Use:   "output HASH",
		Short: "Read terminal output from a supervised CLI",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			proc, err := b.store.Resolve(args[0])
			if err != nil {
				return err
			}

			var content string
			if escapes {
				content, err = b.tmux.CapturePaneEscapes(proc.TmuxSession, lines)
			} else {
				content, err = b.tmux.CapturePan(proc.TmuxSession, lines)
			}
			if err != nil {
				return err
			}

			fmt.Print(content)
			return nil
		},
	}

	cmd.Flags().IntVarP(&lines, "lines", "n", 50, "Number of lines to capture")
	cmd.Flags().BoolVarP(&escapes, "escapes", "e", false, "Include ANSI escape sequences")

	return cmd
}
