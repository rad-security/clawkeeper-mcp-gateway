// Plugin inventory scanning. Plugins in the Anthropic ecosystem are bundles
// of skills + MCP server manifests shipped as a single installable unit. This
// package treats them as first-class inventory entities (separate from the
// flat skill + MCP-server lists) so the dashboard can render provenance — who
// published the plugin, what marketplace it came from, which MCP servers it
// registered on install, what commit SHA is on disk.
//
// Layouts (observed live April 2026):
//
//   Claude Code (terminal):
//     ~/.claude/plugins/installed_plugins.json          (manifest)
//     ~/.claude/plugins/cache/<owner>/<repo>/<ver>/     (installed plugin tree)
//     ~/.claude/plugins/marketplaces/<m>/<plugin>/      (marketplace-available)
//
//   Claude Desktop (Cowork):
//     <appSupport>/local-agent-mode-sessions/<sA>/<sB>/cowork_plugins/
//         installed_plugins.json                        (manifest)
//         cache/<m>/<plugin>/<ver>/                     (installed plugin tree)
//         marketplaces/<m>/<plugin>/                    (marketplace-available)
//         known_marketplaces.json                       (marketplace source repos)
//     <appSupport>/local-agent-mode-sessions/<sA>/<sB>/rpm/plugin_<id>/   (alt install)

package skillinventory

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Plugin represents one installed or available plugin, from either agent
// surface.
type Plugin struct {
	// Identity
	Name        string `json:"name"`                  // "marketing"
	Marketplace string `json:"marketplace,omitempty"` // "knowledge-work-plugins"
	FullName    string `json:"full_name"`             // "marketing@knowledge-work-plugins"
	Version     string `json:"version,omitempty"`
	Scope       string `json:"scope,omitempty"`    // "user", "project", or empty
	Platform    string `json:"platform"`           // "claude_code" | "claude_desktop"
	Source      string `json:"source"`             // "installed" | "marketplace-available" | "rpm"
	InstallPath string `json:"install_path"`

	// History
	InstalledAt  string `json:"installed_at,omitempty"`
	LastUpdated  string `json:"last_updated,omitempty"`
	GitCommitSha string `json:"git_commit_sha,omitempty"`

	// Provenance
	MarketplaceRepo string `json:"marketplace_repo,omitempty"` // e.g. "github:anthropics/knowledge-work-plugins"

	// Content discovered inside the plugin tree
	Description    string   `json:"description,omitempty"`       // first non-heading line of README
	MCPServerNames []string `json:"mcp_server_names,omitempty"`  // names registered in .mcp.json
	SkillNames     []string `json:"skill_names,omitempty"`       // SKILL.md names under skills/
}

// ScanPlugins returns plugins discovered from both Claude Code and Claude
// Desktop trees, deduplicated by (platform, full_name, install_path).
//
// Errors on individual files are swallowed; the scan is fail-open.
func ScanPlugins(home string) []Plugin {
	if home == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		home = h
	}

	out := []Plugin{}
	seen := map[string]bool{}

	add := func(p Plugin) {
		key := p.Platform + "|" + p.FullName + "|" + p.InstallPath
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, p)
	}

	// --- Claude Code plugins -------------------------------------------------
	ccManifest := filepath.Join(home, ".claude", "plugins", "installed_plugins.json")
	ccMarketplaceRepos := map[string]string{} // marketplace name → "github:owner/repo"
	if m := readKnownMarketplaces(filepath.Join(home, ".claude", "plugins", "known_marketplaces.json")); m != nil {
		ccMarketplaceRepos = m
	}
	for _, p := range readInstalledPluginsManifest(ccManifest, PlatformClaudeCode, ccMarketplaceRepos) {
		add(p)
	}
	// Marketplace-available (may overlap with installed; seen-key dedupes)
	for _, p := range walkMarketplacePlugins(
		filepath.Join(home, ".claude", "plugins", "marketplaces"),
		PlatformClaudeCode, ccMarketplaceRepos,
	) {
		add(p)
	}

	// --- Claude Desktop (Cowork) plugins -------------------------------------
	// Desktop scopes plugin trees under each session dir. Walk every
	// cowork_plugins/ we find.
	desktopRoot := coworkAppSupportDir(home)
	if desktopRoot != "" {
		for _, coworkPluginsDir := range findCoworkPluginsDirs(desktopRoot) {
			dsMarketplaceRepos := map[string]string{}
			if m := readKnownMarketplaces(filepath.Join(coworkPluginsDir, "known_marketplaces.json")); m != nil {
				dsMarketplaceRepos = m
			}
			for _, p := range readInstalledPluginsManifest(
				filepath.Join(coworkPluginsDir, "installed_plugins.json"),
				PlatformClaudeDesktop, dsMarketplaceRepos,
			) {
				add(p)
			}
			for _, p := range walkMarketplacePlugins(
				filepath.Join(coworkPluginsDir, "marketplaces"),
				PlatformClaudeDesktop, dsMarketplaceRepos,
			) {
				add(p)
			}
		}
		// rpm/plugin_* alt install path
		for _, p := range findRpmPlugins(desktopRoot) {
			add(p)
		}
	}

	// Deterministic output
	sort.Slice(out, func(i, j int) bool {
		if out[i].Platform != out[j].Platform {
			return out[i].Platform < out[j].Platform
		}
		if out[i].FullName != out[j].FullName {
			return out[i].FullName < out[j].FullName
		}
		return out[i].InstallPath < out[j].InstallPath
	})
	return out
}

