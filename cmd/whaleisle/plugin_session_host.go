package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginui"
	desktopwebview "whaleisle/internal/desktop/webview"
	"whaleisle/internal/harness"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
)

type nativePluginSessionState struct {
	visible, open bool
	owner         *pluginui.Owner
	target        *session.ComposerTarget
	sessions      *session.Store
	endpoint      harness.Endpoint
	generation    int
	subject       pluginui.Subject
	error         string
	pendingTurn   *sessionPluginTurnPending
	turnExpired   bool
}

func pluginSessionSubject(v session.ViewState) pluginui.Subject {
	return pluginui.Subject{Kind: "session", SessionID: v.SelectedID, Epoch: fmt.Sprintf("%d:%d", v.ComposerHostGeneration, v.ComposerSessionGeneration)}
}

func pluginSessionStoreSubject(s *session.Store) pluginui.Subject {
	v := s.SelectionSummary()
	return pluginui.Subject{Kind: "session", SessionID: v.SelectedID, Epoch: fmt.Sprintf("%d:%d", v.ComposerHostGeneration, s.ComposerSessionGeneration())}
}

// syncPluginSession observes identity only; it never creates a browser or opens
// a Session. Retained owners use the cheap captured navigation fence each frame.
func (s *mainState) syncPluginSession(visible bool) {
	s.mu.Lock()
	p := s.nativePlugins.session
	if p == nil {
		p = &nativePluginSessionState{}
		s.nativePlugins.session = p
	}
	sessions, status, win := s.sessions, s.bootStatus, s.win
	ready := visible && status.State == harness.Ready && sessions != nil
	if win != nil && (!win.IsVisible() || win.IsMinimized()) {
		ready = false
	}
	changed := p.sessions != sessions || p.endpoint != status.Endpoint || p.generation != status.Generation
	owner, target, subject, active := p.owner, p.target, p.subject, p.owner != nil || p.open
	s.mu.Unlock()
	if !changed && ready && active {
		if target != nil {
			changed = !target.NavigationCurrent()
		} else {
			changed = subject != pluginSessionStoreSubject(sessions)
		}
		if owner != nil && !owner.Alive() {
			changed = true
		}
	}
	s.mu.Lock()
	old := (*pluginui.Owner)(nil)
	if !ready || changed {
		old = p.owner
		p.owner, p.target, p.open, p.error, p.pendingTurn, p.turnExpired = nil, nil, false, "", nil, false
		p.subject = pluginui.Subject{Kind: "session"}
	}
	p.visible = ready
	p.sessions, p.endpoint, p.generation = sessions, status.Endpoint, status.Generation
	s.mu.Unlock()
	if old != nil {
		old.Close()
	}
}

// Opening is a user operation. Snapshot capture and original owner navigation
// happen here, never while deciding whether a header contribution exists.
func (s *mainState) openPluginSession(p *nativePluginSessionState) {
	s.mu.Lock()
	win, sessions, endpoint, generation := s.win, s.sessions, p.endpoint, p.generation
	valid := s.nativePlugins.session == p && p.visible && sessions != nil && win != nil && s.bootStatus.State == harness.Ready && s.bootStatus.Endpoint == endpoint && s.bootStatus.Generation == generation
	existing, target := p.owner, p.target
	s.mu.Unlock()
	if !valid {
		return
	}
	if existing != nil && existing.Alive() && target != nil && target.NavigationCurrent() {
		s.mu.Lock()
		p.open, p.error, p.pendingTurn, p.turnExpired = true, "", nil, false
		s.mu.Unlock()
		existing.Mask(false)
		return
	}
	subject := pluginSessionStoreSubject(sessions)
	target, err := sessions.CaptureComposerTarget(subject.SessionID)
	if err != nil || subject.SessionID == "" || !target.NavigationCurrent() {
		s.mu.Lock()
		p.open = true
		p.error = i18n.T(i18n.KeyCmdSessionOwnerUnavailable)
		s.mu.Unlock()
		return
	}
	if existing != nil {
		existing.Close()
	}
	owner, err := pluginui.New(pluginui.Options{Parent: win, Origin: endpoint.Origin, Cookie: endpoint.Cookie, Session: true, Inline: true, Changed: func() { win.Update(func() {}) }, PermissionPolicy: desktopwebview.PermissionPolicy{SessionHost: true, NotificationsAllowed: systemNotificationsSupported}.Decide})
	s.mu.Lock()
	current := s.nativePlugins.session == p && p.visible && s.sessions == sessions && p.endpoint == endpoint && p.generation == generation && target.NavigationCurrent()
	if current {
		p.open = true
		if err != nil {
			p.owner, p.target = nil, nil
			p.error = err.Error()
		} else {
			p.owner, p.target, p.subject, p.error = owner, target, subject, ""
		}
	}
	s.mu.Unlock()
	if !current {
		if owner != nil {
			owner.Close()
		}
		return
	}
	if owner != nil {
		owner.Select(subject)
	}
}

