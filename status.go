package agentboss

import (
	"fmt"
	"regexp"
	"time"
)

// queryStatus sends /status to a tmux session, captures the pane output,
// extracts a value using the given regex (first capture group), then dismisses
// the overlay with Escape. Retries up to 3 times if the overlay doesn't render.
func queryStatus(tmux *Tmux, tmuxSession string, re *regexp.Regexp) (string, error) {
	// Send /status and Enter
	if err := tmux.SendText(tmuxSession, "/status"); err != nil {
		return "", err
	}
	time.Sleep(100 * time.Millisecond)
	if err := tmux.SendKeys(tmuxSession, "Enter"); err != nil {
		return "", err
	}

	// Poll for the overlay to render (up to 5 seconds)
	var content string
	var match []string
	for range 10 {
		time.Sleep(500 * time.Millisecond)
		var err error
		content, err = tmux.CapturePan(tmuxSession, 50)
		if err != nil {
			continue
		}
		match = re.FindStringSubmatch(content)
		if match != nil {
			break
		}
	}

	// Dismiss the overlay
	tmux.SendKeys(tmuxSession, "Escape")
	time.Sleep(200 * time.Millisecond)

	if match == nil {
		return "", fmt.Errorf("session ID not found in /status output")
	}
	return match[1], nil
}
