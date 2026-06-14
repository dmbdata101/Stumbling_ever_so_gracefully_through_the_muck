#!/bin/bash
# Pane 2 — CLAUDE's pane
cd "$(dirname "$0")"
source .env 2>/dev/null || true
python3 agents/claude_watcher.py
