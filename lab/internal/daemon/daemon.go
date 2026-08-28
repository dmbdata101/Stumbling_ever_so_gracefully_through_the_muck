// Package daemon is the resident half of the tool.
//
// It exists for one reason: a command must answer in about a millisecond, and
// you cannot do that if every invocation pays process startup plus loading
// and parsing state. Keeping the graph resident collapses that to a socket
// round trip — which is also, conveniently, exactly what "one persistent
// queryable graph, no save, no load" requires.
package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/dmbdata101/lab/internal/driver"
	"github.com/dmbdata101/lab/internal/graph"
	"github.com/dmbdata101/lab/internal/proto"
	"github.com/dmbdata101/lab/internal/verbs"
)

// Config configures a daemon.
type Config struct {
	Socket  string
	Graph   *graph.Graph
	Driver  driver.Driver
	TopoDir string
	// IdleTimeout stops the daemon after this long with no connections, so a
	// forgotten daemon does not outlive its usefulness. Zero disables it.
	IdleTimeout time.Duration
	Logger      *log.Logger
}

// Server serves one socket.
type Server struct {
	cfg      Config
	ln       net.Listener
	lastSeen atomic.Int64
	seq      atomic.Uint64
}

// Listen binds the socket.
//
// A socket left behind by a killed daemon is removed only after a connect
// attempt proves nothing is listening — unlinking a live daemon's socket
// would silently orphan it.
func Listen(cfg Config) (*Server, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.Socket), 0o700); err != nil {
		return nil, fmt.Errorf("create socket dir: %w", err)
	}
	if _, err := os.Stat(cfg.Socket); err == nil {
		if c, derr := net.DialTimeout("unix", cfg.Socket, 250*time.Millisecond); derr == nil {
			c.Close()
			return nil, fmt.Errorf("labd already running on %s", cfg.Socket)
		}
		if err := os.Remove(cfg.Socket); err != nil {
			return nil, fmt.Errorf("remove stale socket: %w", err)
		}
	}
	ln, err := net.Listen("unix", cfg.Socket)
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}
	if err := os.Chmod(cfg.Socket, 0o600); err != nil {
		ln.Close()
		return nil, fmt.Errorf("chmod socket: %w", err)
	}
	if cfg.Logger == nil {
		cfg.Logger = log.New(os.Stderr, "labd ", log.LstdFlags)
	}
	s := &Server{cfg: cfg, ln: ln}
	s.lastSeen.Store(time.Now().UnixNano())
	return s, nil
}

// Serve accepts until the context is cancelled, a signal arrives, or the idle
// timeout expires.
func (s *Server) Serve(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go func() {
		<-ctx.Done()
		s.ln.Close()
	}()
	if s.cfg.IdleTimeout > 0 {
		go s.reapWhenIdle(ctx, cancel)
	}

	s.cfg.Logger.Printf("listening on %s (driver=%s, %d nodes in graph)",
		s.cfg.Socket, s.cfg.Driver.Name(), s.cfg.Graph.Len())

	for {
		conn, err := s.ln.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("accept: %w", err)
		}
		s.lastSeen.Store(time.Now().UnixNano())
		go s.handle(ctx, conn)
	}
}

func (s *Server) reapWhenIdle(ctx context.Context, cancel context.CancelFunc) {
	t := time.NewTicker(s.cfg.IdleTimeout / 4)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			idle := time.Since(time.Unix(0, s.lastSeen.Load()))
			if idle >= s.cfg.IdleTimeout {
				s.cfg.Logger.Printf("idle for %s, stopping", idle.Round(time.Second))
				cancel()
				return
			}
		}
	}
}

func (s *Server) handle(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Minute))

	var req proto.Request
	if err := json.NewDecoder(bufio.NewReader(conn)).Decode(&req); err != nil {
		writeResp(conn, &proto.Response{OK: false, Error: "malformed request: " + err.Error()})
		return
	}

	start := time.Now()
	resp, canonical := s.dispatch(ctx, &req)
	elapsed := time.Since(start)
	resp.ElapsedMS = float64(elapsed.Microseconds()) / 1000

	s.record(canonical, &req, resp, elapsed)
	writeResp(conn, resp)
}

func (s *Server) dispatch(ctx context.Context, req *proto.Request) (*proto.Response, string) {
	v, ok := verbs.Lookup(req.Verb)
	if !ok {
		return &proto.Response{OK: false,
			Error: fmt.Sprintf("unknown verb %q — try `lab help`", req.Verb)}, req.Verb
	}
	resp, err := v.Handler(&verbs.Ctx{
		Ctx: ctx, G: s.cfg.Graph, D: s.cfg.Driver,
		TopoDir: s.cfg.TopoDir, Args: req.Args,
	})
	if err != nil {
		return &proto.Response{OK: false, Error: err.Error()}, v.Name
	}
	if resp == nil {
		return &proto.Response{OK: true}, v.Name
	}
	return resp, v.Name
}

// record writes the latency of every command into the graph itself, so `lab
// stat` reports on real usage rather than a benchmark.
//
// It records the canonical verb name, not what was typed: `u` and `up` are
// the same operation, and splitting them would hide which verbs actually
// dominate usage.
func (s *Server) record(canonical string, req *proto.Request, resp *proto.Response, elapsed time.Duration) {
	if canonical == "stat" {
		return // measuring the measurement is noise
	}
	id := fmt.Sprintf("event/%d-%d", time.Now().UnixNano(), s.seq.Add(1))
	err := s.cfg.Graph.Set(id, graph.KindEvent, map[string]string{
		"verb":  canonical,
		"typed": req.Verb,
		"args":  fmt.Sprint(req.Args),
		"ms":    strconv.FormatFloat(float64(elapsed.Microseconds())/1000, 'f', 3, 64),
		"ok":    strconv.FormatBool(resp.OK),
	})
	if err != nil {
		s.cfg.Logger.Printf("record event: %v", err)
	}
}

func writeResp(conn net.Conn, resp *proto.Response) {
	b, err := json.Marshal(resp)
	if err != nil {
		b, _ = json.Marshal(&proto.Response{OK: false, Error: "marshal response: " + err.Error()})
	}
	conn.Write(append(b, '\n'))
}

// Close releases the socket.
func (s *Server) Close() error {
	err := s.ln.Close()
	os.Remove(s.cfg.Socket)
	return err
}
