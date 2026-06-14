#!/usr/bin/env python3
"""Home Hub — your command center. Run this in your main terminal pane."""

import json
import threading
import time
import uuid
from datetime import datetime, timezone
from pathlib import Path

from rich.console import Console
from rich.text import Text
from rich.panel import Panel
from rich import box
from prompt_toolkit import PromptSession
from prompt_toolkit.patch_stdout import patch_stdout

BASE_DIR = Path(__file__).parent
HOME_DIR = BASE_DIR / "Home"
MESSAGES_DIR = HOME_DIR / "messages"
AGENTS_DIR = HOME_DIR / "agents"

SENDER_STYLES = {
    "user":   ("bold cyan",    "  YOU  "),
    "claude": ("bold yellow",  " CLAUDE"),
    "grok":   ("bold magenta", "  GROK "),
    "system": ("dim white",    "  SYS  "),
}

console = Console()
_seen_ids: set[str] = set()
_running = True


def ensure_dirs():
    MESSAGES_DIR.mkdir(parents=True, exist_ok=True)
    AGENTS_DIR.mkdir(parents=True, exist_ok=True)


def write_message(sender: str, content: str, msg_type: str = "message") -> dict:
    msg_id = str(uuid.uuid4())
    ts = datetime.now(timezone.utc).isoformat()
    msg = {
        "id": msg_id,
        "sender": sender,
        "timestamp": ts,
        "content": content,
        "type": msg_type,
    }
    fname = datetime.now().strftime("%Y%m%d_%H%M%S_%f") + f"_{sender}.json"
    (MESSAGES_DIR / fname).write_text(json.dumps(msg, indent=2))
    return msg


def fmt_msg(msg: dict) -> Text:
    sender = msg.get("sender", "system")
    color, label = SENDER_STYLES.get(sender, ("white", sender[:7].upper().center(7)))
    try:
        ts_str = datetime.fromisoformat(msg["timestamp"]).strftime("%H:%M:%S")
    except Exception:
        ts_str = "??:??:??"
    t = Text()
    t.append(f" {ts_str} ", style="dim")
    t.append(f" {label} ", style=f"{color} reverse")
    t.append(f"  {msg.get('content', '')}", style="white")
    return t


def agent_status(agent: str) -> str:
    f = AGENTS_DIR / f"{agent}.status"
    if not f.exists():
        return "[dim]offline[/dim]"
    try:
        d = json.loads(f.read_text())
        age = (datetime.now(timezone.utc) - datetime.fromisoformat(d["last_seen"])).total_seconds()
        s = d.get("status", "idle")
        if age > 30:
            return "[dim]idle[/dim]"
        return {"working": "[bold green]working ●[/bold green]", "idle": "[yellow]idle ○[/yellow]"}.get(
            s, f"[dim]{s}[/dim]"
        )
    except Exception:
        return "[dim]?[/dim]"


def status_bar() -> str:
    return (
        f"[bold yellow]CLAUDE[/bold yellow] {agent_status('claude')}   "
        f"[bold magenta]GROK[/bold magenta] {agent_status('grok')}"
    )


def poll_loop():
    """Background thread: print new messages as they arrive in Home/."""
    global _running
    while _running:
        for f in sorted(MESSAGES_DIR.glob("*.json")):
            try:
                msg = json.loads(f.read_text())
                mid = msg.get("id")
                if mid and mid not in _seen_ids:
                    _seen_ids.add(mid)
                    if msg.get("sender") != "user":
                        console.print(fmt_msg(msg))
            except Exception:
                pass
        time.sleep(2)


def banner():
    console.print(Panel(
        "[bold cyan]HOME HUB[/bold cyan]  ·  Your command center\n\n"
        "[bold yellow]CLAUDE[/bold yellow]  code · tests · implementation\n"
        "[bold magenta]GROK[/bold magenta]    research · visuals · reviews\n\n"
        "[dim]Type a message → Enter.   /status  /clear  /quit[/dim]",
        border_style="cyan",
        box=box.DOUBLE_EDGE,
        padding=(0, 2),
    ))


def main():
    global _running
    ensure_dirs()
    banner()

    # Show recent history on startup
    for f in sorted(MESSAGES_DIR.glob("*.json")):
        try:
            msg = json.loads(f.read_text())
            mid = msg.get("id")
            if mid:
                _seen_ids.add(mid)
                if msg.get("type") != "status":
                    console.print(fmt_msg(msg))
        except Exception:
            pass

    system_msg = write_message("system", "Hub online. Agents are listening.", "status")
    _seen_ids.add(system_msg["id"])

    threading.Thread(target=poll_loop, daemon=True).start()
    session = PromptSession()

    with patch_stdout():
        while True:
            try:
                text = session.prompt("→ ").strip()
            except (KeyboardInterrupt, EOFError):
                _running = False
                break

            if not text:
                continue

            if text.startswith("/"):
                cmd = text[1:].lower().strip()
                if cmd in ("quit", "exit", "q"):
                    write_message("system", "Hub shutting down.", "status")
                    _running = False
                    break
                elif cmd == "status":
                    console.print(status_bar())
                elif cmd == "clear":
                    console.clear()
                    banner()
                else:
                    console.print(f"[dim]Unknown command: {text}[/dim]")
                continue

            msg = write_message("user", text)
            _seen_ids.add(msg["id"])
            console.print(fmt_msg(msg))


if __name__ == "__main__":
    main()
