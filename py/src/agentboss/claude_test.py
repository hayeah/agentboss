"""Tests for Claude helper pure functions."""

import pytest

from agentboss.claude import parse_highlighted_line, parse_prompt_text


class TestParsePromptText:
    def test_empty_prompt(self):
        content = """\
───────────────────────────────────────
❯
───────────────────────────────────────
⏵⏵ bypass permissions on"""
        assert parse_prompt_text(content) == ""

    def test_with_text(self):
        content = """\
───────────────────────────────────────
❯ fix the auth bug
───────────────────────────────────────
⏵⏵ bypass permissions on"""
        assert parse_prompt_text(content) == "fix the auth bug"

    def test_with_leading_spaces(self):
        content = """\
───────────────────────────────────────
  ❯ some text here
───────────────────────────────────────"""
        assert parse_prompt_text(content) == "some text here"

    def test_no_prompt(self):
        content = "some random output\nwithout a prompt"
        assert parse_prompt_text(content) == ""

    def test_prompt_only_whitespace(self):
        content = "❯   "
        assert parse_prompt_text(content) == ""


class TestParseHighlightedLine:
    def test_inverse_video(self):
        raw = (
            "  Default (Opus 4.6)\n"
            "\x1b[7m  Sonnet (claude-sonnet-4-6)\x1b[0m\n"
            "  Haiku (claude-haiku-4-5)\n"
        )
        assert parse_highlighted_line(raw) == "Sonnet (claude-sonnet-4-6)"

    def test_inverse_with_other_codes(self):
        raw = (
            "  Default (Opus 4.6)\n"
            "\x1b[1;7;34m  Haiku (claude-haiku-4-5)\x1b[0m\n"
        )
        assert parse_highlighted_line(raw) == "Haiku (claude-haiku-4-5)"

    def test_no_highlight(self):
        raw = (
            "  Default (Opus 4.6)\n"
            "  Sonnet (claude-sonnet-4-6)\n"
            "  Haiku (claude-haiku-4-5)\n"
        )
        assert parse_highlighted_line(raw) is None

    def test_empty_highlighted_line(self):
        raw = "\x1b[7m   \x1b[0m\n"
        assert parse_highlighted_line(raw) is None

    def test_sgr7_mid_sequence(self):
        # 7 appears in the middle of the SGR params
        raw = "\x1b[38;7;1mBold Inverse Text\x1b[0m\n"
        assert parse_highlighted_line(raw) == "Bold Inverse Text"

    def test_cursor_prefix_highlight(self):
        # Claude Code v2.x uses ❯ prefix for selected item in model picker
        raw = (
            "\x1b[38;2;177;185;249m❯\x1b[39m \x1b[38;2;153;153;153m1. "
            "\x1b[38;2;78;186;101mDefault (recommended) ✔\x1b[39m  "
            "\x1b[38;2;153;153;153mOpus 4.6\x1b[39m\n"
            "    \x1b[38;2;153;153;153m2.\x1b[39m Sonnet\n"
            "    \x1b[38;2;153;153;153m3.\x1b[39m Haiku\n"
        )
        result = parse_highlighted_line(raw)
        assert result is not None
        assert "Default" in result

    def test_cursor_prefix_non_default(self):
        raw = (
            "    \x1b[38;2;153;153;153m1.\x1b[39m Default\n"
            "\x1b[38;2;177;185;249m❯\x1b[39m \x1b[38;2;153;153;153m2.\x1b[39m Sonnet\n"
            "    \x1b[38;2;153;153;153m3.\x1b[39m Haiku\n"
        )
        result = parse_highlighted_line(raw)
        assert result == "Sonnet"
