// Package server manages the lifecycle of registered MCP servers,
// including process spawning, health checking, and graceful shutdown.
package server

import (
	"github.com/rad-security/clawkeeper-mcp-gateway/internal/config"
)

// Status represents the health state of an MCP server.
type Status struct {
	Name    string `json:"name"`
	Running bool   `json:"running"`
	PID     int    `json:"pid,omitempty"`
	Uptime  string `json:"uptime,omitempty"`
}

// Manager handles MCP server process lifecycle.
type Manager struct {
	servers map[string]*config.Server
}

// NewManager creates a server manager.
func NewManager() *Manager {
	return &Manager{
		servers: make(map[string]*config.Server),
	}
}

// Add registers a new MCP server.
func (m *Manager) Add(name string, srv config.Server) error {
	m.servers[name] = &srv
	return nil
}

// Remove unregisters an MCP server.
func (m *Manager) Remove(name string) error {
	delete(m.servers, name)
	return nil
}

// List returns the status of all registered servers.
func (m *Manager) List() []Status {
	// TODO: implement health checking
	var statuses []Status
	for name := range m.servers {
		statuses = append(statuses, Status{Name: name, Running: false})
	}
	return statuses
}
