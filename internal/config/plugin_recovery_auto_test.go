package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAutomaticRecoveryPersistsRedactedVersionedEvidence(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RecordAutomaticPluginRecovery("v1", "plugin tree failed token=synthetic-secret", []string{"cannot find package '@fixture/plugin' imported from fixture", "apiKey=synthetic-key"}); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reopened.Get().SkipUserPlugins {
		t.Fatal("skip was not persisted")
	}
	ev := reopened.PluginRecoveryEvidence()
	if ev.AppVersion != "v1" || len(ev.LogTail) != 2 {
		t.Fatal("evidence missing")
	}
	b, _ := os.ReadFile(filepath.Join(dir, FileName))
	if strings.Contains(string(b), "synthetic-secret") || strings.Contains(string(b), "synthetic-key") {
		t.Fatal("secret persisted")
	}
	if err = reopened.ReconcilePluginRecovery("v1"); err != nil {
		t.Fatal(err)
	}
	if !reopened.Get().SkipUserPlugins {
		t.Fatal("same-build recovery cleared")
	}
	if err = reopened.ReconcilePluginRecovery("v2"); err != nil {
		t.Fatal(err)
	}
	if reopened.Get().SkipUserPlugins {
		t.Fatal("stale automatic recovery remained active")
	}
}
func TestManualRecoverySurvivesBuildAndWriteFailure(t *testing.T) {
	dir := t.TempDir()
	s, _ := Open(dir)
	if err := s.UpdatePluginRecovery(func(m map[string]any) { m["skipUserPlugins"] = true; m["reason"] = "launcher" }); err != nil {
		t.Fatal(err)
	}
	if err := s.ReconcilePluginRecovery("v2"); err != nil {
		t.Fatal(err)
	}
	if !s.Get().SkipUserPlugins {
		t.Fatal("manual setting was cleared")
	}
	s.rename = func(string, string) error { return os.ErrPermission }
	if err := s.RecordAutomaticPluginRecovery("v2", "test", nil); err == nil {
		t.Fatal("failed write accepted")
	}
	if s.PluginRecoveryEvidence().AppVersion != "" {
		t.Fatal("failed write mutated memory")
	}
}
