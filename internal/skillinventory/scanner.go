// Package skillinventory scans a developer laptop for installed Claude Code
// skills and MCP servers, and reports the inventory to the Clawkeeper
// dashboard via /api/v1/claude-code/checkin.
//
// This exists as a gateway binary subcommand (`scan-inventory`) so the same
// single endpoint agent can handle:
//
//   - MCP proxy (server)
//   - IDE config rewiring (configure-ide)
//   - Inventory reporting (scan-inventory) ← this package
//
// ...without requiring a separate Claude Code plugin install. The Kandji /
// Jamf / MDM path becomes: install one binary, point a SessionStart hook at
// `type: command` -> `/usr/local/bin/clawkeeper-mcp-gateway scan-inventory`.
//
// Logic mirrors plugin/skills/connect/SKILL.md collect_skills + collect_mcp
// byte-for-byte so both paths report identical data to the same endpoint.
package skillinventory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
)

// Skill represents one installed Claude Code skill.
type Skill struct {
	Name    string `json:"name"`
	Source  string `json:"source"`  // "global" or "project"
	Preview string `json:"preview"` // first 300 bytes of SKILL.md
	Hash    string `json:"hash"`    // SHA-256 of SKILL.md
}

// MCPServer represents one configured Claude Code MCP server.
type MCPServer struct {
	Name    string `json:"name"`
	Type    string `json:"type"`              // "stdio" or "http"
	Command string `json:"command,omitempty"` // stdio: joined command+args
	Source  string `json:"source"`            // "global" or "project"
}

// Inventory is the full payload captured from this laptop.
type Inventory struct {
	Skills     []Skill     `json:"installed_skills"`
	MCPServers []MCPServer `json:"installed_mcp_servers"`
}

// ScanOptions controls where the scanner looks.
type ScanOptions struct {
	// Home is the user's home directory. Defaults to os.UserHomeDir().
	Home string
	// CWD is the Claude Code session's working directory. When empty,
	// project-scoped skills + project settings.json are skipped.
	CWD string
}

// Scan walks all known skill + MCP locations and returns the combined inventory.
// Errors on individual files are swallowed by design — the checkin is
// fail-open, and a single unreadable SKILL.md should not drop the whole scan.
func Scan(opts ScanOptions) (Inventory, error) {
	home := opts.Home
	if home == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return Inventory{}, fmt.Errorf("resolving home dir: %w", err)
		}
		home = h
	}

	inv := Inventory{
		Skills:     scanSkills(home, opts.CWD),
		MCPServers: scanMCPServers(home, opts.CWD),
	}
	return inv, nil
}

// --- skills -----------------------------------------------------------------

// scanSkills mirrors collect_skills() in plugin/skills/connect/SKILL.md.
//
// Four locations, in order:
//  1. ~/.claude/skills/*/SKILL.md                            (global standalone)
//  2. <cwd>/.claude/skills/*/SKILL.md                        (project standalone)
//  3. plugin installs read from installed_plugins.json manifest, then fallback-globbed
//  4. <cwd>/.claude/plugins/*/*/skills/*/SKILL.md            (project-level plugin installs)
func scanSkills(home, cwd string) []Skill {
	skills := []Skill{}
	seen := map[string]bool{}

	add := func(skillMDPath, source string) {
		name := filepath.Base(filepath.Dir(skillMDPath))
		key := name + "|" + source
		if seen[key] {
			return
		}
		seen[key] = true
		skills = append(skills, Skill{
			Name:    name,
			Source:  source,
			Preview: readPreview(skillMDPath),
			Hash:    hashFile(skillMDPath),
		})
	}

	// 1 + 2. Well-known standalone locations
	for _, entry := range []struct {
		pattern string
		source  string
	}{
		{filepath.Join(home, ".claude", "skills", "*", "SKILL.md"), "global"},
	} {
		for _, f := range globSafe(entry.pattern) {
			add(f, entry.source)
		}
	}
	if cwd != "" {
		for _, f := range globSafe(filepath.Join(cwd, ".claude", "skills", "*", "SKILL.md")) {
			add(f, "project")
		}
	}

	// 3. Plugin-installed skills via installed_plugins.json manifest.
	manifestPath := filepath.Join(home, ".claude", "plugins", "installed_plugins.json")
	manifestOK := false
	if data, err := os.ReadFile(manifestPath); err == nil {
		var manifest struct {
			Plugins map[string][]struct {
				InstallPath string `json:"installPath"`
				Scope       string `json:"scope"`
			} `json:"plugins"`
		}
		if err := json.Unmarshal(data, &manifest); err == nil {
			manifestOK = true
			for _, installs := range manifest.Plugins {
				for _, inst := range installs {
					if inst.InstallPath == "" {
						continue
					}
					source := "global"
					if inst.Scope == "project" {
						source = "project"
					}
					for _, f := range globSafe(filepath.Join(inst.InstallPath, "skills", "*", "SKILL.md")) {
						add(f, source)
					}
				}
			}
		}
	}
	// Fallback cache glob when the manifest is missing or malformed.
	if !manifestOK {
		cachePattern := filepath.Join(home, ".claude", "plugins", "cache", "*", "*", "*", "skills", "*", "SKILL.md")
		for _, f := range globSafe(cachePattern) {
			add(f, "global")
		}
	}

	// 4. Project-level plugin installs (rare but supported).
	if cwd != "" {
		for _, f := range globSafe(filepath.Join(cwd, ".claude", "plugins", "*", "*", "skills", "*", "SKILL.md")) {
			add(f, "project")
		}
	}

	// Deterministic output so tests and diffs are stable.
	sort.Slice(skills, func(i, j int) bool {
		if skills[i].Source != skills[j].Source {
			return skills[i].Source < skills[j].Source
		}
		return skills[i].Name < skills[j].Name
	})

	return skills
}

