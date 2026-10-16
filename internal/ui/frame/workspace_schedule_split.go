package frame

// A duplicated task pane starts on the same task identities, but creates its
// own editor/history controls lazily. Mutable panels and unsaved drafts are
// never shared between occurrences or serialized into the workspace layout.
func copySchedulePaneTarget(target, source *WorkspacePane) {
	if target.Kind != "schedule-task" || source.Kind != "schedule-task" || source.Schedule == nil {
		return
	}
	target.Schedule = &SchedulePaneState{Targets: make(map[string]ScheduleTaskTarget, len(source.Schedule.Targets))}
	for scope, task := range source.Schedule.Targets {
		target.Schedule.Targets[scope] = task
	}
}
