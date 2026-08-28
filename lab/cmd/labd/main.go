// Command labd holds the graph and serves the socket.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dmbdata101/lab/internal/daemon"
	"github.com/dmbdata101/lab/internal/driver"
	"github.com/dmbdata101/lab/internal/graph"
	"github.com/dmbdata101/lab/internal/paths"
)

func main() {
	socket := flag.String("socket", paths.Socket(), "unix socket to listen on")
	topo := flag.String("topo", paths.TopoDir(), "directory holding topology files")
	drv := flag.String("driver", os.Getenv(paths.EnvDriver), "containerlab|mock (default: autodetect)")
	journal := flag.String("journal", paths.Journal(), "append-only graph log")
	idle := flag.Duration("idle", 0, "stop after this long with no clients (0 = never)")
	flag.Parse()

	if err := run(*socket, *topo, *drv, *journal, *idle); err != nil {
		fmt.Fprintln(os.Stderr, "labd: "+err.Error())
		os.Exit(1)
	}
}

func run(socket, topo, drvName, journalPath string, idle time.Duration) error {
	jrnl, err := graph.OpenJournal(journalPath)
	if err != nil {
		return err
	}
	defer jrnl.Close()

	g, err := graph.New(jrnl)
	if err != nil {
		return err
	}

	d, err := selectDriver(drvName)
	if err != nil {
		return err
	}

	srv, err := daemon.Listen(daemon.Config{
		Socket: socket, Graph: g, Driver: d, TopoDir: topo,
		IdleTimeout: idle,
		Logger:      log.New(os.Stderr, "labd ", log.LstdFlags|log.Lmsgprefix),
	})
	if err != nil {
		return err
	}
	defer srv.Close()

	return srv.Serve(context.Background())
}

// selectDriver honours an explicit choice and otherwise prefers a real
// runtime, falling back to mock so the tool is usable on a machine that has
// not installed containerlab yet.
func selectDriver(name string) (driver.Driver, error) {
	clab := &driver.Containerlab{}
	mock := &driver.Mock{Delay: 40 * time.Millisecond}

	switch name {
	case "containerlab":
		if !clab.Available() {
			return nil, fmt.Errorf("driver containerlab requested but the binary is not on PATH")
		}
		return clab, nil
	case "mock":
		return mock, nil
	case "":
		if clab.Available() {
			return clab, nil
		}
		return mock, nil
	default:
		return nil, fmt.Errorf("unknown driver %q (want containerlab or mock)", name)
	}
}
