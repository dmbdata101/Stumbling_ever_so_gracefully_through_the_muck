package verbs

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dmbdata101/lab/internal/graph"
	"github.com/dmbdata101/lab/internal/proto"
)

func init() {
	Register(&Verb{
		Name: "up", Aliases: []string{"u"},
		Usage: "up [lab]", Short: "bring a topology up",
		Handler: up,
	})
	Register(&Verb{
		Name: "down", Aliases: []string{"d"},
		Usage: "down [lab]", Short: "tear a lab down",
		Handler: down,
	})
	Register(&Verb{
		Name: "go", Aliases: []string{"g"},
		Usage: "go [node]", Short: "open a console on a device",
		Handler: goTo,
	})
	Register(&Verb{
		Name: "ls", Aliases: []string{"l"},
		Usage: "ls", Short: "show topologies, labs and devices",
		Handler: ls,
	})
}

func up(c *Ctx) (*proto.Response, error) {
	if len(c.Args) == 0 {
		tops, err := c.D.Topologies(c.TopoDir)
		if err != nil {
			return nil, err
		}
		if len(tops) == 0 {
			return nil, fmt.Errorf("no topologies in %s", c.TopoDir)
		}
		choices := make([]proto.Choice, 0, len(tops))
		for _, t := range tops {
			label := t.Name
			if n := c.G.Get(labID(t.Name)); n != nil && n.Attr("state") == "up" {
				label += "  (already up)"
			}
			choices = append(choices, proto.Choice{ID: t.Name, Label: label})
		}
		return pick("up", "bring up which lab?", choices), nil
	}

	name := c.Args[0]
	topo, err := findTopology(c, name)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	devices, links, err := c.D.Up(c.Ctx, topo)
	if err != nil {
		return nil, err
	}
	elapsed := time.Since(start)

	// Peers are indexed per device so the graph answers "what is r1 wired to"
	// without re-reading the topology file.
	peers := map[string][]string{}
	for _, l := range links {
		peers[l.A] = append(peers[l.A], fmt.Sprintf("%s->%s:%s", l.APort, l.B, l.BPort))
		peers[l.B] = append(peers[l.B], fmt.Sprintf("%s->%s:%s", l.BPort, l.A, l.APort))
	}

	// A lab coming up replaces whatever devices the graph held for it.
	for _, old := range devicesOf(c.G, name) {
		if err := c.G.Del(old.ID); err != nil {
			return nil, err
		}
	}

	for _, d := range devices {
		n := &graph.Node{
			ID:   deviceID(name, d.Name),
			Kind: graph.KindNode,
			Attrs: map[string]string{
				"lab": name, "name": d.Name, "runtime": d.Runtime,
				"kind": d.Kind, "image": d.Image, "mgmt": d.MgmtIPv4,
				"state": stateWord(d.Reachable), "ports": strings.Join(peers[d.Name], " "),
			},
			Edges: map[string][]string{"lab": {labID(name)}},
		}
		if err := c.G.Put(n); err != nil {
			return nil, err
		}
	}

	if err := c.G.Set(labID(name), graph.KindLab, map[string]string{
		"name": name, "state": "up", "driver": c.D.Name(), "path": topo.Path,
		"devices": fmt.Sprint(len(devices)), "links": fmt.Sprint(len(links)),
		"up_ms": fmt.Sprintf("%.1f", float64(elapsed.Microseconds())/1000),
	}); err != nil {
		return nil, err
	}

	names := make([]string, 0, len(devices))
	for _, d := range devices {
		names = append(names, d.Name)
	}
	return okf("%s up via %s — %d devices (%s), %d links in %s",
		name, c.D.Name(), len(devices), strings.Join(names, " "), len(links),
		elapsed.Round(time.Millisecond)), nil
}

func stateWord(reachable bool) string {
	if reachable {
		return "running"
	}
	return "down"
}

