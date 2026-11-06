package market

import (
	"context"
	"encoding/json"
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

func TestOperationHistoryRedactsWholeAuthorizationAndQuotedCredentials(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	s := NewWithPersonalPath(path)
	if err := s.BeginInstall("attempt", "owner/demo", "install"); err != nil {
		t.Fatal(err)
	}
	log := "Authorization: Basic Zml4dHVyZTpjcmVkZW50aWFs\n" +
		"Proxy-Authorization: Digest username=\"fixture-user\", response=\"fixture-response\"\n" +
		`{"Authorization":"Custom fixture-auth", "result":"diagnostic retained"}` + "\n" +
		`api_key="fixture secret with spaces and an escaped \"value\""` + "\n" +
		"Build failed after authentication\n"
	if err := s.FinishInstall("attempt", "failed", "operation-failed", log); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	exported := string(NewWithPersonalPath(path).ExportHistory())
	for _, secret := range []string{"Zml4dHVyZTpjcmVkZW50aWFs", "fixture-user", "fixture-response", "fixture-auth", "secret with spaces", "value"} {
		if strings.Contains(string(saved), secret) || strings.Contains(exported, secret) {
			t.Fatalf("credential remained in operation history: %s", secret)
		}
	}
	if !strings.Contains(exported, "diagnostic retained") || !strings.Contains(exported, "Build failed after authentication") {
		t.Fatal("ordinary diagnostic context was removed")
	}
}

func TestLoadingHistoryRemovesPreviouslyPersistedCredentials(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	data := personalData{Favorites: map[string]bool{"owner/demo": true}, History: []InstallHistory{{Attempt: "completed", Result: "failed", Log: "Authorization: Basic previous-credential\nBuild failed", Diagnostic: `api_key="previous secret"`}}}
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	s := NewWithPersonalPath(path)
	if s.Personal().Err != "" || !s.Personal().Favorites["owner/demo"] {
		t.Fatalf("history migration lost state: %+v", s.Personal())
	}
	raw, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "previous-credential") || strings.Contains(string(raw), "previous secret") || !strings.Contains(string(raw), "Build failed") {
		t.Fatal("existing history was not safely redacted on disk")
	}
}
