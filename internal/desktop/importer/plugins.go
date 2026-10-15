package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"whaleisle/internal/harness/installctrl"
)

// PluginInstaller must use the confirmed destination and must not auto-approve builds.
type PluginInstaller func(context.Context, CatalogItem, string) PluginInstallResult

type PluginInstallResult struct {
	OK    bool
	Error string
	Log   string
}

type PluginResult struct {
	Name   string
	Status string
	Reason string
	Error  string
	Log    string
}

// PluginReinstallSpec matches Electron's GitHub and pinned registry channels.
// Empty means unsupported; never pass an arbitrary manifest string to the CLI.
func PluginReinstallSpec(name, spec string) string {
	value := strings.TrimSpace(spec)
	if installctrl.IsValidGithubSpec(value) {
		return value
	}
	candidate := name + "@" + value
	if installctrl.RegistryPluginName(candidate) != "" {
		return candidate
	}
	return ""
}

func pluginSkipReason(name, spec string) string {
	switch name {
	case "@deepseek-ai/dsh-base", "@deepseek-ai/dsh-web-app":
		return "template"
	case "@dsh-external/dsh-genui", "@huanlin/dsh-plugin-yet-another-subagent",
		"dshmarket", "@xmanrui/dsh-im", "dsh-im", "xmanrui-dsh-im",
		"dsh-usage-panel", "dshbot", "dsh-whale", "dsh-remote":
		return "dropped"
	}
	value := strings.ToLower(strings.TrimSpace(spec))
	for _, prefix := range []string{"file:", "link:", "workspace:"} {
		if strings.HasPrefix(value, prefix) {
			return "local-spec"
		}
	}
	if PluginReinstallSpec(name, spec) == "" {
		return "unsupported"
	}
	return ""
}

func readPluginDependencies(home string) (map[string]json.RawMessage, error) {
	file := filepath.Join(home, "profiles", "web", "package.json")
	f, err := os.Open(file)
	if os.IsNotExist(err) {
		return map[string]json.RawMessage{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	const limit = 4 << 20
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if len(data) > limit {
		return nil, fmt.Errorf("plugin manifest exceeds size limit")
	}
	var manifest struct {
		Dependencies map[string]json.RawMessage `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("invalid plugin manifest: %w", err)
	}
	return manifest.Dependencies, nil
}

func pluginInstalled(home, name string) (bool, error) {
	deps, err := readPluginDependencies(home)
	if err != nil {
		return false, err
	}
	if _, ok := deps[name]; ok {
		return true, nil
	}
	// The manifest key need not be a valid package name for a GitHub spec.
	// Do not construct a filesystem path from an unvalidated dependency key.
	if installctrl.RegistryPluginName(name+"@0.0.0") == "" {
		return false, nil
	}
	for _, root := range []string{filepath.Join(home, "profiles", "web", "node_modules"), filepath.Join(home, "plugins"), filepath.Join(home, "user-plugins")} {
		if entryExists(filepath.Join(root, filepath.FromSlash(name)), true) {
			return true, nil
		}
	}
	return false, nil
}

func scanPlugins(src, dst string) []CatalogItem {
	manifest := filepath.Join(src, "profiles", "web", "package.json")
	if !entryExists(manifest, false) {
		// Legacy directory-only plugins have no reproducible source spec.
		var out []CatalogItem
		for _, sub := range []string{"plugins", "user-plugins"} {
			dir := filepath.Join(src, sub)
			entries, err := os.ReadDir(dir)
			if err != nil { continue }
			for _, entry := range entries {
				if !entry.IsDir() { continue }
				out = append(out, CatalogItem{
					ID: entry.Name(), Label: entry.Name(), Abs: filepath.Join(dir, entry.Name()),
					Conflict: entryExists(filepath.Join(dst, sub, entry.Name()), true),
					Unsupported: true, Group: "local-spec",
				})
			}
		}
		return out
	}
	deps, err := readPluginDependencies(src)
	if err != nil { return nil }
	names := make([]string, 0, len(deps))
	for name := range deps { names = append(names, name) }
	sort.Strings(names)
	var out []CatalogItem
	for _, name := range names {
		var spec string
		_ = json.Unmarshal(deps[name], &spec)
		spec = strings.TrimSpace(spec)
		reason := pluginSkipReason(name, spec)
		conflict, err := pluginInstalled(dst, name)
		// A broken destination manifest must never imply that overwrite is safe.
		if err != nil { conflict = true }
		out = append(out, CatalogItem{
			ID: name, Label: name, Spec: spec, Group: reason,
			Unsupported: reason != "", Conflict: conflict,
		})
	}
	return out
}
