package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Containerlab shells out to the containerlab binary.
type Containerlab struct{}

func (c *Containerlab) Name() string { return "containerlab" }

func (c *Containerlab) Available() bool {
	_, err := exec.LookPath("containerlab")
	return err == nil
}

func (c *Containerlab) Topologies(dir string) ([]Topology, error) { return TopologiesIn(dir) }

// clabNode is the subset of `containerlab inspect -f json` we consume.
//
// containerlab has shipped two shapes for this: a bare array, and an object
// with a "containers" key. Both are handled — a driver that breaks on a
// version bump is worse than a few extra lines here.
type clabNode struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Image       string `json:"image"`
	IPv4Address string `json:"ipv4_address"`
	State       string `json:"state"`
	LabName     string `json:"lab_name"`
}

func (c *Containerlab) Up(ctx context.Context, t Topology) ([]Device, []Link, error) {
	out, err := exec.CommandContext(ctx, "containerlab", "deploy", "-t", t.Path, "--reconfigure").CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("containerlab deploy: %w: %s", err, strings.TrimSpace(string(out)))
	}
	devices, err := c.inspect(ctx, t)
	if err != nil {
		return nil, nil, err
	}
	// Links come from the topology file; containerlab does not report them
	// back in a stable shape across versions.
	_, links, err := scanTopology(t.Path)
	if err != nil {
		return devices, nil, nil // devices are up; link display is cosmetic
	}
	return devices, links, nil
}

func (c *Containerlab) inspect(ctx context.Context, t Topology) ([]Device, error) {
	out, err := exec.CommandContext(ctx, "containerlab", "inspect", "-t", t.Path, "-f", "json").Output()
	if err != nil {
		return nil, fmt.Errorf("containerlab inspect: %w", err)
	}

	var nodes []clabNode
	if err := json.Unmarshal(out, &nodes); err != nil {
		var wrapped struct {
			Containers []clabNode `json:"containers"`
		}
		if err2 := json.Unmarshal(out, &wrapped); err2 != nil {
			return nil, fmt.Errorf("parse containerlab inspect output: %w", err)
		}
		nodes = wrapped.Containers
	}

	devices := make([]Device, 0, len(nodes))
	for _, n := range nodes {
		devices = append(devices, Device{
			Name:      shortName(n.Name, t.Name),
			Runtime:   n.Name,
			Kind:      n.Kind,
			Image:     n.Image,
			MgmtIPv4:  strings.TrimSuffix(n.IPv4Address, "/24"),
			Reachable: strings.Contains(strings.ToLower(n.State), "running"),
		})
	}
	return devices, nil
}

// shortName turns "clab-ospf-r1" back into "r1" — the name written in the
// topology is the one worth typing.
func shortName(runtime, lab string) string {
	if s, ok := strings.CutPrefix(runtime, "clab-"+lab+"-"); ok {
		return s
	}
	return runtime
}

func (c *Containerlab) Down(ctx context.Context, t Topology) error {
	out, err := exec.CommandContext(ctx, "containerlab", "destroy", "-t", t.Path, "--cleanup").CombinedOutput()
	if err != nil {
		return fmt.Errorf("containerlab destroy: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (c *Containerlab) AttachCmd(labName string, d Device) []string {
	return []string{"docker", "exec", "-it", d.Runtime, "/bin/sh"}
}
