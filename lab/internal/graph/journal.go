package graph

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type opKind string

const (
	opPut opKind = "put"
	opDel opKind = "del"
)

type op struct {
	Op   opKind `json:"op"`
	ID   string `json:"id,omitempty"`
	Node *Node  `json:"node,omitempty"`
}

// Journal is an append-only log of graph mutations, one JSON object per line.
//
// Append-only because it makes a write a single buffered write plus fsync-free
// append: mutations stay off the latency budget. Compaction is a future
// concern; a personal lab produces a trivial number of ops.
type Journal struct {
	mu   sync.Mutex
	path string
	f    *os.File
	w    *bufio.Writer
}

// OpenJournal opens (creating as needed) the journal at path.
func OpenJournal(path string) (*Journal, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create state dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open journal: %w", err)
	}
	return &Journal{path: path, f: f, w: bufio.NewWriter(f)}, nil
}

// Replay reads every op recorded so far.
//
// A truncated final line (killed mid-write) is tolerated and dropped: losing
// the last mutation is strictly better than refusing to start.
func (j *Journal) Replay() ([]op, error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	f, err := os.Open(j.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var ops []op
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var o op
		if err := json.Unmarshal(line, &o); err != nil {
			// Truncated tail. Anything earlier failing to parse is a real
			// corruption, but we cannot tell the difference cheaply, and
			// stopping here preserves every op we did understand.
			break
		}
		ops = append(ops, o)
	}
	return ops, sc.Err()
}

func (j *Journal) Append(o op) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	b, err := json.Marshal(o)
	if err != nil {
		return err
	}
	if _, err := j.w.Write(append(b, '\n')); err != nil {
		return err
	}
	return j.w.Flush()
}

func (j *Journal) Close() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.f == nil {
		return nil
	}
	if err := j.w.Flush(); err != nil {
		j.f.Close()
		j.f = nil
		return err
	}
	err := j.f.Close()
	j.f = nil
	return err
}
