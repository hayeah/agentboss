# Driving OpenAI Codex CLI with agentboss

Reference for automating [OpenAI Codex CLI](https://github.com/openai/codex) through agentboss.

## Spawning

```bash
agentboss run --key codex --detector codex -- \
  pnpm dlx @openai/codex --dangerously-bypass-approvals-and-sandbox
```

The `--dangerously-bypass-approvals-and-sandbox` flag is Codex's "full auto" mode. Without it, Codex will prompt for approval on file edits and commands.

## Sending Messages

```bash
# Send a prompt and return immediately
agentboss send codex "implement a login page"

# Send a prompt and wait for the response
agentboss send codex "fix the auth bug" --wait

# Send with a longer timeout (default 60s)
agentboss send codex "refactor the entire codebase" --wait --timeout 120
```

Text is sent via `tmux send-keys -l` (literal mode), then `Enter` is sent after a 100ms delay. This works reliably with Codex's Ink-based TUI.

## Slash Commands

Codex supports slash commands typed at the prompt. Send them as regular text:

```bash
# Start a new chat (clears history)
agentboss send codex "/new"

# Toggle fast inference mode
agentboss send codex "/fast"

# Open review mode (shows preset picker)
agentboss send codex "/review"
```

## Model Switching

`/model` opens an interactive picker that requires arrow key navigation:

```bash
# Open model picker
agentboss send codex "/model"

# Select a model (e.g. move down one, press Enter)
agentboss send codex --keys Down Enter

# Then select reasoning effort (e.g. High = down once from Medium default)
agentboss send codex --keys Down Enter
```

The model picker shows entries like:

```
  1. gpt-5.3-codex (current)  Latest frontier agentic coding model.
› 2. gpt-5.4                  Latest frontier agentic coding model.
  3. gpt-5.2-codex            Frontier agentic coding model.
```

After selecting a model, a second picker asks for reasoning effort (Low / Medium / High / Extra high).

## Permissions

```bash
# Open permissions picker
agentboss send codex "/permissions"

# Select "Full Access" (down once from Default)
agentboss send codex --keys Down Enter
```

## Interrupting

```bash
# Ctrl+C to interrupt a running task
agentboss send codex --keys C-c
```

Codex prints "Conversation interrupted" and returns to the prompt.

## File References

Use `@filename` in prompts to reference specific files:

```bash
agentboss send codex "summarize @README.md" --wait
```

## Dismissing Menus

Any menu or picker can be dismissed with Escape:

```bash
agentboss send codex --keys Escape
```

## Detector Details

The codex detector (`detectors/codex.py`) recognizes these states:

| State | Indicator |
|---|---|
| `working` | `(esc to interrupt)` in last 20 lines |
| `working` | `Thinking...`, `Working...`, spinner characters |
| `waiting` | `[y/N]`, `[Y/n]`, `approve`, `Allow` |
| `idle` | `›` (U+203A) prompt character |
| `unknown` | None of the above |

**Quirk**: Codex uses `›` (single right-pointing angle quotation mark, U+203A) as its prompt character, not `>` (greater-than sign). The detector checks for this specifically.

## Quirks and Gotchas

- **Fast responses**: Codex can process simple prompts in under 500ms. The `--wait` flag handles this by comparing output content hashes rather than requiring a working→idle state transition.
- **Screen redraws**: Codex redraws the entire terminal, so the `›` prompt line is always visible at the bottom — even while working. The detector checks for `(esc to interrupt)` first, which takes priority over the idle prompt pattern.
- **`/model` and `/permissions` are multi-step**: They open pickers that require arrow key navigation. These don't change detector state (the screen stays "idle" since it's waiting for user input, not processing).
- **`/review` is multi-step**: Opens a preset picker first (PR style, uncommitted changes, etc.), then processes after you select one.
- **`paste-buffer` doesn't work reliably**: Codex's Ink TUI doesn't handle tmux paste-buffer correctly for submitting prompts. agentboss uses `send-keys -l` (literal text) instead.
