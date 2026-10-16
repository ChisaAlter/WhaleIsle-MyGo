package pluginslots

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

const PackageName = "whaleisle-desktop-plugin-views"

//go:embed client.js
var clientSource []byte

//go:embed keyboard.js
var keyboardSource []byte

//go:embed shortcut_import.js
var shortcutImportSource []byte

//go:embed document_viewer.js
var documentViewerSource []byte

func packagedClientSource() []byte {
	return append(append(append(append([]byte(nil), keyboardSource...), shortcutImportSource...), documentViewerSource...), clientSource...)
}

// Prepare adds an application-owned client adapter; user/vendor files are untouched.
func Prepare(profile string) (string, error) {
	if profile == "" {
		return "", fmt.Errorf("plugin views: profile required")
	}
	dir := filepath.Join(profile, "node_modules", PackageName)
	files := map[string][]byte{
		"package.json": []byte(`{"name":"whaleisle-desktop-plugin-views","version":"1.0.0","type":"module","main":"index.js","exports":{".":"./index.js","./client":"./client.js","./package.json":"./package.json"},"dsh":{"client":{"platform":"web","inject":["@deepseek-ai/dsh-client-shortcuts","@deepseek-ai/dsh-client-ui-layout","@deepseek-ai/dsh-client-ui-sidebar","@deepseek-ai/dsh-client-ui-settings-general","@deepseek-ai/dsh-client-ui-sidebar-documentpreview","@deepseek-ai/dsh-client-ui-workspace"],"external":["react"]}}}`),
		"index.js":     []byte("export const name = 'whaleisle-desktop-plugin-views';\nexport function apply() {}\n"),
		"client.js":    packagedClientSource(),
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0644); err != nil {
			return "", err
		}
	}
	patch := filepath.Join(dir, "desktop-plugin-views.patch.yml")
	data := "- insert:\n    - id: whaleisle-desktop-plugin-views\n      name: whaleisle-desktop-plugin-views\n"
	if err := os.WriteFile(patch, []byte(data), 0644); err != nil {
		return "", err
	}
	return patch, nil
}
