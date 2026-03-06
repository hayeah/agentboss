package agentboss

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// Tmux wraps tmux commands.
type Tmux struct{}

// NewTmux creates a Tmux instance.
func NewTmux() *Tmux {
	return &Tmux{}
}

// NewSession creates a new detached tmux session running cmd in cwd.
func (t *Tmux) NewSession(name string, cwd string, cmd []string) error {
	args := []string{"new-session", "-d", "-s", name, "-c", cwd}
	args = append(args, cmd...)
	return t.run(args...)
}

// KillSession kills a tmux session.
func (t *Tmux) KillSession(name string) error {
	return t.run("kill-session", "-t", name)
}

// HasSession checks if a tmux session exists.
func (t *Tmux) HasSession(name string) bool {
	err := t.run("has-session", "-t", name)
	return err == nil
}

// CapturePan captures the last N lines of a tmux pane.
func (t *Tmux) CapturePan(name string, lines int) (string, error) {
	start := fmt.Sprintf("-%d", lines)
	out, err := t.output("capture-pane", "-t", name, "-p", "-S", start)
	if err != nil {
		return "", err
	}
	return out, nil
}

// SendKeys sends raw key names to a tmux session.
func (t *Tmux) SendKeys(name string, keys ...string) error {
	args := []string{"send-keys", "-t", name}
	args = append(args, keys...)
	return t.run(args...)
}

// SendText sends literal text to a tmux session via send-keys -l.
// Unlike PasteBuffer, this works reliably with TUI apps (Codex, Claude).
func (t *Tmux) SendText(name string, text string) error {
	return t.run("send-keys", "-t", name, "-l", text)
}

// PasteBuffer pastes text into a tmux session via an in-memory buffer.
// Uses set-buffer (in-memory) rather than load-buffer (file).
// Better for multi-line content but may not work with all TUI apps.
func (t *Tmux) PasteBuffer(name string, text string) error {
	bufName := "agentboss-paste"
	if err := t.run("set-buffer", "-b", bufName, "--", text); err != nil {
		return err
	}
	if err := t.run("paste-buffer", "-b", bufName, "-d", "-t", name); err != nil {
		return err
	}
	return nil
}

// Attach replaces the current process with tmux attach.
func (t *Tmux) Attach(name string) error {
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return err
	}
	return syscall.Exec(tmuxPath, []string{"tmux", "attach-session", "-t", name}, os.Environ())
}

// WaitForExit polls until the tmux session no longer exists.
func (t *Tmux) WaitForExit(name string) error {
	for {
		if !t.HasSession(name) {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (t *Tmux) run(args ...string) error {
	cmd := exec.Command("tmux", args...)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (t *Tmux) output(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\n"), nil
}
