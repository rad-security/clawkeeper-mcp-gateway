// Package proxy implements the MCP stdio proxy that sits between
// AI clients and MCP servers, forwarding JSON-RPC messages while
// allowing inspection and modification by the detection engine.
package proxy

import (
	"io"
)

// Proxy manages bidirectional stdio communication between a client
// and one or more MCP servers.
type Proxy struct {
	stdin  io.Reader
	stdout io.Writer
}

// New creates a new MCP proxy instance.
func New(stdin io.Reader, stdout io.Writer) *Proxy {
	return &Proxy{
		stdin:  stdin,
		stdout: stdout,
	}
}

// Run starts the proxy loop. It reads JSON-RPC messages from stdin,
// routes them to the appropriate MCP server, and writes responses
// back to stdout.
func (p *Proxy) Run() error {
	// TODO: implement proxy loop
	return nil
}
