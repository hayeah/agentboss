---
name: agentboss
description: Generic tmux-based interactive CLI supervisor. Spawn any CLI in a tmux session, read its output, send it input, and detect its state. Use when you need to drive interactive terminal programs (Claude Code, Codex, Python REPL, etc.) from scripts or an outer AI loop.
---

# agentboss

Spawn any interactive CLI in a tmux session, read its output, send it input. No server, no daemon — every command talks directly to tmux and the filesystem.

Designed as a building block for an outer AI loop that drives terminal programs.

## Install

```bash
go install github.com/hayeah/agentboss/cli/agentboss@latest
```

Or install in editable mode:

```bash
gobin install ./cli/agentboss
```

Requires `tmux` to be installed.

## Quick Start

```bash
# Spawn a Python REPL
agentboss run --key pyrepl --detector generic -- python3 -i

# In another terminal: send a command and wait for the response
agentboss send pyrepl "print('hello')" --wait

# Read the output
agentboss output pyrepl

# Attach to see it live
agentboss attach pyrepl
```

## Commands

### `run` — Spawn a supervised CLI

```bash
agentboss run [flags] -- COMMAND [ARGS...]
```

Spawns COMMAND in a new tmux session. The `run` process stays in the foreground, holding an flock for liveness detection. When the child exits, `run` exits.

```bash
# Spawn Claude Code
agentboss run --key claude --detector claude -- claude --permission-mode acceptEdits

# Spawn Codex
agentboss run --key codex --detector codex -- pnpm dlx @openai/codex --dangerously-bypass-approvals-and-sandbox

# Spawn with explicit working directory
agentboss run --cwd ~/project -- python3 -i
```

Flags:
- `--key` — explicit key for hash identity (instead of hashing cwd+cmd)
- `--cwd` — working directory (default: current)
- `--detector` — named detector script (e.g. `claude`, `codex`, `generic`)

Idempotent: if the same hash is already running, prints info and exits.

### `output` — Read terminal content

```bash
agentboss output HASH [-n LINES]
```

Captures the last N lines (default 50) from the tmux pane via `tmux capture-pane`.

```bash
# Read last 50 lines
agentboss output claude

# Read last 100 lines
agentboss output claude -n 100
```

### `send` — Send input

```bash
agentboss send HASH TEXT [--wait] [--timeout SEC]
agentboss send HASH --keys KEY...
```

Two modes:

**Text mode** (default): sends literal text via `tmux send-keys -l`, then presses Enter.

```bash
# Send a prompt
agentboss send claude "fix the auth bug"

# Send and wait for the agent to finish
agentboss send claude "fix the auth bug" --wait

# Send without pressing Enter
agentboss send claude --no-enter "partial input"
```

**Keys mode**: sends raw tmux key names via `tmux send-keys`.

```bash
# Press Escape twice
agentboss send claude --keys Escape Escape

# Ctrl+C to interrupt
agentboss send claude --keys C-c

# Navigate a menu
agentboss send claude --keys Down Down Enter
```

The `--wait` flag blocks until the agent returns to idle after processing. It works by snapshotting the terminal content right after sending, then polling until both (a) the content has changed and (b) the detector reports `idle`.

### `status` — Check state

```bash
agentboss status HASH [-q]
```

Runs the detector script against the current pane content and reports the state.

```bash
# Full JSON output
agentboss status claude

# Just the state string (for scripting)
agentboss status claude -q
# → idle
```

### `wait` — Block until idle

```bash
agentboss wait HASH [--timeout SEC]
```

Polls the detector every second until it reports `idle`. Times out with exit code 1.

```bash
agentboss wait claude --timeout 120
```

### `ls` — List all processes

```bash
agentboss ls
```

Lists all processes from `~/.agentboss/`, with liveness (via flock) and detector state.

### `attach` — Attach to tmux session

```bash
agentboss attach HASH
```

Replaces the current process with `tmux attach-session`. Use this to observe or manually interact with the supervised CLI.

## Process Identity

Each instance is identified by a 10-char SHA-256 hash of either:
- The `--key` value (if provided), or
- The cwd + command joined with null bytes

