package shortcuts

func functionKey(key string) bool {
	switch key {
	case "F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8", "F9", "F10", "F11", "F12":
		return true
	}
	return false
}

func (e *Editor) BeginRecording(id string) {
	if e.ReadError != "" || e.ResetPending || e.drafts[id] == nil {
		return
	}
	e.Recording = id
	e.Err = ""
	e.Status = ""
}
func (e *Editor) CancelRecording() { e.Recording = "" }

// SaveRow follows the original inline editor: a released recording, clear or
// reset submits only that command against the revision the editor reviewed.
// Other native text drafts remain uncommitted and survive a failed write.
func (e *Editor) SaveRow(id string, reset bool) bool {
	if e.drafts[id] == nil || e.ReadError != "" || e.persist == nil {
		return false
	}
	value := *e.drafts[id]
	issue := ""
	if !reset {
		if e.describe != nil {
			issue = e.describe(id, value)
		} else if !e.shared {
			issue = Validate(e.Values())[id]
		}
	}
	if issue != "" {
		e.Err = issue
		return false
	}
	chord, err := Parse(value)
	if err != nil {
		e.Err = err.Error()
		return false
	}
	value = chord.String()
	patch := value
	if reset && e.shared {
		patch = "\x00reset"
	}
	if err = e.persist(map[string]string{id: patch}); err != nil {
		e.Err = "Save failed; draft retained: " + err.Error()
		return false
	}
	e.active[id], *e.drafts[id] = value, value
	delete(e.resetDrafts, id)
	e.Recording, e.Err = "", ""
	e.Status = "Saved"
	return true
}
func (e *Editor) RecordAndSave(raw string) bool {
	id := e.Recording
	if id == "" || e.drafts[id] == nil {
		return false
	}
	chord, err := Parse(raw)
	if err != nil || chord.Key == "" {
		e.Err = "Unsupported physical shortcut"
		return false
	}
	*e.drafts[id] = chord.String()
	return e.SaveRow(id, false)
}
func (e *Editor) Record(raw string) bool {
	if e.Recording == "" || e.drafts[e.Recording] == nil {
		return false
	}
	chord, err := Parse(raw)
	if err != nil || chord.Key == "" {
		e.Err = "请按有效组合键；Escape 取消 / Press a valid chord; Escape cancels"
		return false
	}
	*e.drafts[e.Recording] = chord.String()
	e.Recording = ""
	e.Err = ""
	e.Status = "已录入草稿；保存后生效 / Recorded as draft; save to apply"
	return true
}
