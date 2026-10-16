package files

import (
	"github.com/egoist/mygo"
	"testing"
)

type fixtureDocumentSurface struct{ visible, closed, recommended bool }

func (s *fixtureDocumentSurface) DocumentViewerRecommended() bool { return s.recommended }
func (s *fixtureDocumentSurface) Alive() bool                     { return !s.closed }
func (s *fixtureDocumentSurface) SetViewport(*mygo.Rectangle)     {}
func (s *fixtureDocumentSurface) SetVisible(v bool)               { s.visible = v }
func (s *fixtureDocumentSurface) Reload()                         {}
func (s *fixtureDocumentSurface) Close() error                    { s.closed = true; return nil }

func TestB25DocumentSurfaceTabLifetime(t *testing.T) {
	first, second := &fixtureDocumentSurface{visible: true}, &fixtureDocumentSurface{visible: true}
	a := &FileTab{Path: "a.md", document: documentSurfaceState{surface: first, enabled: true}}
	b := &FileTab{Path: "b.md", document: documentSurfaceState{surface: second, enabled: true}}
	tabs := &Tabs{binding: "host/session", boundRoot: "/work", Workspaces: map[string]*WorkspaceTabs{"/work": {Open: []*FileTab{a, b}, Active: "a.md"}}}
	selected := "a.md"
	syncDocumentSurfaces(Model{Tabs: tabs, WorkspaceRoot: "/work", SelectedFile: &selected})
	if !first.visible || second.visible || second.closed {
		t.Fatal("switch did not retain hidden tab")
	}
	tabs.Pause()
	if first.visible || first.closed {
		t.Fatal("pause did not retain hidden surface")
	}
	tabs.Bind("replacement/session", "/work")
	if !first.closed || !second.closed || a.document.enabled {
		t.Fatal("Host rebind retained old page authority")
	}
}

func TestB25AutomaticExtensionAndNativeOverride(t *testing.T) {
	surface := &fixtureDocumentSurface{}
	state := documentSurfaceState{surface: surface}
	if documentSurfaceEnabled(state) {
		t.Fatal("native fallback lost without extension")
	}
	surface.recommended = true
	if !documentSurfaceEnabled(state) {
		t.Fatal("matching extension did not win automatically")
	}
	state.choice = 1
	if documentSurfaceEnabled(state) {
		t.Fatal("explicit native choice ignored")
	}
	state.choice = 0
	surface.recommended = false
	if documentSurfaceEnabled(state) {
		t.Fatal("unregistered extension retained automatic surface")
	}
	state.choice = 2
	if !documentSurfaceEnabled(state) {
		t.Fatal("explicit original viewer unavailable")
	}
}
