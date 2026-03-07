"""Claude Code helpers for agentboss.

Higher-level functions for controlling Claude Code's TUI:
- Prompt inspection and clearing
- Model picker navigation with highlight detection
- Permission mode cycling

Usage:
    from agentboss import Agent
    from claude_helpers import Claude

    agent = Agent("claude")
    claude = Claude(agent)

    # Check/clear prompt
    text = claude.prompt_text()
    claude.clear_prompt()

    # Switch model
    claude.switch_model("sonnet")
"""

import re

# ANSI escape sequence for inverse video (SGR 7), used to highlight
# the selected item in Ink-based TUI pickers.
_INVERSE_RE = re.compile(r"\x1b\[[\d;]*7[\d;]*m")

# Known model entries in Claude Code's model picker.
MODELS = {
    "opus": "Opus",
    "sonnet": "Sonnet",
    "haiku": "Haiku",
}

# Order of models in the picker (top to bottom).
MODEL_ORDER = ["opus", "sonnet", "haiku"]


class Claude:
    def __init__(self, agent):
        self.agent = agent

    # -- Prompt inspection --

    def prompt_text(self):
        """Extract text currently typed in the Claude Code input prompt.

        Returns the text after the `❯` prompt character, or empty string
        if the prompt is empty or not visible.
        """
        content = self.agent.output()
        for line in content.split("\n"):
            stripped = line.strip()
            if stripped.startswith("\u276f"):  # ❯
                return stripped[1:].strip()
        return ""

    def has_prompt_text(self):
        """Check if there is text in the input prompt."""
        return bool(self.prompt_text())

    def clear_prompt(self):
        """Clear the current input prompt text.

        Sends Ctrl+U (Unix line-kill) which clears the line in most
        line editors including Claude Code's Ink-based input.
        After clearing, waits for the prompt to reflect the change.
        """
        if not self.has_prompt_text():
            return
        self.agent.send_keys("C-u", expect_change=True)

    # -- Model picker --

    def highlighted_model(self):
        """Detect which model is currently highlighted in the model picker.

        Uses ANSI escape sequences from tmux capture-pane to find the
        line rendered with inverse video (the cursor/highlight indicator).

        Returns the plain-text content of the highlighted line, or None
        if no highlighted line is detected.
        """
        raw = self.agent.output(escapes=True)
        for line in raw.split("\n"):
            if _INVERSE_RE.search(line):
                # Strip ANSI codes to get plain text
                clean = re.sub(r"\x1b\[[0-9;]*m", "", line).strip()
                if clean:
                    return clean
        return None

    def highlighted_model_key(self):
        """Return the model key (opus/sonnet/haiku) of the highlighted model.

        Returns None if the highlight can't be detected or doesn't match
        a known model.
        """
        text = self.highlighted_model()
        if not text:
            return None
        for key, name in MODELS.items():
            if name in text:
                return key
        return None

    def switch_model(self, target):
        """Switch to a target model via the /model picker.

        Args:
            target: Model key — "opus", "sonnet", or "haiku".

        Opens the model picker, detects the currently highlighted model,
        navigates to the target with arrow keys (verifying each step),
        and selects it.
        """
        if target not in MODEL_ORDER:
            raise ValueError(f"unknown model {target!r}, expected one of {MODEL_ORDER}")

        # Open the model picker and wait for it to render
        self.agent.send("/model", expect="|".join(MODELS.values()))

        # Detect current highlight position
        current = self.highlighted_model_key()
        if current is None:
            # Fallback: assume cursor starts on the first item
            current = MODEL_ORDER[0]

        current_idx = MODEL_ORDER.index(current)
        target_idx = MODEL_ORDER.index(target)
        delta = target_idx - current_idx

        # Navigate with arrow keys, verifying highlight moves each step
        if delta > 0:
            for _ in range(delta):
                self.agent.send_keys("Down", expect_change=True)
        elif delta < 0:
            for _ in range(-delta):
                self.agent.send_keys("Up", expect_change=True)

        # Verify we're on the right item before selecting
        actual = self.highlighted_model_key()
        if actual and actual != target:
            raise RuntimeError(
                f"expected highlight on {target}, but got {actual}"
            )

        # Select and wait for picker to close
        self.agent.send_keys("Enter", expect_state="idle")

    # -- Permission mode --

    def cycle_permission_mode(self):
        """Cycle to the next permission mode (Shift+Tab).

        Returns the new mode string from the status bar.
        """
        content = self.agent.send_keys(
            "BTab",
            expect=r"accept edits on|plan mode on|bypass permissions on",
        )
        # Extract the mode from the matched content
        for mode in ["bypass permissions on", "accept edits on", "plan mode on"]:
            if mode in content:
                return mode.replace(" on", "")
        return None

    def set_permission_mode(self, target):
        """Cycle permission modes until the target mode is active.

        Args:
            target: "bypass permissions", "accept edits", or "plan mode"
        """
        for _ in range(4):  # at most 4 cycles to get back
            content = self.agent.output()
            if f"{target} on" in content:
                return
            self.cycle_permission_mode()
        raise RuntimeError(f"could not reach permission mode {target!r}")
