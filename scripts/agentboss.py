"""agentboss Python helper — thin wrapper around the agentboss CLI.

Usage:
    from agentboss import Agent

    agent = Agent("claude")  # or Agent() to use AGENTBOSS_KEY env var
    agent.send("/model", expect=r"Sonnet|Haiku|Opus")
    agent.send_keys("Down", expect=r"> .*Sonnet")
    agent.send_keys("Enter", expect_state="idle")
"""

import json
import os
import re
import subprocess
import sys


class ExpectTimeout(Exception):
    pass


class Agent:
    def __init__(self, key=None):
        self.key = key or os.environ["AGENTBOSS_KEY"]

    def send(self, text, expect=None, expect_state=None, expect_change=False, timeout=None):
        """Send text + Enter. Optionally block until expect condition is met.

        Returns the pane content at match time (for pattern/change), or empty string.
        """
        cmd = ["agentboss", "send", self.key, text]
        cmd += self._expect_flags(expect, expect_state, expect_change, timeout)
        return self._run(cmd)

    def send_keys(self, *keys, expect=None, expect_state=None, expect_change=False, timeout=None):
        """Send raw tmux keys. Optionally block until expect condition is met."""
        cmd = ["agentboss", "send", self.key, "--keys", *keys]
        cmd += self._expect_flags(expect, expect_state, expect_change, timeout)
        return self._run(cmd)

    def expect(self, pattern=None, state=None, change=False, timeout=None):
        """Wait without sending. Block until condition is met."""
        cmd = ["agentboss", "expect", self.key]
        if pattern:
            cmd.append(pattern)
        if state:
            cmd += ["--state", state]
        if change:
            cmd += ["--change"]
        if timeout:
            cmd += ["--timeout", timeout]
        return self._run(cmd)

    def output(self, lines=50, escapes=False):
        """Capture current pane content."""
        cmd = ["agentboss", "output", self.key, "--lines", str(lines)]
        if escapes:
            cmd += ["--escapes"]
        return self._run(cmd)

    def events(self):
        """Yield parsed state-change events from stdin.

        For use with `agentboss drive`.
        """
        for line in sys.stdin:
            line = line.strip()
            if line:
                yield json.loads(line)

    def _expect_flags(self, expect, expect_state, expect_change, timeout):
        flags = []
        if expect:
            flags += ["--expect", expect]
        if expect_state:
            flags += ["--expect-state", expect_state]
        if expect_change:
            flags += ["--expect-change"]
        if timeout:
            flags += ["--timeout", str(timeout)]
        return flags

    def _run(self, cmd):
        r = subprocess.run(cmd, capture_output=True, text=True)
        if r.returncode != 0:
            stderr = r.stderr.strip()
            if "timed out" in stderr:
                raise ExpectTimeout(stderr)
            raise RuntimeError(f"agentboss error: {stderr}")
        return r.stdout
