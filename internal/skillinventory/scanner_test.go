package skillinventory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// seedSkill writes a dummy SKILL.md under `dir/<name>/SKILL.md`.
func seedSkill(t *testing.T, dir, name, content string) {
	t.Helper()
	writeFile(t, filepath.Join(dir, name, "SKILL.md"), content)
}

func TestScan_EmptyHome(t *testing.T) {
	home := t.TempDir()
	inv, err := Scan(ScanOptions{Home: home})
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Skills) != 0 {
		t.Errorf("empty home → expected 0 skills, got %d", len(inv.Skills))
	}
	if len(inv.MCPServers) != 0 {
		t.Errorf("empty home → expected 0 MCP servers, got %d", len(inv.MCPServers))
	}
}

func TestScan_StandaloneGlobalSkills(t *testing.T) {
	home := t.TempDir()
	seedSkill(t, filepath.Join(home, ".claude", "skills"), "my-skill", "# My Skill\nDoes a thing.")
	seedSkill(t, filepath.Join(home, ".claude", "skills"), "another", "# Another\nAlso does things.")

	inv, err := Scan(ScanOptions{Home: home})
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Skills) != 2 {
		t.Fatalf("want 2 skills, got %d", len(inv.Skills))
	}
	for _, s := range inv.Skills {
		if s.Source != "global" {
			t.Errorf("expected global source, got %q for %s", s.Source, s.Name)
		}
		if s.Hash == "" {
			t.Errorf("missing hash for %s", s.Name)
		}
		if s.Preview == "" {
			t.Errorf("missing preview for %s", s.Name)
		}
	}
}

func TestScan_PluginInstalledSkillsViaManifest(t *testing.T) {
	home := t.TempDir()

	// Simulate a plugin install: manifest points at an install path with skills.
	installPath := filepath.Join(home, ".claude", "plugins", "cache", "org", "repo", "v1")
	seedSkill(t, filepath.Join(installPath, "skills"), "plugin-skill-1", "# Plugin 1")
	seedSkill(t, filepath.Join(installPath, "skills"), "plugin-skill-2", "# Plugin 2")

	manifest := map[string]any{
		"plugins": map[string]any{
			"org/repo": []map[string]any{
				{
					"installPath": installPath,
					"scope":       "user",
				},
			},
		},
	}
	data, _ := json.Marshal(manifest)
	writeFile(t, filepath.Join(home, ".claude", "plugins", "installed_plugins.json"), string(data))

	inv, err := Scan(ScanOptions{Home: home})
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Skills) != 2 {
		t.Fatalf("want 2 plugin skills, got %d: %+v", len(inv.Skills), inv.Skills)
	}
	for _, s := range inv.Skills {
		if s.Source != "global" {
			t.Errorf("manifest scope=user should map to global; got %q for %s", s.Source, s.Name)
		}
	}
}

func TestScan_PluginProjectScopeManifest(t *testing.T) {
	home := t.TempDir()

	installPath := filepath.Join(home, ".claude", "plugins", "cache", "proj", "plugin", "v1")
	seedSkill(t, filepath.Join(installPath, "skills"), "proj-skill", "# Proj")

	manifest := map[string]any{
		"plugins": map[string]any{
			"proj/plugin": []map[string]any{
				{"installPath": installPath, "scope": "project"},
			},
		},
	}
	data, _ := json.Marshal(manifest)
	writeFile(t, filepath.Join(home, ".claude", "plugins", "installed_plugins.json"), string(data))

	inv, _ := Scan(ScanOptions{Home: home})
	if len(inv.Skills) != 1 {
		t.Fatalf("want 1 skill, got %d", len(inv.Skills))
	}
	if inv.Skills[0].Source != "project" {
		t.Errorf("scope=project should surface as project source; got %q", inv.Skills[0].Source)
	}
}

func TestScan_ManifestMissing_FallbackCacheGlob(t *testing.T) {
	home := t.TempDir()
	// No manifest at all; the scanner should fall back to the cache glob.
	cachePath := filepath.Join(home, ".claude", "plugins", "cache", "org", "repo", "v1")
	seedSkill(t, filepath.Join(cachePath, "skills"), "cached-skill", "# Cached")

	inv, _ := Scan(ScanOptions{Home: home})
	if len(inv.Skills) != 1 {
		t.Fatalf("fallback glob should find the cached skill; got %d", len(inv.Skills))
	}
}

func TestScan_ProjectScopeSkills(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	seedSkill(t, filepath.Join(cwd, ".claude", "skills"), "proj-only", "# Project only")

	inv, _ := Scan(ScanOptions{Home: home, CWD: cwd})
	if len(inv.Skills) != 1 {
		t.Fatalf("want 1 project skill, got %d", len(inv.Skills))
	}
	if inv.Skills[0].Source != "project" {
		t.Errorf("expected project source, got %q", inv.Skills[0].Source)
	}
}

