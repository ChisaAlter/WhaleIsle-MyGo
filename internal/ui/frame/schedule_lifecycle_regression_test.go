package frame

import (
	"testing"

	schedstore "whaleisle/internal/store/schedule"
	scheduleui "whaleisle/internal/ui/schedule"
)

func TestScheduleMinimizePreservesMountedEditorButSessionSuspendRetiresIt(t *testing.T) {
	w := &WorkspacePanes{}
	w.syncRoute("schedule-task")
	panel := &scheduleui.Panel{}
	panel.OpenTask(schedstore.Entry{ID: "A", SessionID: "owner", Origin: "host"}, 7)
	scope := "host\x00viewer"
	state := &SchedulePaneState{Targets: map[string]ScheduleTaskTarget{scope: {SessionID: "owner", ID: "A"}}, panels: map[string]*scheduleui.Panel{"editor": panel}}
	w.Root.Schedule = state
	// OS minimize/hide calls this transient presentation pause, not unmount.
	w.PauseContent()
	if state.panels["editor"] != panel {
		t.Fatal("OS minimize discarded the mounted task editor")
	}
	w.BeginContentFrame("same-host/same-session", "")
	state.rendered = true
	w.EndContentFrame()
	if state.panels["editor"] != panel {
		t.Fatal("restoring the visible seat replaced its editor")
	}
	w.SuspendSession()
	if len(state.panels) != 0 || state.Targets[scope].ID != "A" {
		t.Fatal("session switch kept old capabilities or lost persisted task binding")
	}
	state.panels = map[string]*scheduleui.Panel{"new-owner": {}}
	w.BeginContentFrame("new-host/new-session", "")
	w.EndContentFrame()
	if len(state.panels) != 0 {
		t.Fatal("truly hidden task seat kept its editor owner")
	}
}
