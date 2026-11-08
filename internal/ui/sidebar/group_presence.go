package sidebar

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/widgets"
)

const groupPresenceDuration = 200 * time.Millisecond
const groupFadeDuration = 100 * time.Millisecond
const sidebarScrollbarLingerDuration = 2 * time.Second

type workspaceListMotionState struct {
	HostGeneration uint64
	Initialized    bool
	Signature      string
	GlideUntil     time.Time
}

// workspaceGroupListMotion mirrors AnimatedRows (ROW_FADE_MS = 100ms,
// ROW_GLIDE_MS = 200ms, ease-out): exiting workspace blocks or session rows
// fade out in 100ms while remaining blocks glide to their new positions over
// 200ms. Collapse/expand leaves the roster signature unchanged so
// workspaceGroupSessionRun owns fold height without double-smoothing.
func workspaceGroupListMotion(c *ui.Context, list ui.Element, hostGeneration uint64, groups []session.WorkspaceGroup) ui.ElementTransition {
	state := ui.Local(list, "workspace-list-motion", func() workspaceListMotionState { return workspaceListMotionState{} })
	var b strings.Builder
	for i, g := range groups {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(g.Key)
		for _, it := range g.Items {
			b.WriteByte('\t')
			b.WriteString(it.SessionID)
		}
	}
	return advanceWorkspaceMotion(c, state, hostGeneration, b.String())
}

func workspaceArchiveSectionMotion(c *ui.Context, section ui.Element, snap session.ViewState) ui.ElementTransition {
	state := ui.Local(section, "workspace-archive-motion", func() workspaceListMotionState { return workspaceListMotionState{} })
	var b strings.Builder
	for i, ws := range snap.Workspaces {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(ws.WorkspaceID)
		for _, id := range ws.SessionIDs {
			b.WriteByte('\t')
			b.WriteString(id)
		}
	}
	b.WriteString("\n#archived:")
	for _, id := range snap.ArchivedSessionIDs {
		b.WriteByte('\t')
		b.WriteString(id)
	}
	return advanceWorkspaceMotion(c, state, snap.WorkspaceLayout.HostGeneration, b.String())
}

func advanceWorkspaceMotion(c *ui.Context, state *workspaceListMotionState, hostGeneration uint64, sig string) ui.ElementTransition {
	now := c.Now()
	if !state.Initialized || state.HostGeneration != hostGeneration {
		state.HostGeneration = hostGeneration
		state.Initialized = true
		state.Signature = sig
		state.GlideUntil = time.Time{}
	} else if sig != state.Signature {
		state.Signature = sig
		if !c.Preferences().ReduceMotion {
			state.GlideUntil = now.Add(groupPresenceDuration)
		}
	}
	glide := !c.Preferences().ReduceMotion && !state.GlideUntil.IsZero() && now.Before(state.GlideUntil)
	return ui.ElementTransition{
		Duration:     groupPresenceDuration,
		ExitDuration: groupFadeDuration,
		Ease:         widgets.DesignEase,
		Position:     glide,
		Colors:       true,
		Enter:        &ui.Motion{Opacity: 0},
		Exit:         &ui.Motion{Opacity: 0},
	}
}

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