func TestScan_MCPServersFromGlobalSettings(t *testing.T) {
	home := t.TempDir()
	settings := map[string]any{
		"mcpServers": map[string]any{
			"github": map[string]any{
				"command": "npx",
				"args":    []string{"-y", "@modelcontextprotocol/server-github"},
			},
			"remote-sse": map[string]any{
				"type": "http",
				"url":  "https://api.example.com/mcp/sse",
			},
		},
	}
	data, _ := json.Marshal(settings)
	writeFile(t, filepath.Join(home, ".claude", "settings.json"), string(data))

	inv, _ := Scan(ScanOptions{Home: home})
	if len(inv.MCPServers) != 2 {
		t.Fatalf("want 2 MCP servers, got %d: %+v", len(inv.MCPServers), inv.MCPServers)
	}

	byName := map[string]MCPServer{}
	for _, s := range inv.MCPServers {
		byName[s.Name] = s
	}

	gh := byName["github"]
	if gh.Type != "stdio" {
		t.Errorf("github type: want stdio, got %q", gh.Type)
	}
	if gh.Command != "npx -y @modelcontextprotocol/server-github" {
		t.Errorf("github command: want joined args, got %q", gh.Command)
	}
	if gh.Source != "global" {
		t.Errorf("github source: want global, got %q", gh.Source)
	}

	remote := byName["remote-sse"]
	if remote.Type != "http" {
		t.Errorf("remote-sse type: want http, got %q", remote.Type)
	}
}

func TestScan_MCPServersFromProjectSettings(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	settings := map[string]any{
		"mcpServers": map[string]any{
			"project-pg": map[string]any{
				"command": "npx",
				"args":    []string{"-y", "@modelcontextprotocol/server-postgres"},
			},
		},
	}
	data, _ := json.Marshal(settings)
	writeFile(t, filepath.Join(cwd, ".claude", "settings.json"), string(data))

	inv, _ := Scan(ScanOptions{Home: home, CWD: cwd})
	if len(inv.MCPServers) != 1 {
		t.Fatalf("want 1 project MCP server, got %d", len(inv.MCPServers))
	}
	if inv.MCPServers[0].Source != "project" {
		t.Errorf("want project source, got %q", inv.MCPServers[0].Source)
	}
}

func TestScan_MalformedSettingsJSON_DoesNotCrash(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".claude", "settings.json"), "{ not valid json")

	inv, err := Scan(ScanOptions{Home: home})
	if err != nil {
		t.Fatalf("malformed json should not error; got %v", err)
	}
	if len(inv.MCPServers) != 0 {
		t.Errorf("malformed json → expected 0 servers, got %d", len(inv.MCPServers))
	}
}

func TestScan_NoDuplicatesAcrossPaths(t *testing.T) {
	home := t.TempDir()

	// Same skill name appears via both the manifest and the fallback glob.
	installPath := filepath.Join(home, ".claude", "plugins", "cache", "org", "repo", "v1")
	seedSkill(t, filepath.Join(installPath, "skills"), "shared", "# Shared")

	manifest := map[string]any{
		"plugins": map[string]any{
			"org/repo": []map[string]any{
				{"installPath": installPath, "scope": "user"},
			},
		},
	}
	data, _ := json.Marshal(manifest)
	writeFile(t, filepath.Join(home, ".claude", "plugins", "installed_plugins.json"), string(data))

	inv, _ := Scan(ScanOptions{Home: home})
	if len(inv.Skills) != 1 {
		t.Fatalf("deduplication broken: expected 1 skill, got %d", len(inv.Skills))
	}
}

func TestBuildPayload_HasAllFields(t *testing.T) {
	inv := Inventory{
		Skills:     []Skill{{Name: "x", Source: "global"}},
		MCPServers: []MCPServer{{Name: "y", Type: "stdio", Source: "global"}},
	}
	p := BuildPayload(inv, "/tmp/cwd", "0.2.76", "abc123")
	if p.CWD != "/tmp/cwd" {
		t.Errorf("cwd not passed through: %q", p.CWD)
	}
	if p.ClaudeVersion != "0.2.76" {
		t.Errorf("claude_version not passed through: %q", p.ClaudeVersion)
	}
	if p.MachineID != "abc123" {
		t.Errorf("machine_id not passed through: %q", p.MachineID)
	}
	if p.Source != "gateway-scan-inventory" {
		t.Errorf("source tag missing / wrong: %q", p.Source)
	}
	if len(p.InstalledSkills) != 1 || len(p.InstalledMCPServer) != 1 {
		t.Errorf("skills/mcp arrays not passed through")
	}
}
