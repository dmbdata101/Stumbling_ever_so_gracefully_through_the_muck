# Stepping Stone — GED Prep App
## Figma Make Prompt (paste-ready)

**App working name:** Stepping Stone
**Tool:** Figma Make (paste the block between the rules into a new project)
**Drafted:** 2026-05-09 with Jar-vez
**Use when:** Figma Make daily/monthly credits reset (free tier: 150/day, 500/month)

---

Build a responsive web app called **"Stepping Stone"** — a personal GED prep tracker for an adult learner working toward eventually pursuing a PsyD. The app should look beautiful on both mobile (375px) and desktop (1440px) — mobile-first responsive, but desktop should feel intentional, not stretched.

## Visual identity

Academic/scholarly aesthetic — think university library meets modern productivity app. Serious but warm, never sterile or corporate.

**Color palette (use these exact values as CSS variables):**
- Primary navy: `#0F2444` (deep, almost-black blue — used for headers, primary buttons, key text)
- Accent gold: `#C8A951` (muted antique gold — used for highlights, progress bars, badges, key metrics)
- Cream background: `#FAF6EE` (off-white with warmth — main app background, NOT pure white)
- Card surface: `#FFFFFF` (clean white for cards floating on the cream)
- Burgundy accent: `#7A2E2E` (sparingly, for "behind/roadblock" states)
- Forest green: `#3A6B47` (for "completed/passed" states)
- Ink text: `#1A1A1A` (body copy)
- Muted text: `#6B6B6B` (secondary text)
- Border: `#E5DFD2` (warm subtle borders)

**Typography:**
- Headings: serif font — use **Playfair Display** or **Cormorant Garamond**. Bold weight for h1/h2.
- Body: clean sans — use **Inter** at 16px base on desktop, 15px on mobile.
- Numbers/stats: serif (Playfair) for that academic-statistics feel.

**Visual touches:**
- Subtle paper-like texture on the cream background (very subtle — 3% opacity noise)
- Cards have a soft drop shadow and 8px rounded corners
- Gold underlines on active nav items instead of background pills
- Use elegant divider lines (1px, border color) between sections
- Iconography: thin/outline style, not filled. Lucide icons preferred.

## App structure — pages

Build these 6 pages with a persistent left sidebar nav (desktop) that collapses to a bottom tab bar (mobile):

### 1. Dashboard (home / "/")
Top of page: a warm greeting with the user's name and a motivational sub-line (e.g. "Good morning, Dave. You're 47 days from your test date.").

Below: a 2x2 grid (stacks on mobile) showing:
- **Overall progress ring** — large gold circular progress indicator showing % of GED prep complete
- **Study streak card** — number of consecutive days studied, with a small flame icon
- **Days until test** — countdown to scheduled GED test date (placeholder: 47 days)
- **Current roadblock** — one card showing the most recent unresolved roadblock with a "view all" link

Below the grid: **"Today's plan"** — a small list of 3 suggested study tasks for the day, with checkboxes.

Below that: **Subject progress bar chart** — horizontal bars for each of the 4 GED subjects (Math, RLA, Science, Social Studies), color-coded gold for in-progress, green for passed.

### 2. Subjects (/subjects)
Grid of 4 subject cards (one per GED test area):
- Mathematical Reasoning
- Reasoning Through Language Arts (RLA)
- Science
- Social Studies

Each card shows: subject name (serif heading), progress ring (gold), # of topics covered / total topics, last studied date, "Continue studying" button.

Clicking a card opens a detail page showing all topics within that subject as a checklist (e.g. for Math: "Quantitative problem solving," "Algebraic problem solving," "Geometry," etc.). Each topic has a status: not started / in progress / mastered. User can mark progress.

### 3. Study Log (/log)
A reverse-chronological log of study sessions. Each entry shows:
- Date and time
- Subject(s) studied
- Duration (minutes)
- Topics covered (tags)
- Optional notes

Top of page: "Log a session" button that opens a modal with a form (subject, duration, topics, notes).

Filter bar: filter by subject, date range, or duration.

