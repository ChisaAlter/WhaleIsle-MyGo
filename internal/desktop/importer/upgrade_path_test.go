package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"whaleisle/internal/config"
)

// TestCoverUpgradePathInvariants encodes M11 “覆盖 Electron 安装”约束：
// 同 appId / 数据目录名，且 NSIS 默认装到 Programs\Deepseek-Harness-Desktop。
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
		"ai.deepseek.harness.gui",
		"Deepseek-Harness-Desktop",
		"LOCALAPPDATA",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("whaleisle.nsi missing %q", want)
		}
	}

	mygo := filepath.Join(repoRoot, "mygo.json")
	mj, err := os.ReadFile(mygo)
	if err != nil {
		t.Fatalf("mygo.json: %v", err)
	}
	if !strings.Contains(string(mj), `"identifier": "ai.deepseek.harness.gui"`) &&
		!strings.Contains(string(mj), `"identifier":"ai.deepseek.harness.gui"`) {
		t.Fatalf("mygo.json identifier must be ai.deepseek.harness.gui; got %s", truncate(string(mj), 200))
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
