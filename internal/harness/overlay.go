package harness

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"whaleisle/internal/startuprec"
)

// DesktopPlugin describes one vendor package mounted via junction + --patch.
type DesktopPlugin struct {
	// VendorName is the directory under runtime/vendor/.
	VendorName string
	// Package is the npm name written into the overlay insert.
	Package string
	// InsertID is the cordis insert id.
	InsertID string
	// OverlayFile is the filename under desktop-plugins/<OverlayDir>/.
	OverlayFile string
	// OverlayDir is the subdirectory under desktop-plugins/ (often VendorName).
	OverlayDir string
	// ModuleLink is the path under profile node_modules/ (may be scoped).
	ModuleLink string
	// Required fails Prepare when the vendor copy is missing.
	Required bool
}

// PrepareOptions selects which optional desktop plugins to mount.
type PrepareOptions struct {
	DisabledPlugins []string
	// VendorDir defaults to paths resolved by the caller.
	VendorDir string
	// ProfileDir is dsh-home/profiles/web.
	ProfileDir string
	Dshbot     bool
	Whale      bool
	// Remote defaults to true when RemoteSet is false (Electron default ON).
	Remote    bool
	RemoteSet bool
	// SkipUserPlugins omits the session-search overlay (Electron skip starts).
	SkipUserPlugins bool
	// DshHome is used by the session-search overlay (session-query.sqlite).
	DshHome string
	// HarnessRoot locates the market fork package for a node_modules junction.
	HarnessRoot string
}

// builtinPlugins are always mounted when present (Electron always-on built-ins
// that ship in runtime/vendor). Market / office / session-search / install
// overlays are not yet ported (sources live inside Electron or harness forks).
func builtinPlugins() []DesktopPlugin {
	return []DesktopPlugin{
		{
			VendorName:  "dsh-task-control",
			Package:     "dsh-task-control",
			InsertID:    "dsh-task-control",
			OverlayFile: "desktop-task-control.patch.yml",
			OverlayDir:  "dsh-task-control",
			ModuleLink:  "dsh-task-control",
			Required:    true,
		},
		{
			VendorName:  "dsh-platform-session",
			Package:     "dsh-platform-session",
			InsertID:    "dsh-platform-session",
			OverlayFile: "desktop-platform-session.patch.yml",
			OverlayDir:  "dsh-platform-session",
			ModuleLink:  "dsh-platform-session",
			Required:    true,
		},
		{
			VendorName:  "dsh-usage-panel",
			Package:     "dsh-usage-panel",
			InsertID:    "usage-stats",
			OverlayFile: "desktop-usage-panel.patch.yml",
			OverlayDir:  "dsh-usage-panel",
			ModuleLink:  "dsh-usage-panel",
			Required:    true,
		},
		{
			VendorName:  "dsh-im",
			Package:     "@xmanrui/dsh-im",
			InsertID:    "xmanrui-dsh-im",
			OverlayFile: "desktop-dsh-im.patch.yml",
			OverlayDir:  "dsh-im",
			ModuleLink:  filepath.Join("@xmanrui", "dsh-im"),
			Required:    true,
		},
	}
}

func optionalPlugins(o PrepareOptions) []DesktopPlugin {
	remote := true
	if o.RemoteSet {
		remote = o.Remote
	}
	var out []DesktopPlugin
	if o.Dshbot {
		out = append(out, DesktopPlugin{
			VendorName:  "dshbot",
			Package:     "dshbot",
			InsertID:    "dsh-bot",
			OverlayFile: "desktop-dshbot.patch.yml",
			OverlayDir:  "dshbot",
			ModuleLink:  "dshbot",
			Required:    true,
		})
	}
	if o.Whale {
		out = append(out, DesktopPlugin{
			VendorName:  "dsh-whale",
			Package:     "dsh-whale",
			InsertID:    "dsh-whale",
			OverlayFile: "desktop-dsh-whale.patch.yml",
			OverlayDir:  "dsh-whale",
			ModuleLink:  "dsh-whale",
			Required:    true,
		})
	}
	if remote {
		out = append(out, DesktopPlugin{
			VendorName:  "dsh-remote",
			Package:     "dsh-remote",
			InsertID:    "dsh-remote",
			OverlayFile: "desktop-dsh-remote.patch.yml",
			OverlayDir:  "dsh-remote",
			ModuleLink:  "dsh-remote",
			Required:    true,
		})
	}
	return out
}

// Prepare mounts desktop plugins into the web profile: junctions under
// node_modules and --patch overlay YAML files. Returns absolute overlay
// paths in Electron push order (built-ins, then optionals).
func Prepare(o PrepareOptions) ([]string, error) {
	if o.VendorDir == "" || o.ProfileDir == "" {
		return nil, fmt.Errorf("harness: VendorDir and ProfileDir are required")
	}
	if err := (startuprec.OfflinePlugins{ProfileDir: o.ProfileDir}).ApplyDisabled(o.DisabledPlugins); err != nil {
		return nil, fmt.Errorf("disabled plugins: %w", err)
	}
	plugins := append(builtinPlugins(), optionalPlugins(o)...)
	var patches []string
	for _, p := range plugins {
		source := filepath.Join(o.VendorDir, p.VendorName)
		if _, err := os.Stat(filepath.Join(source, "package.json")); err != nil {
			if p.Required {
				return nil, fmt.Errorf("desktop plugin %s: missing package.json under %s", p.VendorName, source)
			}
			continue
		}
		link := filepath.Join(o.ProfileDir, "node_modules", p.ModuleLink)
		if _, err := EnsureDirectoryLink(source, link); err != nil {
			return nil, fmt.Errorf("junction %s: %w", p.VendorName, err)
		}
		var overlayPath string
		var err error
		if p.VendorName == "dshbot" {
			overlayPath, err = writeDshbotRoomOverlay(o, p)
		} else {
			overlayPath, err = writeInsertOverlay(o.ProfileDir, p)
		}
		if err != nil {
			return nil, err
		}
		patches = append(patches, overlayPath)
	}
	extra, err := extraOverlays(o)
	if err != nil {
		return nil, err
	}
	return append(patches, extra...), nil
}

func writeInsertOverlay(profileDir string, p DesktopPlugin) (string, error) {
	dir := filepath.Join(profileDir, "desktop-plugins", p.OverlayDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, p.OverlayFile)
	contents := overlayYAML(p.InsertID, p.Package)
	if existing, err := os.ReadFile(path); err == nil && string(existing) == contents {
		return path, nil
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(contents), 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return path, nil
}

func overlayYAML(insertID, packageName string) string {
	var b strings.Builder
	b.WriteString("# Desktop-managed overlay passed to EVERY start via --patch.\n")
	b.WriteString("# Regenerated on every start; do not edit.\n")
	b.WriteString("- insert:\n")
	b.WriteString("    - id: ")
	b.WriteString(insertID)
	b.WriteByte('\n')
	b.WriteString("      name: ")
	b.WriteString(quoteYAMLString(packageName))
	b.WriteByte('\n')
	b.WriteByte('\n')
	return b.String()
}

func quoteYAMLString(s string) string {
	// Mirror Electron's JSON.stringify for the name field.
	escaped := strings.ReplaceAll(s, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}
