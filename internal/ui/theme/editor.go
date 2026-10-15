package theme

type Editor struct {
	Draft    Preferences
	Saved    Preferences
	Selected string
	Status   string
	Preview  bool
	OnSave   func(Preferences) error
}

func NewEditor(p Preferences, save func(Preferences) error) *Editor {
	p = p.Normalized()
	return &Editor{Draft: p.Clone(), Saved: p.Clone(), OnSave: save}
}
func (e *Editor) ApplyPreview() {
	if err := e.Draft.Validate(); err != nil {
		e.Status = err.Error()
		return
	}
	SetPreferences(e.Draft)
	e.Preview = true
	e.Status = "预览中，尚未保存 / Preview, not saved"
}
func (e *Editor) Cancel() {
	e.Draft = e.Saved.Clone()
	e.Preview = false
	SetPreferences(e.Saved)
	e.Status = "已恢复 / Restored"
}
func (e *Editor) Save() {
	if err := e.Draft.Validate(); err != nil {
		e.Status = err.Error()
		return
	}
	if e.OnSave == nil {
		return
	}
	if err := e.OnSave(e.Draft.Clone()); err != nil {
		e.Status = err.Error()
		return
	}
	e.Saved = e.Draft.Clone()
	SetPreferences(e.Saved)
	e.Preview = false
	e.Status = "已保存 / Saved"
}
func (e *Editor) DeleteSelected() {
	for i, f := range e.Draft.Custom {
		if f.ID == e.Selected {
			e.Draft.Custom = append(e.Draft.Custom[:i], e.Draft.Custom[i+1:]...)
			if e.Draft.Light == e.Selected {
				e.Draft.Light = "deepseek"
			}
			if e.Draft.Dark == e.Selected {
				e.Draft.Dark = "deepseek"
			}
			e.Selected = ""
			return
		}
	}
}
