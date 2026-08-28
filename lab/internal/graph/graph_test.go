package graph

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestGraph(t *testing.T) (*Graph, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "graph.jsonl")
	j, err := OpenJournal(path)
	if err != nil {
		t.Fatalf("OpenJournal: %v", err)
	}
	t.Cleanup(func() { j.Close() })
	g, err := New(j)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return g, path
}

func TestPutGetDel(t *testing.T) {
	g, _ := newTestGraph(t)

	if err := g.Set("lab/ospf", KindLab, map[string]string{"state": "up"}); err != nil {
		t.Fatalf("Set: %v", err)
	}
	n := g.Get("lab/ospf")
	if n == nil || n.Attr("state") != "up" {
		t.Fatalf("Get returned %+v, want state=up", n)
	}
	if n.Created.IsZero() || n.Updated.IsZero() {
		t.Error("timestamps not stamped")
	}

	if err := g.Del("lab/ospf"); err != nil {
		t.Fatalf("Del: %v", err)
	}
	if g.Get("lab/ospf") != nil {
		t.Error("node survived Del")
	}
	if err := g.Del("lab/ospf"); err != nil {
		t.Errorf("deleting a missing node should be a no-op, got %v", err)
	}
}

func TestGetReturnsCopy(t *testing.T) {
	g, _ := newTestGraph(t)
	g.Set("n/1", KindNode, map[string]string{"name": "r1"})

	got := g.Get("n/1")
	got.Attrs["name"] = "mutated"

	if again := g.Get("n/1"); again.Attr("name") != "r1" {
		t.Fatalf("mutating a returned node changed graph state: %q", again.Attr("name"))
	}
}

func TestPutPreservesCreated(t *testing.T) {
	g, _ := newTestGraph(t)
	g.Set("n/1", KindNode, map[string]string{"v": "1"})
	created := g.Get("n/1").Created

	g.Set("n/1", KindNode, map[string]string{"v": "2"})
	after := g.Get("n/1")

	if !after.Created.Equal(created) {
		t.Errorf("Created changed on update: %v -> %v", created, after.Created)
	}
	if after.Attr("v") != "2" {
		t.Errorf("value not updated: %q", after.Attr("v"))
	}
}

func TestJournalReplay(t *testing.T) {
	g, path := newTestGraph(t)
	g.Set("lab/ospf", KindLab, map[string]string{"state": "up"})
	g.Set("lab/vlan", KindLab, map[string]string{"state": "up"})
	g.Set("node/ospf/r1", KindNode, map[string]string{"lab": "ospf"})
	g.Del("lab/vlan")

	j2, err := OpenJournal(path)
	if err != nil {
		t.Fatalf("reopen journal: %v", err)
	}
	defer j2.Close()
	g2, err := New(j2)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}

	if got := g2.Len(); got != 2 {
		t.Fatalf("replayed %d nodes, want 2", got)
	}
	if n := g2.Get("lab/ospf"); n == nil || n.Attr("state") != "up" {
		t.Error("lab/ospf did not survive replay")
	}
	if g2.Get("lab/vlan") != nil {
		t.Error("deleted node came back after replay")
	}
	if len(g2.Kind(KindNode)) != 1 {
		t.Error("device node did not survive replay")
	}
}

// A daemon killed mid-write leaves a partial final line. Losing that one
// mutation is fine; refusing to start is not.
func TestJournalTruncatedTail(t *testing.T) {
	g, path := newTestGraph(t)
	g.Set("lab/ospf", KindLab, map[string]string{"state": "up"})

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(`{"op":"put","node":{"id":"lab/hal`)
	f.Close()

	j2, err := OpenJournal(path)
	if err != nil {
		t.Fatalf("OpenJournal on truncated file: %v", err)
	}
	defer j2.Close()
	g2, err := New(j2)
	if err != nil {
		t.Fatalf("New on truncated journal: %v", err)
	}
	if g2.Get("lab/ospf") == nil {
		t.Error("complete op before the truncated tail was lost")
	}
}

func TestKindIsSorted(t *testing.T) {
	g, _ := newTestGraph(t)
	for _, id := range []string{"node/c", "node/a", "node/b"} {
		g.Set(id, KindNode, nil)
	}
	got := g.Kind(KindNode)
	for i, want := range []string{"node/a", "node/b", "node/c"} {
		if got[i].ID != want {
			t.Fatalf("Kind()[%d] = %q, want %q", i, got[i].ID, want)
		}
	}
}
