package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"whaleisle/internal/config"
	"whaleisle/internal/product"
)

// TestCoverUpgradePathInvariants encodes the side-by-side MyGo install + shared
// userData contract: independent AppID and install directory (never overwriting
// Programs\Deepseek-Harness-Desktop), while retaining shared DataDirName.
func TestCoverUpgradePathInvariants(t *testing.T) {
	if got := config.DataDirName; got != "Deepseek-Harness-Desktop" {
		t.Fatalf("DataDirName = %q, want Deepseek-Harness-Desktop (shared with Electron)", got)
	}

	repoRoot := findRepoRoot(t)
	nsi := filepath.Join(repoRoot, "scripts", "nsis", "whaleisle.nsi")
	raw, err := os.ReadFile(nsi)
	if err != nil {
		t.Fatalf("read nsi: %v", err)
	}
	body := string(raw)
	for _, want := range []string{
		product.AppID,
		product.Name,
		`InstallDir "$LOCALAPPDATA\Programs\` + product.InstallDir + `"`,
		"WhaleIsle-MyGo-Setup-",
		"LOCALAPPDATA",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("whaleisle.nsi missing %q", want)
		}
	}
	if strings.Contains(body, `InstallDir "$LOCALAPPDATA\Programs\Deepseek-Harness-Desktop"`) {
		t.Fatal("whaleisle.nsi must not default to Electron's Programs\\Deepseek-Harness-Desktop install directory")
	}

	mygo := filepath.Join(repoRoot, "mygo.json")
	mj, err := os.ReadFile(mygo)
	if err != nil {
		t.Fatalf("mygo.json: %v", err)
	}
	if !strings.Contains(string(mj), `"identifier": "`+product.AppID+`"`) &&
		!strings.Contains(string(mj), `"identifier":"`+product.AppID+`"`) {
		t.Fatalf("mygo.json identifier must be %s; got %s", product.AppID, truncate(string(mj), 200))
	}
	if !strings.Contains(string(mj), `"name": "`+product.EnglishName+`"`) {
		t.Fatalf("mygo.json name must be %s; got %s", product.EnglishName, truncate(string(mj), 200))
	}

	// Simulated Electron userData: ImportReport must see artifacts without reading secrets.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "credentials.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "dsh-home"), 0o755); err != nil {
		t.Fatal(err)
	}
	rep, err := ImportReport(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.DirExists || !rep.ConfigJSON || !rep.CredentialsJSON || !rep.DshHome {
		t.Fatalf("ImportReport incomplete: %+v", rep)
	}
	summary := FormatReport(rep)
	if !strings.Contains(summary, "config.json") {
		t.Fatalf("FormatReport = %q", summary)
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "scripts", "nsis", "whaleisle.nsi")); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("repo root with scripts/nsis/whaleisle.nsi not found")
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