// --- MCP servers ------------------------------------------------------------

// scanMCPServers mirrors collect_mcp() — reads mcpServers from each of the
// three settings.json locations and flattens them into a single list.
func scanMCPServers(home, cwd string) []MCPServer {
	servers := []MCPServer{}
	seen := map[string]bool{}

	sources := []struct {
		scope string
		path  string
	}{
		{"global", filepath.Join(home, ".claude", "settings.json")},
	}
	if cwd != "" {
		sources = append(sources,
			struct{ scope, path string }{"project", filepath.Join(cwd, ".claude", "settings.json")},
			struct{ scope, path string }{"project", filepath.Join(cwd, ".claude", "settings.local.json")},
		)
	}

	for _, src := range sources {
		data, err := os.ReadFile(src.path)
		if err != nil {
			continue
		}
		var settings struct {
			MCPServers map[string]struct {
				Type    string   `json:"type"`
				URL     string   `json:"url"`
				Command string   `json:"command"`
				Args    []string `json:"args"`
			} `json:"mcpServers"`
		}
		if err := json.Unmarshal(data, &settings); err != nil {
			continue
		}
		for name, cfg := range settings.MCPServers {
			key := name + "|" + src.scope
			if seen[key] {
				continue
			}
			seen[key] = true

			serverType := cfg.Type
			if serverType == "" {
				if cfg.URL != "" {
					serverType = "http"
				} else {
					serverType = "stdio"
				}
			}
			command := cfg.Command
			if command != "" && len(cfg.Args) > 0 {
				for _, a := range cfg.Args {
					command += " " + a
				}
			}
			servers = append(servers, MCPServer{
				Name:    name,
				Type:    serverType,
				Command: command,
				Source:  src.scope,
			})
		}
	}

	// Deterministic.
	sort.Slice(servers, func(i, j int) bool {
		if servers[i].Source != servers[j].Source {
			return servers[i].Source < servers[j].Source
		}
		return servers[i].Name < servers[j].Name
	})
	return servers
}

// --- helpers ----------------------------------------------------------------

// readPreview returns the first 300 bytes of the file as a string, or empty
// string on any error (unreadable, missing, etc.). Matches read_preview() in
// connect/SKILL.md.
func readPreview(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	buf := make([]byte, 300)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return ""
	}
	return string(buf[:n])
}

// hashFile returns the hex-encoded SHA-256 of the file contents, or empty
// string on any error.
func hashFile(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}

// globSafe returns filepath.Glob results without the error — any error is
// indistinguishable from "no match" for our purposes.
func globSafe(pattern string) []string {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}
	return matches
}

// Platform returns the OS label expected by the checkin endpoint.
// (Separate from runtime.GOOS only because we may want friendlier labels
// later; for now keep it identical.)
func Platform() string {
	return runtime.GOOS
}
