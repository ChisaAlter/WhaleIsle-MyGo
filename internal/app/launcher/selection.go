package launcherctl

import (
	"context"

	"whaleisle/internal/store/session"
)

type startupSelectionState struct {
	store              *session.Store
	binding            uint64
	boot               int
	panel, gesture     uint64
	observed, consumed bool
}

// ClaimStartupSelection atomically claims the single initial empty-workspace
// blank session creation opportunity if no route/draft/boot fence retired it.
func (c *Controller) ClaimStartupSelection(win WindowRef, sessions *session.Store, receipt session.InitialWorkspaceSession) bool {
	if c == nil || sessions == nil || receipt.Binding == 0 {
		return false
	}
	env := c.hostEnv()
	if !env.HasSessions || env.MainWindow != win {
		return false
	}
	allowed := !env.Quitting && !env.ExclusivePanelOpen && !env.HasOccupiedDrafts

	c.mu.Lock()
	defer c.mu.Unlock()
	initial := &c.selection
	if !initial.observed || initial.store != sessions || initial.binding != receipt.Binding {
		*initial = startupSelectionState{
			store:    sessions,
			binding:  receipt.Binding,
			boot:     env.BootStatus.Generation,
			panel:    env.AssistantPanelGen,
			gesture:  env.WorkspaceOpenGen,
			observed: true,
		}
	}
	if initial.consumed {
		return false
	}
	if initial.boot != env.BootStatus.Generation ||
		initial.panel != env.AssistantPanelGen ||
		initial.gesture != env.WorkspaceOpenGen ||
		!allowed {
		initial.consumed = true
		return false
	}
	if !receipt.Ready {
		return false
	}
	initial.consumed = true
	return true
}

// RetireStartupSelection marks the initial startup blank-session opportunity consumed.
func (c *Controller) RetireStartupSelection() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.selection.consumed = true
	c.mu.Unlock()
}

// MaybeStartupBlankSession checks initial session readiness and asks HostDeps
// to open the initial blank session cleanly off the render path.
func (c *Controller) MaybeStartupBlankSession(win WindowRef, sessions *session.Store) {
	if c == nil || win == nil || sessions == nil || c.deps == nil {
		return
	}
	receipt := sessions.InitialWorkspaceSession()
	if !c.ClaimStartupSelection(win, sessions, receipt) {
		return
	}
	c.deps.OpenStartupBlankSession(win, sessions, receipt.HostGeneration, receipt.WorkspaceID)
}

var _ = context.Background
