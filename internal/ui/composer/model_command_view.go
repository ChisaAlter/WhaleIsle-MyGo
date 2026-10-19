package composer

func handleModelCommand(m Model) bool {
	if m.GroupMembers != nil || m.Editing || m.State == nil || m.Draft == nil {
		return false
	}
	return m.State.HandleNativeCommand(m.Snapshot, m.OnNativeCommandAction)
}
