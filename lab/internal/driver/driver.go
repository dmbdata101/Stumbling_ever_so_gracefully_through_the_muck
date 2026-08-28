// Package driver abstracts whatever actually runs a topology.
//
// containerlab today; GNS3 or libvirt can be added without any verb, graph,
// or client change. The interface is deliberately small — the graph owns all
// state, so a driver only has to make the world match it and report back.
package driver

import "context"

// Device is one node in a running topology.
type Device struct {
	Name      string // short name as written in the topology, e.g. "r1"
	Runtime   string // driver-specific handle, e.g. a container name
	Kind      string
	Image     string
	MgmtIPv4  string
	Reachable bool
}

// Link is one point-to-point connection.
type Link struct {
	A, B         string // device names
	APort, BPort string
}

// Topology is a lab definition that has not been brought up yet.
type Topology struct {
	Name string
	Path string
}

// Driver runs topologies.
type Driver interface {
	// Name identifies the driver in output and in the graph.
	Name() string

	// Available reports whether this driver can run here. A driver that is
	// not available must never be selected.
	Available() bool

	// Topologies lists lab definitions found under dir.
	Topologies(dir string) ([]Topology, error)

	// Up brings up a topology and returns what it created.
	Up(ctx context.Context, t Topology) ([]Device, []Link, error)

	// Down tears a lab down. Tearing down a lab that is not up is not an error.
	Down(ctx context.Context, t Topology) error

	// AttachCmd returns the argv the *client* should exec to get a console on
	// a device. The daemon never owns a TTY, so it hands the command back
	// instead of running it.
	AttachCmd(labName string, d Device) []string
}