A shortest-unique-prefix (`hashid`) is computed among all existing processes (minimum 3 chars). Commands accept hash prefixes or keys to identify targets:

```bash
agentboss output a3f        # by hash prefix
agentboss output claude     # by key
```

## State Directory

```
~/.agentboss/<hash>/
  state.json          # process metadata (hash, key, cwd, cmd, etc.)
  lock                # flock held by the run process for liveness
  detect.py           # per-instance detector (optional, overrides named detector)
```

## State Detection

Detectors are Python scripts that read terminal content from stdin and output a JSON state to stdout:

```python
#!/usr/bin/env python3
import sys, json, re

content = sys.stdin.read()
lines = content.strip().split('\n')
recent = '\n'.join(lines[-20:])

if re.search(r'\(esc\s+to\s+interrupt', recent, re.IGNORECASE):
    state = 'working'
elif recent.strip().endswith('>'):
    state = 'idle'
else:
    state = 'unknown'

json.dump({"state": state}, sys.stdout)
```

### Detector Resolution Order

- `~/.agentboss/<hash>/detect.py` — per-instance override
- `~/.agentboss/detectors/<name>.py` — named detector (from `--detector` flag)
- No detector → state is always `unknown`

### Built-in Detectors

Copy these to `~/.agentboss/detectors/` to use them:

```bash
cp detectors/*.py ~/.agentboss/detectors/
```

| Detector | CLI | Key Patterns |
|---|---|---|
| `claude.py` | Claude Code | `(esc to interrupt)` → working; `[Y/n]`, `Allow` → waiting; `│ >`, `❯` → idle |
| `codex.py` | OpenAI Codex | `(esc to interrupt)` → working; `Thinking...` → working; `›` → idle |
| `generic.py` | Any CLI | `(esc to interrupt)` → working; `$`, `>`, `❯`, `>>>` → idle |

### Writing Custom Detectors

A detector script must:
- Read terminal content from **stdin**
- Output `{"state": "STATE"}` to **stdout**
- Use one of: `idle`, `working`, `waiting`, `unknown`

The script receives the last 50 lines of terminal content. Focus pattern matching on the last 10-20 lines where the current state appears.

The `unknown` state signals the caller (outer AI loop) should use its own LLM to classify. The detector is a **deterministic fast-path** — the LLM only gets involved for genuinely ambiguous states.

## Outer AI Loop Example

```python
import subprocess, json

def boss(cmd, *args):
    result = subprocess.run(["agentboss", cmd, *args], capture_output=True, text=True)
    return result.stdout.strip()

# Spawn Claude
subprocess.Popen(["agentboss", "run", "--key", "claude", "--detector", "claude",
                   "--", "claude", "--permission-mode", "acceptEdits"])

# Send task and wait
subprocess.run(["agentboss", "send", "claude", "fix the auth bug", "--wait"])

# Read result
output = boss("output", "claude", "-n", "30")
state = boss("status", "claude", "-q")
print(f"State: {state}")
print(output)
```

## Agent-Specific Guides

- [CLAUDE.md](CLAUDE.md) — driving Claude Code (model switching, permission modes, overlays, quirks)
- [CODEX.md](CODEX.md) — driving OpenAI Codex CLI (model switching, slash commands, quirks)

## Architecture

```
agentboss/
  process.go       # Process struct, hash, store, flock liveness
  boss.go          # Supervisor: spawn tmux, hold flock, wait for exit
  tmux.go          # Tmux wrappers (capture-pane, send-keys, send-text)
  detect.go        # Detector script runner
  wire.go          # google/wire provider set
  conf/conf.go     # Config types (StateDir)
  cli/agentboss/   # Cobra CLI subcommands + wire injector
  detectors/       # Built-in detector scripts (Python)
```

No server, no daemon, no IPC. Every command resolves a hash prefix → reads `state.json` → calls tmux or runs a detector script directly.

Dependency injection via [google/wire](https://github.com/google/wire). Liveness via [flock](https://github.com/gofrs/flock) — the kernel releases the lock automatically on crash.
