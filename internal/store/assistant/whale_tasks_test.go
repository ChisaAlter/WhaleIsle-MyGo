package assistant

import (
	"errors"
	"testing"

	"whaleisle/internal/desktop/pluginrpc"
)

func TestWhaleTaskPanelPreservesUnknownSubmissionAndHostOwnership(t *testing.T) {
	var panel WhaleTaskPanel
	panel.Bind("host-a")
	panel.Edit("host-a", "goal", "Keep this draft")
	panel.Edit("host-a", "files", "C:\\owned\\notes.md\n C:\\owned\\data.csv ")
	ticket, draft, ok := panel.Begin("host-a", "start")
	if !ok || len(draft.Draft().Files) != 2 {
		t.Fatal("draft capture failed")
	}
	panel.Finish(ticket, "start", nil, false, "", errors.New("connection lost"))
	if _, _, ok = panel.Begin("host-a", "start"); ok {
		t.Fatal("uncertain start was submitted again")
	}
	snap := panel.Snapshot()
	if snap.Goal != "Keep this draft" || !snap.CreateUnknown {
		t.Fatal("lost uncertain draft", snap)
	}
	ticket, _, _ = panel.Begin("host-a", "refresh")
	panel.Finish(ticket, "refresh", &pluginrpc.WhaleTasks{Available: true}, false, "", nil)
	old, _, ok := panel.Begin("host-a", "start")
	if !ok {
		t.Fatal("readback did not release explicit next submission")
	}
	panel.Bind("host-b")
	current, _, _ := panel.Begin("host-b", "refresh")
	panel.Cancel(old)
	if !panel.Current(current) {
		t.Fatal("old completion cancelled new Host request")
	}
	if panel.Finish(old, "start", nil, true, "", nil) {
		t.Fatal("old Host receipt accepted")
	}
	panel.Finish(current, "refresh", &pluginrpc.WhaleTasks{Available: true, Tasks: []pluginrpc.WhaleTask{{ID: "task", Inputs: []pluginrpc.WhaleTaskInput{{ID: "input-1", Name: "original"}}}}}, false, "", nil)
	snap = panel.Snapshot()
	snap.Catalog.Tasks[0].Inputs[0].Name = "mutated"
	if panel.Snapshot().Catalog.Tasks[0].Inputs[0].Name != "original" {
		t.Fatal("render snapshot mutated stored tasks")
	}
}

func TestWhaleTaskPanelAcceptedStartClearsOnlyTaskDraft(t *testing.T) {
	var panel WhaleTaskPanel
	panel.Bind("host")
	panel.Edit("host", "goal", "Summarize")
	panel.Edit("host", "files", "C:\\notes.txt")
	panel.Edit("host", "bot", "bot-id")
	ticket, _, _ := panel.Begin("host", "start")
	panel.Finish(ticket, "start", nil, true, "", errors.New("list readback unavailable"))
	snap := panel.Snapshot()
	if snap.Goal != "" || snap.Files != "" || snap.CreateUnknown || snap.BotID != "bot-id" {
		t.Fatal("accepted mutation not reflected in local draft", snap)
	}
}