func (s *mainState) closePluginSession(p *nativePluginSessionState) {
	s.mu.Lock()
	owner := p.owner
	p.owner, p.target, p.open, p.error, p.pendingTurn, p.turnExpired = nil, nil, false, "", nil, false
	s.mu.Unlock()
	if owner != nil {
		owner.Close()
	}
}

func (s *mainState) collapsePluginSession(p *nativePluginSessionState) {
	s.mu.Lock()
	owner := p.owner
	p.open, p.pendingTurn = false, nil
	s.mu.Unlock()
	if owner != nil {
		owner.Mask(true)
		owner.Show("")
	}
}

func (s *mainState) pluginSessionCurrent(p *nativePluginSessionState) bool {
	s.mu.Lock()
	valid := s.nativePlugins.session == p && p.visible && s.sessions == p.sessions && s.bootStatus.State == harness.Ready && s.bootStatus.Generation == p.generation && s.bootStatus.Endpoint == p.endpoint
	target := p.target
	s.mu.Unlock()
	return valid && target != nil && target.NavigationCurrent()
}

type sessionPluginOutletItem struct {
	Outlet                         pluginui.SessionOutlet
	Title, Attribution, Accessible string
}

// Ordinals only separate otherwise identical visible labels. Sort by the opaque
// occurrence handle so a reorder does not rename the selected instance.
func sessionPluginOutletItems(rows []pluginui.SessionOutlet) []sessionPluginOutletItem {
	items := make([]sessionPluginOutletItem, 0, len(rows))
	type visibleIdentity struct{ title, attribution string }
	groups := make(map[visibleIdentity][]int)
	for _, row := range rows {
		title := strings.TrimSpace(row.Label)
		if title == "" {
			title = strings.TrimSpace(row.EntryID)
		}
		if title == "" {
			title = i18n.T(i18n.KeySessionPluginsOutletUntitled)
		}
		var parts []string
		if row.Turn >= 0 && (row.HasTurn || row.Turn > 0) {
			parts = append(parts, fmt.Sprintf(i18n.T(i18n.KeySessionPluginsOutletTurn), row.Turn))
		}
		if call := strings.TrimSpace(row.CallID); call != "" {
			parts = append(parts, fmt.Sprintf(i18n.T(i18n.KeySessionPluginsOutletCall), call))
		}
		if len(parts) == 0 {
			parts = append(parts, i18n.T(i18n.KeySessionPluginsOutletUnknown))
		}
		attribution := strings.Join(parts, " · ")
		identity := visibleIdentity{title, attribution}
		groups[identity] = append(groups[identity], len(items))
		items = append(items, sessionPluginOutletItem{Outlet: row, Title: title, Attribution: attribution})
	}
	for _, indices := range groups {
		if len(indices) < 2 {
			continue
		}
		sort.Slice(indices, func(a, b int) bool { return items[indices[a]].Outlet.Handle < items[indices[b]].Outlet.Handle })
		for ordinal, index := range indices {
			items[index].Attribution += " · " + fmt.Sprintf(i18n.T(i18n.KeySessionPluginsOutletInstance), ordinal+1)
		}
	}
	for index := range items {
		item := &items[index]
		if !item.Outlet.Available || item.Outlet.Handle == "" {
			item.Attribution += " · " + i18n.T(i18n.KeySessionPluginsOutletUnavailable)
		}
		item.Accessible = item.Title + " · " + item.Attribution
	}
	return items
}

func renderSessionPluginOutlets(c *ui.Context, snapshot pluginui.Snapshot, show func(pluginui.Snapshot, string)) {
	valid := snapshot.Connected && snapshot.Subject != nil && snapshot.Subject.Kind == "session" && snapshot.Subject.SessionID != "" && snapshot.Subject.Epoch != "" && snapshot.Unavailable == ""
	ui.Column(c).FillWidth().MinWidth(0).Gap(4).Children(func() {
		for _, item := range sessionPluginOutletItems(snapshot.Outlets) {
			row := item.Outlet
			button := ui.Button(c.Key("session-plugin-outlet:"+row.Handle), "").Label(item.Accessible).Description(item.Attribution).Tooltip(item.Accessible).FillWidth().MinWidth(0).Padding(6, 10).Disabled(!valid || !row.Available || row.Handle == "" || show == nil)
			button.Children(func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(2).Children(func() {
					ui.Text(c, item.Title).FillWidth().MinWidth(0).NoWrap().Ellipsis("…")
					ui.Text(c, item.Attribution).FillWidth().MinWidth(0).FontSize(12)
				})
			})
			if button.Clicked() {
				show(snapshot, row.Handle)
			}
		}
	})
}

