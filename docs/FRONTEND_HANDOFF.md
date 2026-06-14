# Frontend Handoff for Grok
**File:** `frontend/index.html` (and optionally `frontend/style.css`, `frontend/app.js`)
**Backend runs at:** `http://localhost:8080`
**Your job:** Replace the placeholder UI in `frontend/index.html` with a polished, functional dashboard.

---

## What the app does

This is a local AI workspace hub. The user (Director) types messages in the browser.
Two AI agents — Claude (code/implementation) and Grok (research/visuals) — run in
background terminal panes, pick up those messages, and respond. Everything flows
through a shared folder on disk. The browser polls the backend every 5 seconds to
display new messages.

---

## Color identity

These are the established colors — keep them consistent across all UI:

| Sender | Color | Notes |
|--------|-------|-------|
| USER   | `#39d353` (green) | The Director |
| CLAUDE | `#e3b341` (amber/yellow) | Code agent |
| GROK   | `#d2a8ff` (lavender) | Research agent |
| SYSTEM | `#6e7681` (gray) | Status messages, suppress or de-emphasize |

Background palette (dark theme, GitHub-inspired):
- Page background: `#0d1117`
- Panel/card: `#161b22`
- Border: `#30363d`
- Text primary: `#e6edf3`
- Text muted: `#8b949e`

---

## All API endpoints

Base URL: `http://localhost:8080`

### Projects

```
GET  /api/projects
  → [ { name, description, created } ]

POST /api/projects
  body: { name: string, description?: string }
  → { name, description, created }
  (also auto-switches to the new project)

GET  /api/projects/current
  → { name, description, created }

PUT  /api/projects/current
  body: { name: string }
  → { name, description, created }
```

### Messages

```
GET  /api/messages
  → [ Message ]   (all messages for current project)

GET  /api/messages?since=2026-06-14T09:30:00.000000+00:00
  → [ Message ]   (only messages newer than that ISO timestamp)
  Use this for polling. Store the timestamp of the last received message
  and pass it on each poll call.

POST /api/messages
  body: { content: string }
  → Message
```

**Message object:**
```json
{
  "id": "uuid4-string",
  "sender": "user | claude | grok | system",
  "timestamp": "2026-06-14T09:30:00.123456+00:00",
  "content": "The message text",
  "type": "message | status",
  "project": "ProjectName"
}
```
Filter out `type: "status"` messages — they are internal bookkeeping, not for display.

### Agents

```
GET  /api/agents
  → {
      claude: { status: "working | idle | offline | unknown", last_seen: ISO | null },
      grok:   { status: "working | idle | offline | unknown", last_seen: ISO | null }
    }
```

### Tokens

```
GET  /api/tokens
  → {
      claude: { input_tokens: int, output_tokens: int, estimated_cost_usd: float, response_count: int },
      grok:   { input_tokens: int, output_tokens: int, estimated_cost_usd: float, response_count: int }
    }
```

### Config

```
GET  /api/config
  → {
      dashboard_port: 8080,
      claude_model: "claude-opus-4-8",
      grok_model: "grok-3",
      poll_interval_seconds: 3,
      max_history_messages: 20,
      max_responses_per_minute: 6,
      max_consecutive_agent_chain: 3,
      export_path: "~/Desktop"
    }

PUT  /api/config
  body: { data: { ...full config object... } }
  → updated config object
  (writes to config/settings.json — agents pick up changes on next poll)
```

### Export

```
POST /api/export
  body: { destination: "/Volumes/MyDrive/Archive" }
  → {
      exported_to: "/Volumes/MyDrive/Archive/ProjectName",
      transcript: "/Volumes/MyDrive/Archive/ProjectName/ProjectName_transcript.md",
      message_count: 42
    }
  Copies entire project folder + generates a markdown transcript.
  The destination path must exist on disk (external drive, iCloud folder, etc.)
```

---

## Pages / views to build

### 1. Main Chat View (primary view)
- Full-height message feed, color-coded by sender
- Code blocks rendered as `<pre>` (markdown-lite is fine — no need for a full parser)
- Text input at the bottom (Enter to send, Shift+Enter for newline)
- Agent status badges in header (working/idle/offline with colored dot indicator)
- Token counter per agent in header or sidebar (live, refreshes every 8s)
- Current project name displayed prominently

### 2. Project Switcher / New Project Modal
- Triggered from header or sidebar
- Lists existing projects (from `GET /api/projects`)
- Form to create a new project (name + description)
- Switching project clears the feed and loads the new project's messages

### 3. Settings Panel (can be a slide-out drawer or a `/settings` route)
- Editable fields for every key in config:
  - `dashboard_port` — number input
  - `claude_model` — text input or dropdown (claude-opus-4-8, claude-sonnet-4-6, claude-haiku-4-5-20251001)
  - `grok_model` — text input (grok-3, grok-3-mini)
  - `max_responses_per_minute` — slider (1–20)
  - `max_history_messages` — slider (5–50)
  - `max_consecutive_agent_chain` — slider (1–10)
  - `export_path` — text input (filesystem path)
- Save button → `PUT /api/config`
- Show current estimated cost per agent from `/api/tokens`

### 4. Export Button
- Available in header or project menu
- Shows current export_path from config (editable inline)
- On click → `POST /api/export` with the destination path
- Show success (destination path) or error message

---

## Polling pattern (already in placeholder)

```javascript
let lastTimestamp = null;

async function poll() {
  const url = lastTimestamp
    ? `/api/messages?since=${encodeURIComponent(lastTimestamp)}`
    : '/api/messages';
  const msgs = await fetch(url).then(r => r.json());
  msgs.forEach(msg => {
    if (msg.type === 'status') return;
    appendToFeed(msg);
    if (!lastTimestamp || msg.timestamp > lastTimestamp) lastTimestamp = msg.timestamp;
  });
}

setInterval(poll, 5000);
```

---

## Constraints / notes

- **No build step** — plain HTML + CSS + vanilla JS. No React, no npm, no webpack.
  The backend serves `frontend/index.html` directly. Additional files (style.css, app.js)
  can live in `frontend/` and be linked from index.html with relative paths.
- **Dark theme only** — the terminal panes are dark; keep visual consistency.
- **Mobile-aware but desktop-first** — the user runs this on a Mac.
- **No auth** — it's localhost, no login needed.
- **Markdown in messages** — agents use `\`\`\`code blocks\`\`\`` and `**bold**`. Rendering
  inline code and fenced code blocks is enough. Full markdown is a nice-to-have.
- The backend is already running when Grok opens the browser — all endpoints are live.

---

## File to edit

`frontend/index.html` — replace or extend this file. The current version is a
functional but completely unstyled placeholder so you can see the data flowing.
