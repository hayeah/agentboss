#!/usr/bin/env python3
"""Mock model picker menu for testing expect-based navigation.

Renders a menu with cursor indicator ("> ") and responds to arrow keys.
Designed to be spawned in a tmux session for e2e tests.
"""
import sys
import tty
import termios

models = [
    "Default (Opus 4.6)",
    "Sonnet (claude-sonnet-4-6)",
    "Haiku (claude-haiku-4-5)",
]
cursor = 0


def render():
    sys.stdout.write("\033[2J\033[H")
    sys.stdout.write("Select model:\n")
    for i, model in enumerate(models):
        prefix = "> " if i == cursor else "  "
        sys.stdout.write(f"{prefix}{model}\n")
    sys.stdout.write("\nREADY\n")
    sys.stdout.flush()


fd = sys.stdin.fileno()
old = termios.tcgetattr(fd)
try:
    tty.setraw(fd)
    render()
    while True:
        ch = sys.stdin.read(1)
        if ch == "\x1b":
            seq = sys.stdin.read(2)
            if seq == "[B":  # Down
                cursor = min(cursor + 1, len(models) - 1)
            elif seq == "[A":  # Up
                cursor = max(cursor - 1, 0)
            render()
        elif ch == "\r":  # Enter
            sys.stdout.write("\033[2J\033[H")
            sys.stdout.write(f"SELECTED: {models[cursor]}\n")
            sys.stdout.flush()
            # Wait for any key then exit
            sys.stdin.read(1)
            break
        elif ch == "q" or ch == "\x03":
            break
finally:
    termios.tcsetattr(fd, termios.TCSADRAIN, old)
