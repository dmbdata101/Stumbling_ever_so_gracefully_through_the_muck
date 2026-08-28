// Package graph is the persistent object graph that replaces files.
//
// Design thesis: there are no files and no save/load. Every lab, node,
// capture, and event is a Node in one graph that lives in the daemon's
// memory and is durable because every mutation is appended to a journal as
// it happens. Restarting labd replays the journal; nothing is ever
// serialized or parsed on the read path.
//
// That is also why the daemon exists at all: holding the graph resident is
// what makes a query ~microseconds instead of a process start plus a parse.
package graph

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// Kinds currently in use. Kinds are just strings; adding one requires no
// schema change, which is the point.
const (
	KindLab   = "lab"
	KindNode  = "node"
	KindEvent = "event"
)

// Node is the only structure in the graph.
type Node struct {
	ID      string              `json:"id"`
	Kind    string              `json:"kind"`
	Attrs   map[string]string   `json:"attrs,omitempty"`
	Edges   map[string][]string `json:"edges,omitempty"`
	Created time.Time           `json:"created"`
	Updated time.Time           `json:"updated"`
}

func (n *Node) Attr(k string) string {
	if n.Attrs == nil {
		return ""
	}
	return n.Attrs[k]
}

// Clone returns a deep copy, so callers can never mutate graph state without
// going through Put.
func (n *Node) Clone() *Node {
	c := &Node{ID: n.ID, Kind: n.Kind, Created: n.Created, Updated: n.Updated}
	if n.Attrs != nil {
		c.Attrs = make(map[string]string, len(n.Attrs))
		for k, v := range n.Attrs {
			c.Attrs[k] = v
		}
	}
	if n.Edges != nil {
		c.Edges = make(map[string][]string, len(n.Edges))
		for k, v := range n.Edges {
			c.Edges[k] = append([]string(nil), v...)
		}
	}
	return c
}

// Graph is safe for concurrent use.
type Graph struct {
	mu    sync.RWMutex
	nodes map[string]*Node
	jrnl  *Journal
	now   func() time.Time // swappable in tests
}

// New returns a graph backed by jrnl, replaying whatever the journal holds.
// A nil journal makes the graph purely in-memory (used by tests).
func New(jrnl *Journal) (*Graph, error) {
	g := &Graph{nodes: map[string]*Node{}, jrnl: jrnl, now: time.Now}
	if jrnl == nil {
		return g, nil
	}
	ops, err := jrnl.Replay()
	if err != nil {
		return nil, fmt.Errorf("replay journal: %w", err)
	}
	for _, op := range ops {
		switch op.Op {
		case opPut:
			g.nodes[op.Node.ID] = op.Node
		case opDel:
			delete(g.nodes, op.ID)
		}
	}
	return g, nil
}

// Put inserts or replaces a node and durably records the change.
func (g *Graph) Put(n *Node) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	n = n.Clone()
	now := g.now()
	if prev, ok := g.nodes[n.ID]; ok {
		n.Created = prev.Created
	} else if n.Created.IsZero() {
		n.Created = now
	}
	n.Updated = now

	if g.jrnl != nil {
		if err := g.jrnl.Append(op{Op: opPut, Node: n}); err != nil {
			return fmt.Errorf("journal put %s: %w", n.ID, err)
		}
	}
	g.nodes[n.ID] = n
	return nil
}

// Set is a convenience Put for a node built from attrs.
func (g *Graph) Set(id, kind string, attrs map[string]string) error {
	return g.Put(&Node{ID: id, Kind: kind, Attrs: attrs})
}

// Del removes a node. Deleting a missing node is not an error.
func (g *Graph) Del(id string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, ok := g.nodes[id]; !ok {
		return nil
	}
	if g.jrnl != nil {
		if err := g.jrnl.Append(op{Op: opDel, ID: id}); err != nil {
			return fmt.Errorf("journal del %s: %w", id, err)
		}
	}
	delete(g.nodes, id)
	return nil
}

// Get returns a copy of a node, or nil.
func (g *Graph) Get(id string) *Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if n, ok := g.nodes[id]; ok {
		return n.Clone()
	}
	return nil
}

// Kind returns copies of every node of a kind, sorted by ID for stable output.
func (g *Graph) Kind(kind string) []*Node {
	g.mu.RLock()
	out := make([]*Node, 0, 8)
	for _, n := range g.nodes {
		if n.Kind == kind {
			out = append(out, n.Clone())
		}
	}
	g.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Len reports total node count.
func (g *Graph) Len() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.nodes)
}
