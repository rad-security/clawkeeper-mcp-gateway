// Package config handles loading, merging, and validating the
// gateway configuration from files, environment variables, and
// CLI flags.
package config

// Config holds the gateway configuration.
type Config struct {
	Mode    string            `json:"mode"`    // "audit" or "enforce"
	LogPath string            `json:"logPath"` // path to JSONL event log
	Servers map[string]Server `json:"servers"` // registered MCP servers
}

// Server represents a registered MCP server.
type Server struct {
	Command string            `json:"command"`          // stdio command
	Args    []string          `json:"args,omitempty"`   // command arguments
	Env     map[string]string `json:"env,omitempty"`    // environment variables
	URL     string            `json:"url,omitempty"`    // SSE URL (alternative to command)
	Headers map[string]string `json:"headers,omitempty"` // headers for SSE connections
}

// Load reads configuration from the given path, merging with defaults.
func Load(path string) (*Config, error) {
	// TODO: implement config loading
	return &Config{
		Mode:    "audit",
		Servers: make(map[string]Server),
	}, nil
}

// DefaultPath returns the default config file path.
func DefaultPath() string {
	return "~/.config/clawkeeper/gateway.json"
}
