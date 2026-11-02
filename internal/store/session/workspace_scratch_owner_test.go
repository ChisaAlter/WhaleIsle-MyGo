package session

import (
	"context"
	"testing"

	"whaleisle/internal/dsh"
)

// The Host may decline reuse after the user has navigated elsewhere. That
// stale gesture must not launch a fresh scratch creation, even on an ABA return.
func TestScratchCreationStopsAfterNavigationDuringReuse(t *testing.T) {
	for _, transition := range []string{"selection", "selection-aba", "workspace", "workspace-aba", "same-client-events"} {
		t.Run(transition, func(t *testing.T) {
			h := newWorkspaceCreationHost(t)
			h.store.mu.Lock()
			h.store.items = []Item{{SessionID: "scratch-blank", Cwd: "/scratch", Blank: true}}
			h.store.mu.Unlock()
			target := h.capture("::scratch")
			h.handler = func(call workspaceCreationCall) (map[string]any, *dsh.Error) {
				if call.Method != "session/blankReuse" {
					t.Fatalf("stale gesture sent %s", call.Method)
				}
				switch transition {
				case "selection", "selection-aba":
					h.store.mu.Lock()
					h.store.selectSessionLocked("different")
					if transition == "selection-aba" {
						h.store.selectSessionLocked("current")
					}
					h.store.mu.Unlock()
				case "workspace", "workspace-aba":
					h.store.SetWorkspaceCwd("/different")
					if transition == "workspace-aba" {
						h.store.SetWorkspaceCwd("/configured")
					}
				case "same-client-events":
					h.store.BeginEvents(context.Background(), h.client)
				}
				return map[string]any{"reusable": false}, nil
			}
			if id, err := target.Create(context.Background()); err == nil || id != "" {
				t.Fatalf("obsolete scratch gesture returned %q, %v", id, err)
			}
			if len(h.calls) != 1 || h.calls[0].Method != "session/blankReuse" {
				t.Fatalf("unexpected calls after navigation: %+v", h.calls)
			}
		})
	}
}
