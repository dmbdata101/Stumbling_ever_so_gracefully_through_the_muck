# Handoff — read this first

State as of 2026-08-28, branch `claude/linux-omarchy-thoughts-olgjsz`.

## What this repo is becoming

A keyboard-driven terminal console for network labs and security testing,
built around a small verb grammar. It is **not** a distro, and **not** an OS —
both were considered and ruled out for stated reasons in
`design/00-paradigm-notes.md`. Read that before proposing either again.

## What exists

- `design/00-paradigm-notes.md` — the reasoning, the reading list, the
  decision, and the interaction rules that hold regardless of implementation.
- `lab/` — Go, no external dependencies. First vertical slice: `up`, `down`,
  `go`, `ls`, `stat`, `help`. See `lab/README.md` for design and usage.

Verified: `make check` passes, `go test -race ./...` clean, ~3ms per command
end to end, 128ms rebuild.

## The three decisions the code rests on

1. **One persistent object graph, no files.** No save, no load, no parsing on
   the read path. Durability is an append-only journal replayed on start.
2. **Resident daemon + thin client over a unix socket.** This is what keeps a
   command at ~3ms, and it is the same architecture the graph model requires.
   The client autostarts the daemon.
3. **Every verb invoked bare returns choices, not an error.** Only verbs need
   remembering. The daemon never touches the terminal — it returns choices and
   the client asks. Console attach hands an argv back for the client to exec.

Do not "simplify" any of these away. Each is load-bearing for the others.

## Next session

**Blocked on the user, and genuinely blocking:** they were asked to run

    history | awk '{$1="";print}' | sort | uniq -c | sort -rn | head -60

on their own machine and paste the top 40. The next verbs should come from
what they actually type, not from what seems likely. Ask for it before
designing new verbs.

Ranked, once that lands:

1. `lab check` — did the lab converge. First verb that isn't a wrapper. ~1h.
2. Drop-in verbs — a directory where any script becomes a verb. ~1h.
3. Exercise the containerlab driver against a real containerlab. It is
   written but has only ever run against the mock driver; there was no
   containerlab binary in the build container.
4. `lab cap`, snapshot/restore — after real use, not before.
5. Bubble Tea TUI — deliberately last, once `lab stat` shows what gets stared at.

**Standing instruction from the user's own reasoning: live in it for two weeks
before adding verbs.** Resist building ahead of use.

## Still open, deliberately

- Thesis A (capabilities / per-program namespaces) is a seam, not implemented.
- The from-scratch OS is parked, not dead. Separate project if it returns.
- The Ansible-on-stock-Ubuntu lab platform was recommended and never built.
