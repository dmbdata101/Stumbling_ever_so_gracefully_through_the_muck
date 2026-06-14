#!/usr/bin/env python3
"""
Home Dashboard — FastAPI backend.
Run this as Pane 1. Open http://localhost:8080 in your browser.
"""

import json
import shutil
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional

import uvicorn
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import FileResponse
from fastapi.staticfiles import StaticFiles
from pydantic import BaseModel

BASE_DIR = Path(__file__).parent
HOME_DIR = BASE_DIR / "Home"
PROJECTS_DIR = HOME_DIR / "projects"
AGENTS_DIR = HOME_DIR / "agents"
CURRENT_PROJECT_FILE = HOME_DIR / "current_project.json"
CONFIG_FILE = BASE_DIR / "config" / "settings.json"
FRONTEND_DIR = BASE_DIR / "frontend"

app = FastAPI(title="Home Hub")
app.add_middleware(CORSMiddleware, allow_origins=["*"], allow_methods=["*"], allow_headers=["*"])


# ── helpers ──────────────────────────────────────────────────────────────────

def ensure_dirs():
    for d in (PROJECTS_DIR, AGENTS_DIR, FRONTEND_DIR):
        d.mkdir(parents=True, exist_ok=True)


def load_config() -> dict:
    return json.loads(CONFIG_FILE.read_text()) if CONFIG_FILE.exists() else {}


def get_current_project() -> Optional[dict]:
    return json.loads(CURRENT_PROJECT_FILE.read_text()) if CURRENT_PROJECT_FILE.exists() else None


def messages_dir(project_name: str) -> Path:
    return PROJECTS_DIR / project_name / "messages"


def tokens_dir(project_name: str) -> Path:
    return PROJECTS_DIR / project_name / "tokens"


# ── request models ───────────────────────────────────────────────────────────

class NewProject(BaseModel):
    name: str
    description: str = ""


class SwitchProject(BaseModel):
    name: str


class NewMessage(BaseModel):
    content: str


class ConfigUpdate(BaseModel):
    data: dict


class ExportRequest(BaseModel):
    destination: str  # absolute path to external drive / iCloud folder


# ── projects ─────────────────────────────────────────────────────────────────

@app.get("/api/projects")
def list_projects():
    if not PROJECTS_DIR.exists():
        return []
    result = []
    for d in sorted(PROJECTS_DIR.iterdir()):
        meta_file = d / "meta.json"
        if d.is_dir() and meta_file.exists():
            result.append(json.loads(meta_file.read_text()))
    return result


@app.post("/api/projects", status_code=201)
def create_project(body: NewProject):
    project_dir = PROJECTS_DIR / body.name
    if project_dir.exists():
        raise HTTPException(409, "Project already exists")
    for sub in ("messages", "tokens"):
        (project_dir / sub).mkdir(parents=True)
    meta = {
        "name": body.name,
        "description": body.description,
        "created": datetime.now(timezone.utc).isoformat(),
    }
    (project_dir / "meta.json").write_text(json.dumps(meta, indent=2))
    # Auto-switch to the new project
    CURRENT_PROJECT_FILE.write_text(json.dumps(meta, indent=2))
    return meta


@app.get("/api/projects/current")
def current_project():
    cp = get_current_project()
    if not cp:
        raise HTTPException(404, "No active project. Create one first.")
    return cp


@app.put("/api/projects/current")
def switch_project(body: SwitchProject):
    project_dir = PROJECTS_DIR / body.name
    if not project_dir.exists():
        raise HTTPException(404, "Project not found")
    meta = json.loads((project_dir / "meta.json").read_text())
    CURRENT_PROJECT_FILE.write_text(json.dumps(meta, indent=2))
    return meta


# ── messages ─────────────────────────────────────────────────────────────────

@app.get("/api/messages")
def get_messages(since: Optional[str] = None):
    """
    Polling endpoint. Frontend calls this every 5-10s.
    Pass ?since=ISO_TIMESTAMP to receive only messages newer than that time.
    Omit since to get all messages for the current project.
    """
    cp = get_current_project()
    if not cp:
        raise HTTPException(400, "No active project")

    msg_dir = messages_dir(cp["name"])
    msg_dir.mkdir(parents=True, exist_ok=True)

    result = []
    for f in sorted(msg_dir.glob("*.json")):
        try:
            msg = json.loads(f.read_text())
            if since is None or msg.get("timestamp", "") > since:
                result.append(msg)
        except Exception:
            pass
    return result


