// Package config handles loading, merging, and validating the
// gateway configuration from files, environment variables, and
// CLI flags.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the merged gateway configuration.
type Config struct {
	// Gateway settings
	Mode    string `json:"mode" yaml:"mode"` // "audit" or "enforce"
	Verbose bool   `json:"verbose" yaml:"verbose"`
	LogPath string `json:"log_path" yaml:"log_path"`

	// Detection settings
	Detection DetectionConfig `json:"detection" yaml:"detection"`

	// Server configs
	Servers []ServerEntry `json:"servers" yaml:"servers"`

	// API connection (set by auth login)
	APIKey string `json:"api_key,omitempty" yaml:"api_key,omitempty"`
	APIURL string `json:"api_url,omitempty" yaml:"api_url,omitempty"`

	// Dashboard policy (fetched, not configured locally)
	DashboardPolicy *DashboardPolicy `json:"-" yaml:"-"`
}

// DetectionConfig controls detection behavior.
type DetectionConfig struct {
	Threat         string   `json:"threat" yaml:"threat"`                   // "warn", "block", "monitor"
	SensitiveData  string   `json:"sensitive_data" yaml:"sensitive_data"`   // "warn", "block", "monitor"
	CustomKeywords []string `json:"custom_keywords,omitempty" yaml:"custom_keywords,omitempty"`
}

// ServerEntry is a server in the config file.
type ServerEntry struct {
	Name      string            `json:"name" yaml:"name"`
	Command   string            `json:"command" yaml:"command"`
	Args      []string          `json:"args,omitempty" yaml:"args,omitempty"`
	Env       map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	Transport string            `json:"transport,omitempty" yaml:"transport,omitempty"`
	URL       string            `json:"url,omitempty" yaml:"url,omitempty"`
	Headers   map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
}

// Server is an alias for ServerEntry for backward compatibility.
type Server = ServerEntry

// DashboardPolicy represents policy fetched from the Clawkeeper dashboard.
type DashboardPolicy struct {
	Mode           string              `json:"mode"`
	BlockedServers []string            `json:"blocked_servers"`
	BlockedTools   map[string][]string `json:"blocked_tools"` // server -> tools
	CustomKeywords []string            `json:"custom_keywords"`
	Detection      DetectionConfig     `json:"detection"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		Mode:    "audit",
		Verbose: false,
		APIURL:  "https://clawkeeper.dev",
		Detection: DetectionConfig{
			Threat:        "warn",
			SensitiveData: "warn",
		},
	}
}

// Load reads configuration from the local config file.
// Returns default config if no file exists.
func Load() (Config, error) {
	cfg := DefaultConfig()

	// Try local config file
	home, err := os.UserHomeDir()
	if err != nil {
		return cfg, nil
	}

	configPath := filepath.Join(home, ".config", "clawkeeper-mcp-gateway", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // No config file — use defaults
		}
		return cfg, fmt.Errorf("reading config: %w", err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing config: %w", err)
	}

	return cfg, nil
}

// Save writes the configuration to the local config file.
func Save(cfg Config) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".config", "clawkeeper-mcp-gateway")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.json")
	return os.WriteFile(configPath, data, 0644)
}

// SaveAPIKey stores the API key from auth login.
func SaveAPIKey(apiKey string) error {
	cfg, err := Load()
	if err != nil {
		cfg = DefaultConfig()
	}
	cfg.APIKey = apiKey
	return Save(cfg)
}

// AddServer adds a server to the config.
func AddServer(entry ServerEntry) error {
	cfg, err := Load()
	if err != nil {
		cfg = DefaultConfig()
	}

	// Remove existing server with same name
	filtered := make([]ServerEntry, 0, len(cfg.Servers))
	for _, s := range cfg.Servers {
		if s.Name != entry.Name {
			filtered = append(filtered, s)
		}
	}
	cfg.Servers = append(filtered, entry)

	return Save(cfg)
}

// RemoveServer removes a server from the config.
func RemoveServer(name string) error {
	cfg, err := Load()
	if err != nil {
		return err
	}

	filtered := make([]ServerEntry, 0, len(cfg.Servers))
	for _, s := range cfg.Servers {
		if s.Name != name {
			filtered = append(filtered, s)
		}
	}
	cfg.Servers = filtered

	return Save(cfg)
}

// MergeWithDashboard applies dashboard policy (additive-only: dashboard wins).
func (c *Config) MergeWithDashboard(policy DashboardPolicy) {
	c.DashboardPolicy = &policy

	// Dashboard mode overrides local
	if policy.Mode != "" {
		c.Mode = policy.Mode
	}

	// Dashboard detection settings override local (can only be stricter)
	if isStricter(policy.Detection.Threat, c.Detection.Threat) {
		c.Detection.Threat = policy.Detection.Threat
	}
	if isStricter(policy.Detection.SensitiveData, c.Detection.SensitiveData) {
		c.Detection.SensitiveData = policy.Detection.SensitiveData
	}

	// Merge custom keywords (additive)
	if len(policy.CustomKeywords) > 0 {
		seen := make(map[string]bool)
		for _, k := range c.Detection.CustomKeywords {
			seen[k] = true
		}
		for _, k := range policy.CustomKeywords {
			if !seen[k] {
				c.Detection.CustomKeywords = append(c.Detection.CustomKeywords, k)
			}
		}
	}
}

// isStricter returns true if a is stricter than b.
// block > warn > monitor > ""
func isStricter(a, b string) bool {
	rank := map[string]int{"": 0, "monitor": 1, "warn": 2, "block": 3}
	return rank[a] > rank[b]
}
