package driver

import (
	"context"
	"fmt"
	"time"
)

// Mock runs nothing. It reads the real topology file and reports the devices
// and links it declares, so the whole stack — verbs, graph, picker, client —
// can be exercised on a machine without containerlab installed.
//
// It is not a stub for testing only: developing against it is what keeps the
// iteration loop under a second.
type Mock struct {
	// Delay simulates per-device bring-up cost so latency instrumentation
	// shows something realistic during development.
	Delay time.Duration
}

func (m *Mock) Name() string    { return "mock" }
func (m *Mock) Available() bool { return true }

func (m *Mock) Topologies(dir string) ([]Topology, error) { return TopologiesIn(dir) }

func (m *Mock) Up(ctx context.Context, t Topology) ([]Device, []Link, error) {
	names, links, err := scanTopology(t.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("read topology %s: %w", t.Path, err)
	}
	if len(names) == 0 {
		return nil, nil, fmt.Errorf("topology %s declares no nodes", t.Path)
	}

	devices := make([]Device, 0, len(names))
	for i, n := range names {
		if m.Delay > 0 {
			select {
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			case <-time.After(m.Delay):
			}
		}
		devices = append(devices, Device{
			Name:      n,
			Runtime:   fmt.Sprintf("clab-%s-%s", t.Name, n),
			Kind:      "mock",
			Image:     "mock/none",
			MgmtIPv4:  fmt.Sprintf("172.20.20.%d", i+11),
			Reachable: true,
		})
	}
	return devices, links, nil
}

func (m *Mock) Down(ctx context.Context, t Topology) error { return nil }

func (m *Mock) AttachCmd(labName string, d Device) []string {
	return []string{"sh", "-c", fmt.Sprintf(
		"echo '[mock] console for %s (%s) — no runtime behind this driver'; exec sh",
		d.Name, d.Runtime)}
}
