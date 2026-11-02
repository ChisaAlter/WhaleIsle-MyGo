package main

import (
	"testing"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/frame"
)

func TestPluginSessionOuterNavigationBindingsNeverMountOrRetarget(t *testing.T) {
	sessions := session.New()
	sessions.Select("A")
	s := &mainState{sessions: sessions}
	p := &nativePluginSessionState{visible: true, sessions: sessions, open: true}
	s.nativePlugins.session = p
	m := frame.Model{Store: sessions}
	s.bindPluginSessionNavigation(&m)
	if m.OnOpenSessionPlugins == nil || m.OnCloseSessionPlugins == nil || m.SessionPluginsActive || p.owner != nil {
		t.Fatal("navigation missed explicit operations or mounted an owner")
	}
	sessions.Select("B")
	m.OnCloseSessionPlugins()
	if !p.open {
		t.Fatal("stale navigation callback closed another selected session")
	}
	sessions.Select("A")
	replacement := &nativePluginSessionState{visible: true, sessions: sessions, open: true}
	s.nativePlugins.session = replacement
	m.OnCloseSessionPlugins()
	if !p.open || !replacement.open {
		t.Fatal("stale callback changed a replacement owner")
	}
	s.nativePlugins.session = p
	for _, blocked := range []frame.Model{{}, {Store: session.New()}, {Store: sessions, WorkspacePage: true}} {
		s.bindPluginSessionNavigation(&blocked)
		if blocked.OnOpenSessionPlugins != nil || blocked.OnCloseSessionPlugins != nil {
			t.Fatal("foreign/global navigation received Session operations")
		}
	}
	p.visible = false
	hidden := frame.Model{Store: sessions}
	s.bindPluginSessionNavigation(&hidden)
	if hidden.OnOpenSessionPlugins != nil {
		t.Fatal("hidden Session retained navigation actions")
	}
}
