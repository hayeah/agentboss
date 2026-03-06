package agentboss

import (
	"fmt"
	"os"
	"os/signal"
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
	store *ProcessStore
	tmux  *Tmux
}

// NewBoss creates a Boss.
func NewBoss(store *ProcessStore, tmux *Tmux) *Boss {
	return &Boss{store: store, tmux: tmux}
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
