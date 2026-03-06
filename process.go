package agentboss

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofrs/flock"
	"github.com/hayeah/agentboss/conf"
)

// Process represents a supervised CLI instance.
type Process struct {
	Hash        string    `json:"hash"`
	HashID      string    `json:"hashid"`
	Key         string    `json:"key,omitempty"`
	CWD         string    `json:"cwd"`
	CMD         []string  `json:"cmd"`
	TmuxSession string    `json:"tmux_session"`
	Detector    string    `json:"detector,omitempty"`
	SessionID   string    `json:"session_id,omitempty"`
	PID         int       `json:"pid"`
	CreatedAt   time.Time `json:"created_at"`
}

// ComputeHash computes a 10-char hex hash from key (if set) or cwd+cmd.
func ComputeHash(key, cwd string, cmd []string) string {
	var input string
	if key != "" {
		input = key
	} else {
		input = cwd + "\x00" + strings.Join(cmd, "\x00")
	}
	sum := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", sum[:5])
}

// ShortestUniquePrefix computes the shortest unique prefix for hash
// among all existing hashes, with a minimum of 3 characters.
func ShortestUniquePrefix(hash string, allHashes []string) string {
	minLen := 3
	for l := minLen; l < len(hash); l++ {
		prefix := hash[:l]
		unique := true
		for _, h := range allHashes {
			if h == hash {
				continue
			}
			if strings.HasPrefix(h, prefix) {
				unique = false
				break
			}
		}
		if unique {
			return prefix
		}
	}
	return hash
}

// ProcessStore manages process state on disk.
type ProcessStore struct {
	stateDir conf.StateDir
}

// NewProcessStore creates a ProcessStore.
func NewProcessStore(stateDir conf.StateDir) *ProcessStore {
	return &ProcessStore{stateDir: stateDir}
}

// Dir returns the directory for a given hash.
func (s *ProcessStore) Dir(hash string) string {
	return filepath.Join(string(s.stateDir), hash)
}

// Save writes a process's state.json to disk.
func (s *ProcessStore) Save(p *Process) error {
	dir := s.Dir(p.Hash)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "state.json"), data, 0644)
}

// Load reads a process's state.json from disk.
func (s *ProcessStore) Load(hash string) (*Process, error) {
	data, err := os.ReadFile(filepath.Join(s.Dir(hash), "state.json"))
	if err != nil {
		return nil, err
	}
	var p Process
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// List returns all stored processes.
func (s *ProcessStore) List() ([]*Process, error) {
	entries, err := os.ReadDir(string(s.stateDir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var procs []*Process
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p, err := s.Load(e.Name())
		if err != nil {
			continue
		}
		procs = append(procs, p)
	}
	return procs, nil
}

// AllHashes returns just the hashes of all stored processes.
func (s *ProcessStore) AllHashes() ([]string, error) {
	procs, err := s.List()
	if err != nil {
		return nil, err
	}
	hashes := make([]string, len(procs))
	for i, p := range procs {
		hashes[i] = p.Hash
	}
	return hashes, nil
}

// Resolve matches a hash prefix (min 3 chars) to a full process.
func (s *ProcessStore) Resolve(prefix string) (*Process, error) {
	if len(prefix) < 3 {
		return nil, fmt.Errorf("prefix too short (minimum 3 characters)")
	}

	procs, err := s.List()
	if err != nil {
		return nil, err
	}

	// Try key match first
	for _, p := range procs {
		if p.Key != "" && p.Key == prefix {
			return p, nil
		}
	}

	// Try hash prefix
	var matches []*Process
	for _, p := range procs {
		if strings.HasPrefix(p.Hash, prefix) {
			matches = append(matches, p)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no process matching %q", prefix)
	case 1:
		return matches[0], nil
	default:
		var hashes []string
		for _, m := range matches {
			hashes = append(hashes, m.Hash)
		}
		return nil, fmt.Errorf("ambiguous prefix %q matches: %s", prefix, strings.Join(hashes, ", "))
	}
}

// LockPath returns the lock file path for a hash.
func (s *ProcessStore) LockPath(hash string) string {
	return filepath.Join(s.Dir(hash), "lock")
}

// Lock acquires an exclusive flock for a hash.
func (s *ProcessStore) Lock(hash string) (*flock.Flock, error) {
	if err := os.MkdirAll(s.Dir(hash), 0755); err != nil {
		return nil, err
	}
	fl := flock.New(s.LockPath(hash))
	ok, err := fl.TryLock()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("process %s is already running", hash)
	}
	return fl, nil
}

// IsAlive checks if a process is alive by trying to acquire its flock.
func (s *ProcessStore) IsAlive(hash string) bool {
	fl := flock.New(s.LockPath(hash))
	ok, err := fl.TryLock()
	if err != nil {
		return false
	}
	if ok {
		fl.Unlock()
		return false
	}
	return true
}