// --- manifest parsers -------------------------------------------------------

// readInstalledPluginsManifest parses installed_plugins.json (format shared
// between Claude Code and Claude Desktop Cowork). Each plugin entry is keyed
// by `<plugin>@<marketplace>` and maps to an array of install records.
func readInstalledPluginsManifest(path, platform string, marketplaceRepos map[string]string) []Plugin {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var manifest struct {
		Version int `json:"version"`
		Plugins map[string][]struct {
			Scope        string `json:"scope"`
			InstallPath  string `json:"installPath"`
			Version      string `json:"version"`
			InstalledAt  string `json:"installedAt"`
			LastUpdated  string `json:"lastUpdated"`
			GitCommitSha string `json:"gitCommitSha"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil
	}

	var out []Plugin
	for fullName, installs := range manifest.Plugins {
		name, marketplace := splitPluginFullName(fullName)
		for _, inst := range installs {
			p := Plugin{
				Name:         name,
				Marketplace:  marketplace,
				FullName:     fullName,
				Version:      inst.Version,
				Scope:        inst.Scope,
				Platform:     platform,
				Source:       "installed",
				InstallPath:  inst.InstallPath,
				InstalledAt:  inst.InstalledAt,
				LastUpdated:  inst.LastUpdated,
				GitCommitSha: inst.GitCommitSha,
			}
			if repo, ok := marketplaceRepos[marketplace]; ok {
				p.MarketplaceRepo = repo
			}
			enrichFromInstallDir(&p)
			out = append(out, p)
		}
	}
	return out
}

// walkMarketplacePlugins lists plugins that are cached in a marketplace tree
// but not necessarily formally installed. Shape: <marketplacesRoot>/<m>/<plugin>/
func walkMarketplacePlugins(marketplacesRoot, platform string, marketplaceRepos map[string]string) []Plugin {
	if marketplacesRoot == "" {
		return nil
	}
	if _, err := os.Stat(marketplacesRoot); err != nil {
		return nil
	}

	marketplaces, err := os.ReadDir(marketplacesRoot)
	if err != nil {
		return nil
	}

	var out []Plugin
	for _, m := range marketplaces {
		if !m.IsDir() {
			continue
		}
		marketplaceName := m.Name()
		marketplaceDir := filepath.Join(marketplacesRoot, marketplaceName)

		// Each subdir under the marketplace is a plugin (skip marketplace-level
		// metadata files like README.md, LICENSE).
		pluginDirs, err := os.ReadDir(marketplaceDir)
		if err != nil {
			continue
		}
		for _, pd := range pluginDirs {
			if !pd.IsDir() {
				continue
			}
			pluginName := pd.Name()
			installPath := filepath.Join(marketplaceDir, pluginName)
			p := Plugin{
				Name:        pluginName,
				Marketplace: marketplaceName,
				FullName:    pluginName + "@" + marketplaceName,
				Platform:    platform,
				Source:      "marketplace-available",
				InstallPath: installPath,
			}
			if repo, ok := marketplaceRepos[marketplaceName]; ok {
				p.MarketplaceRepo = repo
			}
			enrichFromInstallDir(&p)
			out = append(out, p)
		}
	}
	return out
}

// readKnownMarketplaces parses known_marketplaces.json and returns
// marketplace name → "github:owner/repo" style source string.
func readKnownMarketplaces(path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var raw map[string]struct {
		Source struct {
			Source string `json:"source"`
			Repo   string `json:"repo"`
		} `json:"source"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	out := map[string]string{}
	for name, v := range raw {
		if v.Source.Source != "" && v.Source.Repo != "" {
			out[name] = v.Source.Source + ":" + v.Source.Repo
		}
	}
	return out
}

// --- discovery helpers ------------------------------------------------------

// findCoworkPluginsDirs returns every path under the Claude Desktop app
// support tree that looks like a `cowork_plugins` root. Cowork scopes them
// per session, so there can be multiple.
func findCoworkPluginsDirs(desktopRoot string) []string {
	root := filepath.Join(desktopRoot, "local-agent-mode-sessions")
	if _, err := os.Stat(root); err != nil {
		return nil
	}
	var found []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if d.Name() == "cowork_plugins" {
			found = append(found, path)
			return filepath.SkipDir
		}
		// Cap depth — cowork_plugins sits at depth 2 below sessions root
		// (<sA>/<sB>/cowork_plugins), so 4 gives us slack.
		rel, relErr := filepath.Rel(root, path)
		if relErr == nil && strings.Count(rel, string(filepath.Separator)) > 4 {
			return filepath.SkipDir
		}
		return nil
	})
	return found
}

