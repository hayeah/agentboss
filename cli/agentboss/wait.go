package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func newWaitCmd(b *Boss) *cobra.Command {
	var timeout int

	cmd := &cobra.Command{
		Use:   "wait HASH",
		Short: "Block until the supervised CLI is idle",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			proc, err := b.store.Resolve(args[0])
			if err != nil {
				return err
			}

			deadline := time.Now().Add(time.Duration(timeout) * time.Second)
			start := time.Now()

			for time.Now().Before(deadline) {
				if !b.store.IsAlive(proc.Hash) {
					return fmt.Errorf("process %s is no longer running", proc.HashID)
				}

				result, err := b.detector.Detect(proc)
				if err != nil {
					return err
				}

				if result.State == "idle" {
					elapsed := time.Since(start).Milliseconds()
					fmt.Printf("idle (waited %dms)\n", elapsed)
					return nil
				}

				time.Sleep(1 * time.Second)
			}

			return fmt.Errorf("timed out after %ds waiting for %s to become idle", timeout, proc.HashID)
		},
	}

	cmd.Flags().IntVar(&timeout, "timeout", 60, "Timeout in seconds")

	return cmd
}
