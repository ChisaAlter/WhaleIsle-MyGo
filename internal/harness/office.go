package harness

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	officeOverlayFile = "desktop-office.patch.yml"
	officeOverlayDir  = "office"
)

// EnsureOfficeOverlay writes the desktop Office --patch overlay when a bundled
// primary-runtime payload is present. Missing payload in development is OK
// (returns "", nil); an explicit empty DSH_PRIMARY_RUNTIME disables Office.
func EnsureOfficeOverlay(o PrepareOptions) (string, error) {
	if o.ProfileDir == "" {
		return "", nil
	}
	overlayPath := filepath.Join(o.ProfileDir, "desktop-plugins", officeOverlayDir, officeOverlayFile)
	if disabled, candidates := officeRuntimeCandidates(); disabled {
		_ = os.Remove(overlayPath)
		return "", nil
	} else {
		source := ""
		for _, c := range candidates {
			if _, err := os.Stat(filepath.Join(c, "runtime.json")); err == nil {
				source = c
				break
			}
		}
		if source == "" {
			_ = os.Remove(overlayPath)
			return "", nil
		}
		dshHome := o.DshHome
		if dshHome == "" {
			dshHome = filepath.Dir(filepath.Dir(o.ProfileDir)) // profiles/web → dsh-home
		}
		root := filepath.Join(dshHome, "dsh-runtimes", "dsh-primary-runtime")
		assetRoot := filepath.Join(filepath.Dir(source), "office-skills")
		nodeName := "node"
		if runtime.GOOS == "windows" {
			nodeName = "node.exe"
		}
		node := filepath.Join(source, "dependencies", "node", "bin", nodeName)
		cli := ""
		if o.HarnessRoot != "" {
			cli = filepath.Join(o.HarnessRoot, "node_modules", "@deepseek-ai", "libreoffice-kit", "lib", "cli.js")
			if _, err := os.Stat(cli); err != nil {
				cli = filepath.Join(o.HarnessRoot, "apps", "cli", "node_modules", "@deepseek-ai", "libreoffice-kit", "lib", "cli.js")
			}
		}
		if _, err := os.Stat(node); err != nil {
			return "", fmt.Errorf("office runtime missing node: %s", node)
		}
		if cli == "" {
			return "", fmt.Errorf("office runtime: harness root required to locate libreoffice-kit cli")
		}
		contents := strings.Join([]string{
			"# Desktop-managed overlay passed to every start (full and skip) via",
			"# --patch: the bundled Office runtime insert rows. Regenerated on every start.",
			"- insert:",
			"    - id: workspace-dependencies",
			`      name: "@deepseek-ai/dsh-tool-workspace-dependencies"`,
			"      config:",
			"        source: " + quoteYAMLString(filepath.ToSlash(source)),
			"        root: " + quoteYAMLString(filepath.ToSlash(root)),
			"    - id: skill-office",
			`      name: "@deepseek-ai/dsh-skill-office"`,
			"      config:",
			"        assetRoot: " + quoteYAMLString(filepath.ToSlash(assetRoot)),
			"        node: " + quoteYAMLString(filepath.ToSlash(node)),
			"        cli: " + quoteYAMLString(filepath.ToSlash(cli)),
			"",
		}, "\n")
		return writeRawOverlay(o.ProfileDir, officeOverlayDir, officeOverlayFile, contents)
	}
}

func officeRuntimeCandidates() (disabled bool, candidates []string) {
	if v, ok := os.LookupEnv("DSH_PRIMARY_RUNTIME"); ok {
		if strings.TrimSpace(v) == "" {
			return true, nil
		}
		return false, []string{filepath.Clean(v)}
	}
	wd, _ := os.Getwd()
	candidates = []string{
		filepath.Join(wd, "build", "office-runtime", "primary-runtime"),
		filepath.Join(wd, "resources", "runtime", "primary-runtime"),
	}
	return false, candidates
}
