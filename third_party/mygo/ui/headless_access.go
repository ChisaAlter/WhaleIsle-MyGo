package ui

import (
	"fmt"

	"github.com/egoist/mygo/internal/platform"
)

// SetValue replaces the complete value of the text field labeled s through its
// accessibility action, as a screen reader does. It enables accessibility and
// reports an error for a missing, disabled or read-only field.
func (t *Tester) SetValue(s, value string) error {
	t.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	for _, n := range t.h.access.Nodes {
		if n.Label != s || n.Role != platform.RoleTextField && n.Role != platform.RoleComboBox {
			continue
		}
		if n.Actions&platform.ActionSetValue == 0 {
			return fmt.Errorf("ui: text field %q is disabled or read-only", s)
		}
		t.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: n.ID, Action: platform.AccessSetValue, Text: value})
		return nil
	}
	return fmt.Errorf("ui: no text field is labeled %q", s)
}
