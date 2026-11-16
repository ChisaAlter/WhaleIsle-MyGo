package pluginmanager

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func copyTestFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverSidebarPluginsRealQAPanelFixture(t *testing.T) {
	profileDir := t.TempDir()
	qaSrc := filepath.Join("..", "..", "cmd", "whaleisle", "testdata", "qa-panel")
	qaDst := filepath.Join(profileDir, "node_modules", "qa-panel")
	copyTestFile(t, filepath.Join(qaSrc, "package.json"), filepath.Join(qaDst, "package.json"))
	copyTestFile(t, filepath.Join(qaSrc, "client", "client.js"), filepath.Join(qaDst, "client", "client.js"))

	// Built-in desktop overlays and non-dsh npm libraries must be excluded.
	copyTestFile(t, filepath.Join(qaSrc, "package.json"), filepath.Join(profileDir, "node_modules", "dsh-whale", "package.json"))
	pureHostDir := filepath.Join(profileDir, "node_modules", "pure-host-tool")
	if err := os.MkdirAll(pureHostDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pureHostDir, "package.json"), []byte(`{"name":"pure-host-tool","version":"1.0.0","main":"index.js","dsh":{"bundle":{}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	plainLibDir := filepath.Join(profileDir, "node_modules", "react")
	if err := os.MkdirAll(plainLibDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(plainLibDir, "package.json"), []byte(`{"name":"react","version":"19.0.0","exports":{"./client":"./client.js"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got := DiscoverSidebarPlugins(profileDir, "zh-CN", nil)
	if len(got) != 1 || got[0].ID != "qa-panel" || got[0].Label != "QA Panel" || got[0].Order != 60 {
		t.Fatalf("unexpected discovered sidebar plugins: %+v", got)
	}

	// Explicit disable from Host bundle/inventory suppresses the entry.
	disabled := DiscoverSidebarPlugins(profileDir, "zh-CN", []SidebarPluginCandidate{
		{Package: "qa-panel", ExplicitDisabled: true},
	})
	if len(disabled) != 0 {
		t.Fatalf("expected disabled plugin to be hidden, got %+v", disabled)
	}
}

func TestDiscoverSidebarPluginsScopedAndRemoteCandidateFallback(t *testing.T) {
	profileDir := t.TempDir()
	scopedDir := filepath.Join(profileDir, "node_modules", "@acme", "notes-panel")
	if err := os.MkdirAll(filepath.Join(scopedDir, "client"), 0o755); err != nil {
		t.Fatal(err)
	}
	pkgJSON := `{"name":"@acme/notes-panel","version":"1.0.0","exports":{"./client":"./client/index.js"},"dsh":{"client":{"platform":"web"},"title":{"zh-CN":"笔记面板","en":"Notes Panel"}}}`
	if err := os.WriteFile(filepath.Join(scopedDir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scopedDir, "client", "index.js"), []byte(`export function apply(){}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Path traversal in exports["./client"] must never read outside pkgDir.
	outsideFile := filepath.Join(profileDir, "escape.js")
	if err := os.WriteFile(outsideFile, []byte(`slots.inject('sidebar.panellist', () => slots.register({label:'ESCAPED',order:1}))`), 0o644); err != nil {
		t.Fatal(err)
	}
	trapDir := filepath.Join(profileDir, "node_modules", "trap-pkg")
	if err := os.MkdirAll(trapDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(trapDir, "package.json"), []byte(`{"name":"trap-pkg","exports":{"./client":"../../escape.js"},"dsh":{"client":{"platform":"web"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got := DiscoverSidebarPlugins(profileDir, "zh-CN", []SidebarPluginCandidate{
		{Package: "@third/remote-board", Title: json.RawMessage(`{"zh-CN":"看板","en":"Board"}`), Enabled: true, Removable: true, Capability: CapabilityWebUIOnly},
		{Package: "@deepseek-ai/builtin-web", Enabled: true, Official: true, Removable: false, Capability: CapabilityWebUIOnly},
	})
	if len(got) != 3 {
		t.Fatalf("expected 3 sidebar entries (local @acme, local trap fallback, remote @third), got %+v", got)
	}
	byID := map[string]SidebarPluginEntry{}
	for _, e := range got {
		byID[e.ID] = e
	}
	if byID["@acme/notes-panel"].Label != "笔记面板" || byID["@acme/notes-panel"].Order != defaultSidebarPluginOrder {
		t.Fatalf("unexpected localized scoped plugin: %+v", byID["@acme/notes-panel"])
	}
	if byID["trap-pkg"].Label != "trap-pkg" {
		t.Fatalf("path traversal escaped package dir: %+v", byID["trap-pkg"])
	}
	if byID["@third/remote-board"].Label != "看板" {
		t.Fatalf("unexpected remote candidate fallback: %+v", byID["@third/remote-board"])
	}
}
