package agentboss

import (
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"
)

// RunOpts are options for the Boss.Run command.
type RunOpts struct {
	Key      string
	CWD      string
	CMD      []string
	Detector string
}

// Boss supervises a single CLI in a tmux session.
type Boss struct {
	store    *ProcessStore
	tmux     *Tmux
	detector *DetectorRunner
}

// NewBoss creates a Boss.
func NewBoss(store *ProcessStore, tmux *Tmux, detector *DetectorRunner) *Boss {
	return &Boss{store: store, tmux: tmux, detector: detector}
}

// Run spawns the CLI in tmux, holds flock, and waits for exit.
func (b *Boss) Run(opts RunOpts) error {
	hash := ComputeHash(opts.Key, opts.CWD, opts.CMD)

	// Check if already running
	if b.store.IsAlive(hash) {
		proc, err := b.store.Load(hash)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "already running: %s (session: %s)\n", proc.HashID, proc.TmuxSession)
		return nil
	}

	// Acquire flock
	fl, err := b.store.Lock(hash)
	if err != nil {
		return err
	}
	defer fl.Unlock()

	// Compute unique short ID
	allHashes, err := b.store.AllHashes()
	if err != nil {
		return err
	}
	hashID := ShortestUniquePrefix(hash, allHashes)
	sessionName := "boss-" + hashID

	// Create tmux session
	if err := b.tmux.NewSession(sessionName, opts.CWD, opts.CMD); err != nil {
		return fmt.Errorf("create tmux session: %w", err)
	}

	// Save state
	proc := &Process{
		Hash:        hash,
		HashID:      hashID,
		Key:         opts.Key,
		CWD:         opts.CWD,
		CMD:         opts.CMD,
		TmuxSession: sessionName,
		Detector:    opts.Detector,
		PID:         os.Getpid(),
		CreatedAt:   time.Now(),
	}
	if err := b.store.Save(proc); err != nil {
		b.tmux.KillSession(sessionName)
		return err
	}

	fmt.Fprintf(os.Stderr, "agentboss: started %s (session: %s)\n", hashID, sessionName)
	fmt.Fprintf(os.Stderr, "agentboss: attach with: agentboss attach %s\n", hashID)

	// Capture agent session ID in background once it becomes idle
	if opts.Detector != "" {
		go b.captureSessionID(proc)
	}

	// Handle SIGINT/SIGTERM — kill tmux session
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintf(os.Stderr, "\nagentboss: shutting down %s...\n", hashID)
		b.tmux.KillSession(sessionName)
	}()

	// Block until session exits
	return b.tmux.WaitForExit(sessionName)
}

// sessionIDPatterns maps detector names to regex patterns for extracting
// the session ID from /status output.
var sessionIDPatterns = map[string]*regexp.Regexp{
	"claude": claudeSessionIDRe,
	"codex":  codexSessionIDRe,
}

// captureSessionID waits for the agent to become idle, then queries /status
// to extract and store the session ID.
func (b *Boss) captureSessionID(proc *Process) {
	re, ok := sessionIDPatterns[proc.Detector]
	if !ok {
		return
	}

	// Wait for idle (up to 120s for initial startup)
	deadline := time.Now().Add(120 * time.Second)
	for time.Now().Before(deadline) {
		result, err := b.detector.Detect(proc)
		if err == nil && result.State == "idle" {
			break
		}
		time.Sleep(2 * time.Second)
	}

	sessionID, err := queryStatus(b.tmux, proc.TmuxSession, re)
	if err != nil {
		fmt.Fprintf(os.Stderr, "agentboss: failed to capture session ID: %v\n", err)
		return
	}

	proc.SessionID = sessionID
	if err := b.store.Save(proc); err != nil {
		fmt.Fprintf(os.Stderr, "agentboss: failed to save session ID: %v\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "agentboss: captured session ID: %s\n", sessionID)
}
