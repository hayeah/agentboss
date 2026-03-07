package agentboss

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/hayeah/agentboss/conf"
)

func setupMenuTest(t *testing.T) (*Boss, *Process, func()) {
	t.Helper()

	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not found")
	}

	tmx := NewTmux()
	session := fmt.Sprintf("test-expect-%d", time.Now().UnixNano())

	absPath, err := filepath.Abs("testdata/mock_menu.py")
	if err != nil {
		t.Fatal(err)
	}

	if err := tmx.NewSession(session, ".", []string{"python3", absPath}); err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	store := NewProcessStore(conf.StateDir(tmpDir))
	detector := NewDetectorRunner(store, tmx, conf.StateDir(tmpDir))
	boss := NewBoss(store, tmx, detector)

	proc := &Process{TmuxSession: session}

	cleanup := func() {
		tmx.KillSession(session)
	}

	return boss, proc, cleanup
}

func TestExpectPattern(t *testing.T) {
	boss, proc, cleanup := setupMenuTest(t)
	defer cleanup()

	// Wait for menu to render
	content, err := boss.Expect(proc, ExpectOpts{
		Pattern: regexp.MustCompile(`READY`),
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal("menu did not show READY:", err)
	}

	// Verify initial state: cursor on Default
	if !regexp.MustCompile(`> Default`).MatchString(content) {
		t.Fatal("expected '> Default' in initial state, got:\n", content)
	}
}

func TestExpectTimeout(t *testing.T) {
	boss, proc, cleanup := setupMenuTest(t)
	defer cleanup()

	// Wait for menu to be ready first
	_, err := boss.Expect(proc, ExpectOpts{
		Pattern: regexp.MustCompile(`READY`),
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Expect a pattern that will never appear — should timeout
	_, err = boss.Expect(proc, ExpectOpts{
		Pattern: regexp.MustCompile(`NEVER_APPEARS`),
		Timeout: 500 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestSendAndExpectPattern(t *testing.T) {
	boss, proc, cleanup := setupMenuTest(t)
	defer cleanup()

	tmx := boss.tmux

	// Wait for menu ready
	_, err := boss.Expect(proc, ExpectOpts{
		Pattern: regexp.MustCompile(`READY`),
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Send Down arrow, expect cursor to move to Sonnet
	_, err = boss.SendAndExpect(proc, func() error {
		return tmx.SendKeys(proc.TmuxSession, "Down")
	}, ExpectOpts{
		Pattern: regexp.MustCompile(`> Sonnet`),
	})
	if err != nil {
		t.Fatal("expected '> Sonnet' after Down:", err)
	}
}

func TestSendAndExpectChange(t *testing.T) {
	boss, proc, cleanup := setupMenuTest(t)
	defer cleanup()

	tmx := boss.tmux

	// Wait for menu ready
	_, err := boss.Expect(proc, ExpectOpts{
		Pattern: regexp.MustCompile(`READY`),
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Send Down arrow, expect any content change
	content, err := boss.SendAndExpect(proc, func() error {
		return tmx.SendKeys(proc.TmuxSession, "Down")
	}, ExpectOpts{
		Change: true,
	})
	if err != nil {
		t.Fatal("expected content change after Down:", err)
	}

	// Verify the cursor moved
	if !regexp.MustCompile(`> Sonnet`).MatchString(content) {
		t.Fatal("expected '> Sonnet' in changed content, got:\n", content)
	}
}

func TestMenuNavigation(t *testing.T) {
	boss, proc, cleanup := setupMenuTest(t)
	defer cleanup()

	tmx := boss.tmux

	// Wait for menu ready
	_, err := boss.Expect(proc, ExpectOpts{
		Pattern: regexp.MustCompile(`READY`),
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Navigate: Down to Sonnet
	_, err = boss.SendAndExpect(proc, func() error {
		return tmx.SendKeys(proc.TmuxSession, "Down")
	}, ExpectOpts{
		Pattern: regexp.MustCompile(`> Sonnet`),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Navigate: Down to Haiku
	_, err = boss.SendAndExpect(proc, func() error {
		return tmx.SendKeys(proc.TmuxSession, "Down")
	}, ExpectOpts{
		Pattern: regexp.MustCompile(`> Haiku`),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Navigate: Down again — stays on Haiku (already at bottom).
	// The menu redraws so content changes, but cursor stays put.
	_, err = boss.SendAndExpect(proc, func() error {
		return tmx.SendKeys(proc.TmuxSession, "Down")
	}, ExpectOpts{
		Pattern: regexp.MustCompile(`> Haiku`),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Navigate: Up back to Sonnet
	_, err = boss.SendAndExpect(proc, func() error {
		return tmx.SendKeys(proc.TmuxSession, "Up")
	}, ExpectOpts{
		Pattern: regexp.MustCompile(`> Sonnet`),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Select: Enter on Sonnet
	_, err = boss.SendAndExpect(proc, func() error {
		return tmx.SendKeys(proc.TmuxSession, "Enter")
	}, ExpectOpts{
		Pattern: regexp.MustCompile(`SELECTED: Sonnet`),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestExpectChangeNoSend(t *testing.T) {
	boss, proc, cleanup := setupMenuTest(t)
	defer cleanup()

	tmx := boss.tmux

	// Wait for menu ready
	_, err := boss.Expect(proc, ExpectOpts{
		Pattern: regexp.MustCompile(`READY`),
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Start expect --change, then send a key from a goroutine
	done := make(chan error, 1)
	var result string
	go func() {
		var err error
		result, err = boss.Expect(proc, ExpectOpts{
			Change:  true,
			Timeout: 5 * time.Second,
		})
		done <- err
	}()

	// Give expect a moment to snapshot baseline
	time.Sleep(200 * time.Millisecond)
	tmx.SendKeys(proc.TmuxSession, "Down")

	if err := <-done; err != nil {
		t.Fatal("expected change detection:", err)
	}

	if !regexp.MustCompile(`> Sonnet`).MatchString(result) {
		t.Fatal("expected '> Sonnet' after change, got:\n", result)
	}
}