@app.post("/api/messages", status_code=201)
def send_message(body: NewMessage):
    """User sends a message from the browser. Agents pick it up via polling."""
    cp = get_current_project()
    if not cp:
        raise HTTPException(400, "No active project")

    msg_dir = messages_dir(cp["name"])
    msg_dir.mkdir(parents=True, exist_ok=True)

    msg = {
        "id": str(uuid.uuid4()),
        "sender": "user",
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "content": body.content,
        "type": "message",
        "project": cp["name"],
    }
    fname = datetime.now().strftime("%Y%m%d_%H%M%S_%f") + "_user.json"
    (msg_dir / fname).write_text(json.dumps(msg, indent=2))
    return msg


# ── tokens ────────────────────────────────────────────────────────────────────

@app.get("/api/tokens")
def get_tokens():
    """Returns token usage for each agent in the current project."""
    cp = get_current_project()
    if not cp:
        return {}
    tok_dir = tokens_dir(cp["name"])
    result = {}
    for agent in ("claude", "grok"):
        f = tok_dir / f"{agent}_tokens.json"
        result[agent] = json.loads(f.read_text()) if f.exists() else {
            "input_tokens": 0,
            "output_tokens": 0,
            "estimated_cost_usd": 0.0,
            "response_count": 0,
        }
    return result


# ── agents ────────────────────────────────────────────────────────────────────

@app.get("/api/agents")
def get_agents():
    """Returns live status of Claude and Grok agent processes."""
    result = {}
    for agent in ("claude", "grok"):
        f = AGENTS_DIR / f"{agent}.status"
        if not f.exists():
            result[agent] = {"status": "offline", "last_seen": None}
            continue
        try:
            data = json.loads(f.read_text())
            age = (datetime.now(timezone.utc) - datetime.fromisoformat(data["last_seen"])).total_seconds()
            if age > 30:
                data["status"] = "idle"
            result[agent] = data
        except Exception:
            result[agent] = {"status": "unknown", "last_seen": None}
    return result


# ── config ────────────────────────────────────────────────────────────────────

@app.get("/api/config")
def get_config():
    return load_config()


@app.put("/api/config")
def update_config(body: ConfigUpdate):
    CONFIG_FILE.write_text(json.dumps(body.data, indent=2))
    return body.data


# ── export ────────────────────────────────────────────────────────────────────

@app.post("/api/export")
def export_project(body: ExportRequest):
    """
    Copies the entire project folder to an external drive / iCloud path
    and generates a human-readable markdown transcript alongside it.
    """
    cp = get_current_project()
    if not cp:
        raise HTTPException(400, "No active project")

    dest = Path(body.destination).expanduser()
    if not dest.exists():
        raise HTTPException(400, f"Destination not found: {dest}")

    name = cp["name"]
    src = PROJECTS_DIR / name
    archive_dest = dest / name
    shutil.copytree(str(src), str(archive_dest), dirs_exist_ok=True)

    # Build markdown transcript
    lines = [f"# {name}\n\n", f"*Exported: {datetime.now().isoformat()}*\n\n---\n\n"]
    for f in sorted((src / "messages").glob("*.json")):
        try:
            msg = json.loads(f.read_text())
            sender = msg.get("sender", "?").upper()
            ts = msg.get("timestamp", "")[:19].replace("T", " ")
            content = msg.get("content", "")
            lines.append(f"**[{ts}] {sender}**\n\n{content}\n\n---\n\n")
        except Exception:
            pass

    transcript = archive_dest / f"{name}_transcript.md"
    transcript.write_text("".join(lines))

    return {
        "exported_to": str(archive_dest),
        "transcript": str(transcript),
        "message_count": len(list((src / "messages").glob("*.json"))),
    }


# ── frontend ──────────────────────────────────────────────────────────────────

app.mount("/static", StaticFiles(directory=str(FRONTEND_DIR)), name="static")

@app.get("/")
def index():
    index_file = FRONTEND_DIR / "index.html"
    if not index_file.exists():
        return {"error": "Frontend not built yet. See frontend/index.html"}
    return FileResponse(str(index_file))


# ── entry ─────────────────────────────────────────────────────────────────────

if __name__ == "__main__":
    ensure_dirs()
    port = load_config().get("dashboard_port", 8080)
    print(f"[DASHBOARD] http://localhost:{port}")
    uvicorn.run(app, host="0.0.0.0", port=port, log_level="warning")
