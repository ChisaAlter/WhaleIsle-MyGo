package installctrl

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// ListNodeModuleNames returns top-level package names under profile/node_modules
// (including @scope/name). Mirrors Electron marketplace-install listNodeModuleNames.
func ListNodeModuleNames(profileDir string) []string {
	root := filepath.Join(profileDir, "node_modules")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == ".bin" || entry.Name() == ".pnpm" {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "@") {
			nested, err := os.ReadDir(filepath.Join(root, name))
			if err != nil {
				continue
			}
			for _, child := range nested {
				if child.IsDir() {
					names = append(names, name+"/"+child.Name())
				}
			}
			continue
		}
		names = append(names, name)
	}
	return names
}

type packageJSON struct {
	Name    string          `json:"name"`
	Version string          `json:"version"`
	Main    string          `json:"main"`
	Exports json.RawMessage `json:"exports"`
	Dsh     *struct {
		Client json.RawMessage `json:"client"`
		Bundle *struct {
			Patch string `json:"patch"`
		} `json:"bundle"`
	} `json:"dsh"`
}

func readPackageJSON(dir string) (*packageJSON, error) {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil, err
	}
	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}
	return &pkg, nil
}

func packageInstallDir(profileDir, packageName string) string {
	return filepath.Join(profileDir, "node_modules", filepath.FromSlash(packageName))
}

func isExistingFile(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func resolveExportFile(pkg *packageJSON, dir, key string) string {
	if len(pkg.Exports) == 0 {
		return ""
	}
	var asString string
	if err := json.Unmarshal(pkg.Exports, &asString); err == nil {
		if key == "." {
			return filepath.Join(dir, filepath.FromSlash(asString))
		}
		return ""
	}
	var asMap map[string]json.RawMessage
	if err := json.Unmarshal(pkg.Exports, &asMap); err != nil {
		return ""
	}
	entry, ok := asMap[key]
	if !ok {
		return ""
	}
	var rel string
	if err := json.Unmarshal(entry, &rel); err == nil {
		return filepath.Join(dir, filepath.FromSlash(rel))
	}
	var obj map[string]string
	if err := json.Unmarshal(entry, &obj); err != nil {
		return ""
	}
	for _, k := range []string{"default", "import", "require"} {
		if v := obj[k]; v != "" {
			return filepath.Join(dir, filepath.FromSlash(v))
		}
	}
	return ""
}

// HasLoadableEntry reports whether packageName under the web profile has a
// dsh patch/client, main, or exports entry that resolves to an existing file.
func HasLoadableEntry(profileDir, packageName string) bool {
	dir := packageInstallDir(profileDir, packageName)
	pkg, err := readPackageJSON(dir)
	if err != nil || pkg == nil {
		return false
	}
	if pkg.Dsh != nil && pkg.Dsh.Bundle != nil {
		if patch := strings.TrimSpace(pkg.Dsh.Bundle.Patch); patch != "" {
			if isExistingFile(filepath.Join(dir, filepath.FromSlash(patch))) {
				return true
			}
		}
	}
	if pkg.Dsh != nil && len(pkg.Dsh.Client) > 0 {
		var clientPath string
		if err := json.Unmarshal(pkg.Dsh.Client, &clientPath); err == nil && clientPath != "" {
			if isExistingFile(filepath.Join(dir, filepath.FromSlash(clientPath))) {
				return true
			}
		} else if isExistingFile(resolveExportFile(pkg, dir, "./client")) {
			return true
		}
	}
	if main := strings.TrimSpace(pkg.Main); main != "" {
		if isExistingFile(filepath.Join(dir, filepath.FromSlash(main))) {
			return true
		}
	}
	return isExistingFile(resolveExportFile(pkg, dir, "."))
}

// HasInventoryIdentity reports whether the package manifest has non-empty
// name and version (required by dsh request inventory).
func HasInventoryIdentity(profileDir, packageName string) bool {
	pkg, err := readPackageJSON(packageInstallDir(profileDir, packageName))
	if err != nil || pkg == nil {
		return false
	}
	return strings.TrimSpace(pkg.Name) != "" && strings.TrimSpace(pkg.Version) != ""
}

// ResolveAddedPackageNames returns package names that appeared after install
// plus the github repo name heuristic for github:owner/repo specs.
func ResolveAddedPackageNames(spec string, before, after []string) []string {
	prev := make(map[string]struct{}, len(before))
	for _, n := range before {
		prev[n] = struct{}{}
	}
	added := make([]string, 0)
	seen := make(map[string]struct{})
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		added = append(added, name)
	}
	for _, n := range after {
		if _, ok := prev[n]; !ok {
			add(n)
		}
	}
	if m := githubSpecPattern.FindStringSubmatch(strings.TrimSpace(spec)); m != nil {
		repo := m[2]
		for _, n := range after {
			if n == repo || strings.HasSuffix(n, "/"+repo) {
				add(n)
			}
		}
	} else if name := RegistryPluginName(spec); name != "" {
		add(name)
	} else if isValidPackageName(spec) {
		add(spec)
	}
	return added
}

// ValidateLoadablePackages ensures every name is loadable and has identity.
// Returns a Chinese error message when validation fails (empty on success).
func ValidateLoadablePackages(profileDir string, names []string) string {
	if len(names) == 0 {
		return "该包不是可加载的 dsh 插件"
	}
	for _, name := range names {
		if !HasLoadableEntry(profileDir, name) {
			return "该包不是可加载的 dsh 插件"
		}
		if !HasInventoryIdentity(profileDir, name) {
			return "插件包 " + name + " 缺少 name 或 version 声明"
		}
	}
	return ""
}
