"""Core Agent class — thin wrapper around the agentboss CLI."""

from __future__ import annotations

import json
import os
import subprocess
import sys
from dataclasses import dataclass


class ExpectTimeout(Exception):
    pass


@dataclass
class Event:
    state: str
    prev_state: str
    timestamp: float

    @property
    def state_changed(self) -> bool:
        return self.state != self.prev_state


class Agent:
    """Wraps a running agentboss process for sending input and waiting on output.

    Args:
        key: Process key or hash prefix. Defaults to AGENTBOSS_KEY env var.
    """

    def __init__(self, key: str | None = None):
        self.key = key or os.environ["AGENTBOSS_KEY"]

    def send(
        self,
        text: str,
        *,
        expect: str | None = None,
        expect_state: str | None = None,
        expect_change: bool = False,
        timeout: int | None = None,
    ) -> str:
        """Send text + Enter. Optionally block until expect condition is met.

        Returns the pane content at match time (for pattern/change), or empty string.
        """
        cmd = ["agentboss", "send", self.key, text]
        cmd += _expect_flags(expect, expect_state, expect_change, timeout)
        return _run(cmd)

    def send_keys(
        self,
        *keys: str,
        expect: str | None = None,
        expect_state: str | None = None,
        expect_change: bool = False,
        timeout: int | None = None,
    ) -> str:
        """Send raw tmux keys. Optionally block until expect condition is met."""
        cmd = ["agentboss", "send", self.key, "--keys", *keys]
        cmd += _expect_flags(expect, expect_state, expect_change, timeout)
        return _run(cmd)

    def expect(
        self,
        pattern: str | None = None,
        *,
        state: str | None = None,
        change: bool = False,
        timeout: int | None = None,
    ) -> str:
        """Wait without sending. Block until condition is met."""
        cmd = ["agentboss", "expect", self.key]
        if pattern:
            cmd.append(pattern)
        if state:
            cmd += ["--state", state]
        if change:
            cmd += ["--change"]
        if timeout:
            cmd += ["--timeout", f"{timeout}s"]
        return _run(cmd)

    def output(self, lines: int = 50, *, escapes: bool = False) -> str:
        """Capture current pane content."""
        cmd = ["agentboss", "output", self.key, "--lines", str(lines)]
        if escapes:
            cmd += ["--escapes"]
        return _run(cmd)

    def events(self) -> "EventIterator":
        """Yield parsed state-change events from stdin.

        For use with `agentboss drive`.
        """
        return EventIterator()


class EventIterator:
    """Iterator over JSON-line events from stdin."""

    def __iter__(self) -> EventIterator:
        return self

    def __next__(self) -> Event:
        for line in sys.stdin:
            line = line.strip()
            if line:
                data = json.loads(line)
                return Event(
                    state=data["state"],
                    prev_state=data.get("prev_state", ""),
                    timestamp=data.get("timestamp", 0.0),
                )
        raise StopIteration


def _expect_flags(
    expect: str | None,
    expect_state: str | None,
    expect_change: bool,
    timeout: int | None,
) -> list[str]:
    flags: list[str] = []
    if expect:
        flags += ["--expect", expect]
    if expect_state:
        flags += ["--expect-state", expect_state]
    if expect_change:
        flags += ["--expect-change"]
    if timeout:
        flags += ["--timeout", str(timeout)]
    return flags


def _run(cmd: list[str]) -> str:
    r = subprocess.run(cmd, capture_output=True, text=True)
    if r.returncode != 0:
        stderr = r.stderr.strip()
        if "timed out" in stderr:
            raise ExpectTimeout(stderr)
        raise RuntimeError(f"agentboss error: {stderr}")
    return r.stdout
