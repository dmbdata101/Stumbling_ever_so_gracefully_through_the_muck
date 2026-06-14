#!/bin/bash
# Pane 1 — Dashboard (replaces hub.py as primary interface)
cd "$(dirname "$0")"
source .env 2>/dev/null || true
python3 dashboard.py