func down(c *Ctx) (*proto.Response, error) {
	if len(c.Args) == 0 {
		labs := runningLabs(c.G)
		if len(labs) == 0 {
			return okf("nothing is up"), nil
		}
		choices := make([]proto.Choice, 0, len(labs))
		for _, l := range labs {
			choices = append(choices, proto.Choice{
				ID:    l.Attr("name"),
				Label: fmt.Sprintf("%s  (%s devices)", l.Attr("name"), l.Attr("devices")),
			})
		}
		return pick("down", "tear down which lab?", choices), nil
	}

	name := c.Args[0]
	topo, err := findTopology(c, name)
	if err != nil {
		return nil, err
	}
	if err := c.D.Down(c.Ctx, topo); err != nil {
		return nil, err
	}
	for _, d := range devicesOf(c.G, name) {
		if err := c.G.Del(d.ID); err != nil {
			return nil, err
		}
	}
	if err := c.G.Set(labID(name), graph.KindLab, map[string]string{
		"name": name, "state": "down", "driver": c.D.Name(), "path": topo.Path,
	}); err != nil {
		return nil, err
	}
	return okf("%s down", name), nil
}

func goTo(c *Ctx) (*proto.Response, error) {
	devices := c.G.Kind(graph.KindNode)

	if len(c.Args) == 0 {
		if len(devices) == 0 {
			return nil, fmt.Errorf("nothing is up — try `lab up`")
		}
		choices := make([]proto.Choice, 0, len(devices))
		for _, d := range devices {
			choices = append(choices, proto.Choice{
				ID: d.Attr("name"),
				Label: fmt.Sprintf("%-8s %-10s %s",
					d.Attr("name"), d.Attr("lab"), d.Attr("mgmt")),
			})
		}
		return pick("go", "console on which device?", choices), nil
	}

	want := c.Args[0]
	var matches []*graph.Node
	for _, d := range devices {
		if d.Attr("name") == want {
			matches = append(matches, d)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no running device named %q", want)
	case 1:
	default:
		// Same short name in two labs: ask rather than guess.
		choices := make([]proto.Choice, 0, len(matches))
		for _, d := range matches {
			choices = append(choices, proto.Choice{
				ID:    d.Attr("lab") + "/" + d.Attr("name"),
				Label: fmt.Sprintf("%s in %s", d.Attr("name"), d.Attr("lab")),
			})
		}
		return pick("go", fmt.Sprintf("%q is in several labs — which?", want), choices), nil
	}

	d := matches[0]
	dev := deviceFromNode(d)
	argv := c.D.AttachCmd(d.Attr("lab"), dev)
	return &proto.Response{OK: true, Exec: argv}, nil
}

func ls(c *Ctx) (*proto.Response, error) {
	tops, err := c.D.Topologies(c.TopoDir)
	if err != nil {
		return nil, err
	}

	var b strings.Builder
	rows := [][]string{{"LAB", "STATE", "DEVICES", "DRIVER"}}
	seen := map[string]bool{}
	for _, t := range tops {
		seen[t.Name] = true
		state, devs, drv := "defined", "-", "-"
		if n := c.G.Get(labID(t.Name)); n != nil {
			state = n.Attr("state")
			drv = n.Attr("driver")
			if state == "up" {
				devs = n.Attr("devices")
			}
		}
		rows = append(rows, []string{t.Name, state, devs, drv})
	}
	b.WriteString(column(rows))

	devices := c.G.Kind(graph.KindNode)
	if len(devices) > 0 {
		sort.Slice(devices, func(i, j int) bool {
			if a, c2 := devices[i].Attr("lab"), devices[j].Attr("lab"); a != c2 {
				return a < c2
			}
			return devices[i].Attr("name") < devices[j].Attr("name")
		})
		drows := [][]string{{"DEVICE", "LAB", "MGMT", "STATE", "LINKS"}}
		for _, d := range devices {
			links := d.Attr("ports")
			if links == "" {
				links = "-"
			}
			drows = append(drows, []string{
				d.Attr("name"), d.Attr("lab"), d.Attr("mgmt"), d.Attr("state"), links,
			})
		}
		b.WriteString("\n\n")
		b.WriteString(column(drows))
	}
	return &proto.Response{OK: true, Output: b.String()}, nil
}
