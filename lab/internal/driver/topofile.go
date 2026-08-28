package driver

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TopologiesIn finds *.clab.yml / *.clab.yaml under dir. Shared by every
// driver, since the topology file format is the source of truth regardless
// of who runs it.
func TopologiesIn(dir string) ([]Topology, error) {
	var out []Topology
	for _, pat := range []string{"*.clab.yml", "*.clab.yaml"} {
		matches, err := filepath.Glob(filepath.Join(dir, pat))
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			out = append(out, Topology{Name: topoName(m), Path: m})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func topoName(path string) string {
	b := filepath.Base(path)
	b = strings.TrimSuffix(b, ".yaml")
	b = strings.TrimSuffix(b, ".yml")
	return strings.TrimSuffix(b, ".clab")
}

// scanTopology pulls device and link names out of a containerlab file.
//
// This is an intentionally minimal indentation scan rather than a YAML
// dependency: the client/daemon pair stays stdlib-only so the binaries start
// in ~2ms with nothing to vendor. The real containerlab driver never uses
// this — it asks containerlab itself. Only the mock driver reads files, so a
// scanner that handles the shapes we actually write is enough.
func scanTopology(path string) (devices []string, links []Link, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	const (
		sectionNone = iota
		sectionNodes
		sectionLinks
	)
	section := sectionNone
	sectionIndent, nodeIndent := 0, -1

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		raw := sc.Text()
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))

		if section != sectionNone && indent <= sectionIndent {
			section = sectionNone
			nodeIndent = -1
		}

		switch {
		case trimmed == "nodes:":
			section, sectionIndent, nodeIndent = sectionNodes, indent, -1
			continue
		case trimmed == "links:":
			section, sectionIndent = sectionLinks, indent
			continue
		}

		switch section {
		case sectionNodes:
			// The first deeper indent level under nodes: is the device names.
			if nodeIndent == -1 {
				nodeIndent = indent
			}
			if indent == nodeIndent && strings.HasSuffix(trimmed, ":") {
				devices = append(devices, strings.TrimSuffix(trimmed, ":"))
			}
		case sectionLinks:
			if l, ok := parseEndpoints(trimmed); ok {
				links = append(links, l)
			}
		}
	}
	return devices, links, sc.Err()
}

// parseEndpoints reads `- endpoints: ["r1:eth1", "r2:eth1"]`.
func parseEndpoints(line string) (Link, bool) {
	i := strings.Index(line, "endpoints:")
	if i < 0 {
		return Link{}, false
	}
	rest := line[i+len("endpoints:"):]
	rest = strings.Trim(strings.TrimSpace(rest), "[]")
	parts := strings.Split(rest, ",")
	if len(parts) != 2 {
		return Link{}, false
	}
	a, aPort, ok1 := splitEndpoint(parts[0])
	b, bPort, ok2 := splitEndpoint(parts[1])
	if !ok1 || !ok2 {
		return Link{}, false
	}
	return Link{A: a, APort: aPort, B: b, BPort: bPort}, true
}

func splitEndpoint(s string) (dev, port string, ok bool) {
	s = strings.Trim(strings.TrimSpace(s), `"'`)
	dev, port, found := strings.Cut(s, ":")
	if !found || dev == "" {
		return "", "", false
	}
	return dev, port, true
}
