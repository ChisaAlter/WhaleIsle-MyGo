package frame

import (
	"testing"
	scheduleui "whaleisle/internal/ui/schedule"
)

func TestSchedulePaneIndependentSplit(t *testing.T) {
	for _, vertical := range []bool{false, true} {
		kind := "schedule-task"
		w := &WorkspacePanes{}
		w.syncRoute(kind)
		scope := "host\x00viewer"
		w.Root.Schedule = &SchedulePaneState{Targets: map[string]ScheduleTaskTarget{scope: {SessionID: "s1", ID: "task1"}}, panels: map[string]*scheduleui.Panel{"editor": {}}}
		m := Model{RightPaneOpen: true, RightPaneTab: &kind, Sizes: &LayoutSizes{}}
		m.Sizes.Workspace = w
		original, before := w.Root, string(w.Export())
		originalSchedule := original.Schedule
		SplitWorkspacePane(m, vertical)
		if vertical {
			if w.Root != original || string(w.Export()) != before || w.Root.Schedule != originalSchedule || w.Root.Schedule.panels["editor"] == nil {
				t.Fatal("rejected vertical split changed retained task or editor")
			}
			continue
		}
		first, second := w.Root.First, w.Root.Second
		if first == nil || second == nil || w.Root.Axis != "horizontal" || second.Kind != "schedule-task" || first.ID == second.ID {
			t.Fatal("task pane was not split")
		}
		before = string(w.Export())
		SplitWorkspacePane(m, false)
		if w.Root.First != first || w.Root.Second != second || string(w.Export()) != before {
			t.Fatal("third dock split changed retained task owners")
		}
		if second.Schedule == first.Schedule || second.Schedule.panels != nil || second.Schedule.Targets[scope] != first.Schedule.Targets[scope] {
			t.Fatal("task target/editor not independently copied")
		}
		second.Schedule.Targets[scope] = ScheduleTaskTarget{SessionID: "s2", ID: "task2"}
		if first.Schedule.Targets[scope].ID != "task1" {
			t.Fatal("editing second pane changed first")
		}
		restored, err := RestoreWorkspacePanes(w.Export())
		if err != nil {
			t.Fatal(err)
		}
		if restored.Root.First.Schedule.Targets[scope].ID != "task1" || restored.Root.Second.Schedule.Targets[scope].ID != "task2" {
			t.Fatal("restore collapsed task identities")
		}
	}
}
