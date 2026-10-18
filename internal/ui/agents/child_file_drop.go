package agents

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

// childFileDropTarget owns the blank/history area of an addressed child pane.
// The child editor remains the innermost target; this only extends the same
// image intake to the rest of the child panel, as the original agent panel's
// FileDropZone did. The exact navigation token/address and Store owner are
// captured for every drop, so a stale child cannot feed another session.
func childFileDropTarget(c *ui.Context, area ui.Element, owner *session.Store, v session.AgentNavigationView) bool {
	if owner == nil || v.Depth == 0 || v.Address.Mode != "continuable" {
		return false
	}
	disabled := (v.RetainedReadOnly && !v.RetainedPromptAllowed) || v.Busy || v.ActionBusy || !v.ParentAvailabilityKnown || !v.ParentAvailable
	stateRef := ui.Local(area, "child-file-drop-state", func() *composer.DraftState { return composer.NewDraftState("") })
	if stateRef == nil || *stateRef == nil {
		return false
	}
	state := *stateRef
	state.Sending = disabled
	over := composer.FileDropTarget(area, composer.Model{
		Draft: &state.Text,
		State: state,
		OnAttachFiles: func(paths []string) {
			owner.AddAgentNavigationImages(v.Token, v.Address, paths, nil)
		},
	})
	if !over {
		return false
	}
	bounds := area.Bounds()
	ui.Overlay(c, func() {
		ui.Box(c).Absolute().Left(bounds.X+12).Top(bounds.Y+12).Padding(10).
			Background(c.Theme().Background).Border(1, c.Theme().Accent).PassThrough().Children(func() {
			label := "Drop images into this child conversation"
			if disabled {
				label = "Wait for child continuation"
			}
			ui.Text(c, label).PassThrough()
		})
	})
	return true
}
