#!/usr/bin/env python3
"""
Grok Agent Watcher — run this in Grok's dedicated terminal pane.
Polls Home/ for new messages and responds via the xAI Grok API.
"""

import json
import os
import sys
import time
import uuid
from datetime import datetime, timezone
from pathlib import Path

from openai import OpenAI, APIError

BASE_DIR = Path(__file__).parent.parent
HOME_DIR = BASE_DIR / "Home"
MESSAGES_DIR = HOME_DIR / "messages"
AGENTS_DIR = HOME_DIR / "agents"
CONFIG_FILE = BASE_DIR / "config" / "settings.json"
PROCESSED_FILE = AGENTS_DIR / "grok_processed.json"

AGENT_NAME = "grok"
POLL_INTERVAL = 2
MAX_HISTORY = 20
MAX_RPM = 6
RPM_WINDOW = 60
MAX_AGENT_CHAIN = 3

SYSTEM_PROMPT = """\
You are Grok, the Research & Visual Design specialist in a shared AI workspace called "Home."

Your responsibilities:
- Deep research, fact-checking, market analysis, technical deep-dives
- Web and app UI/UX layout and design system recommendations
- Visual asset feedback, mockup descriptions, image analysis
- Content review, proofreading, and documentation critique
- Synthesizing research into clear, actionable recommendations

Your collaborator Claude handles code execution, testing, and implementation.
The user is the Director — follow their instructions and coordinate with Claude as needed.

Communication style:
- Lead with the most important finding or recommendation.
- Use bullet points and numbered lists for research findings.
- To hand off to Claude: start with "CLAUDE:" so they know you're addressing them.
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
        if sender in ("user", "claude") and msg.get("id") not in processed:
            return True
    return False


def consecutive_agent_chain(messages: list[dict]) -> int:
    count = 0
    for msg in reversed(messages):
        if msg.get("sender") == "user":
            break
        if msg.get("sender") in ("claude", "grok"):
            count += 1
    return count


def build_conversation(messages: list[dict]) -> list[dict]:
    """Build an OpenAI-format messages array from recent history."""
    chat = [{"role": "system", "content": SYSTEM_PROMPT}]
    for msg in messages[-MAX_HISTORY:]:
        sender = msg.get("sender", "")
        content = msg.get("content", "")
        if msg.get("type") == "status":
            continue
        if sender == AGENT_NAME:
            chat.append({"role": "assistant", "content": content})
        elif sender in ("user", "claude"):
            label = "USER" if sender == "user" else "CLAUDE"
            chat.append({"role": "user", "content": f"[{label}]: {content}"})

    # Merge consecutive same-role messages (required by API)
    merged = [chat[0]]  # keep system message
    for m in chat[1:]:
        if merged[-1]["role"] == m["role"] and m["role"] != "system":
            merged[-1]["content"] += "\n\n" + m["content"]
        else:
            merged.append(m)

    # Must end with a user message
    if merged[-1]["role"] != "user":
        return []

    return merged


def main():
    config = load_config()
    model = config.get("grok_model", "grok-3")

    api_key = os.environ.get("XAI_API_KEY")
    if not api_key:
        print("[GROK] ERROR: XAI_API_KEY not set. See .env.example")
        sys.exit(1)

    client = OpenAI(api_key=api_key, base_url="https://api.x.ai/v1")
    ensure_dirs()
    update_status("idle")
    processed = load_processed()
    response_times: list[float] = []

    print(f"[GROK] Online · model: {model} · polling every {POLL_INTERVAL}s")
    print("[GROK] Press Ctrl+C to stop.\n")

    while True:
        try:
            update_status("idle")
            messages = load_messages()

            if not has_unprocessed(messages, processed):
                time.sleep(POLL_INTERVAL)
                continue

            if consecutive_agent_chain(messages) >= MAX_AGENT_CHAIN:
                time.sleep(5)
                continue

            now = time.time()
            response_times = [t for t in response_times if now - t < RPM_WINDOW]
            if len(response_times) >= MAX_RPM:
                print(f"[GROK] Rate limit ({MAX_RPM}/min). Pausing...")
                time.sleep(10)
                continue

            update_status("working")

            for msg in messages:
                if msg.get("sender") in ("user", "claude") and msg.get("id") not in processed:
                    processed.add(msg["id"])
            save_processed(processed)

            conversation = build_conversation(messages)
            if not conversation:
                time.sleep(POLL_INTERVAL)
                continue

            response = client.chat.completions.create(
                model=model,
                messages=conversation,
                max_tokens=4096,
            )

            reply = response.choices[0].message.content
            write_response(reply)
            response_times.append(time.time())
            print(f"[GROK] Responded at {datetime.now().strftime('%H:%M:%S')}")

        except APIError as e:
            print(f"[GROK] API error: {e}")
            time.sleep(5)
        except KeyboardInterrupt:
            print("\n[GROK] Shutting down.")
            update_status("offline")
            break
        except Exception as e:
            print(f"[GROK] Unexpected error: {e}")
            time.sleep(POLL_INTERVAL)


if __name__ == "__main__":
    main()
