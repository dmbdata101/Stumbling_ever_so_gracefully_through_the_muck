#!/usr/bin/env python3
"""
Claude Agent Watcher — run this in Claude's dedicated terminal pane.
Polls Home/ for new messages and responds via the Anthropic API.
"""

import json
import os
import sys
import time
import uuid
from datetime import datetime, timezone
from pathlib import Path

import anthropic

BASE_DIR = Path(__file__).parent.parent
HOME_DIR = BASE_DIR / "Home"
MESSAGES_DIR = HOME_DIR / "messages"
AGENTS_DIR = HOME_DIR / "agents"
CONFIG_FILE = BASE_DIR / "config" / "settings.json"
PROCESSED_FILE = AGENTS_DIR / "claude_processed.json"

AGENT_NAME = "claude"
POLL_INTERVAL = 2       # seconds between polls
MAX_HISTORY = 20        # messages of context sent to API
MAX_RPM = 6             # max responses per minute (cost guard)
RPM_WINDOW = 60         # seconds for the rate-limit window
MAX_AGENT_CHAIN = 3     # max consecutive cross-agent replies before pausing for user

SYSTEM_PROMPT = """\
You are Claude, the Code & Implementation specialist in a shared AI workspace called "Home."

Your responsibilities:
- Write, debug, test, and deploy code
- Architect solutions and optimize performance
- Write unit/integration tests and validate functionality
- Process and transform data and files

Your collaborator Grok handles research, visual design, web/app layouts, and content reviews.
The user is the Director — follow their instructions and coordinate with Grok as needed.

Communication style:
- Lead with action or answer. Be direct and concise.
- Use code blocks for all code.
- To hand off to Grok: start your message with "GROK:" so they know you're addressing them.
- To request user input: start with "USER:" and explain what you need.
"""


def load_config() -> dict:
    if CONFIG_FILE.exists():
        return json.loads(CONFIG_FILE.read_text())
    return {}


def ensure_dirs():
    MESSAGES_DIR.mkdir(parents=True, exist_ok=True)
    AGENTS_DIR.mkdir(parents=True, exist_ok=True)


def update_status(status: str):
    data = {
        "agent": AGENT_NAME,
        "status": status,
        "last_seen": datetime.now(timezone.utc).isoformat(),
    }
    (AGENTS_DIR / f"{AGENT_NAME}.status").write_text(json.dumps(data, indent=2))


def load_processed() -> set[str]:
    if PROCESSED_FILE.exists():
        try:
            return set(json.loads(PROCESSED_FILE.read_text()))
        except Exception:
            pass
    return set()


def save_processed(ids: set[str]):
    # Keep only the last 1000 IDs to prevent unbounded growth
    trimmed = list(ids)[-1000:]
    PROCESSED_FILE.write_text(json.dumps(trimmed, indent=2))


def write_response(content: str) -> dict:
    msg_id = str(uuid.uuid4())
    ts = datetime.now(timezone.utc).isoformat()
    msg = {
        "id": msg_id,
        "sender": AGENT_NAME,
        "timestamp": ts,
        "content": content,
        "type": "message",
    }
    fname = datetime.now().strftime("%Y%m%d_%H%M%S_%f") + f"_{AGENT_NAME}.json"
    (MESSAGES_DIR / fname).write_text(json.dumps(msg, indent=2))
    return msg


def load_messages() -> list[dict]:
    msgs = []
    for f in sorted(MESSAGES_DIR.glob("*.json")):
        try:
            msgs.append(json.loads(f.read_text()))
        except Exception:
            pass
    return msgs


def has_unprocessed(messages: list[dict], processed: set[str]) -> bool:
    for msg in messages:
        sender = msg.get("sender", "")
        if sender in ("user", "grok") and msg.get("id") not in processed:
            return True
    return False


def consecutive_agent_chain(messages: list[dict]) -> int:
    """Count how many recent messages in a row have been from agents (not user)."""
    count = 0
    for msg in reversed(messages):
        if msg.get("sender") == "user":
            break
        if msg.get("sender") in ("claude", "grok"):
            count += 1
    return count


def build_conversation(messages: list[dict]) -> list[dict]:
    """Build an Anthropic messages array from recent history."""
    api_msgs = []
    for msg in messages[-MAX_HISTORY:]:
        sender = msg.get("sender", "")
        content = msg.get("content", "")
        if msg.get("type") == "status":
            continue
        if sender == AGENT_NAME:
            api_msgs.append({"role": "assistant", "content": content})
        elif sender in ("user", "grok"):
            label = "USER" if sender == "user" else "GROK"
            api_msgs.append({"role": "user", "content": f"[{label}]: {content}"})

    # Anthropic requires conversation to start with a user message
    while api_msgs and api_msgs[0]["role"] == "assistant":
        api_msgs.pop(0)

    # Merge consecutive same-role messages (API requires alternating)
    merged = []
    for m in api_msgs:
        if merged and merged[-1]["role"] == m["role"]:
            merged[-1]["content"] += "\n\n" + m["content"]
        else:
            merged.append(m)

    return merged


def main():
    config = load_config()
    model = config.get("claude_model", "claude-opus-4-8")

    api_key = os.environ.get("ANTHROPIC_API_KEY")
    if not api_key:
        print("[CLAUDE] ERROR: ANTHROPIC_API_KEY not set. See .env.example")
        sys.exit(1)

    client = anthropic.Anthropic(api_key=api_key)
    ensure_dirs()
    update_status("idle")
    processed = load_processed()
    response_times: list[float] = []

    print(f"[CLAUDE] Online · model: {model} · polling every {POLL_INTERVAL}s")
    print("[CLAUDE] Press Ctrl+C to stop.\n")

    while True:
        try:
            update_status("idle")
            messages = load_messages()

            if not has_unprocessed(messages, processed):
                time.sleep(POLL_INTERVAL)
                continue

            # Safety: pause if agents have been talking too much without user input
            if consecutive_agent_chain(messages) >= MAX_AGENT_CHAIN:
                time.sleep(5)
                continue

            # Rate limit check
            now = time.time()
            response_times = [t for t in response_times if now - t < RPM_WINDOW]
            if len(response_times) >= MAX_RPM:
                print(f"[CLAUDE] Rate limit ({MAX_RPM}/min). Pausing...")
                time.sleep(10)
                continue

            update_status("working")

            # Mark new messages as processed before responding
            for msg in messages:
                if msg.get("sender") in ("user", "grok") and msg.get("id") not in processed:
                    processed.add(msg["id"])
            save_processed(processed)

            conversation = build_conversation(messages)
            if not conversation:
                time.sleep(POLL_INTERVAL)
                continue

            response = client.messages.create(
                model=model,
                max_tokens=4096,
                system=SYSTEM_PROMPT,
                messages=conversation,
            )

            reply = response.content[0].text
            write_response(reply)
            response_times.append(time.time())
            print(f"[CLAUDE] Responded at {datetime.now().strftime('%H:%M:%S')}")

        except anthropic.APIError as e:
            print(f"[CLAUDE] API error: {e}")
            time.sleep(5)
        except KeyboardInterrupt:
            print("\n[CLAUDE] Shutting down.")
            update_status("offline")
            break
        except Exception as e:
            print(f"[CLAUDE] Unexpected error: {e}")
            time.sleep(POLL_INTERVAL)


if __name__ == "__main__":
    main()
