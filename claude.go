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

// ClaudeSession reads Claude Code's session JSONL logs.
type ClaudeSession struct {
	cwd string
}

// NewClaudeSession creates a ClaudeSession for the given working directory.
func NewClaudeSession(cwd string) *ClaudeSession {
	return &ClaudeSession{cwd: cwd}
}

// ClaudeBookmark holds a position in a session JSONL file.
type ClaudeBookmark struct {
	Path   string
	Offset int64
}

// claudeProjectDir returns the encoded project directory path.
func (c *ClaudeSession) claudeProjectDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	// Claude Code encodes paths by replacing / and . with -
	encoded := strings.NewReplacer("/", "-", ".", "-").Replace(c.cwd)
	return filepath.Join(home, ".claude", "projects", encoded), nil
}

var claudeSessionIDRe = regexp.MustCompile(`Session ID:\s+([0-9a-f-]{36})`)

// Bookmark records the current end-of-file position for the given session ID.
func (c *ClaudeSession) Bookmark(sessionID string) (*ClaudeBookmark, error) {
	dir, err := c.claudeProjectDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, sessionID+".jsonl")

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("session file %s: %w", path, err)
	}
	return &ClaudeBookmark{Path: path, Offset: info.Size()}, nil
}

type claudeEntry struct {
	Type    string    `json:"type"`
	Message claudeMsg `json:"message"`
}

type claudeMsg struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type claudeContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ReadReply reads new JSONL entries after the bookmark and extracts assistant text.
// It polls briefly since Claude Code may flush the JSONL slightly after the UI shows idle.
func (bm *ClaudeBookmark) ReadReply() (string, error) {
	// Poll up to 10 seconds for assistant entries to appear in the JSONL.
	// Claude Code flushes the log asynchronously after the UI shows idle.
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

func (bm *ClaudeBookmark) readReplyOnce() (string, error) {
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
		var entry claudeEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if entry.Type != "assistant" {
			continue
		}

		// Content can be a string or array of blocks
		var blocks []claudeContentBlock
		if err := json.Unmarshal(entry.Message.Content, &blocks); err != nil {
			// Try as plain string
			var s string
			if err := json.Unmarshal(entry.Message.Content, &s); err == nil && s != "" {
				texts = append(texts, s)
			}
			continue
		}

		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				texts = append(texts, b.Text)
			}
		}
	}

	return strings.Join(texts, "\n"), nil
}
