package theme

import (
	"reflect"

	"whaleisle/internal/ui/i18n"
)

type Editor struct {
	Draft             Preferences
	Saved             Preferences
	Selected          string
	Status            string
	Preview           bool
	OnSave            func(Preferences) error
	OnImport          func() (string, error)
	Pending           func() bool
	FlushWrites       func()
	pendingSave       *Preferences
	pendingFamilySave string
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
	e.Status = i18n.T(i18n.KeyThemeEditorPreview)
}
func (e *Editor) Cancel() {
	e.Draft = e.Saved.Clone()
	e.Preview = false
	SetPreferences(e.Saved)
	RefreshWindows()
	e.Status = i18n.T(i18n.KeyThemeEditorRestored)
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
	e.Status = i18n.T(i18n.KeyThemeEditorSaved)
	if pending {
		e.Status = i18n.T(i18n.KeyThemeEditorSaving)
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
	} else if e.Selected != "" {
		// Imports and unrelated saves rebase the library, never the selected
		// family's independent editor. Failed imports cannot hitchhike on Save.
		var family *Family
		for _, f := range e.Draft.Custom {
			if f.ID == e.Selected {
				copy := f
				family = &copy
				break
			}
		}
		e.Draft.Custom = p.Clone().Custom
		if family != nil {
			found := false
			for i := range e.Draft.Custom {
				if e.Draft.Custom[i].ID == family.ID {
					e.Draft.Custom[i] = *family
					found = true
					break
				}
			}
			if !found {
				e.Draft.Custom = append(e.Draft.Custom, *family)
			}
		}
	}
	if failure != "" {
		e.Status = failure
	} else if e.pendingSave != nil {
		e.Status = i18n.T(i18n.KeyThemeEditorSaved)
	}
	if ownerChanged {
		e.Selected = ""
		e.pendingFamilySave = ""
		e.generalFontEdits = nil
		e.generalFontRevision, e.generalFontAttempted = 0, 0
	} else if failure == "" && e.pendingSave != nil {
		if submitted && e.pendingFamilySave == e.Selected {
			e.Selected = ""
		}
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
	e.pendingFamilySave = ""
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
