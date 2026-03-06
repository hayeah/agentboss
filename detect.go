package agentboss

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hayeah/agentboss/conf"
)

// DetectorResult is the JSON output from a detector script.
type DetectorResult struct {
	State string `json:"state"`
}

// DetectorRunner runs detector scripts against pane output.
type DetectorRunner struct {
	store *ProcessStore
	tmux  *Tmux
	stateDir conf.StateDir
}

// NewDetectorRunner creates a DetectorRunner.
func NewDetectorRunner(store *ProcessStore, tmux *Tmux, stateDir conf.StateDir) *DetectorRunner {
	return &DetectorRunner{store: store, tmux: tmux, stateDir: stateDir}
}

// Detect runs the detector script for a process and returns the result.
func (d *DetectorRunner) Detect(proc *Process) (*DetectorResult, error) {
	script := d.findScript(proc)
	if script == "" {
		return &DetectorResult{State: "unknown"}, nil
	}

	content, err := d.tmux.CapturePan(proc.TmuxSession, 50)
	if err != nil {
		return nil, fmt.Errorf("capture pane: %w", err)
	}

	cmd := exec.Command("python3", script)
	cmd.Stdin = bytes.NewBufferString(content)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("detector script failed: %w", err)
	}

	var result DetectorResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("detector output invalid: %w", err)
	}
	return &result, nil
}

// findScript locates the detector script for a process.
// Priority: per-instance detect.py > named detector > none.
func (d *DetectorRunner) findScript(proc *Process) string {
	// Per-instance detector
	perInstance := filepath.Join(d.store.Dir(proc.Hash), "detect.py")
	if _, err := os.Stat(perInstance); err == nil {
		return perInstance
	}

	// Named detector
	if proc.Detector != "" {
		named := filepath.Join(string(d.stateDir), "detectors", proc.Detector+".py")
		if _, err := os.Stat(named); err == nil {
			return named
		}
	}

	return ""
}
