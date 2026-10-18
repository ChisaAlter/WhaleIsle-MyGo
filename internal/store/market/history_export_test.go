package market

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestExportHistoryIncludesPersistedSafeReceipts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "market.json")
	s := NewWithPersonalPath(path)
	if err := s.RecordInstall("owner/plugin", "applied", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordInstall("https://secret:password@example.com", "failed", "token=secret"); err != nil {
		t.Fatal(err)
	}
	s = NewWithPersonalPath(path)
	s.SetManagementFilter("updates")
	data := string(s.ExportHistory())
	for _, want := range []string{"owner/plugin", "applied", "manual-install", "failed", s.Personal().History[0].At.Format("2006-01-02T15:04:05Z07:00")} {
		if !strings.Contains(data, want) {
			t.Fatalf("export missing %q: %s", want, data)
		}
	}
	for _, secret := range []string{"secret", "password", "token=", "https://"} {
		if strings.Contains(data, secret) {
			t.Fatalf("export leaked %q", secret)
		}
	}
	if len(NewWithPersonalPath(filepath.Join(t.TempDir(), "empty.json")).ExportHistory()) != 0 {
		t.Fatal("empty history export must be empty")
	}
}
