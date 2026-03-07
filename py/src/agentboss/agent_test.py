"""Tests for Agent helper functions."""

from agentboss.agent import Event, _expect_flags


class TestExpectFlags:
    def test_no_flags(self):
        assert _expect_flags(None, None, False, None) == []

    def test_pattern(self):
        assert _expect_flags("foo.*bar", None, False, None) == ["--expect", "foo.*bar"]

    def test_state(self):
        assert _expect_flags(None, "idle", False, None) == ["--expect-state", "idle"]

    def test_change(self):
        assert _expect_flags(None, None, True, None) == ["--expect-change"]

    def test_timeout(self):
        assert _expect_flags(None, None, False, 30) == ["--timeout", "30"]

    def test_combined(self):
        flags = _expect_flags("pattern", "idle", True, 10)
        assert "--expect" in flags
        assert "--expect-state" in flags
        assert "--expect-change" in flags
        assert "--timeout" in flags


class TestEvent:
    def test_state_changed(self):
        e = Event(state="idle", prev_state="working", timestamp=0.0)
        assert e.state_changed is True

    def test_state_not_changed(self):
        e = Event(state="idle", prev_state="idle", timestamp=0.0)
        assert e.state_changed is False

    def test_first_event(self):
        e = Event(state="idle", prev_state="", timestamp=0.0)
        assert e.state_changed is True
