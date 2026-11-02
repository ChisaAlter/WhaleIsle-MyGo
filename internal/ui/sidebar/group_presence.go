package sidebar

import (
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/widgets"
)

const groupPresenceDuration = 200 * time.Millisecond
const groupFadeDuration = 100 * time.Millisecond
const sidebarScrollbarLingerDuration = 2 * time.Second

type groupPresenceState struct {
	Initialized, Open bool
	Last              workspaceWatermarkRows
	Height            float32
}

// workspaceGroupSessionRun keeps the last open rows during a logical fold.
// The caller marks descendants hidden and clears watermark state immediately;
// this run only retains inert outgoing paint and takes no batch/drop authority.
func workspaceGroupSessionRun(c *ui.Context, host ui.Element, m Model, group session.WorkspaceGroup, collapsed bool, draw func(workspaceWatermarkRows)) {
	if group.Kind == "flat" {
		draw(visibleWorkspaceSessionRows(m, group))
		return
	}
	state := ui.Local(host, workspaceRowKey{HostGeneration: m.Snapshot.WorkspaceLayout.HostGeneration, ID: "presence/" + group.Key}, func() groupPresenceState { return groupPresenceState{} })
	open := !collapsed
	if open {
		state.Last = visibleWorkspaceSessionRows(m, group)
		retainWorkspaceWatermark(m, group.Key, state.Last.Watermark)
	}
	target := float32(0)
	if open {
		target = 1
	}
	track := host.AnimateWith("session-run-track", target, groupPresenceDuration, widgets.DesignEase)
	fade := host.AnimateWith("session-run-fade", target, groupFadeDuration, widgets.DesignEase)
	state.Initialized, state.Open = true, open
	if !open && (c.Preferences().ReduceMotion || track <= 0) {
		state.Last = workspaceWatermarkRows{}
		state.Height = 0
		return
	}
	run := ui.Column(c.Key("workspace-session-run/" + group.Key)).FillWidth().MinWidth(0).Shrink(0).ClipY().Opacity(fade).Disabled(!open).Inert(!open)
	if track < 1 {
		run.Height(track * state.Height)
	}
	run.Children(func() {
		inner := ui.Column(c.Key("workspace-session-inner/" + group.Key)).FillWidth().Shrink(0).Gap(2)
		inner.Children(func() { draw(state.Last) })
		if height := inner.Bounds().H; height > 0 && (open || state.Height == 0) {
			state.Height = height
		}
	})
}

type sidebarScrollbarState struct {
	Visible bool
	HideAt  time.Time
}

func (s *sidebarScrollbarState) update(now time.Time, inside bool) (visible bool, after time.Duration) {
	if inside {
		s.Visible, s.HideAt = true, time.Time{}
	} else if s.Visible {
		if s.HideAt.IsZero() {
			s.HideAt = now.Add(sidebarScrollbarLingerDuration)
		}
		if !now.Before(s.HideAt) {
			s.Visible, s.HideAt = false, time.Time{}
		}
	}
	if !s.HideAt.IsZero() {
		after = s.HideAt.Sub(now)
	}
	return s.Visible, after
}

// sidebarScrollbarLinger is scoped to the sidebar column's box, not DOM-like
// ancestry or the smaller scroll lane. Only its single deadline wakes a frame.
func sidebarScrollbarLinger(c *ui.Context, root ui.Element) bool {
	state := ui.Local(root, "sidebar-scrollbar-linger", func() sidebarScrollbarState { return sidebarScrollbarState{} })
	_, _, inside := root.PointerPosition()
	visible, after := state.update(c.Now(), inside)
	if after > 0 {
		c.After(after)
	}
	return visible
}
