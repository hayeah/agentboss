#!/usr/bin/env python3
"""State detector for OpenAI Codex CLI.

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

# Working: "(esc to interrupt)" is universal across agents.
# Also check for Codex-specific progress patterns.
if re.search(r'\(esc\s+to\s+interrupt', recent, re.IGNORECASE):
    state = 'working'
elif re.search(r'(Thinking|Working|Processing|Running)\.\.\.',  bottom, re.IGNORECASE):
    state = 'working'
elif re.search(r'[⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏⏳🔄]', bottom):
    state = 'working'

# Waiting: confirmation prompts.
elif any(p in bottom for p in ['[y/N]', '[Y/n]', 'approve', 'Approve', 'Allow']):
    state = 'waiting'

# Idle: open prompt. Codex uses "›" (U+203A) as prompt, not ">".
elif re.search(r'›\s', bottom):
    state = 'idle'
elif bottom.strip().endswith('>'):
    state = 'idle'

else:
    state = 'unknown'

json.dump({"state": state}, sys.stdout)
