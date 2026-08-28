// Package client is the thin half.
//
// It must stay tiny: stdlib only, no init work, no config parsing. Everything
// it does is on the critical path of every command you run, and the whole
// point of the daemon is that this side costs nothing.
package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/dmbdata101/lab/internal/proto"
)

// Client talks to labd over a unix socket.
type Client struct {
	Socket string
	// AutoStart launches labd when nothing is listening. The daemon is an
	// implementation detail; you should never have to think about it.
	AutoStart bool
}

// Send performs one request/response round trip, starting labd if needed.
func (c *Client) Send(req *proto.Request) (*proto.Response, error) {
	conn, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Minute))

	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Write(append(b, '\n')); err != nil {
		return nil, fmt.Errorf("send: %w", err)
	}

	var resp proto.Response
	if err := json.NewDecoder(bufio.NewReader(conn)).Decode(&resp); err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	return &resp, nil
}

func (c *Client) dial() (net.Conn, error) {
	conn, err := net.DialTimeout("unix", c.Socket, 500*time.Millisecond)
	if err == nil {
		return conn, nil
	}
	if !c.AutoStart {
		return nil, fmt.Errorf("labd not running (%s)", c.Socket)
	}
	if err := c.startDaemon(); err != nil {
		return nil, err
	}
	return c.waitForSocket(3 * time.Second)
}

// startDaemon launches labd detached, preferring the copy sitting next to
// this binary so a built tree never picks up a stale one from PATH.
func (c *Client) startDaemon() error {
	bin := "labd"
	if self, err := os.Executable(); err == nil {
		sibling := filepath.Join(filepath.Dir(self), "labd")
		if _, err := os.Stat(sibling); err == nil {
			bin = sibling
		}
	}
	if bin == "labd" {
		if _, err := exec.LookPath("labd"); err != nil {
			return fmt.Errorf("labd not found on PATH or beside %s", os.Args[0])
		}
	}

	cmd := exec.Command(bin)
	cmd.Stdin = nil
	// The daemon's log is its own business; inheriting our stderr would
	// interleave its output into every command you run.
	logPath := c.Socket + ".log"
	if lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); err == nil {
		cmd.Stdout, cmd.Stderr = lf, lf
		defer lf.Close()
	}
	cmd.SysProcAttr = detach()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start labd: %w", err)
	}
	// Not waited on deliberately: the daemon outlives this process. Release
	// so it is reparented to init rather than left a zombie.
	return cmd.Process.Release()
}

func (c *Client) waitForSocket(limit time.Duration) (net.Conn, error) {
	deadline := time.Now().Add(limit)
	delay := 2 * time.Millisecond
	for time.Now().Before(deadline) {
		if conn, err := net.DialTimeout("unix", c.Socket, 250*time.Millisecond); err == nil {
			return conn, nil
		}
		time.Sleep(delay)
		if delay < 100*time.Millisecond {
			delay *= 2
		}
	}
	return nil, fmt.Errorf("labd did not come up within %s (see %s.log)", limit, c.Socket)
}
