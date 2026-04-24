package skillinventory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// helper — seed a minimal plugin install tree.
func seedPluginInstall(t *testing.T, installPath string, mcpServers map[string]any, skills []string, readme string) {
	t.Helper()
	if err := os.MkdirAll(installPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if mcpServers != nil {
		data, _ := json.Marshal(map[string]any{"mcpServers": mcpServers})
		if err := os.WriteFile(filepath.Join(installPath, ".mcp.json"), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, sk := range skills {
		if err := os.MkdirAll(filepath.Join(installPath, "skills", sk), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(installPath, "skills", sk, "SKILL.md"), []byte("# "+sk), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if readme != "" {
		if err := os.WriteFile(filepath.Join(installPath, "README.md"), []byte(readme), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func seedManifest(t *testing.T, manifestPath string, plugins map[string][]map[string]any) {
	t.Helper()
	data, _ := json.Marshal(map[string]any{"version": 2, "plugins": plugins})
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func seedKnownMarketplaces(t *testing.T, path string, m map[string]any) {
	t.Helper()
	data, _ := json.Marshal(m)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// --- Claude Code plugin scanning -------------------------------------------

func TestScanPlugins_ClaudeCodeInstalled(t *testing.T) {
	home := t.TempDir()

	installPath := filepath.Join(home, ".claude", "plugins", "cache", "acme", "auditor", "1.2.3")
	seedPluginInstall(t, installPath,
		map[string]any{
			"github": map[string]any{"command": "npx", "args": []string{"-y", "@x/gh"}},
			"slack":  map[string]any{"type": "http", "url": "https://mcp.slack.com/mcp"},
		},
		[]string{"audit-policy", "scan-secrets"},
		"# Acme Auditor\n\nSecurity auditing for your codebase.\n",
	)

	seedManifest(t, filepath.Join(home, ".claude", "plugins", "installed_plugins.json"),
		map[string][]map[string]any{
			"auditor@acme-marketplace": {
				{
					"scope":        "user",
					"installPath":  installPath,
					"version":      "1.2.3",
					"installedAt":  "2026-04-01T12:00:00Z",
					"lastUpdated":  "2026-04-15T08:00:00Z",
					"gitCommitSha": "deadbeef1234",
				},
			},
		})

	seedKnownMarketplaces(t, filepath.Join(home, ".claude", "plugins", "known_marketplaces.json"),
		map[string]any{
			"acme-marketplace": map[string]any{
				"source": map[string]any{"source": "github", "repo": "acme/claude-plugins"},
			},
		})

	plugins := ScanPlugins(home)
	if len(plugins) != 1 {
		t.Fatalf("want 1 plugin, got %d: %+v", len(plugins), plugins)
	}
	p := plugins[0]
	if p.Name != "auditor" || p.Marketplace != "acme-marketplace" || p.FullName != "auditor@acme-marketplace" {
		t.Errorf("identity mismatch: %+v", p)
	}
	if p.Version != "1.2.3" || p.Scope != "user" {
		t.Errorf("version/scope mismatch: %+v", p)
	}
	if p.InstalledAt != "2026-04-01T12:00:00Z" || p.LastUpdated != "2026-04-15T08:00:00Z" {
		t.Errorf("timestamps mismatch: %+v", p)
	}
	if p.GitCommitSha != "deadbeef1234" {
		t.Errorf("sha mismatch: %q", p.GitCommitSha)
	}
	if p.Platform != PlatformClaudeCode {
		t.Errorf("platform: want claude_code, got %q", p.Platform)
	}
	if p.Source != "installed" {
		t.Errorf("source: want installed, got %q", p.Source)
	}
	if p.MarketplaceRepo != "github:acme/claude-plugins" {
		t.Errorf("marketplace_repo mismatch: %q", p.MarketplaceRepo)
	}
	if got := p.MCPServerNames; len(got) != 2 || got[0] != "github" || got[1] != "slack" {
		t.Errorf("mcp_server_names: got %+v (want sorted [github, slack])", got)
	}
	if got := p.SkillNames; len(got) != 2 || got[0] != "audit-policy" || got[1] != "scan-secrets" {
		t.Errorf("skill_names: got %+v (want sorted [audit-policy, scan-secrets])", got)
	}
	if p.Description != "Security auditing for your codebase." {
		t.Errorf("description: %q", p.Description)
	}
}

func TestScanPlugins_ClaudeCodeMarketplaceAvailable(t *testing.T) {
	home := t.TempDir()

	// Marketplace-cached plugin (not registered in installed_plugins.json)
	mktPlugin := filepath.Join(home, ".claude", "plugins", "marketplaces", "some-mkt", "graph-viz")
	seedPluginInstall(t, mktPlugin,
		map[string]any{"viz": map[string]any{"command": "graph-viz-mcp"}},
		[]string{"render-graph"},
		"# Graph Viz\n\nRender graphs.\n",
	)

	plugins := ScanPlugins(home)
	if len(plugins) != 1 {
		t.Fatalf("want 1, got %d", len(plugins))
	}
	p := plugins[0]
	if p.Source != "marketplace-available" {
		t.Errorf("source: %q", p.Source)
	}
	if p.Marketplace != "some-mkt" || p.Name != "graph-viz" {
		t.Errorf("identity: %+v", p)
	}
	if len(p.MCPServerNames) != 1 || p.MCPServerNames[0] != "viz" {
		t.Errorf("mcp: %+v", p.MCPServerNames)
	}
}

func TestScanPlugins_InstalledTakesPrecedenceOverMarketplace(t *testing.T) {
	// A plugin present in BOTH the installed manifest and the marketplace
	// cache should surface once per unique install_path. Installed + its
	// cache subpath have different install paths, so we expect 2 rows
	// with distinct sources — they represent different facts (formally
	// installed vs. also available in the marketplace).
	home := t.TempDir()

	installedPath := filepath.Join(home, ".claude", "plugins", "cache", "mkt", "foo", "1.0.0")
	seedPluginInstall(t, installedPath, nil, []string{"only-skill"}, "")
	seedManifest(t, filepath.Join(home, ".claude", "plugins", "installed_plugins.json"),
		map[string][]map[string]any{
			"foo@mkt": {{"scope": "user", "installPath": installedPath, "version": "1.0.0"}},
		})

	marketplacePath := filepath.Join(home, ".claude", "plugins", "marketplaces", "mkt", "foo")
	seedPluginInstall(t, marketplacePath, nil, []string{"only-skill"}, "")

	plugins := ScanPlugins(home)
	countBySource := map[string]int{}
	for _, p := range plugins {
		if p.FullName == "foo@mkt" {
			countBySource[p.Source]++
		}
	}
	if countBySource["installed"] != 1 {
		t.Errorf("want 1 installed row, got %d", countBySource["installed"])
	}
	if countBySource["marketplace-available"] != 1 {
		t.Errorf("want 1 marketplace-available row, got %d", countBySource["marketplace-available"])
	}
}

// --- Claude Desktop (Cowork) plugin scanning -------------------------------

func TestScanPlugins_DesktopInstalled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows cowork path conventions differ; covered separately if needed")
	}
	home := t.TempDir()
	appSupport := coworkAppSupportDir(home)
	if appSupport == "" {
		t.Skip("no cowork app-support dir on this OS")
	}

	coworkPlugins := filepath.Join(appSupport, "local-agent-mode-sessions", "sA", "sB", "cowork_plugins")
	installPath := filepath.Join(coworkPlugins, "cache", "knowledge-work-plugins", "marketing", "1.1.0")
	seedPluginInstall(t, installPath,
		map[string]any{
			"gmail":           map[string]any{"type": "http", "url": "https://gmail.mcp.claude.com/mcp"},
			"google-calendar": map[string]any{"type": "http", "url": "https://gcal.mcp.claude.com/mcp"},
			"slack":           map[string]any{"type": "http", "url": "https://mcp.slack.com/mcp"},
		},
		[]string{"brand-voice", "campaign-planning"},
		"# Marketing Plugin\n\nContent creation and campaigns.\n",
	)
	seedManifest(t, filepath.Join(coworkPlugins, "installed_plugins.json"),
		map[string][]map[string]any{
			"marketing@knowledge-work-plugins": {
				{
					"scope":        "user",
					"installPath":  installPath,
					"version":      "1.1.0",
					"installedAt":  "2026-04-24T19:02:51.035Z",
					"gitCommitSha": "477c893b7a",
				},
			},
		})
	seedKnownMarketplaces(t, filepath.Join(coworkPlugins, "known_marketplaces.json"),
		map[string]any{
			"knowledge-work-plugins": map[string]any{
				"source": map[string]any{"source": "github", "repo": "anthropics/knowledge-work-plugins"},
			},
		})

	plugins := ScanPlugins(home)
	if len(plugins) == 0 {
		t.Fatal("desktop plugins not discovered")
	}
	var found *Plugin
	for i, p := range plugins {
		if p.FullName == "marketing@knowledge-work-plugins" && p.Source == "installed" {
			found = &plugins[i]
		}
	}
	if found == nil {
		t.Fatalf("marketing plugin not found among %d plugins", len(plugins))
	}
	if found.Platform != PlatformClaudeDesktop {
		t.Errorf("platform: %q", found.Platform)
	}
	if found.MarketplaceRepo != "github:anthropics/knowledge-work-plugins" {
		t.Errorf("marketplace_repo: %q", found.MarketplaceRepo)
	}
	if got := found.MCPServerNames; len(got) != 3 {
		t.Errorf("mcp count: got %d (%+v)", len(got), got)
	}
	hasConnector := false
	for _, n := range found.MCPServerNames {
		if n == "gmail" || n == "google-calendar" {
			hasConnector = true
		}
	}
	if !hasConnector {
		t.Errorf("expected HTTP connector MCP entries, got %+v", found.MCPServerNames)
	}
}

func TestScanPlugins_DesktopRpmPlugin(t *testing.T) {
	home := t.TempDir()
	appSupport := coworkAppSupportDir(home)
	if appSupport == "" {
		t.Skip("no cowork app-support dir on this OS")
	}

	rpmDir := filepath.Join(appSupport, "local-agent-mode-sessions", "sA", "sB", "rpm", "plugin_XYZ123")
	seedPluginInstall(t, rpmDir, nil, []string{"some-skill"}, "# RPM plugin\n\nDescription.\n")

	plugins := ScanPlugins(home)
	if len(plugins) != 1 {
		t.Fatalf("want 1 rpm plugin, got %d: %+v", len(plugins), plugins)
	}
	p := plugins[0]
	if p.Source != "rpm" {
		t.Errorf("source: %q", p.Source)
	}
	if p.Name != "plugin_XYZ123" {
		t.Errorf("name: %q", p.Name)
	}
	if p.Platform != PlatformClaudeDesktop {
		t.Errorf("platform: %q", p.Platform)
	}
	if len(p.SkillNames) != 1 || p.SkillNames[0] != "some-skill" {
		t.Errorf("skills: %+v", p.SkillNames)
	}
}

// --- Parsing edge cases -----------------------------------------------------

func TestScanPlugins_MalformedManifestIsIgnored(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".claude", "plugins"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(home, ".claude", "plugins", "installed_plugins.json"),
		[]byte("{ not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}
	plugins := ScanPlugins(home)
	if len(plugins) != 0 {
		t.Errorf("malformed manifest should yield 0 plugins, got %d", len(plugins))
	}
}

func TestSplitPluginFullName(t *testing.T) {
	cases := []struct {
		in       string
		wantName string
		wantMkt  string
	}{
		{"auditor@acme", "auditor", "acme"},
		{"foo@bar@baz", "foo@bar", "baz"}, // split on LAST @
		{"no-at-sign", "no-at-sign", ""},
		{"", "", ""},
	}
	for _, c := range cases {
		name, mkt := splitPluginFullName(c.in)
		if name != c.wantName || mkt != c.wantMkt {
			t.Errorf("splitPluginFullName(%q) = (%q, %q), want (%q, %q)",
				c.in, name, mkt, c.wantName, c.wantMkt)
		}
	}
}

func TestReadReadmeDescription_SkipsHeadingsAndBanners(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")
	if err := os.WriteFile(path, []byte(
		"# Heading\n\n"+
			"<!-- banner -->\n\n"+
			"Actual description line.\n\n"+
			"Second paragraph ignored.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := readReadmeDescription(path)
	if got != "Actual description line." {
		t.Errorf("got %q", got)
	}
}

func TestScan_EmptyHomeNoPlugins(t *testing.T) {
	inv, err := Scan(ScanOptions{Home: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Plugins) != 0 {
		t.Errorf("empty home → expected 0 plugins, got %d", len(inv.Plugins))
	}
}
