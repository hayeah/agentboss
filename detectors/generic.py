#!/usr/bin/env python3
"""Generic state detector — checks for common patterns.

Works with any interactive CLI by looking for universal indicators.
Focus on the last 20 lines where the current state appears.
"""
import sys
import json
import re

content = sys.stdin.read()
lines = content.strip().split('\n')
recent = '\n'.join(lines[-20:]) if lines else ''
bottom = '\n'.join(lines[-10:]) if lines else ''
last_line = lines[-1].strip() if lines else ''

# Working: "(esc to interrupt)" is universal for AI coding agents.
if re.search(r'\(esc\s+to\s+interrupt', recent, re.IGNORECASE):
    state = 'working'

# Waiting: common confirmation patterns.
elif re.search(r'\[(Y/n|y/N|yes/no)\]', bottom, re.IGNORECASE):
    state = 'waiting'

# Idle: common prompt endings.
elif re.search(r'[$>❯#%]\s*$', last_line):
    state = 'idle'
elif last_line.endswith('>>>'):
    state = 'idle'

else:
    state = 'unknown'

json.dump({"state": state}, sys.stdout)
