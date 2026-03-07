package main

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"time"

	"github.com/hayeah/agentboss"
	"github.com/spf13/cobra"
)

func newSendCmd(b *Boss) *cobra.Command {
	var flagKeys bool
	var flagNoEnter bool
	var flagWait bool
	var flagWaitReply bool
	var flagTimeout int
	var flagExpect string
	var flagExpectState string
	var flagExpectChange bool

	cmd := &cobra.Command{
		Use:   "send HASH TEXT|--keys KEY...",
		Short: "Send input to a supervised CLI",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			proc, err := b.store.Resolve(args[0])
			if err != nil {
				return err
			}

			if !b.store.IsAlive(proc.Hash) {
				return fmt.Errorf("process %s is not running", proc.HashID)
			}

			// Build the send function
			sendFn := func() error {
				if flagKeys {
					return b.tmux.SendKeys(proc.TmuxSession, args[1:]...)
				}
				text := args[1]
				if err := b.tmux.SendText(proc.TmuxSession, text); err != nil {
					return err
				}
				if !flagNoEnter {
					time.Sleep(100 * time.Millisecond)
					return b.tmux.SendKeys(proc.TmuxSession, "Enter")
				}
				return nil
			}

			// -- Expect mode: send + poll for condition --
			if flagExpect != "" || flagExpectState != "" || flagExpectChange {
				opts := agentboss.ExpectOpts{
					State:  flagExpectState,
					Change: flagExpectChange,
				}
				if flagExpect != "" {
					re, err := regexp.Compile(flagExpect)
					if err != nil {
						return fmt.Errorf("invalid expect pattern: %w", err)
					}
					opts.Pattern = re
				}
				if flagTimeout > 0 {
					opts.Timeout = time.Duration(flagTimeout) * time.Second
				}

				content, err := b.boss.SendAndExpect(proc, sendFn, opts)
				if err != nil {
					return err
				}
				if content != "" {
					fmt.Print(content)
				}
				return nil
			}

			// -- Legacy --wait / --wait-reply mode --

			// Bookmark the session log before sending (for --wait-reply)
			type replyReader interface {
				ReadReply() (string, error)
			}
			var bookmark replyReader
			if flagWaitReply {
				flagWait = true // --wait-reply implies --wait
				if proc.SessionID == "" {
					return fmt.Errorf("no session ID recorded for %s (restart the process to capture it)", proc.HashID)
				}
				switch proc.Detector {
				case "claude":
					session := agentboss.NewClaudeSession(proc.CWD)
					bookmark, err = session.Bookmark(proc.SessionID)
				case "codex":
					session := agentboss.NewCodexSession()
					bookmark, err = session.Bookmark(proc.SessionID)
				default:
					return fmt.Errorf("--wait-reply not supported for detector %q (supported: claude, codex)", proc.Detector)
				}
				if err != nil {
					return fmt.Errorf("bookmark session: %w", err)
				}
			}

			// Send
			if err := sendFn(); err != nil {
				return err
			}

			if !flagWait {
				return nil
			}

			// Snapshot output right after sending
			time.Sleep(200 * time.Millisecond)
			afterContent, _ := b.tmux.CapturePan(proc.TmuxSession, 50)
			afterHash := sha256.Sum256([]byte(afterContent))

			// Wait until: output changed AND state is idle
			deadline := time.Now().Add(time.Duration(flagTimeout) * time.Second)
			start := time.Now()

			for time.Now().Before(deadline) {
				if !b.store.IsAlive(proc.Hash) {
					return fmt.Errorf("process %s is no longer running", proc.HashID)
				}

				content, _ := b.tmux.CapturePan(proc.TmuxSession, 50)
				currentHash := sha256.Sum256([]byte(content))
				outputChanged := currentHash != afterHash

				result, err := b.detector.Detect(proc)
				if err != nil {
					return err
				}

				if outputChanged && result.State == "idle" {
					elapsed := time.Since(start).Milliseconds()

					if flagWaitReply && bookmark != nil {
						reply, err := bookmark.ReadReply()
						if err != nil {
							return fmt.Errorf("read reply: %w", err)
						}
						fmt.Print(reply)
					} else {
						fmt.Printf("idle (waited %dms)\n", elapsed)
					}
					return nil
				}

				time.Sleep(500 * time.Millisecond)
			}

			return fmt.Errorf("timed out after %ds waiting for %s to become idle", flagTimeout, proc.HashID)
		},
	}

	cmd.Flags().BoolVar(&flagKeys, "keys", false, "Send raw tmux key names")
	cmd.Flags().BoolVar(&flagNoEnter, "no-enter", false, "Don't press Enter after text")
	cmd.Flags().BoolVar(&flagWait, "wait", false, "Block until the agent returns to idle after processing")
	cmd.Flags().BoolVar(&flagWaitReply, "wait-reply", false, "Wait for idle and print the agent's reply (implies --wait)")
	cmd.Flags().IntVar(&flagTimeout, "timeout", 60, "Timeout in seconds")
	cmd.Flags().StringVar(&flagExpect, "expect", "", "Wait for regex pattern in pane after sending")
	cmd.Flags().StringVar(&flagExpectState, "expect-state", "", "Wait for detector state after sending (idle, working, waiting)")
	cmd.Flags().BoolVar(&flagExpectChange, "expect-change", false, "Wait for any pane content change after sending")

	return cmd
}
