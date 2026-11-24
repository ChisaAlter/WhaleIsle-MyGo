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
	// Dynamic label resolved at runtime: still registers sidebar.panellist, so
	// label falls back to localized package title.
	if err := os.WriteFile(filepath.Join(scopedDir, "client", "index.js"), []byte(`export function apply(ctx){ctx.slots.inject('sidebar.panellist', () => ctx.slots.register({name:'sidebar.panellist',id:'notes',label:()=>t('nav')}, Icon))}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Code-split client entrypoint re-exporting a local chunk within pkgDir.
	splitDir := filepath.Join(profileDir, "node_modules", "split-panel")
	if err := os.MkdirAll(filepath.Join(splitDir, "client", "chunks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(splitDir, "package.json"), []byte(`{"name":"split-panel","version":"1.0.0","exports":{"./client":"./client/entry.js"},"dsh":{"client":{"platform":"web"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(splitDir, "client", "entry.js"), []byte(`export * from './chunks/panel.js';`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(splitDir, "client", "chunks", "panel.js"), []byte(`ctx.slots.register({name:"sidebar.panellist",id:"split",label:"Split Board",order:45}, View)`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Path traversal in exports["./client"] must never read outside pkgDir or
	// claim a sidebar slot from outside the package root.
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
		{Package: "@third/remote-hybrid-tool", Title: json.RawMessage(`{"zh-CN":"混合工具","en":"Hybrid Tool"}`), Enabled: true, Removable: true, Capability: CapabilityHybrid},
		{Package: "@deepseek-ai/builtin-web", Enabled: true, Official: true, Removable: false, Capability: CapabilityWebUIOnly},
		{Package: "trap-pkg", Enabled: true, Removable: true, Capability: CapabilityWebUIOnly},
	})
	if len(got) != 3 {
		t.Fatalf("expected 3 sidebar entries (split-panel, @acme/notes-panel, remote @third/remote-board), got %+v", got)
	}
	byID := map[string]SidebarPluginEntry{}
	for _, e := range got {
		byID[e.ID] = e
	}
	if byID["split-panel"].Label != "Split Board" || byID["split-panel"].Order != 45 {
		t.Fatalf("unexpected code-split sidebar plugin: %+v", byID["split-panel"])
	}
	if byID["@acme/notes-panel"].Label != "笔记面板" || byID["@acme/notes-panel"].Order != defaultSidebarPluginOrder {
		t.Fatalf("unexpected localized scoped plugin: %+v", byID["@acme/notes-panel"])
	}
	if _, leaked := byID["trap-pkg"]; leaked {
		t.Fatalf("path traversal trap-pkg must not enter sidebar: %+v", byID["trap-pkg"])
	}
	if _, leaked := byID["@third/remote-hybrid-tool"]; leaked {
		t.Fatalf("remote hybrid candidate without sidebar slot evidence must not enter sidebar: %+v", byID["@third/remote-hybrid-tool"])
	}
	if byID["@third/remote-board"].Label != "看板" {
		t.Fatalf("unexpected remote candidate fallback: %+v", byID["@third/remote-board"])
	}
}

func TestDiscoverSidebarPluginsExcludesNonSidebarSlotPluginsByProtocol(t *testing.T) {
	profileDir := t.TempDir()

	// Real third_party/dsh-deepseek-tools registers only settings.section, never
	// a left-sidebar extension space slot. It must be excluded purely by slot
	// protocol, even when Host inventory reports it enabled as hybrid/web-ui.
	dsSrc := filepath.Join("..", "..", "third_party", "dsh-deepseek-tools")
	dsDst := filepath.Join(profileDir, "node_modules", "dsh-deepseek-tools")
	copyTestFile(t, filepath.Join(dsSrc, "package.json"), filepath.Join(dsDst, "package.json"))
	copyTestFile(t, filepath.Join(dsSrc, "client", "client.js"), filepath.Join(dsDst, "client", "client.js"))

	// Generic third-party plugins contributing only settings.section,
	// sidebar.panellist.actions (More-menu action), sidebar.footer.action, or
	// conversation slots must NOT appear as left-sidebar extension spaces.
	nonSidebar := map[string]string{
		"custom-settings-only": `ctx.slots.inject('settings.section', () => ctx.slots.register({name:'settings.section',id:'foo',label:'Foo'}, Comp))`,
		"custom-more-action":   `ctx.slots.inject('sidebar.panellist.actions', () => ctx.slots.register({name:'sidebar.panellist.actions',id:'act',label:'Act'}, Comp))`,
		"custom-footer-badge":  `ctx.slots.inject('sidebar.footer.action', () => ctx.slots.register({name:'sidebar.footer.action',id:'foot'}, Comp))`,
		"custom-empty-client":  `export function apply(){}`,
	}
	for name, js := range nonSidebar {
		dir := filepath.Join(profileDir, "node_modules", name)
		if err := os.MkdirAll(filepath.Join(dir, "client"), 0o755); err != nil {
			t.Fatal(err)
		}
		pkg := `{"name":"` + name + `","version":"1.0.0","exports":{"./client":"./client/client.js"},"dsh":{"client":{"platform":"web"}}}`
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "client", "client.js"), []byte(js), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got := DiscoverSidebarPlugins(profileDir, "zh-CN", []SidebarPluginCandidate{
		{Package: "dsh-deepseek-tools", Enabled: true, Removable: true, Capability: CapabilityHybrid},
		{Package: "custom-settings-only", Enabled: true, Removable: true, Capability: CapabilityWebUIOnly},
		{Package: "custom-more-action", Enabled: true, Removable: true, Capability: CapabilityWebUIOnly},
		{Package: "custom-footer-badge", Enabled: true, Removable: true, Capability: CapabilityWebUIOnly},
		{Package: "custom-empty-client", Enabled: true, Removable: true, Capability: CapabilityWebUIOnly},
	})
	if len(got) != 0 {
		t.Fatalf("expected non-sidebar plugins to stay out of left sidebar, got %+v", got)
	}
}
