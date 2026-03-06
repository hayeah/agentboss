package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

func newLsCmd(b *Boss) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List all supervised processes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			procs, err := b.store.List()
			if err != nil {
				return err
			}

			if len(procs) == 0 {
				fmt.Fprintln(os.Stderr, "no processes")
				return nil
			}

			type entry struct {
				Hash      string   `json:"hash"`
				HashID    string   `json:"hashid"`
				Key       string   `json:"key,omitempty"`
				CWD       string   `json:"cwd"`
				CMD       []string `json:"cmd"`
				SessionID string   `json:"session_id,omitempty"`
				State     string   `json:"state"`
				Age       string   `json:"age"`
			}

			var entries []entry
			for _, proc := range procs {
				state := "dead"
				if b.store.IsAlive(proc.Hash) {
					result, err := b.detector.Detect(proc)
					if err == nil {
						state = result.State
					} else {
						state = "alive"
					}
				}

				entries = append(entries, entry{
					Hash:      proc.Hash,
					HashID:    proc.HashID,
					Key:       proc.Key,
					CWD:       proc.CWD,
					CMD:       proc.CMD,
					SessionID: proc.SessionID,
					State:     state,
					Age:       formatAge(proc.CreatedAt),
				})
			}

			data, err := json.MarshalIndent(entries, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		},
	}

	return cmd
}

func formatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
