#!/bin/bash
# Pane 1 — YOUR pane (the command center)
cd "$(dirname "$0")"
source .env 2>/dev/null || true
python3 hub.py
