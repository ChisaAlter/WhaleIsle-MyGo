package shortcuts

func functionKey(key string) bool {
	switch key {
	case "F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8", "F9", "F10", "F11", "F12":
		return true
	}
	return false
}

func (e *Editor) BeginRecording(id string) {
	if e.ReadError != "" || e.drafts[id] == nil {
		return
	}
	e.Recording = id
	e.Err = ""
	e.Status = ""
}
func (e *Editor) CancelRecording() { e.Recording = "" }
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
