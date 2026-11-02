package main

import (
	"github.com/egoist/mygo/ui"
	"time"
	"whaleisle/internal/desktop/pluginui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
	"whaleisle/internal/ui/frame"
)

// withPluginSessionFrame offers an explicit menu operation before any owner
// exists and renders only proven current header controls after activation.
func (s *mainState) withPluginSessionFrame(m frame.Model) frame.Model {
	if m.WorkspacePage || frame.MainConversationChildActive(m) {
		s.syncPluginSession(false)
		return m
	}
	s.mu.Lock()
	p := s.nativePlugins.session
	if p == nil || !p.visible || p.sessions != m.Store {
		s.mu.Unlock()
		return m
	}
	owner := p.owner
	s.mu.Unlock()
	m.SessionPluginsActive = owner != nil && owner.Alive() && s.pluginSessionCurrent(p)
	m.OnOpenSessionPlugins = func() { s.openPluginSession(p) }
	m.OnCloseSessionPlugins = func() { s.closePluginSession(p) }
	if m.SessionPluginsActive {
		if owner.Inline() {
			width, height := int(m.ViewportW), int(m.ViewportH)
			if width < 1 || height < 1 {
				s.mu.Lock()
				win := s.win
				s.mu.Unlock()
				if win != nil {
					bounds := win.ContentBounds()
					width, height = bounds.Width, bounds.Height
				}
			}
			owner.RequestMirror(width, height)
		}
		captured := owner.Snapshot()
		mirror := owner.MirrorLayout()
		// This closure is created for this frame only. Parse its raw event window
		// once, then reuse the exact matches across all visible footer rows.
		var turnRows map[pluginui.SessionTurnTail][]sessionPluginOutletItem
		prepared := false
		focusRequested := false
		m.ConversationTurnContributions = func(c *ui.Context, snapshot session.ViewState, msg session.Message) {
			if !s.pluginSessionCurrent(p) {
				return
			}
			s.mu.Lock()
			sameOwner := p.owner == owner
			s.mu.Unlock()
			tail, known := pluginSessionTurnEnd(msg)
			if !sameOwner || !known {
				return
			}
			if !prepared {
				turnRows, prepared = pluginSessionTurnRows(captured, snapshot), true
			}
			items := turnRows[tail]
			if owner.Inline() {
				c.After(400 * time.Millisecond)
				if mirror != nil && len(items) > 0 {
					// Only an intersecting painted row, not List's measurement or
					// overscan builds, may select the source scroll anchor.
					ui.Box(c.Key("inline-tail-anchor:" + items[0].Outlet.Handle)).FillWidth().Height(0).DrawOver(func(painter *ui.Painter, r ui.Rect) {
						clip := painter.NativeSurfaceClip()
						if !focusRequested && r.Y >= clip.Y && r.Y < clip.Y+clip.H {
							focusRequested = true
							owner.NoteMirrorNativeFragment(tail, 0, float64(r.Y))
						}
					})
				}
				if chat.PluginInlineTurn(c, owner, mirror, tail) {
					return
				}
			}
			renderSessionPluginTurnOutlets(c, items, func(row pluginui.SessionOutlet) {
				request := sessionPluginTurnRequest{Page: captured.Page, Revision: captured.Revision, Subject: *captured.Subject, Tail: tail, Handle: row.Handle}
				_ = s.showPluginSessionTurn(p, owner, request)
			})
		}
	}
	m.ConversationContributions = func(c *ui.Context, snapshot session.ViewState, slot string) {
		if !s.pluginSessionCurrent(p) {
			return
		}
		s.mu.Lock()
		owner := p.owner
		s.mu.Unlock()
		if owner == nil {
			return
		}
		state := owner.Snapshot()
		for _, control := range pluginSessionControlRows(state, pluginSessionSubject(snapshot), slot) {
			row := control
			button := ui.Button(c.Key("session-plugin-control:"+row.Handle), row.Label).Label(row.Label).Tooltip(row.Label).
				Height(28).MinWidth(28).MaxWidth(160).Shrink(1).Padding(0, 6).Radius(6).BorderWidth(0).Background(ui.Transparent).Disabled(row.Disabled)
			if button.Clicked() && s.pluginSessionCurrent(p) {
				owner.Mask(false)
				err := owner.ClickSessionControl(state.Page, state.Revision, *state.Subject, row.Handle)
				s.mu.Lock()
				p.open = true
				if err != nil {
					p.error = err.Error()
				}
				s.mu.Unlock()
			}
		}
	}
	return m
}

func pluginSessionControlRows(state pluginui.Snapshot, subject pluginui.Subject, slot string) []pluginui.SessionControl {
	if !state.Connected || !state.Observation || state.Subject == nil || *state.Subject != subject {
		return nil
	}
	var rows []pluginui.SessionControl
	for _, control := range state.SessionControls {
		if control.Slot == slot {
			rows = append(rows, control)
		}
	}
	return rows
}
