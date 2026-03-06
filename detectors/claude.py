#!/usr/bin/env python3
"""State detector for Claude Code.

Based on dmux's deterministic detection patterns.
Focus on the last 20 lines where the current state appears.
"""
import sys
import json
import re

content = sys.stdin.read()
lines = content.strip().split('\n')
recent = '\n'.join(lines[-20:]) if lines else ''
bottom = '\n'.join(lines[-10:]) if lines else ''

# Working: "(esc to interrupt)" is the most reliable indicator.
# It ONLY appears when Claude is actively processing.
if re.search(r'\(esc\s+to\s+interrupt', recent, re.IGNORECASE):
    state = 'working'

# Waiting: option dialogs, permission prompts, confirmation prompts.
# Must have clear choices presented.
elif any(p in bottom for p in [
    '[Y/n]', '[y/N]', '[A]ccept', '[R]eject',
    'Allow', 'Deny',
    'Do you trust', 'Continue?',
]):
    state = 'waiting'

# Idle: open prompt. Claude Code shows "❯" on its own line between
# two divider lines, with a status bar below. Check for the prompt
# character anywhere in the bottom lines, not just at the very end.
elif re.search(r'^❯\s*$', bottom, re.MULTILINE):
    state = 'idle'
# Also check for the │ > pattern (Claude's bordered prompt, older versions).
elif re.search(r'[│|]\s*>\s*$', bottom, re.MULTILINE):
    state = 'idle'
elif bottom.strip().endswith('>') or bottom.strip().endswith('❯'):
    state = 'idle'

else:
    state = 'unknown'

json.dump({"state": state}, sys.stdout)
