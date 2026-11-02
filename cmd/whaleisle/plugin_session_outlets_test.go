package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginui"
	"whaleisle/internal/ui/i18n"
)

func TestPluginSessionOutletAttributionDistinguishesTurnsCallsAndUnknown(t *testing.T) {
	previous := i18n.Locale()
	defer i18n.SetLocale(previous)
	rows := []pluginui.SessionOutlet{
		{Handle: "turn-zero", Label: "Deliverable", Turn: 0, HasTurn: true, Available: true},
		{Handle: "turn-two", Label: "Deliverable", Turn: 2, Available: true},
		{Handle: "tool-a", Label: "Tool output", CallID: "call-a", Available: true},
		{Handle: "tool-b", Label: "Tool output", CallID: "call-b", Available: true},
		{Handle: "both", Label: "Tool output", CallID: "call-c", Turn: 3, HasTurn: true, Available: true},
		{Handle: "unknown-b", Label: "Other", Available: true},
		{Handle: "unknown-a", Label: "Other", Available: true},
		{Handle: "fallback", EntryID: "Entry title", Available: false},
		{Handle: "untitled", Available: true},
		{Handle: "invalid-turn", Label: "Invalid", Turn: -1, HasTurn: true, Available: true},
		{Handle: "same-turn-b", Label: "Same turn", Turn: 2, HasTurn: true, Available: true},
		{Handle: "same-turn-a", Label: "Same turn", Turn: 2, HasTurn: true, Available: true},
	}
	for _, locale := range []string{i18n.LocaleZH, i18n.LocaleEN} {
		t.Run(locale, func(t *testing.T) {
			i18n.SetLocale(locale)
			items := sessionPluginOutletItems(rows)
			if len(items) != len(rows) {
				t.Fatal("instances collapsed")
			}
			seen := map[string]bool{}
			for _, item := range items {
				if seen[item.Accessible] {
					t.Fatalf("ambiguous occurrence: %s", item.Accessible)
				}
				seen[item.Accessible] = true
			}
			if locale == i18n.LocaleEN {
				for index, want := range map[int]string{0: "Turn 0", 1: "Turn 2", 2: "Tool call call-a", 3: "Tool call call-b", 4: "Turn 3 · Tool call call-c", 5: "Location unknown · Instance 2", 6: "Location unknown · Instance 1", 7: "Location unknown · Unavailable"} {
					if items[index].Attribution != want {
						t.Fatalf("item %d: %q want %q", index, items[index].Attribution, want)
					}
				}
			} else if items[0].Attribution != "第 0 轮" || items[4].Attribution != "第 3 轮 · 工具调用 call-c" || items[5].Attribution != "归属未知 · 实例 2" {
				t.Fatalf("unlocalized attribution: %+v", items)
			}
			if items[7].Title != "Entry title" || items[8].Title != i18n.T(i18n.KeySessionPluginsOutletUntitled) {
				t.Fatal("missing label lost readable fallback")
			}
			if items[9].Attribution != i18n.T(i18n.KeySessionPluginsOutletUnknown) || strings.Contains(items[10].Attribution, "-1") {
				t.Fatal("invalid turn gained a displayed identity")
			}
			if items[10].Accessible == items[11].Accessible {
				t.Fatal("same-label same-turn instances remained indistinguishable")
			}
			if strings.Contains(items[5].Attribution, "0") {
				t.Fatal("absent turn claimed turn zero")
			}
			reversed := append([]pluginui.SessionOutlet(nil), rows...)
			for a, b := 0, len(reversed)-1; a < b; a, b = a+1, b-1 {
				reversed[a], reversed[b] = reversed[b], reversed[a]
			}
			byHandle := map[string]string{}
			for _, item := range items {
				byHandle[item.Outlet.Handle] = item.Accessible
			}
			for _, item := range sessionPluginOutletItems(reversed) {
				if byHandle[item.Outlet.Handle] != item.Accessible {
					t.Fatalf("reorder renamed %s", item.Outlet.Handle)
				}
			}
		})
	}
	if len(sessionPluginOutletItems(nil)) != 0 {
		t.Fatal("empty data invented content")
	}
}

func sessionOutletTestSnapshot() pluginui.Snapshot {
	subject := pluginui.Subject{Kind: "session", SessionID: "s", Epoch: "3:4"}
	return pluginui.Snapshot{Connected: true, Page: 7, Revision: 8, Subject: &subject, Outlets: []pluginui.SessionOutlet{
		{Handle: "first", Slot: "conversation.chat.turnTail", Label: "Deliverable", Turn: 0, HasTurn: true, Available: true},
		{Handle: "second", Slot: "conversation.chat.turnTail", Label: "Deliverable", Turn: 2, HasTurn: true, Available: true},
		{Handle: "tool-a", Slot: "tool.call.toolview", Label: "Tool output", CallID: "call-a", Available: true},
		{Handle: "tool-b", Slot: "tool.call.toolview", Label: "Tool output", CallID: "call-b", Available: true},
	}}
}

