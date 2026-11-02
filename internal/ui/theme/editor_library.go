package theme

import "reflect"

func (e *Editor) SelectThemeHalf(dark bool, id string) {
	next, draft := e.Saved.Clone(), e.Draft.Clone()
	if dark {
		next.Dark, draft.Dark = id, id
	} else {
		next.Light, draft.Light = id, id
	}
	e.saveFontField(next, draft)
}

func (e *Editor) StartFamily(f Family) {
	e.Draft = e.Saved.Clone()
	e.Selected = f.ID
	f.Light, f.Dark = f.Light.Clone(), f.Dark.Clone()
	found := false
	for i := range e.Draft.Custom {
		if e.Draft.Custom[i].ID == f.ID {
			e.Draft.Custom[i] = f
			found = true
			break
		}
	}
	if !found {
		e.Draft.Custom = append(e.Draft.Custom, f)
	}
	e.PreviewSelectedFamily()
}

func (e *Editor) PreviewSelectedFamily() {
	if e.Selected == "" {
		return
	}
	preview := e.Draft.Clone()
	preview.Light, preview.Dark = e.Selected, e.Selected
	if preview.Validate() != nil {
		return
	}
	if !reflect.DeepEqual(CurrentPreferences(), preview) {
		SetPreferences(preview)
		RefreshWindows()
	}
	e.Preview = true
}

func (e *Editor) SaveSelectedFamily() {
	if e.Selected == "" {
		return
	}
	e.Draft.Light, e.Draft.Dark = e.Selected, e.Selected
	e.Save()
	if e.Status != "" && !e.Preview {
		e.Selected = ""
	}
}

// DeleteFamily removes only the addressed persisted custom family. Library
// actions must not save another family's open editor draft or activate a source.
func (e *Editor) DeleteFamily(id string) bool {
	next, draft := e.writeBaseline(), e.Draft.Clone()
	remove := func(p *Preferences) bool {
		for i, family := range p.Custom {
			if family.ID != id {
				continue
			}
			p.Custom = append(p.Custom[:i], p.Custom[i+1:]...)
			if p.Light == id {
				p.Light = Defaults().Light
			}
			if p.Dark == id {
				p.Dark = Defaults().Dark
			}
			return true
		}
		return false
	}
	if !remove(&next) {
		return false
	}
	remove(&draft)
	if !e.saveFontField(next, draft) {
		return false
	}
	if e.Selected == id {
		e.Selected = ""
	}
	return true
}

func (e *Editor) ResetAppearanceFonts() {
	next, draft := e.Saved.Clone(), e.Draft.Clone()
	next.Interface, draft.Interface = FontPreference{Size: 16}, FontPreference{Size: 16}
	next.Chat.Family, draft.Chat.Family = "", ""
	next.Code, draft.Code = FontPreference{Size: 13}, FontPreference{Size: 13}
	next.Composer.Family, draft.Composer.Family = "", ""
	next.Terminal.Family, draft.Terminal.Family = "", ""
	e.saveFontField(next, draft)
}
