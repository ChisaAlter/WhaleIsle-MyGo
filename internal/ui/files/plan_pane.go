package files

import "whaleisle/internal/ui/planpreview"

// CopyPlanPane copies read-only resource presentation, never a file draft or a
// resource reader. Splitting is an explicit navigation; recovery remains gated
// by AllowPlanRead as before.
func (t *Tabs) CopyPlanPane() Tabs {
	var out Tabs
	out.Bind(t.binding, t.boundRoot)
	out.Preferences = t.Preferences
	out.Plans = t.Plans
	out.Plans.Tabs = append([]string(nil), t.Plans.Tabs...)
	out.Plans.Active = make(map[string]string, len(t.Plans.Active))
	for k, v := range t.Plans.Active {
		out.Plans.Active[k] = v
	}
	out.Plans.Documents = make(map[string]planpreview.Document, len(t.Plans.Documents))
	for k, v := range t.Plans.Documents {
		out.Plans.Documents[k] = v
	}
	out.planSession = t.planSession
	if address := out.Plans.ActiveAddress(out.planSession); address != "" {
		out.AllowPlanRead(address)
	}
	return out
}
