package agentboss

import (
	"fmt"
	"regexp"
	"time"
)

// ExpectOpts configures what to wait for when polling a tmux pane.
type ExpectOpts struct {
	Pattern *regexp.Regexp
	State   string
	Change  bool
	Timeout time.Duration
	Poll    time.Duration
	Lines   int
}

func (o *ExpectOpts) applyDefaults() {
	if o.Poll == 0 {
		o.Poll = 75 * time.Millisecond
	}
	if o.Timeout == 0 {
		if o.State != "" {
			o.Timeout = 60 * time.Second
		} else {
			o.Timeout = 5 * time.Second
		}
	}
	if o.Lines == 0 {
		o.Lines = 50
	}
}

// Expect polls the tmux pane until a condition is met.
// Returns the pane content at the time of match.
func (b *Boss) Expect(proc *Process, opts ExpectOpts) (string, error) {
	opts.applyDefaults()

	var baseline string
	if opts.Change {
		baseline, _ = b.tmux.CapturePan(proc.TmuxSession, opts.Lines)
	}

	return b.expectLoop(proc, opts, baseline)
}

// SendAndExpect sends input then polls until a condition is met.
// The baseline for change detection is captured before sending.
func (b *Boss) SendAndExpect(proc *Process, sendFn func() error, opts ExpectOpts) (string, error) {
	opts.applyDefaults()

	// For change detection, snapshot before sending
	var baseline string
	if opts.Change {
		baseline, _ = b.tmux.CapturePan(proc.TmuxSession, opts.Lines)
	}

	if err := sendFn(); err != nil {
		return "", fmt.Errorf("send: %w", err)
	}

	return b.expectLoop(proc, opts, baseline)
}

func (b *Boss) expectLoop(proc *Process, opts ExpectOpts, changeBaseline string) (string, error) {
	deadline := time.Now().Add(opts.Timeout)

	for time.Now().Before(deadline) {
		content, err := b.tmux.CapturePan(proc.TmuxSession, opts.Lines)
		if err != nil {
			time.Sleep(opts.Poll)
			continue
		}

		matched := false
		switch {
		case opts.Pattern != nil:
			matched = opts.Pattern.MatchString(content)
		case opts.State != "":
			result, err := b.detector.Detect(proc)
			matched = err == nil && result.State == opts.State
		case opts.Change:
			matched = content != changeBaseline
		}

		if matched {
			return content, nil
		}

		time.Sleep(opts.Poll)
	}

	return "", fmt.Errorf("expect timed out after %s", opts.Timeout)
}
