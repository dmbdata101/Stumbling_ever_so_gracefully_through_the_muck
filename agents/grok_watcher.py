#!/usr/bin/env python3
"""Grok Agent Watcher — run in Grok's dedicated terminal pane."""

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
PROJECTS_DIR = HOME_DIR / "projects"
AGENTS_DIR = HOME_DIR / "agents"
CURRENT_PROJECT_FILE = HOME_DIR / "current_project.json"
CONFIG_FILE = BASE_DIR / "config" / "settings.json"
PROCESSED_FILE = AGENTS_DIR / "grok_processed.json"

AGENT_NAME = "grok"
POLL_INTERVAL = 3
MAX_HISTORY = 20
MAX_RPM = 6
RPM_WINDOW = 60
MAX_AGENT_CHAIN = 3

# xAI pricing — update if it changes
INPUT_COST_PER_M = 3.0
OUTPUT_COST_PER_M = 15.0

SYSTEM_PROMPT = """\
You are Grok, the Research & Visual Design specialist in a shared AI workspace called "Home."

Your responsibilities:
- Deep research, fact-checking, technical deep-dives, market analysis
- Web and app UI/UX layout and design system recommendations
- Visual asset feedback, mockup descriptions, image analysis
- Content review, proofreading, documentation critique

Your collaborator Claude handles code execution, testing, and implementation.
The user is the Director — follow their instructions.

Rules:
- Lead with the most important finding. Be incisive.
- Use bullet points and numbered lists for research.
- To address Claude directly: start your message with "CLAUDE:"
- To request user input: start with "USER:"
"""


def load_config() -> dict:
    return json.loads(CONFIG_FILE.read_text()) if CONFIG_FILE.exists() else {}


def get_current_project() -> dict | None:
    return json.loads(CURRENT_PROJECT_FILE.read_text()) if CURRENT_PROJECT_FILE.exists() else None


def messages_dir(project_name: str) -> Path:
    return PROJECTS_DIR / project_name / "messages"


def tokens_file(project_name: str) -> Path:
    return PROJECTS_DIR / project_name / "tokens" / f"{AGENT_NAME}_tokens.json"


def ensure_dirs(project_name: str):
    AGENTS_DIR.mkdir(parents=True, exist_ok=True)
    messages_dir(project_name).mkdir(parents=True, exist_ok=True)
    tokens_file(project_name).parent.mkdir(parents=True, exist_ok=True)


def update_status(status: str):
    data = {"agent": AGENT_NAME, "status": status, "last_seen": datetime.now(timezone.utc).isoformat()}
    (AGENTS_DIR / f"{AGENT_NAME}.status").write_text(json.dumps(data, indent=2))


def load_processed() -> set[str]:
    if PROCESSED_FILE.exists():
        try:
            return set(json.loads(PROCESSED_FILE.read_text()))
        except Exception:
            pass
    return set()


def save_processed(ids: set[str]):
    PROCESSED_FILE.write_text(json.dumps(list(ids)[-1000:], indent=2))


def track_tokens(project_name: str, usage):
    tf = tokens_file(project_name)
    data = json.loads(tf.read_text()) if tf.exists() else {
        "input_tokens": 0, "output_tokens": 0,
        "estimated_cost_usd": 0.0, "response_count": 0,
    }
    data["input_tokens"] += usage.prompt_tokens
    data["output_tokens"] += usage.completion_tokens
    data["response_count"] += 1
    data["estimated_cost_usd"] = round(
        (data["input_tokens"] / 1_000_000) * INPUT_COST_PER_M
        + (data["output_tokens"] / 1_000_000) * OUTPUT_COST_PER_M,
        4,
    )
    tf.write_text(json.dumps(data, indent=2))


def write_response(project_name: str, content: str) -> dict:
    msg = {
        "id": str(uuid.uuid4()),
        "sender": AGENT_NAME,
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "content": content,
        "type": "message",
        "project": project_name,
    }
    fname = datetime.now().strftime("%Y%m%d_%H%M%S_%f") + f"_{AGENT_NAME}.json"
    (messages_dir(project_name) / fname).write_text(json.dumps(msg, indent=2))
    return msg


def load_messages(project_name: str) -> list[dict]:
    msgs = []
    for f in sorted(messages_dir(project_name).glob("*.json")):
        try:
            msgs.append(json.loads(f.read_text()))
        except Exception:
            pass
    return msgs


def has_unprocessed(messages: list[dict], processed: set[str]) -> bool:
    return any(
        m.get("sender") in ("user", "claude") and m.get("id") not in processed
        for m in messages
    )


def consecutive_agent_chain(messages: list[dict]) -> int:
    count = 0
    for m in reversed(messages):
        if m.get("sender") == "user":
            break
        if m.get("sender") in ("claude", "grok"):
            count += 1
    return count


def build_conversation(messages: list[dict]) -> list[dict]:
    chat = [{"role": "system", "content": SYSTEM_PROMPT}]
    for m in messages[-MAX_HISTORY:]:
        if m.get("type") == "status":
            continue
        sender = m.get("sender", "")
        content = m.get("content", "")
        if sender == AGENT_NAME:
            chat.append({"role": "assistant", "content": content})
        elif sender in ("user", "claude"):
            label = "USER" if sender == "user" else "CLAUDE"
            chat.append({"role": "user", "content": f"[{label}]: {content}"})

    # Merge consecutive same-role messages (API requirement)
    merged = [chat[0]]
    for m in chat[1:]:
        if merged[-1]["role"] == m["role"] and m["role"] != "system":
            merged[-1]["content"] += "\n\n" + m["content"]
        else:
            merged.append(m)

    return merged if merged[-1]["role"] == "user" else []


def main():
    config = load_config()
    model = config.get("grok_model", "grok-3")

    api_key = os.environ.get("XAI_API_KEY")
    if not api_key:
        print("[GROK] ERROR: XAI_API_KEY not set")
        sys.exit(1)

    client = OpenAI(api_key=api_key, base_url="https://api.x.ai/v1")
    AGENTS_DIR.mkdir(parents=True, exist_ok=True)
    update_status("idle")
    processed = load_processed()
    response_times: list[float] = []

    print(f"[GROK] Online · model: {model}")
    print("[GROK] Waiting for an active project...\n")

    while True:
        try:
            cp = get_current_project()
            if not cp:
                update_status("idle")
                time.sleep(POLL_INTERVAL)
                continue

            project_name = cp["name"]
            ensure_dirs(project_name)
            update_status("idle")
            messages = load_messages(project_name)

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

            for m in messages:
                if m.get("sender") in ("user", "claude") and m.get("id") not in processed:
                    processed.add(m["id"])
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
            write_response(project_name, reply)
            track_tokens(project_name, response.usage)
            response_times.append(time.time())

            in_tok = response.usage.prompt_tokens
            out_tok = response.usage.completion_tokens
            print(f"[GROK] {datetime.now().strftime('%H:%M:%S')}  in={in_tok} out={out_tok}")

        except APIError as e:
            print(f"[GROK] API error: {e}")
            time.sleep(5)
        except KeyboardInterrupt:
            print("\n[GROK] Shutting down.")
            update_status("offline")
            break
        except Exception as e:
            print(f"[GROK] Error: {e}")
            time.sleep(POLL_INTERVAL)


if __name__ == "__main__":
    main()
