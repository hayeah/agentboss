package main

import (
	"fmt"
	"regexp"
	"time"

	"github.com/hayeah/agentboss"
	"github.com/spf13/cobra"
)

func newExpectCmd(b *Boss) *cobra.Command {
	var flagState string
	var flagChange bool
	var flagTimeout string
	var flagPoll string
	var flagLines int

	cmd := &cobra.Command{
		Use:   "expect HASH [PATTERN]",
		Short: "Wait for a pattern, state, or content change in the pane",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			proc, err := b.store.Resolve(args[0])
			if err != nil {
				return err
			}

			if !b.store.IsAlive(proc.Hash) {
				return fmt.Errorf("process %s is not running", proc.HashID)
			}

			opts := agentboss.ExpectOpts{
				State:  flagState,
				Change: flagChange,
				Lines:  flagLines,
			}

			if len(args) > 1 {
				re, err := regexp.Compile(args[1])
				if err != nil {
					return fmt.Errorf("invalid pattern: %w", err)
				}
				opts.Pattern = re
			}

			if flagTimeout != "" {
				d, err := time.ParseDuration(flagTimeout)
				if err != nil {
					return fmt.Errorf("invalid timeout: %w", err)
				}
				opts.Timeout = d
			}

			if flagPoll != "" {
				d, err := time.ParseDuration(flagPoll)
				if err != nil {
					return fmt.Errorf("invalid poll interval: %w", err)
				}
				opts.Poll = d
			}

			if opts.Pattern == nil && opts.State == "" && !opts.Change {
				return fmt.Errorf("specify a PATTERN, --state, or --change")
			}

			content, err := b.boss.Expect(proc, opts)
			if err != nil {
				return err
			}

			if content != "" {
				fmt.Print(content)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&flagState, "state", "", "Wait for detector state (idle, working, waiting)")
	cmd.Flags().BoolVar(&flagChange, "change", false, "Wait for any pane content change")
	cmd.Flags().StringVar(&flagTimeout, "timeout", "", "Timeout duration (default: 5s for pattern, 60s for state)")
	cmd.Flags().StringVar(&flagPoll, "poll", "", "Poll interval (default: 75ms)")
	cmd.Flags().IntVar(&flagLines, "lines", 50, "Pane capture depth")

	return cmd
}
