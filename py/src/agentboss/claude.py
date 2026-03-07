"""Claude Code helpers for agentboss.

Higher-level functions for controlling Claude Code's TUI:
- Prompt inspection and clearing
- Model picker navigation with highlight detection
- Permission mode cycling
"""

from __future__ import annotations

import re

from agentboss.agent import Agent

# ANSI escape sequence for inverse video (SGR 7), used to highlight
# the selected item in Ink-based TUI pickers.
# Match SGR parameter 7 as a standalone value, not as part of multi-digit
# numbers like 17 or 177.
_INVERSE_RE = re.compile(r"\x1b\[(?:\d+;)*7(?:;\d+)*m")

# Known model entries in Claude Code's model picker.
MODELS: dict[str, str] = {
    "opus": "Opus",
    "sonnet": "Sonnet",
    "haiku": "Haiku",
}

# Order of models in the picker (top to bottom).
MODEL_ORDER = ["opus", "sonnet", "haiku"]


class Claude:
    def __init__(self, agent: Agent):
        self.agent = agent

    # -- Prompt inspection --

    def prompt_text(self) -> str:
        """Extract text currently typed in the Claude Code input prompt.

        Returns the text after the `❯` prompt character, or empty string
        if the prompt is empty or not visible.
        """
        content = self.agent.output()
        return parse_prompt_text(content)

    def has_prompt_text(self) -> bool:
        """Check if there is text in the input prompt."""
        return bool(self.prompt_text())

    def clear_prompt(self) -> None:
        """Clear the current input prompt text.

        Sends Ctrl+U (Unix line-kill) which clears the line in most
        line editors including Claude Code's Ink-based input.
        After clearing, waits for the prompt to reflect the change.
        """
        if not self.has_prompt_text():
            return
        self.agent.send_keys("C-u", expect_change=True)

    # -- Model picker --

    def highlighted_model(self) -> str | None:
        """Detect which model is currently highlighted in the model picker.

        Uses ANSI escape sequences from tmux capture-pane to find the
        line rendered with inverse video (the cursor/highlight indicator).

        Returns the plain-text content of the highlighted line, or None
        if no highlighted line is detected.
        """
        raw = self.agent.output(escapes=True)
        return parse_highlighted_line(raw)

    def highlighted_model_key(self) -> str | None:
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

    def switch_model(self, target: str) -> None:
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
            raise RuntimeError(f"expected highlight on {target}, but got {actual}")

        # Select and wait for picker to close
        self.agent.send_keys("Enter", expect_state="idle")

    # -- Permission mode --

    def cycle_permission_mode(self) -> str | None:
        """Cycle to the next permission mode (Shift+Tab).

        Returns the new mode string from the status bar, or "default" for
        the unnamed default mode (shown as "? for shortcuts").
        """
        content = self.agent.send_keys("BTab", expect_change=True)
        for mode in ["bypass permissions on", "accept edits on", "plan mode on"]:
            if mode in content:
                return mode.replace(" on", "")
        return "default"

    def set_permission_mode(self, target: str) -> None:
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


# -- Pure functions for testing --


def parse_prompt_text(content: str) -> str:
    """Extract text after the ❯ prompt character from the active prompt.

    Scans from the bottom of the pane content to find the actual input prompt
    (not earlier ❯ lines from conversation history).
    """
    lines = content.split("\n")
    for line in reversed(lines):
        stripped = line.strip()
        if stripped.startswith("\u276f"):  # ❯
            return stripped[1:].strip()
    return ""


def parse_highlighted_line(raw: str) -> str | None:
    """Find the highlighted line in the model picker and return its plain text.

    Detection methods (tried in order):
    - ANSI inverse video (SGR 7) — older Claude Code versions
    - ❯ cursor prefix — current Claude Code model picker uses ❯ to mark
      the selected item (e.g. "❯ 1. Default (recommended) ✔")
    """
    for line in raw.split("\n"):
        if _INVERSE_RE.search(line):
            clean = re.sub(r"\x1b\[[0-9;]*m", "", line).strip()
            if clean:
                return clean

    # Fallback: look for ❯ cursor prefix in picker lines (not the input prompt).
    # Picker lines contain numbered items like "❯ 1. Default..." while the
    # input prompt is just "❯" or "❯ <user text>" on its own line.
    for line in raw.split("\n"):
        clean = re.sub(r"\x1b\[[0-9;]*m", "", line).strip()
        if clean.startswith("\u276f") and re.search(r"\d+\.", clean):
            # Strip the ❯ prefix and number
            text = re.sub(r"^\u276f\s*\d+\.\s*", "", clean).strip()
            if text:
                return text
    return None
