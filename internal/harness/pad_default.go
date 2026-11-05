package harness

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"whaleisle/internal/desktop/openui"
)

// PreparePadDefault supplies the web-profile default layer, before profile/home
// overrides. It deliberately does not use --patch, which would override users.
func PreparePadDefault(o PrepareOptions) (string, error) {
	if slices.Contains(o.DisabledPlugins, "dsh-pad") {
		return "", nil
	}
	var profile struct {
		Dependencies map[string]json.RawMessage `json:"dependencies"`
		DSH          struct {
			Profile struct {
				Bundles []string `json:"bundles"`
			} `json:"profile"`
		} `json:"dsh"`
	}
	if raw, err := os.ReadFile(filepath.Join(o.ProfileDir, "package.json")); err == nil {
		if err = json.Unmarshal(raw, &profile); err != nil {
			return "", fmt.Errorf("pad profile: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	_, installed := profile.Dependencies["dsh-pad"]
	selected := slices.Contains(profile.DSH.Profile.Bundles, "dsh-pad")
	if installed && !selected || selected && !o.SkipUserPlugins {
		return "", nil
	}
	source, err := filepath.Abs(filepath.Join(o.VendorDir, "dsh-pad"))
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(filepath.Join(source, "package.json"))
	// Custom/non-desktop compositions remain usable without this optional asset.
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var manifest struct {
		Name string `json:"name"`
	}
	if err = json.Unmarshal(raw, &manifest); err != nil || manifest.Name != "dsh-pad" {
		return "", fmt.Errorf("invalid bundled Pad manifest: %s", source)
	}
	for _, name := range []string{"lib/index.js", "lib/client.js", "cordis.patch.yml"} {
		if _, err = os.Stat(filepath.Join(source, filepath.FromSlash(name))); err != nil {
			return "", fmt.Errorf("bundled Pad %s: %w", name, err)
		}
	}
	if _, err = EnsureDirectoryLink(source, filepath.Join(o.ProfileDir, "node_modules", "whaleisle-desktop-pad")); err != nil {
		return "", err
	}
	path := filepath.ToSlash(filepath.Join(source, "lib", "index.js"))
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	module := (&url.URL{Scheme: "file", Path: path}).String()
	return writeRawOverlay(o.ProfileDir, "dsh-pad", "desktop-dsh-pad.patch.yml", "# Desktop defaults, before user configuration.\n- insert:\n    - id: dsh-pad\n      name: "+quoteYAMLString(module)+"\n      config:\n        updateCheck: false\n")
}

func prepareWhaleUIHost(home string) (string, error) {
	path := filepath.Join(home, "runtime", "whale-ui", "host.cjs")
	data := openui.WhaleHostValidator()
	if old, err := os.ReadFile(path); err == nil && string(old) == string(data) {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path+".tmp", data, 0600); err != nil {
		return "", err
	}
	if err := os.Rename(path+".tmp", path); err != nil {
		return "", err
	}
	return path, nil
}
