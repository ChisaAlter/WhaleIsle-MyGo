package harness

import (
	"os"
	"path/filepath"
	"strings"
	"whaleisle/internal/desktop/pluginslots"
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
	out := []string{viewPatch}
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
	return out, nil
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
