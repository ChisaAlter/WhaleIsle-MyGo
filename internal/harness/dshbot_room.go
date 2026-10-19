package harness

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed dshbot-room-adapter.mjs
var dshbotRoomAdapter string

// The original dshbot package still provisions .agent-presets on disk. rc.2
// needs the same definition registered with the live registry instead. Keep
// this adaptation inside the already selected dsh-bot entry and its profile.
func writeDshbotRoomOverlay(o PrepareOptions, plugin DesktopPlugin) (string, error) {
	root := o.HarnessRoot
	if root == "" {
		root = filepath.Join(o.VendorDir, "deepseek-harness")
	}
	registry, err := os.ReadFile(filepath.Join(root, "packages", "preset", "agent-preset-registry", "package.json"))
	if os.IsNotExist(err) {
		return writeInsertOverlay(o.ProfileDir, plugin)
	}
	if err != nil {
		return "", err
	}
	var identity struct{ Name, Version string }
	if err = json.Unmarshal(registry, &identity); err != nil {
		return "", err
	}
	if identity.Name != "@deepseek-ai/dsh-agent-preset-registry" || identity.Version != "0.2.1-rc.2" {
		return writeInsertOverlay(o.ProfileDir, plugin)
	}
	source := filepath.Join(o.VendorDir, plugin.VendorName)
	preset := filepath.Join(source, "presets", "dshbot-room")
	metadata, err := os.ReadFile(filepath.Join(preset, "preset.yml"))
	if err != nil {
		return "", fmt.Errorf("dshbot room preset metadata: %w", err)
	}
	composition, err := os.ReadFile(filepath.Join(preset, "agent.cordis.yml"))
	if err != nil {
		return "", fmt.Errorf("dshbot room composition: %w", err)
	}
	var definition map[string]any
	var plugins []map[string]any
	if err = yaml.Unmarshal(metadata, &definition); err != nil {
		return "", err
	}
	if err = yaml.Unmarshal(composition, &plugins); err != nil {
		return "", err
	}
	if definition == nil || len(plugins) == 0 {
		return "", fmt.Errorf("dshbot room preset definition is empty")
	}
	definition["id"], definition["plugins"] = "dshbot-room", plugins
	definitionJSON, err := json.Marshal(definition)
	if err != nil {
		return "", err
	}
	entry, err := filepath.Abs(filepath.Join(source, "lib", "index.js"))
	if err != nil {
		return "", err
	}
	filePath := filepath.ToSlash(entry)
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}
	entryJSON, _ := json.Marshal((&url.URL{Scheme: "file", Path: filePath}).String())
	module := strings.ReplaceAll(dshbotRoomAdapter, "__ORIGINAL_DSHBOT_MODULE__", string(entryJSON))
	module = strings.ReplaceAll(module, "__ORIGINAL_ROOM_DEFINITION__", string(definitionJSON))
	adapter, err := writeRawOverlay(o.ProfileDir, plugin.OverlayDir, "room-adapter.mjs", module)
	if err != nil {
		return "", err
	}

	// The compatibility preflight resolves a file entry's nearest manifest.
	// Preserve the original identity and requirements rather than allowing a
	// loose file to bypass that check. The original node_modules junction and
	// all bare dshbot subpath imports remain unchanged.
	raw, err := os.ReadFile(filepath.Join(source, "package.json"))
	if err != nil {
		return "", err
	}
	var manifest map[string]json.RawMessage
	if err = json.Unmarshal(raw, &manifest); err != nil {
		return "", err
	}
	delete(manifest, "exports") // Do not turn bare dshbot imports into self imports.
	manifest["main"] = json.RawMessage(`"./room-adapter.mjs"`)
	var dsh map[string]json.RawMessage
	if len(manifest["dsh"]) > 0 {
		if err = json.Unmarshal(manifest["dsh"], &dsh); err != nil {
			return "", err
		}
		delete(dsh, "bundle") // The existing optional entry owns activation.
		manifest["dsh"], err = json.Marshal(dsh)
		if err != nil {
			return "", err
		}
	}
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", err
	}
	if _, err = writeRawOverlay(o.ProfileDir, plugin.OverlayDir, "package.json", string(manifestJSON)+"\n"); err != nil {
		return "", err
	}
	client, err := os.ReadFile(filepath.Join(source, "client", "client.js"))
	if err != nil {
		return "", err
	}
	if _, err = writeRawOverlay(o.ProfileDir, filepath.Join(plugin.OverlayDir, "client"), "client.js", string(client)); err != nil {
		return "", err
	}
	return writeRawOverlay(o.ProfileDir, plugin.OverlayDir, plugin.OverlayFile, overlayYAML(plugin.InsertID, filepath.ToSlash(adapter)))
}
