package harness

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"whaleisle/internal/desktop/pluginslots"
	"whaleisle/internal/desktop/pluginui"
)

const (
	sessionSearchOverlayFile = "desktop-session-search.patch.yml"
	marketOverlayFile        = "desktop-dsh-market.patch.yml"
	marketPackage            = "@deepseek-ai/dsh-client-ui-settings-market"
	marketInsertID           = "ui-settings-market"
)

func extraOverlays(o PrepareOptions) ([]string, error) {
	viewPatch, err := pluginslots.Prepare(o.ProfileDir)
	if err != nil {
		return nil, err
	}
	nativePatch, err := pluginui.Prepare(o.ProfileDir)
	if err != nil {
		return nil, err
	}
	out := []string{viewPatch, nativePatch}
	proactivePatch, err := writeProactiveOverlay(o)
	if err != nil {
		return nil, err
	}
	out = append(out, proactivePatch)
	if !o.SkipUserPlugins {
		p, err := writeSessionSearchOverlay(o)
		if err != nil {
			return nil, err
		}
		if p != "" {
			out = append(out, p)
		}
	}
	p, err := writeMarketOverlay(o)
	if err != nil {
		return nil, err
	}
	if p != "" {
		out = append(out, p)
	}
	office, err := EnsureOfficeOverlay(o)
	if err != nil {
		return nil, err
	}
	if office != "" {
		out = append(out, office)
	}
	extra, err := writeEnvOverlay(o)
	if err != nil {
		return nil, err
	}
	out = append(out, extra...)
	return out, nil
}

// EnvExtraOverlay is a development/QA hook: it names a directory holding one
// complete client plugin package (package.json + index.js + client.js + a
// *.patch.yml insert). The package is linked into the profile's node_modules
// and its patch joins the launch overlays, so the Host loads a real
// third-party plugin through the ordinary loader.
const EnvExtraOverlay = "WHALEISLE_EXTRA_OVERLAY"

func writeEnvOverlay(o PrepareOptions) ([]string, error) {
	dir := strings.TrimSpace(os.Getenv(EnvExtraOverlay))
	if dir == "" {
		return nil, nil
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("%s: %q is not a plugin directory", EnvExtraOverlay, dir)
	}
	link := filepath.Join(o.ProfileDir, "node_modules", filepath.Base(dir))
	if _, err := EnsureDirectoryLink(dir, link); err != nil {
		return nil, err
	}
	var patches []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".patch.yml") {
			patches = append(patches, filepath.Join(dir, entry.Name()))
		}
	}
	if len(patches) == 0 {
		return nil, fmt.Errorf("%s: %q has no *.patch.yml insert", EnvExtraOverlay, dir)
	}
	return patches, nil
}

func writeSessionSearchOverlay(o PrepareOptions) (string, error) {
	home := strings.TrimSpace(o.DshHome)
	if home == "" {
		return "", nil
	}
	index := filepath.Join(home, "session-query.sqlite")
	posix := filepath.ToSlash(index)
	contents := strings.Join([]string{
		"# Desktop-managed overlay passed to full starts via --patch.",
		"# Enables Web content search (session.search snippets). Skip starts do",
		"# not pass this file. Regenerated on every full start; do not edit.",
		"- id: session-query-sqlite",
		"  config:",
		"    path: " + quoteYAMLString(posix),
		"    openAt: first-search",
		"",
	}, "\n")
	return writeRawOverlay(o.ProfileDir, "session-search", sessionSearchOverlayFile, contents)
}

func writeMarketOverlay(o PrepareOptions) (string, error) {
	if o.HarnessRoot != "" {
		source := resolveMarketSource(o.HarnessRoot)
		if source != "" {
			link := filepath.Join(o.ProfileDir, "node_modules", "@deepseek-ai", "dsh-client-ui-settings-market")
			if _, err := EnsureDirectoryLink(source, link); err != nil {
				return "", err
			}
		}
	}
	p := DesktopPlugin{
		Package:     marketPackage,
		InsertID:    marketInsertID,
		OverlayFile: marketOverlayFile,
		OverlayDir:  "dsh-market",
	}
	return writeInsertOverlay(o.ProfileDir, p)
}

func resolveMarketSource(harnessRoot string) string {
	candidates := []string{
		filepath.Join(harnessRoot, "packages", "client", "ui-settings-market"),
		filepath.Join(harnessRoot, "node_modules", "@deepseek-ai", "dsh-client-ui-settings-market"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "package.json")); err == nil {
			return c
		}
	}
	return ""
}

func writeRawOverlay(profileDir, overlayDir, filename, contents string) (string, error) {
	dir := filepath.Join(profileDir, "desktop-plugins", overlayDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, filename)
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
