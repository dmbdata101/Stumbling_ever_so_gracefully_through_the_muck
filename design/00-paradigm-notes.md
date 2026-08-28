# Design Notes: A System, Not a Distro

Working notes from the design conversation. Nothing here is committed to as a
decision yet — this is the map, not the route.

---

## What we ruled out, and why

**A custom distro (Tier 2).** Own package repo, own update channel, own
security-patch pipeline. Months up front and it never stops. Kali's value was
never the ISO — it's the repo of packaged tools kept working together. Not
worth rebuilding.

**An OS from scratch (Tier 3), as the *primary* project.** Achievable in
12–18 months if scoped to QEMU `virt` + one arch, but it will not run nmap,
Wireshark, Python, or any lab tooling for years. The lab goal and the
from-scratch goal do not merge. Still viable as a separate long game.

**Rose-tinted assumption, corrected:** AI does not let us bypass past mistakes.
Those mistakes weren't from ignorance — they came from hardware that lies,
errata, and a hostile debug environment. AI is roughly a 2–3x multiplier on
the grind, and an active hazard where it produces plausible-but-wrong kernel
code. Verification burden stays with the human.

---

## The real objection

Every OS in use today descends from decisions made ~1964–1974. macOS *is*
Unix. Linux is a Unix reimplementation. NT came from VMS. "Faster" almost
always means optimizing *within* those assumptions.

The assumptions that are genuinely arbitrary:

| Assumption | Origin | Alternative lineage |
|---|---|---|
| Files as opaque byte arrays in a tree | cheap in 1970 | IBM i single-level store; content-addressed; object graphs |
| Text streams as universal interface | teletypes | PowerShell, Nushell (structured pipelines) |
| Save / load as an explicit act | limited core memory | Smalltalk images; orthogonal persistence; EROS checkpointing |
| Ambient authority (a process inherits *all* your rights) | convenience, not security | capability systems: seL4, Genode, KeyKOS |
| The terminal (VT100 emulation) | a 1978 printer | Oberon, Acme, Arcan, DomTerm |
| The application as unit of work | shrink-wrap software economics | document-centric; verb-centric; Oberon text-as-command-surface |

**The good news:** these alternatives were built, and many were better. They
lost on network effects, application compatibility, and hardware economics —
not on merit. The graveyard is full of good ideas that lost on compatibility,
and compatibility is the exact constraint a personal system already discards.

---

## The practical unlock

The objection is to the **Unix userland paradigm**, not to the kernel.

Almost nothing on that table lives below the syscall boundary. Linux already
provides per-process namespaces (containers are just namespaces), 9P, FUSE,
io_uring, eBPF. Plan 9's most radical ideas run fine as userspace on Linux.

Build the different thing on top of a boring kernel. Replace the substrate
later only if it ever actually gets in the way — by then we'll know exactly
what we need. This is what turns a decade into a year.

---

## Systems to study

- **Plan 9 from Bell Labs** — the Unix authors' own do-over. Per-process
  namespaces, everything genuinely is a file, distribution as mounting.
  Most relevant item on this list. Papers are short.
- **Oberon** (Wirth) — complete OS + compiler + UI in ~10k lines. *Project
  Oberon* is free and fully readable. Any text on screen is executable.
- **Acme** (Pike, Plan 9) — studies as a *challenge to our own premise*:
  mouse-heavy, modeless, arguably more efficient because there is nothing
  to memorize.
- **IBM i / System/38** — the mainframe thread. Single-level store, no
  files, the database *is* the filesystem, capability-addressed objects.
  Shipping since 1979 and still radical.
- **Smalltalk / Squeak** — live image, everything inspectable and editable
  while running. No save, no load, no restart.
- **Erlang/OTP** — isolated processes, supervision trees, let-it-crash, hot
  code reload, distribution as a primitive. Different model of computation
  and completely practical today.

---

## The open question — this is the next real decision

"Totally different" is not a design. Every system that actually broke from the
lineage did it with **one thesis**, from which everything else follows:

- Plan 9 — *everything is a file, and every process builds its own namespace.*
- Smalltalk — *everything is a live object, all the way down.*
- Erlang — *everything is an isolated process that is allowed to crash.*
- IBM i — *there is one persistent address space and no such thing as a file.*

### Candidate theses

**A — "Every program runs in a world I built for it."**
Namespaces + capabilities. Nothing has ambient access to anything; the world
each tool sees is composed explicitly, and can be snapshotted, diffed, and
replayed. For security testing this is close to ideal: every test runs in a
constructed, reproducible universe.

**B — "There are no files. There is one persistent, queryable graph, and
commands are verbs over it."**
Captures, configs, topologies, and results stop being files to parse and
become live objects to query. No save, no load, no serialization tax.
IBM i's idea aimed at this specific domain.

These are compatible. **B is the data model, A is the security model.**

---

## Interaction design (holds regardless of thesis)

Carried over from the shortcuts discussion — this part is settled enough to
build against.

**Shortcuts as a grammar, not a list.** A flat pile of 200 aliases dies by
March. Vim has ~15 verbs and ~20 motions and yields thousands of operations
by composition. Target ~10 verbs × objects, not 200 commands.

**Every verb with no arguments drops into a fuzzy picker.**
`g r1` → straight to R1. `g` → pick from a list.
This means only *verbs* ever need memorizing, and the tool teaches itself.
Single most important interaction decision.

**Drop-in commands.** A directory where any script placed in it automatically
becomes a verb. Adding a shortcut is: write a file. No core edit, no rebuild,
no registration. This is what lets the vocabulary accumulate over years
instead of collapsing.

**Extract the vocabulary, don't invent it.**
```sh
history | awk '{$1="";print}' | sort | uniq -c | sort -rn | head -60
```
The real vocabulary is whatever has already been typed 400 times. It will not
match what gets designed from imagination — it never does, for anyone.

**Automate the scaffolding, never the learning.** Topology spin-up, snapshot,
restore, console jump, capture start — compress all of it. Leave the IOS
commands themselves uncompressed; typing them *is* the CCNA practice.

---

## Status

**Decided.** Thesis B is the data model; A's namespace/capability idea is left
as a seam for later. Go, because a single static binary starts in ~2ms and the
CLI latency problem disappears before the daemon is even needed.

The convergence that settled it: "one persistent queryable graph, no save, no
load" and "never pay process startup on the hot path" turn out to describe the
same architecture — a resident daemon holding the graph, with a thin client
talking to it over a unix socket. The philosophy and the performance answer
were the same decision.

First vertical slice is in `../lab/`: `up`, `down`, `go`, `ls`, `stat`, `help`
through every layer, mock and containerlab drivers, latency instrumented from
the first commit. ~3ms per invocation end to end.

Next, per the interaction notes above: **live in it for a couple of weeks
before adding verbs**, so the vocabulary comes from what actually gets typed.