// renderPluginSessionSlot renders only an explicitly opened modal. The entry
// lives in the conversation's normal menu and actual contributions in its header.
func (s *mainState) renderPluginSessionSlot(c *ui.Context) {
	s.mu.Lock()
	p, win := s.nativePlugins.session, s.win
	if p == nil || !p.visible || !p.open || win == nil || !win.IsVisible() || win.IsMinimized() {
		s.mu.Unlock()
		return
	}
	owner, message := p.owner, p.error
	s.mu.Unlock()
	if owner != nil && owner.Inline() {
		return
	}
	w, h := c.Size()
	open := true
	ui.DialogBase(c.Key("session-plugin-local-seat"), &open, func(backdrop, panel ui.Element) {
		chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), max(260, min(float32(860), w-48)), 16).MaxHeight(max(0, h-48)).Gap(12).Children(func() {
			ui.Row(c).FillWidth().Gap(12).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyCmdSessionPluginTitle)).Grow(1).MinWidth(0)
				if ui.Button(c, i18n.T(i18n.KeyCmdClose)).Clicked() {
					open = false
				}
			})
			if message != "" {
				ui.Text(c, message).FillWidth().TextColor(c.Theme().Warning)
				return
			}
			if owner == nil {
				ui.Text(c, i18n.T(i18n.KeyCmdSessionContentUnavailable)).FillWidth()
				return
			}
			snapshot := owner.Snapshot()
			s.mu.Lock()
			pending := p.pendingTurn
			s.mu.Unlock()
			if pending != nil {
				ready, expired := pending.settled(snapshot, c.Now())
				if !ready && !expired {
					owner.Hide()
					c.After(100 * time.Millisecond)
					ui.Text(c, i18n.T(i18n.KeyCmdConnectingSessionContent))
					return
				}
				s.mu.Lock()
				if p.pendingTurn == pending {
					p.pendingTurn = nil
				}
				s.mu.Unlock()
				if expired {
					owner.Mask(true)
					owner.Show("")
					s.mu.Lock()
					p.turnExpired = true
					s.mu.Unlock()
				}
			}
			s.mu.Lock()
			expired := p.turnExpired
			s.mu.Unlock()
			if expired {
				snapshot.Surface = ""
				ui.Text(c, i18n.T(i18n.KeySessionPluginsTurnExpired)).FillWidth()
			}
			unmask := func() {
				owner.Mask(false)
				s.mu.Lock()
				p.turnExpired = false
				s.mu.Unlock()
			}
			if snapshot.Unavailable == "session-version-incompatible" {
				ui.Text(c, i18n.T(i18n.KeySessionPluginsIncompatible)).FillWidth()
				return
			}
			if !snapshot.Connected {
				ui.Text(c, i18n.T(i18n.KeyCmdConnectingSessionContent))
				return
			}
			if snapshot.Subject == nil || snapshot.Unavailable != "" || !s.pluginSessionCurrent(p) {
				label := i18n.T(i18n.KeyCmdSessionOwnerUnavailable)
				ui.Text(c, label).FillWidth()
				return
			}
			if ui.Button(c, i18n.T(i18n.KeySessionPluginsFullConversation)).Clicked() {
				if err := selectSessionPluginSurface(func() error { return owner.ShowSessionPage(snapshot.Page, snapshot.Revision, *snapshot.Subject) }, unmask); err != nil {
					s.mu.Lock()
					p.error = err.Error()
					s.mu.Unlock()
				}
			}
			if len(snapshot.Outlets) == 0 && snapshot.Surface == "" {
				ui.Text(c, i18n.T(i18n.KeyCmdNoSessionPluginContent))
			}
			if len(snapshot.Outlets) > 0 {
				ui.Scroll(c.Key("session-plugin-outlet-list")).FillWidth().Height(min(float32(132), max(48, h*.2))).MinHeight(0).Children(func() {
					renderSessionPluginOutlets(c, snapshot, func(captured pluginui.Snapshot, handle string) {
						if !s.pluginSessionCurrent(p) {
							return
						}
						if err := selectSessionPluginSurface(func() error {
							return owner.ShowSessionOutlet(captured.Page, captured.Revision, *captured.Subject, handle)
						}, unmask); err != nil {
							s.mu.Lock()
							p.error = err.Error()
							s.mu.Unlock()
						}
					})
				})
			}
			if snapshot.Surface == "" {
				return
			}
			height := min(max(float32(140), h-340), float32(560))
			ui.Box(c.Key("session-plugin-browser-seat")).FillWidth().Height(height).MinWidth(0).Role(ui.RoleGroup).Label(i18n.T(i18n.KeyCmdSessionPluginContent)).DrawOver(func(painter *ui.Painter, r ui.Rect) {
				rect := func(v ui.Rect) mygo.Rectangle {
					return mygo.Rectangle{X: int(math.Ceil(float64(v.X))), Y: int(math.Ceil(float64(v.Y))), Width: max(0, int(math.Floor(float64(v.X+v.W))-math.Ceil(float64(v.X)))), Height: max(0, int(math.Floor(float64(v.Y+v.H))-math.Ceil(float64(v.Y))))}
				}
				if s.pluginSessionCurrent(p) {
					owner.Present(rect(r), rect(painter.NativeSurfaceClip()), nil)
				} else {
					owner.Hide()
				}
			})
		})
	})
	if !open {
		s.collapsePluginSession(p)
	}
}
