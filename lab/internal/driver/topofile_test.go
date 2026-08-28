package driver

import (
	"os"
	"path/filepath"
	"testing"
)

const ospfTopo = `# comment line
name: ospf

topology:
  nodes:
    r1:
      kind: linux
      image: frrouting/frr:latest
    r2:
      kind: linux
  links:
    - endpoints: ["r1:eth1", "r2:eth1"]
    - endpoints: ["r2:eth2", "r1:eth2"]
`

func writeTopo(t *testing.T, name, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestScanTopology(t *testing.T) {
	path := writeTopo(t, "ospf.clab.yml", ospfTopo)

	devices, links, err := scanTopology(path)
	if err != nil {
		t.Fatalf("scanTopology: %v", err)
	}
	if len(devices) != 2 || devices[0] != "r1" || devices[1] != "r2" {
		t.Fatalf("devices = %v, want [r1 r2]", devices)
	}
	if len(links) != 2 {
		t.Fatalf("links = %v, want 2", links)
	}
	if links[0] != (Link{A: "r1", APort: "eth1", B: "r2", BPort: "eth1"}) {
		t.Errorf("links[0] = %+v", links[0])
	}
}

// Keys nested under a node (kind:, image:) must not be mistaken for devices.
func TestScanTopologyIgnoresNestedKeys(t *testing.T) {
	path := writeTopo(t, "x.clab.yml", ospfTopo)
	devices, _, err := scanTopology(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range devices {
		if d == "kind" || d == "image" {
			t.Fatalf("nested key %q parsed as a device: %v", d, devices)
		}
	}
}

func TestScanTopologyNoLinks(t *testing.T) {
	path := writeTopo(t, "solo.clab.yml", "name: solo\ntopology:\n  nodes:\n    r1:\n      kind: linux\n")
	devices, links, err := scanTopology(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 1 || devices[0] != "r1" {
		t.Errorf("devices = %v", devices)
	}
	if len(links) != 0 {
		t.Errorf("links = %v, want none", links)
	}
}

func TestTopologiesIn(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"ospf.clab.yml", "vlan.clab.yaml", "notes.md"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("name: x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	tops, err := TopologiesIn(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(tops) != 2 {
		t.Fatalf("found %d topologies, want 2: %+v", len(tops), tops)
	}
	if tops[0].Name != "ospf" || tops[1].Name != "vlan" {
		t.Errorf("names = %q %q", tops[0].Name, tops[1].Name)
	}
}

func TestParseEndpoints(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Link
		ok   bool
	}{
		{`- endpoints: ["r1:eth1", "r2:eth1"]`, Link{A: "r1", APort: "eth1", B: "r2", BPort: "eth1"}, true},
		{`- endpoints: [r1:eth1, r2:eth1]`, Link{A: "r1", APort: "eth1", B: "r2", BPort: "eth1"}, true},
		{`- endpoints: ["r1:eth1"]`, Link{}, false},
		{`kind: linux`, Link{}, false},
	} {
		got, ok := parseEndpoints(tc.in)
		if ok != tc.ok {
			t.Errorf("parseEndpoints(%q) ok = %v, want %v", tc.in, ok, tc.ok)
			continue
		}
		if ok && got != tc.want {
			t.Errorf("parseEndpoints(%q) = %+v, want %+v", tc.in, got, tc.want)
		}
	}
}

func TestShortName(t *testing.T) {
	if got := shortName("clab-ospf-r1", "ospf"); got != "r1" {
		t.Errorf("shortName = %q, want r1", got)
	}
	if got := shortName("something-else", "ospf"); got != "something-else" {
		t.Errorf("shortName passthrough = %q", got)
	}
}
