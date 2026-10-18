package dshbot

import (
	"os"
	"path/filepath"
	"testing"

	"whaleisle/internal/desktop/pluginrpc"
)

func TestSetFromDescribePersists(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	res := pluginrpc.DshbotDescribeResult{
		View: pluginrpc.DshbotSettingsView{
			Revision: 3,
			Value: pluginrpc.DshbotCatalogValue{
				Items: []pluginrpc.DshbotCatalogItem{
					{ID: "b1", Kind: "bot", Name: "Alpha", SessionID: "s1", Title: "Helper"},
					{ID: "r1", Kind: "room", Name: "War Room", SessionID: "s2", MemberBotIDs: []string{"b1", "b2"}},
				},
			},
		},
	}
	s.SetFromDescribe(res, false)
	snap := s.Snapshot()
	if len(snap.Rows) != 2 || snap.Revision != 3 {
		t.Fatalf("snap = %+v", snap)
	}
	if snap.Rows[0].Kind != "bot" || snap.Rows[0].SessionID != "s1" {
		t.Fatalf("first row = %+v", snap.Rows[0])
	}

	s2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	snap2 := s2.Snapshot()
	if len(snap2.Rows) != 2 {
		t.Fatalf("reload rows = %d", len(snap2.Rows))
	}
	if _, err := os.ReadFile(filepath.Join(Dir(dir), rosterFile)); err != nil {
		t.Fatal(err)
	}
}

func TestMergeActivityAndGroupRuntime(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.SetFromDescribe(pluginrpc.DshbotDescribeResult{
		View: pluginrpc.DshbotSettingsView{
			Value: pluginrpc.DshbotCatalogValue{
				Items: []pluginrpc.DshbotCatalogItem{
					{ID: "b1", Kind: "bot", Name: "Alpha", SessionID: "s1"},
					{ID: "r1", Kind: "room", Name: "Room", SessionID: "s2"},
				},
			},
		},
	}, false)
	s.MergeActivity(pluginrpc.DshbotActivityResult{
		Entries: []pluginrpc.DshbotActivityEntry{
			{BotID: "b1", Preview: "hello from bot"},
		},
	})
	s.ApplyGroupRuntime(pluginrpc.DshbotGroupRuntimeResult{
		RoomID:    "r1",
		SessionID: "s2-live",
		Members: []pluginrpc.DshbotGroupRuntimeMember{
			{BotID: "b1", Status: "idle"},
			{BotID: "b2", Status: "busy"},
		},
	})
	snap := s.Snapshot()
	var bot, room Row
	for _, r := range snap.Rows {
		switch r.ID {
		case "b1":
			bot = r
		case "r1":
			room = r
		}
	}
	if bot.Detail != "hello from bot" {
		t.Fatalf("bot detail = %q", bot.Detail)
	}
	if room.MemberCount != 2 || room.SessionID != "s2-live" {
		t.Fatalf("room = %+v", room)
	}
	if room.Detail != "runtime: idle,busy" {
		t.Fatalf("room detail = %q", room.Detail)
	}
}
