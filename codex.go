package agentboss

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// CodexSession reads Codex's session JSONL logs.
type CodexSession struct{}

// NewCodexSession creates a CodexSession.
func NewCodexSession() *CodexSession {
	return &CodexSession{}
}

// CodexBookmark holds a position in a session JSONL file.
type CodexBookmark struct {
	Path      string
	SessionID string
	Offset    int64
}

var codexSessionIDRe = regexp.MustCompile(`Session:\s+([0-9a-f-]{36})`)

// sessionFilePath finds the JSONL file for a given session ID.
// Codex stores sessions as ~/.codex/sessions/YYYY/MM/DD/rollout-*-<id>.jsonl
func sessionFilePath(sessionID string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	sessionsDir := filepath.Join(home, ".codex", "sessions")

	var found string
	filepath.Walk(sessionsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.Contains(filepath.Base(path), sessionID) {
			found = path
			return filepath.SkipAll
		}
		return nil
	})

	if found == "" {
		return "", fmt.Errorf("no codex session file found for ID %s", sessionID)
	}
	return found, nil
}

// Bookmark records the current end-of-file position for the given session ID.
// If the session file doesn't exist yet (Codex creates it on first message),
// it stores the session ID and resolves the path lazily in ReadReply.
func (c *CodexSession) Bookmark(sessionID string) (*CodexBookmark, error) {
	path, err := sessionFilePath(sessionID)
	if err != nil {
		// File doesn't exist yet — will resolve lazily
		return &CodexBookmark{SessionID: sessionID, Offset: 0}, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return &CodexBookmark{SessionID: sessionID, Offset: 0}, nil
	}
	return &CodexBookmark{Path: path, SessionID: sessionID, Offset: info.Size()}, nil
}

type codexEntry struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type codexResponseItem struct {
	Type    string              `json:"type"`
	Role    string              `json:"role"`
	Content []codexContentBlock `json:"content"`
}

type codexContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ReadReply reads new JSONL entries after the bookmark and extracts assistant text.
func (bm *CodexBookmark) ReadReply() (string, error) {
	// Poll up to 10 seconds for assistant entries to appear in the JSONL.
	for range 20 {
		reply, err := bm.readReplyOnce()
		if err != nil {
			return "", err
		}
		if reply != "" {
			return reply, nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return "", nil
}

func (bm *CodexBookmark) readReplyOnce() (string, error) {
	// Resolve path lazily if not yet set
	if bm.Path == "" {
		path, err := sessionFilePath(bm.SessionID)
		if err != nil {
			return "", nil // file doesn't exist yet, retry later
		}
		bm.Path = path
	}

	f, err := os.Open(bm.Path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := f.Seek(bm.Offset, io.SeekStart); err != nil {
		return "", err
	}

	var texts []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		var entry codexEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if entry.Type != "response_item" {
			continue
		}

		var item codexResponseItem
		if err := json.Unmarshal(entry.Payload, &item); err != nil {
			continue
		}
		if item.Type != "message" || item.Role != "assistant" {
			continue
		}

		for _, b := range item.Content {
			if b.Type == "output_text" && b.Text != "" {
				texts = append(texts, b.Text)
			}
		}
	}

	return strings.Join(texts, "\n"), nil
}
