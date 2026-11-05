package pluginui

import (
	"encoding/json"
	"sort"

	"github.com/egoist/mygo/ui"
)

func (o *Owner) RenderModelExtras(c *ui.Context, provider, namespace string) bool {
	if o == nil {
		return false
	}
	if provider != "" {
		o.mu.Lock()
		if o.modelTargets == nil {
			o.modelTargets = map[string]bool{}
		}
		o.modelTargets[provider] = true
		o.mu.Unlock()
	}
	subject := Subject{Kind: "navigation", Name: "models"}
	o.Select(subject)
	s := o.Snapshot()
	if s.Subject == nil || !s.Subject.equal(subject) {
		return false
	}
	slot := "settings.models.provider-card"
	if provider == "" {
		slot = "settings.models.footer"
	}
	s.Documents = nil
	s.Outlets = nil
	for _, d := range s.ModelDocuments {
		if d.Slot == slot && (provider == "" || d.Provider == provider && d.Namespace == namespace) {
			s.Documents = append(s.Documents, d)
		}
	}
	return o.renderNativeSnapshot(c, s, slot, "", -1, -1, "")
}

func (o *Owner) flushModelTargets() {
	o.mu.Lock()
	targets := []string{}
	for id := range o.modelTargets {
		targets = append(targets, id)
	}
	sort.Strings(targets)
	raw, _ := json.Marshal(targets)
	encoded := string(raw)
	page := o.page
	send := page != nil && !o.closed && encoded != o.modelTargetsLast && (o.selected.Kind == "navigation" && o.selected.Name == "models" || o.modelTargetsLast != "")
	if send {
		o.modelTargetsLast = encoded
	}
	o.mu.Unlock()
	if send {
		_ = o.sendToPage(page, map[string]any{"type": "modelTargets", "providers": targets})
	}
}
