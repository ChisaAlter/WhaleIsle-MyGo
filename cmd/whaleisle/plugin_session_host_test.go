package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/ui/frame"

	"whaleisle/internal/harness"
	"whaleisle/internal/store/session"
)

func TestPluginSessionIdentityIncludesHostAndSelectionGeneration(t *testing.T) {
	v := session.ViewState{SelectedID: "same", ComposerHostGeneration: 1, ComposerSessionGeneration: 2}
	first := pluginSessionSubject(v)
	v.ComposerSessionGeneration++
	if pluginSessionSubject(v) == first {
		t.Fatal("same-id navigation retained old contribution owner")
	}
	v.ComposerSessionGeneration = 2
	v.ComposerHostGeneration++
	if pluginSessionSubject(v) == first {
		t.Fatal("Host replacement retained old contribution owner")
	}
}

func TestPluginSessionSwitchClearsStaleState(t *testing.T) {
	sessions := session.New()
	sessions.Select("session-a")
	status := harness.Status{State: harness.Ready}
	p := &nativePluginSessionState{
		visible:    true,
		open:       true,
		sessions:   sessions,
		endpoint:   status.Endpoint,
		generation: status.Generation,
		subject:    pluginSessionSubject(sessions.Snapshot()),
		error:      "old warning",
	}
	s := &mainState{sessions: sessions, bootStatus: status}
	s.nativePlugins.session = p

	sessions.Select("session-b")
	s.syncPluginSession(true)
	if !p.visible || p.open || p.error != "" || p.owner != nil {
		t.Fatalf("session switch retained old UI: %+v", p)
	}
	if p.subject.SessionID != "" {
		t.Fatalf("session switch retained selected owner: %+v", p.subject)
	}
}

func TestPluginSessionNoOwnerDoesNotReadFullSnapshotOrHideExplicitMenu(t *testing.T) {
	sessions := session.New()
	sessions.Select("session-a")
	s := &mainState{sessions: sessions, bootStatus: harness.Status{State: harness.Ready}}
	before, _ := sessions.SnapshotStats()
	for i := 0; i < 12; i++ {
		s.syncPluginSession(true)
	}
	after, _ := sessions.SnapshotStats()
	if after != before {
		t.Fatal("inactive discovery built full session snapshots")
	}
	model := s.withPluginSessionFrame(frame.Model{Store: sessions})
	if model.OnOpenSessionPlugins == nil || model.SessionPluginsActive {
		t.Fatal("explicit activation unavailable before owner creation")
	}
}

func TestPluginSessionControlRowsRejectForeignEpochAndSlot(t *testing.T) {
	subject := pluginui.Subject{Kind: "session", SessionID: "a", Epoch: "1:2"}
	state := pluginui.Snapshot{Connected: true, Observation: true, Subject: &subject, SessionControls: []pluginui.SessionControl{{Slot: "conversation.session.header.actions", Handle: "1", Label: "Action"}, {Slot: "conversation.session.header.utilities", Handle: "2", Label: "Utility"}}}
	if rows := pluginSessionControlRows(state, subject, "conversation.session.header.actions"); len(rows) != 1 || rows[0].Label != "Action" {
		t.Fatal("wrong header zone")
	}
	other := subject
	other.Epoch = "1:3"
	if len(pluginSessionControlRows(state, other, "conversation.session.header.actions")) != 0 {
		t.Fatal("stale selection leaked controls")
	}
}

func TestPluginSessionClosedSeatRendersNoFloatingEntry(t *testing.T) {
	s := &mainState{}
	tt := ui.NewTester(func(c *ui.Context) { s.renderPluginSessionSlot(c) }, 800, 600)
	if tt.HasText("会话插件") || tt.HasText("Session plugins") {
		t.Fatal("closed seat rendered a floating entry")
	}
}

func TestPluginSessionCollapseRetainsAndStopDestroysOwner(t *testing.T) {
	owner := &pluginui.Owner{}
	p := &nativePluginSessionState{owner: owner, open: true}
	s := &mainState{}
	s.collapsePluginSession(p)
	if p.open || p.owner != owner {
		t.Fatal("collapse destroyed retained owner")
	}
	s.closePluginSession(p)
	if p.owner != nil || p.target != nil || p.open {
		t.Fatal("explicit stop retained owner")
	}
	owner.Select(pluginui.Subject{Kind: "session", SessionID: "retired", Epoch: "1:1"})
	if owner.Snapshot().Subject != nil || owner.Snapshot().Connected {
		t.Fatal("stopped owner revived")
	}
	s.closePluginSession(p)
}

func TestPluginSessionCapturedOwnerRetiresOnHostAndSameIDNavigation(t *testing.T) {
	for _, change := range []string{"same-session-navigation", "host-generation", "route"} {
		t.Run(change, func(t *testing.T) {
			sessions := session.New()
			client := &dsh.Client{}
			sessions.Attach(client, nil)
			sessions.Select("a")
			target, err := sessions.CaptureComposerTarget("a")
			if err != nil {
				t.Fatal(err)
			}
			owner := &pluginui.Owner{}
			status := harness.Status{State: harness.Ready, Generation: 1}
			p := &nativePluginSessionState{visible: true, open: true, owner: owner, target: target, sessions: sessions, endpoint: status.Endpoint, generation: status.Generation, subject: pluginSessionSubject(sessions.Snapshot())}
			s := &mainState{sessions: sessions, bootStatus: status}
			s.nativePlugins.session = p
			visible := true
			switch change {
			case "same-session-navigation":
				sessions.Select("b")
				sessions.Select("a")
			case "host-generation":
				s.bootStatus.Generation++
			case "route":
				visible = false
			}
			s.syncPluginSession(visible)
			if p.owner != nil || p.target != nil || p.open {
				t.Fatal("stale owner survived identity change")
			}
		})
	}
}

func TestPluginSessionGlobalMainContentClosesAndReturnDoesNotRemount(t *testing.T) {
	sessions := session.New()
	sessions.Select("a")
	status := harness.Status{State: harness.Ready, Generation: 1}
	owner := &pluginui.Owner{}
	p := &nativePluginSessionState{visible: true, open: true, owner: owner, sessions: sessions, generation: 1}
	s := &mainState{sessions: sessions, bootStatus: status}
	s.nativePlugins.session = p
	global := s.withPluginSessionFrame(frame.Model{Store: sessions, WorkspacePage: true})
	if p.owner != nil || p.open || p.visible || global.OnOpenSessionPlugins != nil {
		t.Fatal("global content retained parent plugin owner")
	}
	s.syncPluginSession(true)
	parent := s.withPluginSessionFrame(frame.Model{Store: sessions})
	if p.owner != nil || p.open || parent.OnOpenSessionPlugins == nil || parent.SessionPluginsActive {
		t.Fatal("return created owner without an explicit operation")
	}
}

func TestPluginSessionHiddenRouteClearsOpenState(t *testing.T) {
	s := &mainState{}
	s.bootStatus = harness.Status{State: harness.Ready}
	s.nativePlugins.session = &nativePluginSessionState{visible: true, open: true, error: "old"}
	s.syncPluginSession(false)
	p := s.nativePlugins.session
	if p.open || p.visible || p.error != "" || p.owner != nil {
		t.Fatal("hidden route retained interactive Session UI")
	}
}
