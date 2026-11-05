// Package pluginui presents live Plugins contributions through their original
// browser page owner. It never manufactures a Host RPC or executes plugin text.
package pluginui

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

const PackageName = "whaleisle-native-plugin-ui"

//go:embed client.js
var clientSource []byte

//go:embed session.js
var sessionSource []byte

//go:embed right_pane.js
var rightPaneSource []byte

//go:embed navigation.js
var navigationSource []byte

//go:embed workspace.js
var workspaceSource []byte

//go:embed public_slots.js
var publicSlotsSource []byte

//go:embed rich_controls.js
var richControlsSource []byte

//go:embed native_document.js
var nativeDocumentSource []byte

// Prepare writes only an application-owned package in the selected profile.
func Prepare(profile string) (string, error) {
	if profile == "" {
		return "", fmt.Errorf("plugin UI: profile required")
	}
	dir := filepath.Join(profile, "node_modules", PackageName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	files := map[string][]byte{
		"package.json": []byte(`{"name":"whaleisle-native-plugin-ui","version":"1.0.0","type":"module","main":"index.js","exports":{".":"./index.js","./client":"./client.js","./package.json":"./package.json"},"dsh":{"client":{"platform":"web","inject":["@deepseek-ai/dsh-client-ui-plugin-manager","@deepseek-ai/dsh-client-locale","@deepseek-ai/dsh-client-ui-layout"]}}}`),
		"index.js":     []byte("export const name = 'whaleisle-native-plugin-ui';\nexport function apply() {}\n"),
		"client.js":    bytes.Join([][]byte{richControlsSource, nativeDocumentSource, rightPaneSource, workspaceSource, publicSlotsSource, navigationSource, clientSource, sessionSource}, []byte("\n")),
	}
	for name, contents := range files {
		if err := os.WriteFile(filepath.Join(dir, name), contents, 0644); err != nil {
			return "", err
		}
	}
	patch := filepath.Join(dir, "native-plugin-ui.patch.yml")
	if err := os.WriteFile(patch, []byte("- insert:\n    - id: "+PackageName+"\n      name: "+PackageName+"\n"), 0644); err != nil {
		return "", err
	}
	return patch, nil
}
