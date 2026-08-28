# lab

A keyboard-driven console for network labs and security testing. Terminal
only, no mouse, built around a small verb grammar rather than a pile of
aliases.

This is the first vertical slice: `up`, `down`, `go`, `ls`, `stat`, `help`
working end to end through every layer. It is deliberately thin rather than
broad — the next verb costs an hour because the layers underneath already
exist.

## Design

Three decisions drive everything else.

**One persistent graph, no files.** Labs, devices, links, and command events
are all nodes in a single graph held in `labd`. There is no save, no load, and
no parsing on the read path. Durability comes from appending every mutation to
a journal as it happens (`internal/graph`), replayed on start.

**A resident daemon and a thin client.** `lab` opens a unix socket, sends one
JSON message, and exits. All state lives in `labd`. This is why a command
answers in ~1ms instead of paying process startup plus loading state — and it
is the same architecture the graph model requires anyway.

**Every verb with no argument opens a picker.** `go r1` jumps straight to a
console; bare `go` lists the devices and lets you choose. Only *verbs* ever
have to be remembered, and the fast path gets learned by using the slow one.
The daemon never touches the terminal — it returns choices and the client
asks.

## Use

    make build
    eval $(make devenv)     # scratch socket/journal, mock driver, in-tree topologies

    lab up ospf             # bring up a topology       (alias: u)
    lab ls                  # what exists and what is running   (l)
    lab go r1               # console on a device       (g)
    lab go                  # ...or pick one
    lab down ospf           # tear it down              (d)
    lab stat                # latency of every verb you have run
    lab help                # the vocabulary            (h)

`labd` starts itself on first use and stops after its idle timeout. You should
never have to think about it.

### Environment

| Variable | Meaning |
|---|---|
| `LAB_SOCKET` | socket path (default `$XDG_RUNTIME_DIR/lab.sock`) |
| `LAB_STATE_DIR` | journal location (default `~/.local/share/lab`) |
| `LAB_TOPO_DIR` | topology files (default `~/.config/lab/topologies`) |
| `LAB_DRIVER` | `containerlab` or `mock` (default: containerlab if installed) |
| `LAB_TIME` | set to print each command's server-side latency |

## Drivers

`driver.Driver` abstracts whatever actually runs a topology. Two exist:

- **containerlab** — shells out to `containerlab deploy` / `destroy` / `inspect`.
- **mock** — runs nothing, but reads the real topology file and reports the
  devices and links it declares. Not only for tests: developing against it is
  what keeps the iteration loop under a second, and it means the tool works on
  a machine that has not installed containerlab yet.

GNS3 or libvirt slot in here without touching the verbs, graph, or client.

## Latency budget

`lab stat` reports p50/p90/max per verb from real usage, because optimising a
personal tool by vibes is how it ends up fast in the wrong places.

| Interaction | Budget |
|---|---|
| command → result, local | < 100ms |
| anything slower | must show progress |
| > 1s | must be cancellable, must not block |

## Adding a verb

One `Register` call in `internal/verbs`. Return `pick(...)` when an argument is
missing and the picker happens for free. That seam is what lets the vocabulary
grow for years without the core changing.

## Layout

    cmd/lab            thin client — send, print, prompt, or exec
    cmd/labd           daemon — holds the graph, serves the socket
    internal/proto     wire format
    internal/graph     the object graph and its journal
    internal/verbs     the command vocabulary
    internal/driver    containerlab / mock
    internal/picker    fzf when present, builtin otherwise
    internal/daemon    socket server, dispatch, latency recording
    internal/client    dial, autostart
    topologies/        lab definitions

## Not yet

Captures, verification (`lab check`), snapshot/restore, the TUI dashboard, and
drop-in external commands. Deliberately: the design notes in `../design/` argue
for living in this for a couple of weeks first, so the next verbs come from
what actually gets typed rather than what seemed likely.
