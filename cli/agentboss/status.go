package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newStatusCmd(b *Boss) *cobra.Command {
	var flagQuiet bool

	cmd := &cobra.Command{
		Use:   "status HASH",
		Short: "Show status of a supervised CLI",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			proc, err := b.store.Resolve(args[0])
			if err != nil {
				return err
			}

			alive := b.store.IsAlive(proc.Hash)
			state := "dead"
			if alive {
				result, err := b.detector.Detect(proc)
				if err != nil {
					return err
				}
				state = result.State
			}

			if flagQuiet {
				fmt.Println(state)
				return nil
			}

			info := struct {
				Hash      string `json:"hash"`
				HashID    string `json:"hashid"`
				Key       string `json:"key,omitempty"`
				CWD       string `json:"cwd"`
				CMD       []string `json:"cmd"`
				State     string `json:"state"`
				CreatedAt string `json:"created_at"`
			}{
				Hash:      proc.Hash,
				HashID:    proc.HashID,
				Key:       proc.Key,
				CWD:       proc.CWD,
				CMD:       proc.CMD,
				State:     state,
				CreatedAt: proc.CreatedAt.Format("2006-01-02T15:04:05Z"),
			}

			data, err := json.MarshalIndent(info, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(os.Stdout, string(data))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagQuiet, "quiet", "q", false, "Print only the state")

	return cmd
}
