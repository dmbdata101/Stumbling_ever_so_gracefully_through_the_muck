// Package paths resolves where the socket, state, and topologies live.
//
// Every path is overridable by environment variable so a second instance can
// be run against a scratch directory without touching real state — which is
// what the test suite and the dev target do.
package paths

import (
	"os"
	"path/filepath"
	"strconv"
)

const (
	EnvSocket  = "LAB_SOCKET"
	EnvState   = "LAB_STATE_DIR"
	EnvTopoDir = "LAB_TOPO_DIR"
	EnvDriver  = "LAB_DRIVER"
)

// Socket is the unix socket the client talks to.
//
// XDG_RUNTIME_DIR is preferred because it is already per-user and cleaned on
// logout; the /tmp fallback carries the uid to stay per-user on machines
// without it.
func Socket() string {
	if s := os.Getenv(EnvSocket); s != "" {
		return s
	}
	if rt := os.Getenv("XDG_RUNTIME_DIR"); rt != "" {
		return filepath.Join(rt, "lab.sock")
	}
	return filepath.Join(os.TempDir(), "lab-"+strconv.Itoa(os.Getuid())+".sock")
}

// StateDir holds the journal.
func StateDir() string {
	if d := os.Getenv(EnvState); d != "" {
		return d
	}
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, "lab")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "lab-state")
	}
	return filepath.Join(home, ".local", "share", "lab")
}

// Journal is the append-only graph log.
func Journal() string { return filepath.Join(StateDir(), "graph.jsonl") }

// TopoDir holds topology definitions.
func TopoDir() string {
	if d := os.Getenv(EnvTopoDir); d != "" {
		return d
	}
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "lab", "topologies")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "topologies"
	}
	return filepath.Join(home, ".config", "lab", "topologies")
}
