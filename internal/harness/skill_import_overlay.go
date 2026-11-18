package harness

import (
	_ "embed"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

//go:embed native-skill-import.mjs
var nativeSkillImportModule string

//go:embed native-mcp-import.mjs
var nativeMCPImportModule string

func writeMCPImportOverlay(o PrepareOptions) (string, error) {
	document := filepath.Join(o.HarnessRoot, "packages", "mcp", "mcp-servers-file", "lib", "types", "document.js")
	if o.HarnessRoot == "" {
		return "", nil
	}
	if st, err := os.Stat(document); err != nil || !st.Mode().IsRegular() {
		return "", nil
	}
	documentURL, _ := json.Marshal(overlayFileURL(document))
	source := strings.ReplaceAll(nativeMCPImportModule, "'__MYGO_MCP_DOCUMENT__'", string(documentURL))
	entry, err := writeRawOverlay(o.ProfileDir, "mygo-mcp-import", "index.mjs", source)
	if err != nil {
		return "", err
	}
	module, _ := json.Marshal(overlayFileURL(entry))
	return writeRawOverlay(o.ProfileDir, "mygo-mcp-import", "desktop-mcp-import.patch.yml", "- insert:\n    - id: mygo-mcp-import\n      name: "+string(module)+"\n")
}

func overlayFileURL(path string) string {
	absolute, _ := filepath.Abs(path)
	p := filepath.ToSlash(absolute)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return (&url.URL{Scheme: "file", Path: p}).String()
}

// The optional Adapter extends known filesystem-skill Hosts; missing support
// only disables directory import, never the existing Host startup contract.
func writeSkillImportOverlay(o PrepareOptions) (string, error) {
	frontmatter := filepath.Join(o.HarnessRoot, "packages", "host", "skill-inventory", "lib", "types", "frontmatter.js")
	homePaths := filepath.Join(o.HarnessRoot, "packages", "util", "home-paths", "lib", "index.js")
	provider := filepath.Join(o.HarnessRoot, "packages", "skill", "skill-filesystem", "lib", "types", "index.js")
	if o.HarnessRoot == "" {
		return "", nil
	}
	for _, path := range []string{frontmatter, homePaths, provider} {
		if st, err := os.Stat(path); err != nil || !st.Mode().IsRegular() {
			return "", nil
		}
	}
	frontmatterURL, _ := json.Marshal(overlayFileURL(frontmatter))
	homeURL, _ := json.Marshal(overlayFileURL(homePaths))
	providerURL, _ := json.Marshal(overlayFileURL(provider))
	source := strings.ReplaceAll(nativeSkillImportModule, "'__MYGO_SKILL_FRONTMATTER__'", string(frontmatterURL))
	source = strings.ReplaceAll(source, "'__MYGO_HOME_PATHS__'", string(homeURL))
	source = strings.ReplaceAll(source, "'__MYGO_SKILL_PROVIDER__'", string(providerURL))
	entry, err := writeRawOverlay(o.ProfileDir, "mygo-skill-import", "index.mjs", source)
	if err != nil {
		return "", err
	}
	module, _ := json.Marshal(overlayFileURL(entry))
	return writeRawOverlay(o.ProfileDir, "mygo-skill-import", "desktop-skill-import.patch.yml", "- insert:\n    - id: mygo-skill-import\n      name: "+string(module)+"\n")
}
