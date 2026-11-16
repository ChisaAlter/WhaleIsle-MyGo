package harness

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"whaleisle/internal/desktop/pluginslots"
)

const (
	sessionSearchOverlayFile = "desktop-session-search.patch.yml"
	routeAOverlayFile        = "desktop-route-a.patch.yml"
)

func extraOverlays(o PrepareOptions) ([]string, error) {
	viewPatch, err := pluginslots.Prepare(o.ProfileDir)
	if err != nil {
		return nil, err
	}
	routeAPatch, err := writeRouteAHostOverlay(o)
	if err != nil {
		return nil, err
	}
	out := []string{viewPatch, routeAPatch}
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

func writeRouteAHostOverlay(o PrepareOptions) (string, error) {
	lines := []string{
		"# Desktop-managed Route A overlay passed via --patch.",
		"# Disables background @deepseek-ai/dsh-client-hmr 500ms disk polling while",
		"# retaining client module metadata for on-demand compat views.",
		"- id: client-hmr",
		"  disabled: true",
	}

	if o.HarnessRoot != "" {
		hasLibreOffice := hasPackageJSON(o.HarnessRoot, "node_modules/@deepseek-ai/libreoffice-kit/package.json") ||
			hasPackageJSON(o.HarnessRoot, "apps/cli/node_modules/@deepseek-ai/libreoffice-kit/package.json")
		if !hasLibreOffice {
			lines = append(lines,
				"- id: office-to-pdf",
				"  disabled: true",
			)
		}

		hasSherpaOnnx := hasPackageJSON(o.HarnessRoot, "node_modules/sherpa-onnx-node/package.json") ||
			hasPackageJSON(o.HarnessRoot, "packages/experimental/speech-to-text-sensevoice/node_modules/sherpa-onnx-node/package.json")
		if !hasSherpaOnnx {
			lines = append(lines,
				"- id: speech-to-text-sensevoice",
				"  disabled: true",
			)
		}
	}

	lines = append(lines, "")
	contents := strings.Join(lines, "\n")
	return writeRawOverlay(o.ProfileDir, "route-a", routeAOverlayFile, contents)
}

func hasPackageJSON(root, rel string) bool {
	p := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
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

func writeRawOverlay(profileDir, overlayDir, filename, contents string) (string, error) {
	dir := filepath.Join(profileDir, "desktop-plugins", overlayDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, filename)
	if existing, err := os.ReadFile(path); err == nil && string(existing) == contents {
		return path, nil
	}
	if err := writeFileAtomic(path, []byte(contents), 0o644); err != nil {
		return "", err
	}
	return path, nil
}
