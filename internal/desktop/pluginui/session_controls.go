package pluginui

// SessionControl names an enabled original DOM action and its actual rendered
// contribution. Header hosts use Slot to keep actions in their original zone.
type SessionControl struct {
	Handle   string `json:"handle"`
	Outlet   string `json:"outlet"`
	Slot     string `json:"slot"`
	Label    string `json:"label"`
	Disabled bool   `json:"disabled"`
}

func validSessionControls(controls []SessionControl, outlets []SessionOutlet) bool {
	if len(controls) > 128 {
		return false
	}
	seen := map[string]bool{}
	for _, control := range controls {
		if control.Handle == "" || len(control.Handle) > 64 || seen[control.Handle] || control.Label == "" || len(control.Label) > 8192 {
			return false
		}
		seen[control.Handle] = true
		found := false
		for _, row := range outlets {
			if row.Handle == control.Outlet && row.Slot == control.Slot && row.Available {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// ClickSessionControl routes one native header gesture to its still-mounted
// original control. Browser checks repeat the page's entry and scope identity.
func (o *Owner) ClickSessionControl(page, revision uint64, subject Subject, handle string) error {
	if o == nil {
		return errSessionUnavailable
	}
	o.mu.Lock()
	valid := !o.closed && !o.incompatible && o.state.Unavailable == "" && o.state.Connected && o.state.Page == page && o.state.Revision == revision && o.state.Subject != nil && o.state.Subject.equal(subject) && o.selected.equal(subject) && subject.Kind == "session"
	found := false
	for _, row := range o.state.SessionControls {
		if row.Handle == handle && !row.Disabled {
			found = true
			break
		}
	}
	o.mu.Unlock()
	if !valid || !found {
		return errSessionUnavailable
	}
	return o.send(map[string]any{"type": "sessionClick", "revision": revision, "subject": subject, "handle": handle})
}

// ShowSessionPage keeps the original page usable when a renderer version does
// not support entry observation, and is always an explicit user operation.
func (o *Owner) ShowSessionPage(page, revision uint64, subject Subject) error {
	if o == nil {
		return errSessionUnavailable
	}
	o.mu.Lock()
	valid := !o.closed && !o.incompatible && o.state.Unavailable == "" && o.state.Connected && o.state.Page == page && o.state.Revision == revision && o.state.Subject != nil && o.state.Subject.equal(subject) && o.selected.equal(subject) && subject.Kind == "session"
	o.mu.Unlock()
	if !valid {
		return errSessionUnavailable
	}
	return o.send(map[string]any{"type": "sessionPage", "revision": revision, "subject": subject})
}