// findRpmPlugins walks Cowork's rpm/plugin_<id>/ alt-install subtree.
// These plugin trees do NOT have installed_plugins.json manifests — they
// are registered server-side and rendered locally. We surface them with
// source="rpm" so the dashboard can distinguish.
func findRpmPlugins(desktopRoot string) []Plugin {
	root := filepath.Join(desktopRoot, "local-agent-mode-sessions")
	if _, err := os.Stat(root); err != nil {
		return nil
	}
	var out []Plugin
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		base := d.Name()
		if strings.HasPrefix(base, "plugin_") && filepath.Base(filepath.Dir(path)) == "rpm" {
			// plugin_<id> is the install root; no friendly name (Cowork
			// assigns opaque ids server-side). Surface the id as both Name
			// and FullName; users can cross-reference in the Cowork UI.
			p := Plugin{
				Name:        base,
				FullName:    base,
				Platform:    PlatformClaudeDesktop,
				Source:      "rpm",
				InstallPath: path,
			}
			enrichFromInstallDir(&p)
			out = append(out, p)
			return filepath.SkipDir
		}
		// Depth cap
		rel, relErr := filepath.Rel(root, path)
		if relErr == nil && strings.Count(rel, string(filepath.Separator)) > 4 {
			return filepath.SkipDir
		}
		return nil
	})
	return out
}

// enrichFromInstallDir populates MCPServerNames, SkillNames, and Description
// by reading files inside the plugin's install path. Fail-open on any read
// error — a plugin without a .mcp.json or README is still a valid row.
func enrichFromInstallDir(p *Plugin) {
	if p.InstallPath == "" {
		return
	}
	p.MCPServerNames = readPluginMCPServerNames(filepath.Join(p.InstallPath, ".mcp.json"))
	p.SkillNames = listSkillNames(filepath.Join(p.InstallPath, "skills"))
	p.Description = readReadmeDescription(filepath.Join(p.InstallPath, "README.md"))
}

// readPluginMCPServerNames returns the sorted list of MCP server names the
// plugin registers via its .mcp.json.
func readPluginMCPServerNames(mcpJSONPath string) []string {
	data, err := os.ReadFile(mcpJSONPath)
	if err != nil {
		return nil
	}
	var cfg struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}
	if len(cfg.MCPServers) == 0 {
		return nil
	}
	names := make([]string, 0, len(cfg.MCPServers))
	for n := range cfg.MCPServers {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// listSkillNames returns names of immediate subdirs under skillsDir that
// contain a SKILL.md. Silent on missing/unreadable dirs.
func listSkillNames(skillsDir string) []string {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillMD := filepath.Join(skillsDir, e.Name(), "SKILL.md")
		if _, err := os.Stat(skillMD); err == nil {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names
}

// readReadmeDescription returns the first substantive line of a README —
// typically the one-line pitch under the main heading. Strips markdown
// leading characters and caps at 200 chars. Empty string on any failure.
func readReadmeDescription(readmePath string) string {
	f, err := os.Open(readmePath)
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 64*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue // skip heading(s)
		}
		if strings.HasPrefix(line, "<") {
			continue // skip HTML/markdown banners
		}
		line = strings.TrimSpace(line)
		if len(line) > 200 {
			line = line[:200] + "..."
		}
		return line
	}
	return ""
}

// splitPluginFullName parses "<plugin>@<marketplace>" into its parts. If the
// input doesn't contain @, returns (input, "").
func splitPluginFullName(s string) (name, marketplace string) {
	at := strings.LastIndex(s, "@")
	if at < 0 {
		return s, ""
	}
	return s[:at], s[at+1:]
}
