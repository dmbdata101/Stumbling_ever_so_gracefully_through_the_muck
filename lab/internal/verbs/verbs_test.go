package verbs

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func newCtx(t *testing.T, args ...string) *Ctx {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ospf.clab.yml"), []byte(topo), 0o600); err != nil {
		t.Fatal(err)
	}
	g, err := graph.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &Ctx{Ctx: context.Background(), G: g, D: &driver.Mock{}, TopoDir: dir, Args: args}
}

func run(t *testing.T, c *Ctx, verb string, args ...string) *proto.Response {
	t.Helper()
	v, ok := Lookup(verb)
	if !ok {
		t.Fatalf("verb %q not registered", verb)
	}
	c.Args = args
	resp, err := v.Handler(c)
	if err != nil {
		t.Fatalf("%s %v: %v", verb, args, err)
	}
	return resp
}

func TestAliasesResolve(t *testing.T) {
	for alias, want := range map[string]string{
		"u": "up", "d": "down", "g": "go", "l": "ls", "h": "help", "?": "stat",
	} {
		v, ok := Lookup(alias)
		if !ok {
			t.Errorf("alias %q not registered", alias)
			continue
		}
		if v.Name != want {
			t.Errorf("alias %q -> %q, want %q", alias, v.Name, want)
		}
	}
}

func TestUpPopulatesGraph(t *testing.T) {
	c := newCtx(t)
	resp := run(t, c, "up", "ospf")

	if !resp.OK || !strings.Contains(resp.Output, "2 devices") {
		t.Fatalf("up output = %q", resp.Output)
	}
	lab := c.G.Get("lab/ospf")
	if lab == nil || lab.Attr("state") != "up" {
		t.Fatalf("lab node = %+v", lab)
	}
	devices := c.G.Kind(graph.KindNode)
	if len(devices) != 2 {
		t.Fatalf("got %d devices, want 2", len(devices))
	}
	// Peer wiring should be queryable from the graph, not re-read from disk.
	r1 := c.G.Get("node/ospf/r1")
	if r1 == nil || !strings.Contains(r1.Attr("ports"), "r2:eth1") {
		t.Errorf("r1 ports = %q, want a link to r2", r1.Attr("ports"))
	}
}

func TestUpIsIdempotent(t *testing.T) {
	c := newCtx(t)
	run(t, c, "up", "ospf")
	run(t, c, "up", "ospf")

	if n := len(c.G.Kind(graph.KindNode)); n != 2 {
		t.Fatalf("after two ups there are %d devices, want 2", n)
	}
}

func TestUpUnknownLab(t *testing.T) {
	c := newCtx(t, "nope")
	v, _ := Lookup("up")
	if _, err := v.Handler(c); err == nil {
		t.Fatal("expected an error for an unknown topology")
	}
}

func TestBareVerbOffersChoices(t *testing.T) {
	c := newCtx(t)

	resp := run(t, c, "up")
	if len(resp.Choices) != 1 || resp.PickVerb != "up" {
		t.Fatalf("bare up = %+v, want a picker for 1 topology", resp)
	}

	run(t, c, "up", "ospf")
	resp = run(t, c, "go")
	if len(resp.Choices) != 2 || resp.PickVerb != "go" {
		t.Fatalf("bare go = %+v, want a picker over 2 devices", resp)
	}
	resp = run(t, c, "down")
	if len(resp.Choices) != 1 || resp.PickVerb != "down" {
		t.Fatalf("bare down = %+v, want a picker over 1 running lab", resp)
	}
}

func TestGoReturnsExec(t *testing.T) {
	c := newCtx(t)
	run(t, c, "up", "ospf")

	resp := run(t, c, "go", "r1")
	if len(resp.Exec) == 0 {
		t.Fatal("go returned no exec command")
	}
	if resp.Output != "" {
		t.Error("go should hand back a command, not print")
	}
}

func TestGoUnknownDevice(t *testing.T) {
	c := newCtx(t)
	run(t, c, "up", "ospf")
	v, _ := Lookup("go")
	c.Args = []string{"r9"}
	if _, err := v.Handler(c); err == nil {
		t.Fatal("expected an error for an unknown device")
	}
}

// The same short name in two labs must ask rather than pick one.
func TestGoAmbiguousAsks(t *testing.T) {
	c := newCtx(t)
	run(t, c, "up", "ospf")
	c.G.Set("node/other/r1", graph.KindNode, map[string]string{
		"lab": "other", "name": "r1", "state": "running",
	})

	resp := run(t, c, "go", "r1")
	if len(resp.Choices) != 2 {
		t.Fatalf("ambiguous go = %+v, want 2 choices", resp)
	}
}

func TestDownClearsDevices(t *testing.T) {
	c := newCtx(t)
	run(t, c, "up", "ospf")
	run(t, c, "down", "ospf")

	if n := len(c.G.Kind(graph.KindNode)); n != 0 {
		t.Errorf("%d devices left after down", n)
	}
	if lab := c.G.Get("lab/ospf"); lab.Attr("state") != "down" {
		t.Errorf("lab state = %q after down", lab.Attr("state"))
	}
}

func TestDownWithNothingUp(t *testing.T) {
	c := newCtx(t)
	resp := run(t, c, "down")
	if len(resp.Choices) != 0 || !strings.Contains(resp.Output, "nothing is up") {
		t.Fatalf("bare down with nothing up = %+v", resp)
	}
}

func TestLsShowsDefinedAndRunning(t *testing.T) {
	c := newCtx(t)

	resp := run(t, c, "ls")
	if !strings.Contains(resp.Output, "defined") {
		t.Errorf("ls before up = %q", resp.Output)
	}

	run(t, c, "up", "ospf")
	resp = run(t, c, "ls")
	for _, want := range []string{"ospf", "up", "r1", "r2"} {
		if !strings.Contains(resp.Output, want) {
			t.Errorf("ls output missing %q:\n%s", want, resp.Output)
		}
	}
}

func TestStatSummarisesEvents(t *testing.T) {
	c := newCtx(t)
	for _, ms := range []string{"1.0", "5.0", "9.0"} {
		c.G.Set("event/"+ms, graph.KindEvent, map[string]string{"verb": "ls", "ms": ms, "ok": "true"})
	}
	resp := run(t, c, "stat")
	if !strings.Contains(resp.Output, "ls") || !strings.Contains(resp.Output, "3") {
		t.Fatalf("stat = %q", resp.Output)
	}
}

func TestStatWithNoEvents(t *testing.T) {
	c := newCtx(t)
	if resp := run(t, c, "stat"); !strings.Contains(resp.Output, "no commands recorded") {
		t.Errorf("stat with no events = %q", resp.Output)
	}
}

func TestHelpListsEveryVerb(t *testing.T) {
	c := newCtx(t)
	resp := run(t, c, "help")
	for _, v := range All() {
		if !strings.Contains(resp.Output, v.Name) {
			t.Errorf("help omits %q", v.Name)
		}
	}
}

func TestBudget(t *testing.T) {
	for _, tc := range []struct {
		p90  float64
		want string
	}{{50, "ok"}, {150, "watch"}, {500, "OVER"}} {
		if got := budget(tc.p90); got != tc.want {
			t.Errorf("budget(%v) = %q, want %q", tc.p90, got, tc.want)
		}
	}
}
