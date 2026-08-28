package verbs

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/dmbdata101/lab/internal/driver"
	"github.com/dmbdata101/lab/internal/graph"
	"github.com/dmbdata101/lab/internal/proto"
)

func init() {
	Register(&Verb{
		Name: "stat", Aliases: []string{"?"},
		Usage: "stat", Short: "latency of every verb you have run",
		Handler: stat,
	})
	Register(&Verb{
		Name: "help", Aliases: []string{"h"},
		Usage: "help", Short: "the vocabulary",
		Handler: help,
	})
}

func deviceFromNode(n *graph.Node) driver.Device {
	return driver.Device{
		Name:      n.Attr("name"),
		Runtime:   n.Attr("runtime"),
		Kind:      n.Attr("kind"),
		Image:     n.Attr("image"),
		MgmtIPv4:  n.Attr("mgmt"),
		Reachable: n.Attr("state") == "running",
	}
}

// stat reports what you actually run and what is actually slow.
//
// Instrumented from the first commit on purpose: optimising a personal tool
// by vibes is how it ends up fast in the wrong places.
func stat(c *Ctx) (*proto.Response, error) {
	events := c.G.Kind(graph.KindEvent)
	if len(events) == 0 {
		return okf("no commands recorded yet"), nil
	}

	byVerb := map[string][]float64{}
	for _, e := range events {
		ms, err := strconv.ParseFloat(e.Attr("ms"), 64)
		if err != nil {
			continue
		}
		byVerb[e.Attr("verb")] = append(byVerb[e.Attr("verb")], ms)
	}

	names := make([]string, 0, len(byVerb))
	for v := range byVerb {
		names = append(names, v)
	}
	// Most-run first: the top of this list is where effort pays off.
	sort.Slice(names, func(i, j int) bool {
		if a, b := len(byVerb[names[i]]), len(byVerb[names[j]]); a != b {
			return a > b
		}
		return names[i] < names[j]
	})

	rows := [][]string{{"VERB", "RUNS", "P50", "P90", "MAX", "BUDGET"}}
	for _, v := range names {
		s := byVerb[v]
		sort.Float64s(s)
		p50, p90, max := pctile(s, 0.50), pctile(s, 0.90), s[len(s)-1]
		rows = append(rows, []string{
			v, strconv.Itoa(len(s)),
			ms(p50), ms(p90), ms(max), budget(p90),
		})
	}
	return &proto.Response{OK: true, Output: column(rows) +
		"\n\nbudget: <100ms local command, >300ms must show progress"}, nil
}

func pctile(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	i := int(q * float64(len(sorted)-1))
	return sorted[i]
}

func ms(v float64) string { return fmt.Sprintf("%.1fms", v) }

func budget(p90 float64) string {
	switch {
	case p90 < 100:
		return "ok"
	case p90 < 300:
		return "watch"
	default:
		return "OVER"
	}
}

func help(c *Ctx) (*proto.Response, error) {
	rows := [][]string{{"VERB", "ALIAS", "USAGE", ""}}
	for _, v := range All() {
		rows = append(rows, []string{v.Name, strings.Join(v.Aliases, ","), v.Usage, v.Short})
	}
	return &proto.Response{OK: true, Output: column(rows) +
		"\n\nEvery verb with no argument opens a picker, so only verbs need remembering."}, nil
}
