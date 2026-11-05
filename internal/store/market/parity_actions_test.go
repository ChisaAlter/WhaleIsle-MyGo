package market

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	desktopmarket "whaleisle/internal/desktop/market"
)

type gitUpdateFixture struct {
	updateFixture
	commit string
}

func (f gitUpdateFixture) LatestCommit(context.Context, desktopmarket.CatalogItem) (string, error) {
	return f.commit, nil
}
func TestGitAndPrereleaseUpdatesReachCapturedBatchTargets(t *testing.T) {
	s := updateStore(updateFixture{version: "1.1.0-rc.1"})
	s.CheckUpdates()
	if !awaitUpdates(t, s).Rows[0].Available {
		t.Fatal("prerelease not available")
	}
	old, latest := strings.Repeat("a", 40), strings.Repeat("b", 40)
	item := desktopmarket.CatalogItem{ID: "owner/demo", Homepage: "https://github.com/owner/demo", InstallSpec: "github:owner/demo"}
	s.discovery.service = gitUpdateFixture{commit: latest}
	s.discovery.catalog.Items = []desktopmarket.CatalogItem{item}
	s.rows[0].Source = "github:owner/demo#" + old
	s.CheckUpdates()
	u := awaitUpdates(t, s)
	if len(u.Rows) != 1 || u.Rows[0].Kind != "github" || !u.Rows[0].Available {
		t.Fatal(u)
	}
	target, err := s.CaptureUpdateTarget(item.ID)
	if err != nil || target.Spec != "github:owner/demo#"+latest {
		t.Fatal(target, err)
	}
	if err = s.ValidateTarget(target); err != nil {
		t.Fatal(err)
	}
	s.SelectAvailableUpdates(true)
	if !s.UpdateSelected(item.ID) {
		t.Fatal("not selectable in batch")
	}
	s.SetUpdateProfileDir(t.TempDir())
	if s.ValidateTarget(target) == nil {
		t.Fatal("source proof replacement did not expire target")
	}
}
func TestOperationLogsPersistRedactedAndExportAfterRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	s := NewWithPersonalPath(path)
	if err := s.BeginInstall("attempt", "owner/demo", "install"); err != nil {
		t.Fatal(err)
	}
	log := "Built successfully\nhttps://user:fixture-password@example.invalid/p?token=fixture-token\nAuthorization: Bearer fixture-bearer\napi_key=fixture-key\nghp_fixturegithub"
	if err := s.FinishInstall("attempt", "failed", "operation-failed", log, "password=fixture-secret"); err != nil {
		t.Fatal(err)
	}
	s = NewWithPersonalPath(path)
	p := s.Personal()
	if len(p.History) != 1 || !strings.Contains(p.History[0].Log, "Built successfully") {
		t.Fatal("operation log not restored")
	}
	saved, _ := os.ReadFile(path)
	exported := string(s.ExportHistory())
	for _, secret := range []string{"fixture-password", "fixture-token", "fixture-bearer", "fixture-key", "fixture-secret", "ghp_fixturegithub"} {
		if strings.Contains(string(saved), secret) || strings.Contains(exported, secret) {
			t.Fatal("fixture secret retained")
		}
	}
	if !strings.Contains(exported, "Built successfully") || !strings.Contains(exported, "[redacted]") {
		t.Fatal("redacted log missing from export")
	}
	if len([]rune(RedactOperationLog(strings.Repeat("界", 17000), 16000))) != 16000 {
		t.Fatal("unicode log budget")
	}
}