Show a small heatmap calendar at the top (like GitHub's contribution graph but in gold/navy) showing study activity over the past 12 weeks.

### 4. Calendar (/calendar)
Full-page month view calendar (Cal.com / Google Calendar style but in navy/gold).

Show:
- Test dates (gold star icons)
- Study sessions (small dots colored by subject)
- Milestones (e.g. "Begin practice tests," "Complete RLA," "GED test day")
- Personal events / blocked days

Click any day to view/add events. Has month/week/day toggle.

Top of page: a horizontal "milestone timeline" showing:
- Today
- Practice test #1 (placeholder date)
- Subject completion milestones
- GED test date
- Beyond: "Begin undergrad" placeholder marker

### 5. Roadblock Journal (/roadblocks)
A list of "roadblocks" — things blocking progress. Each roadblock entry shows:
- Title (e.g. "Stuck on quadratic equations")
- Subject tag
- Date logged
- Status: Open / Working on it / Resolved
- Description (what's blocking, what I've tried)
- Resolution notes (filled in when resolved)

Top of page: "Add roadblock" button → modal form.

Filter: by status, by subject. Default view shows Open + Working on it.

When status is "Resolved," card gets a subtle gold checkmark and moves to a "Resolved" section with date.

### 6. AI Tutor (/tutor)
Two main modes (toggle at top):

**Ask mode:** A chat interface where the user can ask any GED-related question. Responses appear in the academic style (serif font for AI replies, with a small "Tutor" label). Include a placeholder note: "AI Tutor will be powered by Claude API — connect API key in Settings to enable."

**Quiz mode:** Pick a subject and a topic, then take a 5-question multiple-choice quiz. Show questions one at a time with elegant transitions. After each question: explanation of the correct answer. End with a score summary and a button to "Try another quiz" or "Add this topic to study log."

Make both modes look great as placeholder UIs — they don't need to actually work yet, but build the interface so it's ready.

## Navigation

**Desktop:** Left sidebar nav, 240px wide, navy background with cream text. Logo at top ("Stepping Stone" in serif gold). Nav items: Dashboard, Subjects, Study Log, Calendar, Roadblocks, Tutor, Settings. Active item has a gold left-border accent and slightly brighter text. Profile pill at bottom with avatar + name.

**Mobile:** Bottom tab bar with 5 most-used items (Dashboard, Subjects, Log, Calendar, Tutor) — Roadblocks and Settings live in a "More" tab or hamburger menu. Active tab gets a gold dot above the icon.

## Header (mobile + desktop both)

Right side of top bar: small profile avatar, notification bell (with a subtle gold dot if there are alerts), and a dark/light mode toggle (although we're starting in light mode only — toggle can be UI-only for now).

## Tone of copy throughout

Encouraging but not cheesy. Treats the user like an intelligent adult on a serious journey. No emoji. Examples of good microcopy:
- Empty state for Roadblocks: "No roadblocks logged. Either everything's smooth, or you're hiding from yourself. Be honest."
- Empty state for Study Log: "Your first session starts whenever you do."
- Loading state: "Pulling your progress…"
- Success after logging a session: "Session logged. Keep climbing."

## Data

Use realistic placeholder data for a user named "Dave Brown" prepping for a GED test 47 days from today. Include 12 logged study sessions over the last 3 weeks across all 4 subjects, 3 open roadblocks (one in Math, one in Science, one in RLA), 1 resolved roadblock, and progress between 30-70% across subjects (Math lowest, RLA highest, to feel realistic).

## Final polish

Add subtle animations: progress rings fill in on page load, cards have a gentle lift on hover, page transitions fade in smoothly. Nothing flashy — this is a serious app for serious work.

Make sure everything looks great on:
- iPhone 14 (390x844)
- iPad (768x1024)
- Desktop (1440x900)

That's the brief. Build it.

---

## After Figma Make builds it — first round of tweaks to ask for

If anything's off after the first generation, these are typically the prompts to follow up with (each one costs additional credits, so be selective):

1. "The navy is too dark — lighten the primary navy by about 15% so it feels less heavy on white surfaces."
2. "Add the paper texture I asked for — right now the background is flat cream. Add a 3% noise overlay."
3. "Make the serif font weight bolder on h1 — currently it looks too thin."
4. "The progress rings need to actually animate from 0 to their value on page load — currently they just appear."
5. "On mobile, the bottom tab bar is too tall — reduce padding so it doesn't eat 12% of the screen."

## Notes (don't paste these into Figma Make)

- **App name "Stepping Stone"** is a working title. Other options: *Foundation*, *First Class*, *Groundwork*, *The Climb*, or just *GED Path*. Easy to rename later.
- The **AI Tutor section** is placeholder UI only. To make it actually work, we'll need an Anthropic API key (console.anthropic.com) and either Figma Make or a developer to wire it up.
- When you expand from "Just GED" to full GED → PsyD roadmap, the structure is already friendly to that — the calendar timeline already shows "Begin undergrad" as a future milestone.
- **Pair with your existing dispatch system:** since you already do daily morning briefs and end-of-day logs in `02_Daily/`, the Stepping Stone study log could eventually pull from / write to those files for a single source of truth.
