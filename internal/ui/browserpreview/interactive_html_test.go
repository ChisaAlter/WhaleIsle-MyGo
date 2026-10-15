package browserpreview

import (
	"context"
	"testing"
)

func TestInteractivePreviewCancellationDoesNotReplaceStaticTab(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	op := &interactivePreview{ctx:ctx, cancel:cancel}
	tab := &Tab{ID:7, Draft:"saved.html", interactive:op}
	controller := &Controller{Tabs:[]*Tab{tab}, Active:0}
	controller.cancelInteractivePreviews()
	if ctx.Err() == nil || tab.interactive != nil { t.Fatal("interactive owner was not cancelled") }
	if len(controller.Tabs) != 1 || controller.Tabs[0] != tab || tab.Draft != "saved.html" { t.Fatal("cancellation replaced the static tab") }
	controller.cancelInteractivePreviews() // Idempotent; no helper or native window starts.
}

func TestInteractiveStateStartsOff(t *testing.T) {
	tab := &Tab{}
	if tab.interactive != nil { t.Fatal("interactive preview enabled by default") }
	controller := &Controller{Tabs:[]*Tab{tab}}
	controller.startInteractiveHTML(tab, false)
	if tab.interactive != nil { t.Fatal("missing parent/document launched an operation") }
}
