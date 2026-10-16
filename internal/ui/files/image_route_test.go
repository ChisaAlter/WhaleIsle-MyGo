package files

import (
	"path/filepath"
	"testing"
	"whaleisle/internal/desktop/workspacefs"
)

func TestImageDefaultIsolatedRouteOnce(t *testing.T) {
	for _, name := range []string{"animation.gif", "vector.svg", "icon.ico", "photo.avif"} {
		calls := 0
		root := t.TempDir()
		m := Model{WorkspaceRoot: root, OnOpenDoc: func(path string, kind workspacefs.DocKind, web bool) {
			calls++
			if path != filepath.Join(root, name) || kind != workspacefs.DocNone || !web {
				t.Fatal(path, kind, web)
			}
		}}
		tab := &FileTab{Path: name}
		if !routeDefaultImage(m, tab, name, "windows") {
			t.Fatal("format not routed", name)
		}
		if !routeDefaultImage(m, tab, name, "windows") || calls != 1 {
			t.Fatal("repeated frame reopened image", calls)
		}
		if tab.image == nil || !tab.image.isolated {
			t.Fatal("route state absent")
		}
	}
}

func TestImageDefaultPreservesDraftAndPlatformFallback(t *testing.T) {
	preview, draft := "original", "edited"
	calls := 0
	m := Model{WorkspaceRoot: t.TempDir(), Preview: &preview, EditDraft: &draft, OnOpenDoc: func(string, workspacefs.DocKind, bool) { calls++ }}
	if routeDefaultImage(m, &FileTab{}, "a.svg", "windows") {
		t.Fatal("draft hidden")
	}
	draft = preview
	for _, platform := range []string{"linux", "darwin"} {
		if routeDefaultImage(m, &FileTab{}, "a.gif", platform) {
			t.Fatal("unsupported backend advertised")
		}
	}
	if routeDefaultImage(m, &FileTab{}, "a.png", "windows") {
		t.Fatal("native format rerouted")
	}
	if routeDefaultImage(m, nil, "a.gif", "windows") {
		t.Fatal("stateless rerender loop allowed")
	}
	if calls != 0 {
		t.Fatal("unexpected opener")
	}
}
