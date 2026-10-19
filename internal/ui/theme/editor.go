package theme

import "reflect"

type Editor struct {
	Draft       Preferences
	Saved       Preferences
	Selected    string
	Status      string
	Preview     bool
	OnSave      func(Preferences) error
	OnImport    func() (string, error)
	Pending     func() bool
	FlushWrites func()
	pendingSave *Preferences
	// Only General's explicitly edited family inputs enter this pending set.
	generalFontEdits     map[string]string
	generalFontRevision  uint64
	generalFontAttempted uint64
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
	RefreshWindows()
	e.Preview = true
	e.Status = "预览中，尚未保存 / Preview, not saved"
}
func (e *Editor) Cancel() {
	e.Draft = e.Saved.Clone()
	e.Preview = false
	SetPreferences(e.Saved)
	RefreshWindows()
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
	pending := e.Pending != nil && e.Pending()
	if !pending {
		e.Saved = e.Draft.Clone()
	} else {
		next := e.Draft.Clone()
		e.pendingSave = &next
	}
	SetPreferences(e.Draft)
	RefreshWindows()
	e.Preview = false
	e.Status = "已保存 / Saved"
	if pending {
		e.Status = "正在保存… / Saving…"
	}
}

// AcceptHost settles an asynchronous writer with the accepted namespace value.
// A failed write retains its draft; a later edit is never acknowledged by an
// older submission. Owner replacement deliberately discards that owner's draft.
func (e *Editor) AcceptHost(p Preferences, ownerChanged bool, failure string) {
	clean := reflect.DeepEqual(e.Draft, e.Saved)
	submitted := e.pendingSave != nil && reflect.DeepEqual(e.Draft, *e.pendingSave)
	e.Saved = p.Clone()
	if ownerChanged || clean || failure == "" && submitted {
		e.Draft = p.Clone()
		e.Preview = false
	}
	if failure != "" {
		e.Status = failure
	} else if e.pendingSave != nil {
		e.Status = "已保存 / Saved"
	}
	if ownerChanged {
		e.generalFontEdits = nil
		e.generalFontRevision, e.generalFontAttempted = 0, 0
	} else if failure == "" && e.pendingSave != nil {
		for role, value := range e.generalFontEdits {
			accepted := ""
			switch role {
			case "text":
				accepted = p.Interface.Family
			case "code":
				accepted = p.Code.Family
			case "terminal":
				accepted = p.Terminal.Family
			}
			if normalizeFontFamily(value) == accepted {
				delete(e.generalFontEdits, role)
			}
		}
	}
	e.pendingSave = nil
}

func (e *Editor) writeBaseline() Preferences {
	if e.pendingSave != nil {
		return e.pendingSave.Clone()
	}
	return e.Saved.Clone()
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
