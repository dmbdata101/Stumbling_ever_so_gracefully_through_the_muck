package daemon_test

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dmbdata101/lab/internal/client"
	"github.com/dmbdata101/lab/internal/daemon"
	"github.com/dmbdata101/lab/internal/driver"
	"github.com/dmbdata101/lab/internal/graph"
	"github.com/dmbdata101/lab/internal/proto"
)

const topo = `name: ospf
topology:
  nodes:
    r1:
      kind: linux
    r2:
      kind: linux
  links:
    - endpoints: ["r1:eth1", "r2:eth1"]
`

// start brings up a real daemon on a real socket and returns a client for it.
// Exercising the actual protocol matters more than mocking it: the wire
// format is the contract between the two binaries.
func start(t *testing.T) (*client.Client, *graph.Graph) {
	t.Helper()

	dir := t.TempDir()
	topoDir := filepath.Join(dir, "topologies")
	if err := os.MkdirAll(topoDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(topoDir, "ospf.clab.yml"), []byte(topo), 0o600); err != nil {
		t.Fatal(err)
	}

	// Unix socket paths are capped near 100 bytes; t.TempDir() can be long.
	sock := filepath.Join(os.TempDir(), "labtest-"+strings.ReplaceAll(t.Name(), "/", "_")+".sock")
	os.Remove(sock)

	jrnl, err := graph.OpenJournal(filepath.Join(dir, "graph.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	g, err := graph.New(jrnl)
	if err != nil {
		t.Fatal(err)
	}

	srv, err := daemon.Listen(daemon.Config{
		Socket: sock, Graph: g, Driver: &driver.Mock{}, TopoDir: topoDir,
		Logger: log.New(io.Discard, "", 0),
	})
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); srv.Serve(ctx) }()
	t.Cleanup(func() {
		cancel()
		<-done
		srv.Close()
		jrnl.Close()
		os.Remove(sock)
	})

	return &client.Client{Socket: sock}, g
}

func send(t *testing.T, c *client.Client, verb string, args ...string) *proto.Response {
	t.Helper()
	resp, err := c.Send(&proto.Request{Verb: verb, Args: args})
	if err != nil {
		t.Fatalf("send %s %v: %v", verb, args, err)
	}
	return resp
}

func TestRoundTrip(t *testing.T) {
	c, _ := start(t)

	resp := send(t, c, "ls")
	if !resp.OK {
		t.Fatalf("ls failed: %s", resp.Error)
	}
	if !strings.Contains(resp.Output, "ospf") {
		t.Errorf("ls output = %q", resp.Output)
	}
	if resp.ElapsedMS <= 0 {
		t.Error("elapsed time not reported")
	}
}

func TestUpThenGoOverTheWire(t *testing.T) {
	c, _ := start(t)

	if resp := send(t, c, "up", "ospf"); !resp.OK {
		t.Fatalf("up failed: %s", resp.Error)
	}
	resp := send(t, c, "go", "r1")
	if !resp.OK || len(resp.Exec) == 0 {
		t.Fatalf("go = %+v, want an exec command", resp)
	}
}

func TestPickerRoundTrip(t *testing.T) {
	c, _ := start(t)

	resp := send(t, c, "up")
	if len(resp.Choices) == 0 || resp.PickVerb != "up" {
		t.Fatalf("bare up = %+v, want choices", resp)
	}
	// What the client does after the picker: re-send with the chosen ID.
	resp = send(t, c, resp.PickVerb, resp.Choices[0].ID)
	if !resp.OK || !strings.Contains(resp.Output, "up via mock") {
		t.Fatalf("picked up = %+v", resp)
	}
}

func TestUnknownVerb(t *testing.T) {
	c, _ := start(t)
	resp := send(t, c, "frobnicate")
	if resp.OK || !strings.Contains(resp.Error, "unknown verb") {
		t.Fatalf("unknown verb = %+v", resp)
	}
}

// Aliases and canonical names must land in the same bucket, or stat hides
// which verbs actually dominate usage.
func TestEventsRecordCanonicalVerb(t *testing.T) {
	c, g := start(t)

	send(t, c, "ls")
	send(t, c, "l")

	var ls int
	for _, e := range g.Kind(graph.KindEvent) {
		if e.Attr("verb") == "ls" {
			ls++
		}
		if e.Attr("verb") == "l" {
			t.Error("event recorded the alias instead of the canonical verb")
		}
	}
	if ls != 2 {
		t.Errorf("recorded %d ls events, want 2", ls)
	}
}

func TestStatIsNotSelfRecording(t *testing.T) {
	c, g := start(t)
	send(t, c, "stat")
	send(t, c, "?")

	for _, e := range g.Kind(graph.KindEvent) {
		if e.Attr("verb") == "stat" {
			t.Fatal("stat recorded itself")
		}
	}
}

func TestClientRefusesWhenDaemonAbsent(t *testing.T) {
	c := &client.Client{Socket: filepath.Join(t.TempDir(), "nope.sock")}
	if _, err := c.Send(&proto.Request{Verb: "ls"}); err == nil {
		t.Fatal("expected an error when no daemon is listening and AutoStart is off")
	}
}

func TestSecondDaemonRefusesLiveSocket(t *testing.T) {
	c, _ := start(t)
	// c.Socket is live; a second Listen on it must refuse rather than unlink.
	_, err := daemon.Listen(daemon.Config{
		Socket: c.Socket, Graph: mustGraph(t), Driver: &driver.Mock{},
		Logger: log.New(io.Discard, "", 0),
	})
	if err == nil {
		t.Fatal("second daemon bound a live socket")
	}
	if !strings.Contains(err.Error(), "already running") {
		t.Errorf("error = %v, want 'already running'", err)
	}
}

func TestStaleSocketIsReclaimed(t *testing.T) {
	sock := filepath.Join(os.TempDir(), "labtest-stale.sock")
	os.Remove(sock)
	if err := os.WriteFile(sock, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(sock)

	srv, err := daemon.Listen(daemon.Config{
		Socket: sock, Graph: mustGraph(t), Driver: &driver.Mock{},
		Logger: log.New(io.Discard, "", 0),
	})
	if err != nil {
		t.Fatalf("Listen over a stale socket: %v", err)
	}
	srv.Close()
}

func TestIdleTimeoutStopsDaemon(t *testing.T) {
	sock := filepath.Join(os.TempDir(), "labtest-idle.sock")
	os.Remove(sock)
	defer os.Remove(sock)

	srv, err := daemon.Listen(daemon.Config{
		Socket: sock, Graph: mustGraph(t), Driver: &driver.Mock{},
		IdleTimeout: 200 * time.Millisecond,
		Logger:      log.New(io.Discard, "", 0),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	done := make(chan error, 1)
	go func() { done <- srv.Serve(context.Background()) }()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Serve returned %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("daemon did not stop after its idle timeout")
	}
}

func mustGraph(t *testing.T) *graph.Graph {
	t.Helper()
	g, err := graph.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return g
}
