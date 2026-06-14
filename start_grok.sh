#!/bin/bash
# Pane 3 — GROK's pane
cd "$(dirname "$0")"
source .env 2>/dev/null || true
python3 agents/grok_watcher.py
