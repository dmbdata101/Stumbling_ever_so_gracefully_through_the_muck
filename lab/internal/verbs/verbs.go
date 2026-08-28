// Package verbs is the command vocabulary.
//
// Design rule from the outset: shortcuts are a grammar, not a list. A small
// set of verbs composes over objects, and every verb invoked with no argument
// returns Choices instead of an error — so only verbs ever have to be
// remembered, and the tool teaches itself.
//
// Adding a verb is one Register call. That is the seam that lets the
// vocabulary accumulate for years without the core changing.
package verbs

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/dmbdata101/lab/internal/driver"
	"github.com/dmbdata101/lab/internal/graph"
	"github.com/dmbdata101/lab/internal/proto"
)

// Ctx is what a handler gets.
type Ctx struct {
	Ctx     context.Context
	G       *graph.Graph
	D       driver.Driver
	TopoDir string
	Args    []string
}

// Handler runs a verb.
type Handler func(*Ctx) (*proto.Response, error)

// Verb is one entry in the vocabulary.
type Verb struct {
	Name    string
	Aliases []string
	Usage   string
	Short   string
	Handler Handler
}

var registry = map[string]*Verb{}
var order []*Verb

// Register adds a verb. Panics on a duplicate name or alias, because a
// silently shadowed verb is the kind of bug you find six months later.
func Register(v *Verb) {
	claim := func(k string) {
		if _, dup := registry[k]; dup {
			panic("verbs: duplicate name or alias " + k)
		}
		registry[k] = v
	}
	claim(v.Name)
	for _, a := range v.Aliases {
		claim(a)
	}
	order = append(order, v)
	sort.Slice(order, func(i, j int) bool { return order[i].Name < order[j].Name })
}

// Lookup resolves a name or alias.
func Lookup(name string) (*Verb, bool) {
	v, ok := registry[name]
	return v, ok
}

// All returns every registered verb, once each, sorted by name.
func All() []*Verb { return append([]*Verb(nil), order...) }

// ---- helpers shared by handlers ----

func okf(format string, a ...any) *proto.Response {
	return &proto.Response{OK: true, Output: fmt.Sprintf(format, a...)}
}

// pick builds the "you didn't give me an argument" response.
func pick(verb, title string, choices []proto.Choice) *proto.Response {
	return &proto.Response{OK: true, PickVerb: verb, PickTitle: title, Choices: choices}
}

func labID(name string) string         { return "lab/" + name }
func deviceID(lab, node string) string { return "node/" + lab + "/" + node }

// runningLabs returns lab nodes currently marked up.
func runningLabs(g *graph.Graph) []*graph.Node {
	var out []*graph.Node
	for _, n := range g.Kind(graph.KindLab) {
		if n.Attr("state") == "up" {
			out = append(out, n)
		}
	}
	return out
}

// devicesOf returns device nodes belonging to a lab.
func devicesOf(g *graph.Graph, lab string) []*graph.Node {
	var out []*graph.Node
	for _, n := range g.Kind(graph.KindNode) {
		if n.Attr("lab") == lab {
			out = append(out, n)
		}
	}
	return out
}

// findTopology resolves a lab name against the topology directory.
func findTopology(c *Ctx, name string) (driver.Topology, error) {
	tops, err := c.D.Topologies(c.TopoDir)
	if err != nil {
		return driver.Topology{}, err
	}
	for _, t := range tops {
		if t.Name == name {
			return t, nil
		}
	}
	return driver.Topology{}, fmt.Errorf("no topology named %q in %s", name, c.TopoDir)
}

// column renders aligned rows without a table dependency.
func column(rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}
	widths := make([]int, len(rows[0]))
	for _, r := range rows {
		for i, cell := range r {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	var b strings.Builder
	for _, r := range rows {
		for i, cell := range r {
			if i == len(r)-1 {
				b.WriteString(cell)
			} else {
				b.WriteString(cell)
				b.WriteString(strings.Repeat(" ", widths[i]-len(cell)+2))
			}
		}
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}