func TestPluginSessionOutletUIInvokesAddressedHandleAndCapturedIdentity(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(previous)
	snapshot := sessionOutletTestSnapshot()
	var received []string
	tt := ui.NewTester(func(c *ui.Context) {
		renderSessionPluginOutlets(c, snapshot, func(captured pluginui.Snapshot, handle string) {
			if captured.Page != snapshot.Page || captured.Revision != snapshot.Revision || captured.Subject != snapshot.Subject {
				t.Fatal("list changed action identity")
			}
			received = append(received, handle)
		})
	}, 390, 500)
	for _, item := range sessionPluginOutletItems(snapshot.Outlets) {
		if !tt.HasText(item.Attribution) {
			t.Fatalf("visible attribution missing: %s", item.Attribution)
		}
		r, ok := tt.Find(item.Accessible)
		if !ok || r.X < 0 || r.X+r.W > 390 || r.H < 30 {
			t.Fatalf("instance row does not fit: %+v", r)
		}
		if err := tt.Click(item.Accessible); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(received, []string{"first", "second", "tool-a", "tool-b"}) {
		t.Fatalf("same-label clicks mapped to %v", received)
	}
}

func TestPluginSessionOutletUIReorderKeepsKeyboardFocusOnHandle(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(previous)
	snapshot := sessionOutletTestSnapshot()
	snapshot.Outlets = snapshot.Outlets[:2]
	var received []string
	var revision uint64
	tt := ui.NewTester(func(c *ui.Context) {
		renderSessionPluginOutlets(c, snapshot, func(captured pluginui.Snapshot, handle string) {
			received = append(received, handle)
			revision = captured.Revision
		})
	}, 390, 300)
	firstLabel := sessionPluginOutletItems(snapshot.Outlets)[0].Accessible
	tt.Key(0, ui.KeyTab)
	if !tt.Focused(firstLabel) {
		t.Fatal("first instance did not take keyboard focus")
	}
	snapshot.Outlets[0], snapshot.Outlets[1] = snapshot.Outlets[1], snapshot.Outlets[0]
	snapshot.Revision++
	tt.Frame()
	if !tt.Focused(firstLabel) {
		t.Fatal("reorder transferred keyboard focus to another handle")
	}
	tt.Key(0, ui.KeyEnter)
	if !reflect.DeepEqual(received, []string{"first"}) || revision != snapshot.Revision {
		t.Fatalf("reorder activated %v at revision %d", received, revision)
	}
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeySpace)
	if !reflect.DeepEqual(received, []string{"first", "second"}) {
		t.Fatalf("keyboard order did not follow reordered rows: %v", received)
	}
}

func TestPluginSessionOutletUIUnavailableRemovedAndMissingOwnerDoNotActivate(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(previous)
	snapshot := sessionOutletTestSnapshot()
	snapshot.Outlets = snapshot.Outlets[:2]
	snapshot.Outlets[1].Available = false
	calls := 0
	tt := ui.NewTester(func(c *ui.Context) {
		renderSessionPluginOutlets(c, snapshot, func(pluginui.Snapshot, string) { calls++ })
	}, 390, 300)
	items := sessionPluginOutletItems(snapshot.Outlets)
	if err := tt.Click(items[1].Accessible); err != nil {
		t.Fatal(err)
	}
	if calls != 0 || !tt.HasText("Unavailable") {
		t.Fatal("unavailable occurrence activated or lacked status")
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused(items[0].Accessible) {
		t.Fatal("enabled instance not focused")
	}
	snapshot.Outlets = snapshot.Outlets[1:]
	snapshot.Outlets[0].Available = true
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	if calls != 0 {
		t.Fatal("removed focused instance activated its replacement")
	}
	availableSnapshot := snapshot
	for _, unavailable := range []string{"missing-owner", "disconnected", "version"} {
		copy := availableSnapshot
		switch unavailable {
		case "missing-owner":
			copy.Subject = nil
		case "disconnected":
			copy.Connected = false
		case "version":
			copy.Unavailable = "session-version-incompatible"
		}
		snapshot = copy
		tt.Frame()
		if err := tt.Click(sessionPluginOutletItems(snapshot.Outlets)[0].Accessible); err != nil {
			t.Fatal(err)
		}
		tt.Key(0, ui.KeyEnter)
		if calls != 0 {
			t.Fatalf("%s state admitted action", unavailable)
		}
	}
	snapshot.Outlets = nil
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	if calls != 0 || tt.HasText("Deliverable") {
		t.Fatal("empty list retained occurrence")
	}
}
