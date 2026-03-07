# Driving Claude Code with agentboss

Reference for automating [Claude Code](https://docs.anthropic.com/en/docs/claude-code) through agentboss.

## Spawning

```bash
agentboss run --key claude --detector claude -- \
  claude --dangerously-skip-permissions
```

Permission mode flags:

| Flag | Mode |
|---|---|
| `--dangerously-skip-permissions` | Full bypass (no prompts at all) |
| `--permission-mode acceptEdits` | Auto-accept file edits, prompt for commands |
| `--permission-mode plan` | Read-only, no edits or commands |
| (none) | Default interactive mode |

## Sending Messages

```bash
# Send a prompt and return immediately
agentboss send claude "fix the auth bug"

# Send a prompt and wait for the response
agentboss send claude "fix the auth bug" --wait

# Send with a longer timeout (default 60s)
agentboss send claude "refactor the entire codebase" --wait --timeout 300
```

Text is sent via `tmux send-keys -l` (literal mode), then `Enter` is sent after a 100ms delay.

## Expect: Atomic Send + Verify

Every `send` can include an expect condition. Go sends the input, then polls tmux at 75ms intervals until the condition is met:

```bash
# Send text, wait for pattern to appear in pane
agentboss send claude "/model" --expect "Sonnet|Haiku|Opus"

# Send keys, wait for pattern
agentboss send claude --keys Down --expect "Sonnet"

# Send text, wait for detector state
agentboss send claude "fix the auth bug" --expect-state working

# Send keys, wait for state
agentboss send claude --keys Enter --expect-state idle

# Send text, wait for any content change
agentboss send claude "hello" --expect-change
```

Standalone expect (wait without sending):

```bash
agentboss expect claude "Session ID:"
agentboss expect claude --state idle --timeout 60s
agentboss expect claude --change
```

Default timeouts: 5s for `--expect` (pattern/change), 60s for `--expect-state`.

### Pane output with ANSI escapes

```bash
# Plain text (default)
agentboss output claude

# Include ANSI escape sequences (for highlight detection)
agentboss output claude --escapes
```

## Slash Commands

Claude Code supports slash commands at the prompt. Send them as regular text:

```bash
# Clear conversation history
agentboss send claude "/clear"

# Switch model (opens picker)
agentboss send claude "/model"

# Toggle fast mode (requires native binary, not bunx)
agentboss send claude "/fast"
```

**Important**: Claude Code has autocomplete on `/` commands. When sending slash commands via agentboss, the literal `send-keys -l` approach works correctly because it types the full command before Enter.

## Model Switching

`/model` opens an interactive picker:

```bash
# Open model picker
agentboss send claude "/model"

# Navigate and select (e.g. move to Sonnet)
agentboss send claude --keys Down Enter
```

With expect (no sleeps, verifies each step):

```bash
# Open picker, wait for it to render
agentboss send claude "/model" --expect "Sonnet|Haiku|Opus"

# Navigate down, wait for content to update
agentboss send claude --keys Down --expect-change

# Select and wait for picker to close
agentboss send claude --keys Enter --expect-state idle
```

The model picker shows entries like:

```
  Default (Opus 4.6 · Most capable for complex work)
✔ Sonnet (claude-sonnet-4-6)
  Haiku (claude-haiku-4-5-20251001)
```

The cursor starts on the currently selected item (marked with checkmark). Navigation is relative to current position. The highlighted item is rendered with ANSI inverse video — use `agentboss output claude --escapes` to detect it.

## Permission Mode Cycling

Shift+Tab cycles through permission modes at the prompt:

```bash
# Cycle to next permission mode
agentboss send claude --keys BTab
```

The cycle order (shown in the status bar):

```
bypass permissions → (default/no indicator) → accept edits → plan mode → bypass permissions
```

The current mode appears at the bottom of the terminal:

```
⏵⏵ bypass permissions on (shift+tab to cycle)
⏵⏵ accept edits on (shift+tab to cycle)
⏸ plan mode on (shift+tab to cycle)
? for shortcuts
```

## Interrupting

```bash
# Press Escape to interrupt a running task
agentboss send claude --keys Escape
```

Claude prints "Interrupted - What should Claude do instead?" and returns to the idle prompt. The follow-up prompt expects a redirect instruction, but you can also just send a new prompt.

## Shell Commands

Prefix with `!` to run a shell command:

```bash
agentboss send claude "! echo hello from shell" --wait
```

## Status and Config Overlays

`/status` opens a modal overlay that does **not** return to idle:

```bash
# Open status overlay
agentboss send claude "/status"

# Dismiss overlay
agentboss send claude --keys Escape
```

Similarly, `/help` opens a modal dialog that must be dismissed with Escape.

**Do not use `--wait` with overlay commands** — they don't transition back to the idle prompt automatically.

## Clearing History

```bash
# Clear conversation and start fresh
agentboss send claude "/clear" --wait
```

## Resuming Sessions

```bash
# Resume the last conversation
agentboss run --key claude --detector claude -- \
  claude --continue --dangerously-skip-permissions
```

## Detector Details

The claude detector (`detectors/claude.py`) recognizes these states:

| State | Indicator |
|---|---|
| `working` | `(esc to interrupt)` in last 20 lines |
| `waiting` | `[Y/n]`, `[y/N]`, `Allow`, `Deny`, `Do you trust` |
| `idle` | `❯` on its own line (between divider lines) |
| `idle` | Line ending with `>` or `❯` |
| `unknown` | None of the above |

**Key pattern**: Claude Code renders its prompt as `❯` on a standalone line between two horizontal divider lines (`───...───`). The detector uses `re.search(r'^❯\s*$', bottom, re.MULTILINE)` to match this layout.

### Cross-reference with dmux

dmux's approach to Claude detection (from `PaneWorker.ts`):

- **Working**: Same `(esc to interrupt)` universal pattern. dmux also checks for Claude-specific activity words (`Germinating`, `Thinking`, `Planning`, etc.) combined with the interrupt indicator, but the universal pattern alone is sufficient.
- **Trust prompts**: dmux has `autoApproveTrustPrompt()` that polls for "Do you trust the files" patterns and auto-sends Enter. With `--dangerously-skip-permissions`, this prompt doesn't appear.
- **LLM fallback**: dmux uses a two-tier system — deterministic patterns first, then LLM classification for ambiguous states. agentboss uses `unknown` state to signal the caller's AI loop should classify.

## Python Helpers

`scripts/agentboss.py` provides an `Agent` class wrapping the CLI. `scripts/claude_helpers.py` adds Claude-specific helpers.

### Agent class

```python
from agentboss import Agent

agent = Agent("claude")  # or Agent() to use AGENTBOSS_KEY env var

# Send with expect (atomic action + verify)
agent.send("/model", expect=r"Sonnet|Haiku|Opus")
agent.send_keys("Down", expect_change=True)
agent.send_keys("Enter", expect_state="idle")

# Send fire-and-forget
agent.send("fix the auth bug")

# Wait without sending
agent.expect(pattern=r"Session ID:")
agent.expect(state="idle", timeout="60s")
agent.expect(change=True)

# Read pane content
content = agent.output()
raw = agent.output(escapes=True)  # with ANSI codes
```

### Claude helpers

```python
from agentboss import Agent
from claude_helpers import Claude

agent = Agent("claude")
claude = Claude(agent)

# Prompt inspection
text = claude.prompt_text()      # text after the ❯ prompt
claude.has_prompt_text()          # True if prompt has text
claude.clear_prompt()             # Ctrl+U, verifies with expect-change

# Model switching (detects highlight, navigates, verifies each step)
claude.switch_model("sonnet")     # "opus", "sonnet", or "haiku"
claude.switch_model("haiku")

# Highlight detection (uses ANSI inverse video from --escapes output)
claude.highlighted_model()        # "Sonnet (claude-sonnet-4-6)" or None
claude.highlighted_model_key()    # "sonnet" or None

# Permission modes
claude.cycle_permission_mode()           # Shift+Tab, returns new mode
claude.set_permission_mode("plan mode")  # cycles until target is active
```

### Using in scripts

The helpers are standalone Python files with no dependencies. Copy them or add to `PYTHONPATH`:

```bash
export PYTHONPATH=/path/to/agentboss/scripts
export AGENTBOSS_KEY=claude
python3 my_driver.py
```

Example driver script:

```python
#!/usr/bin/env python3
from agentboss import Agent
from claude_helpers import Claude

agent = Agent()
claude = Claude(agent)

# Switch to Haiku for a quick task
claude.switch_model("haiku")
agent.send("fix the typo in README.md", expect_state="working")
agent.expect(state="idle", timeout="120s")

# Switch to Opus for a complex task
claude.switch_model("opus")
agent.send("redesign the auth architecture", expect_state="working")
agent.expect(state="idle", timeout="300s")
```

## Quirks and Gotchas

- **Overlay commands block idle detection**: `/status`, `/help`, and similar commands open modal overlays. The detector will report `unknown` (not `idle`) until you dismiss them with Escape.
- **Autocomplete on `/`**: Claude Code shows autocomplete suggestions when you type `/`. With `send-keys -l`, the full command is typed before pressing Enter, so autocomplete doesn't interfere.
- **`/fast` requires native binary**: When running via `bunx`, `/fast` may not work. Install Claude Code natively (`npm i -g @anthropic-ai/claude-code`) for full feature support.
- **Trust prompt on first launch**: When running without `--dangerously-skip-permissions` in a new workspace, Claude may show a "Do you trust the files in this folder?" prompt. The detector catches this as `waiting` state. Send Enter or `y` + Enter to approve.
- **`❯` prompt layout**: The idle prompt is rendered as a standalone `❯` between two full-width divider lines. This is different from a trailing `>` — the detector checks for both patterns.
- **MCP connector notices**: The status bar may show notices like "2 claude.ai connectors need auth - /mcp". These are informational and don't affect state detection.
