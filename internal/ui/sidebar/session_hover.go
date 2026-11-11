// Session hover cards share one sidebar-local owner with workspace previews.
package sidebar

import (
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/egoist/mygo/ui"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

const (
	sessionHoverDelay         = 800 * time.Millisecond
	archivedSessionHoverDelay = 500 * time.Millisecond
	sessionHoverLeave         = 200 * time.Millisecond
	sessionHoverCopied        = time.Second
)

// One sidebar-local seat; the Model carries only a pointer to it during a frame.
// Each frame resets anchors, so borrowed Elements never survive their epoch.
// The ordinary-row warm group survives until pointer leaves sidebar/card.
// Archived-management rows are deliberately ungrouped.
type sessionHoverState struct {
	generation                             uint64
	candidate, shown, dismissed            string
	since, leaveAt, copiedUntil, sampledAt time.Time
	warm                                   bool
	dragging                               bool
	panelHeight                            float32
	anchors                                []sessionHoverAnchor
	workspace                              workspaceHoverState
	workspaceAnchors                       []workspaceHoverAnchor
	workspaceList                          ui.Element
}

type sessionHoverAnchor struct {
	row                                 ui.Element
	item                                session.Item
	displayTitle                        string
	archived, grouped, blocked, pressed bool
	tasks                               []schedstore.Entry
	statuses                            []sessionHoverStatus
}

type sessionHoverStatus struct {
	state widgets.State
	label string
}

// beginSessionHover resets frame-local anchors once on the sidebar root before
// rows register themselves. Model copies share the same retained owner.
func beginSessionHover(root ui.Element, m Model) Model {
	s := ui.Local(root, "session-hover-state", func() sessionHoverState { return sessionHoverState{} })
	if s.generation != m.Snapshot.WorkspaceLayout.HostGeneration {
		*s = sessionHoverState{generation: m.Snapshot.WorkspaceLayout.HostGeneration}
	}
	if m.Collapsed {
		s.reset()
		s.workspace.reset()
	}
	s.anchors = s.anchors[:0]
	s.workspaceAnchors = s.workspaceAnchors[:0]
	s.workspaceList = ui.Element{}
	s.dragging = false
	m.sessionHover = s
	return m
}

func (s *sessionHoverState) close() {
	s.shown, s.leaveAt, s.copiedUntil, s.sampledAt, s.panelHeight = "", time.Time{}, time.Time{}, time.Time{}, 0
}

func (s *sessionHoverState) reset() {
	s.close()
	s.candidate, s.since, s.warm = "", time.Time{}, false
}

func (s *sessionHoverState) dismiss(key string) {
	s.close()
	s.candidate, s.since, s.dismissed = "", time.Time{}, key
}

func (a sessionHoverAnchor) key() string {
	if a.grouped {
		return "ordinary:" + a.item.SessionID
	}
	return "archived:" + a.item.SessionID
}

func sessionHoverCopyTitle(item session.Item) string {
	if item.Blank {
		return ""
	}
	return item.Title // Exact original value, including whitespace; never fallback label.
}

// sessionHoverCard registers a row; the root renders once after all anchors
// are known. Optional press values come only from existing click owners; pure
// layout/text must not be made clickable merely to observe a preview dismissal.
func sessionHoverCard(c *ui.Context, m Model, anchor ui.Element, item session.Item, label, _ string, _ theme.Colors, anchorPressed ...bool) {
	registerSessionHover(m, anchor, item, label, false, true, anchorPressed...)
}

// Archived-management branch uses the source's default 500ms, ungrouped gate.
// Ordinary rows that happen to be archived remain grouped800; derive archived
// content from actual archived IDs rather than using this distinction as data.
func archivedSessionHoverCard(c *ui.Context, m Model, anchor ui.Element, item session.Item, label string, anchorPressed ...bool) {
	registerSessionHover(m, anchor, item, label, true, false, anchorPressed...)
}

func registerSessionHover(m Model, anchor ui.Element, item session.Item, label string, archived, grouped bool, anchorPressed ...bool) {
	if m.sessionHover == nil {
		return
	}
	archived = archived || slices.Contains(m.Snapshot.ArchivedSessionIDs, item.SessionID)
	pressed := anchor.Pressed()
	for _, value := range anchorPressed {
		pressed = pressed || value
	}
	m.sessionHover.anchors = append(m.sessionHover.anchors, sessionHoverAnchor{
		row: anchor, item: item, displayTitle: label, archived: archived, grouped: grouped,
		blocked: m.Batch.Enabled || m.Batch.Busy || m.Batch.Confirming || anchor.MenuOpen(), pressed: pressed,
		tasks: schedstore.ActiveForSession(m.Schedules, item.SessionID), statuses: sessionHoverStatuses(m, item, archived),
	})
}

// Workspace anchors register beside session anchors for the root's one family
// decision. Standalone workspace-only callers retain their local owner.
func registerWorkspaceHover(c *ui.Context, m Model, list ui.Element, state *workspaceHoverState, anchors []workspaceHoverAnchor) {
	if m.sessionHover == nil {
		workspaceHoverCard(c, m, list, state, anchors)
		return
	}
	m.sessionHover.workspaceList = list
	m.sessionHover.workspaceAnchors = append(m.sessionHover.workspaceAnchors, anchors...)
}

// Both families are registered before this one decision, so the losing card is
// closed before any overlay is emitted. Warmth passes only across grouped rows.
func renderSessionHover(c *ui.Context, m Model, root ui.Element, suppressSessionHoverForDrag bool) {
	s := m.sessionHover
	if s == nil {
		return
	}
	defer func() {
		clear(s.workspaceAnchors)
		s.workspaceAnchors = s.workspaceAnchors[:0]
		s.workspaceList = ui.Element{}
	}()
	if m.Collapsed || m.Batch.Enabled || m.Batch.Busy || m.Batch.Confirming || root.MenuOpen() || suppressSessionHoverForDrag {
		s.reset()
		s.workspace.reset()
		clear(s.anchors)
		s.anchors = s.anchors[:0]
		return
	}
	warm := s.warm || s.workspace.warm
	workspaceHovered, sessionHovered, ordinaryHovered := false, false, false
	for _, a := range s.workspaceAnchors {
		workspaceHovered = workspaceHovered || a.row.Hovered()
	}
	for _, a := range s.anchors {
		if a.row.Hovered() && !a.blocked {
			sessionHovered = true
			ordinaryHovered = ordinaryHovered || a.grouped
		}
	}
	// A cold dwell ends as soon as its anchor is left, even when no card
	// has opened and the dispatcher skips that family's renderer entirely.
	// Keep displayed cards and group warmth for their existing leave grace.
	if !sessionHovered && s.shown == "" {
		s.candidate, s.since = "", time.Time{}
	}
	if !workspaceHovered && s.workspace.shown == "" {
		s.workspace.candidate, s.workspace.since = "", time.Time{}
	}
	if workspaceHovered || (!sessionHovered && s.workspace.shown != "") {
		s.close()
		s.candidate = ""
		s.warm = warm
		s.workspace.warm = warm
		workspaceHoverCard(c, m, s.workspaceList, &s.workspace, s.workspaceAnchors)
		s.warm = s.workspace.warm
		clear(s.anchors)
		s.anchors = s.anchors[:0]
		return
	}
	if sessionHovered || s.shown != "" {
		s.workspace.close()
		s.workspace.candidate = ""
		s.workspace.since = time.Time{}
		if ordinaryHovered {
			s.warm = warm
		}
		renderSessionHoverCard(c, m, root, suppressSessionHoverForDrag)
		s.workspace.warm = s.warm
		return
	}
	clear(s.anchors)
	s.anchors = s.anchors[:0]
	if !root.Hovered() {
		s.warm = false
		s.workspace.warm = false
	}
}

func renderSessionHoverCard(c *ui.Context, m Model, root ui.Element, suppressSessionHoverForDrag bool) {
	s := m.sessionHover
	if s == nil {
		return
	}
	defer func() { clear(s.anchors); s.anchors = s.anchors[:0] }()
	if m.Collapsed || m.Batch.Enabled || m.Batch.Busy || m.Batch.Confirming || root.MenuOpen() || suppressSessionHoverForDrag {
		s.reset()
		return
	}
	var hovered, shown *sessionHoverAnchor
	for i := range s.anchors {
		a := &s.anchors[i]
		if a.pressed {
			s.dismiss(a.key())
			return
		}
		if a.blocked {
			continue
		}
		if a.row.Hovered() {
			hovered = a
		}
		if a.key() == s.shown {
			shown = a
		}
	}
	if hovered == nil || hovered.key() != s.dismissed {
		s.dismissed = ""
	}
	now := c.Now()
	if hovered != nil && hovered.key() != s.dismissed {
		key := hovered.key()
		if s.candidate != key {
			s.candidate, s.since = key, now
		}
		delay := archivedSessionHoverDelay
		if hovered.grouped {
			delay = sessionHoverDelay
		}
		if s.shown == key {
			s.leaveAt = time.Time{}
		} else if (hovered.grouped && s.warm) || now.Sub(s.since) >= delay {
			s.shown, s.leaveAt, s.copiedUntil, s.sampledAt, s.panelHeight = key, time.Time{}, time.Time{}, now, 0
			shown = hovered
			if hovered.grouped {
				s.warm = true
			}
		} else {
			c.After(delay - now.Sub(s.since))
		}
	}
	if s.shown == "" {
		if hovered == nil {
			s.candidate = ""
			s.since = time.Time{}
		}
		if !root.Hovered() {
			s.warm = false
		}
		return
	}
	if shown == nil {
		s.reset()
		return
	} // Removed/filtered/folded row cannot leave a detached card.
	a := *shown
	ui.Overlay(c.Key("session-hover-overlay:"+s.shown), func() {
		width, height := m.viewport(c)
		// Keep the existing observer identity on a plain wrapper. The whole
		// child is the one actual copy button when an original title exists.
		box := ui.Box(c.Key("session-preview:"+s.shown)).Label("Session preview: "+a.item.SessionID).
			AttachTo(a.row, ui.AnchorTopRight, ui.AnchorTopLeft).Margin(0, 0, 0, 8).
			Width(min(float32(244), max(float32(0), width-16))).MaxHeight(max(float32(0), height-16))
		copyTitle := sessionHoverCopyTitle(a.item)
		copying := now.Before(s.copiedUntil)
		var panel ui.Element
		box.Children(func() {
			if copyTitle != "" {
				panel = ui.ButtonBase(c.Key("session-preview-copy:" + s.shown)).Selectable().Cursor(ui.CursorPointer).Label("Copy session title " + a.item.SessionID).
					Description(workspaceText("复制：", "Copy: ") + copyTitle)
			} else {
				panel = ui.Box(c.Key("session-preview-body:" + s.shown))
			}
			panel.Column().FillWidth().MinWidth(0).Padding(12, 16).Gap(8).Radius(12).Clip().
				AlignItems(ui.Start).Justify(ui.Start).Background(ui.RGB(44, 44, 46)).TextColor(ui.RGB(255, 255, 255)).
				BorderWidth(0).Shadow(0, 0, 1, 0, ui.RGBA(0, 0, 0, .2)).Shadow(0, 0, 4, 0, ui.RGBA(0, 0, 0, .02)).Shadow(0, 12, 32, 0, ui.RGBA(0, 0, 0, .08))
			if copying {
				panel.MinHeight(s.panelHeight).Justify(ui.Center).AlignItems(ui.Center)
				c.After(s.copiedUntil.Sub(now))
			} else {
				panel.ObserveLayout(func(r ui.Rect) { s.panelHeight = r.H })
			}
			panel.Children(func() {
				if copying {
					ui.Text(c, workspaceText("已复制", "Copied")).FontSize(14).FixedLineHeight(20).TextColor(ui.RGB(255, 255, 255))
					return
				}
				publicSidebarRowContext(m, a.item.SessionID, false, true)
				drawSessionHoverBodyWithPublic(c, m, a, s.sampledAt)
			})
		})
		if panel.OverlayShortcut(0, ui.KeyEscape) || panel.PressedOutside() {
			s.dismiss(a.key())
			box.Invisible()
			panel.Invisible()
			c.After(0)
			return
		}
		// Scoped keys also activate while selectable paragraphs hold focus.
		// Pointer activation is suppressed by the selection scope when dragging
		// produces a range; keyboard activation deliberately ignores that range.
		clicked := panel.Clicked()
		enter := panel.Shortcut(0, ui.KeyEnter)
		space := panel.Shortcut(0, ui.KeySpace)
		if copyTitle != "" && (clicked || enter || space) && !copying {
			c.WriteClipboard(copyTitle)
			s.copiedUntil = now.Add(sessionHoverCopied)
			c.After(0)
			c.After(sessionHoverCopied)
		}
		if a.row.Hovered() || panel.Hovered() {
			s.leaveAt = time.Time{}
			if a.grouped {
				s.warm = true
			}
			return
		}
		if s.leaveAt.IsZero() {
			s.leaveAt = now
		}
		if wait := sessionHoverLeave - now.Sub(s.leaveAt); wait > 0 {
			c.After(wait)
		} else {
			s.close()
			if !root.Hovered() {
				s.warm = false
			}
			box.Invisible()
			panel.Invisible()
			c.After(0)
		}
	})
}

func sessionHoverAgo(updatedAt float64, now time.Time) string {
	if updatedAt <= 0 || math.IsNaN(updatedAt) || math.IsInf(updatedAt, 0) {
		return ""
	}
	label := sessionRelativeTime(updatedAt, float64(now.UnixMilli()))
	if label == workspaceText("刚刚", "now") {
		return label
	}
	return fmt.Sprintf(workspaceText("%s前", "%s ago"), label)
}

func sessionHoverStatuses(m Model, item session.Item, archived bool) []sessionHoverStatus {
	var out []sessionHoverStatus
	children := runningSubagentCount(m, item.SessionID)
	childPrimary := false
	switch status := sessionStatus(m, item); status {
	case "approval", "plan-review", "question":
		out = append(out, sessionHoverStatus{widgets.StateWarning, statusLabel(status)})
	case "running":
		label := workspaceText("运行中", "Running")
		if !item.Running && children > 0 {
			label = fmt.Sprintf(workspaceText("%d 个子代理运行中", "%d running subagents"), children)
			childPrimary = true
		}
		out = append(out, sessionHoverStatus{widgets.StateRunning, label})
	case "completed":
		if !archived {
			out = append(out, sessionHoverStatus{widgets.StateDone, statusLabel(status)})
		}
	default:
		if !archived {
			out = append(out, sessionHoverStatus{widgets.StateIdle, workspaceText("空闲", "Idle")})
		}
	}
	// Counts come only from the parent's owned catalog/status projection.
	// Unknown activity must never be displayed as an exact child count.
	if children > 0 && !childPrimary {
		out = append(out, sessionHoverStatus{widgets.StateRunning,
			fmt.Sprintf(workspaceText("%d 个子代理运行中", "%d running subagents"), children)})
	}
	return out
}

// Exact IconClockOutlineRegular (ub -> cb, strokeWidth 1) from the installed
// client primitive bundle; parsed once, without executing bundled JavaScript.
var sessionHoverTaskClockArtwork = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke-width="1" xmlns="http://www.w3.org/2000/svg"><path d="M8 14C11.3137 14 14 11.3137 14 8C14 4.68629 11.3137 2 8 2C4.68629 2 2 4.68629 2 8C2 11.3137 4.68629 14 8 14Z" stroke="currentColor"/><path d="M8 4.31V8.46L11 10.08" stroke="currentColor"/></svg>`))

func drawSessionHoverBody(c *ui.Context, a sessionHoverAnchor, sampledAt time.Time) {
	drawSessionHoverBodyWithPublic(c, Model{}, a, sampledAt)
}

func drawSessionHoverBodyWithPublic(c *ui.Context, m Model, a sessionHoverAnchor, sampledAt time.Time) {
	white := ui.RGB(255, 255, 255)
	secondary := ui.RGBA(255, 255, 255, .65)
	ui.Text(c, a.displayTitle).FillWidth().MinWidth(0).FontSize(14).FixedLineHeight(20).TextColor(white)
	if !a.item.Blank {
		if ago := sessionHoverAgo(a.item.UpdatedAt, sampledAt); ago != "" {
			ui.Text(c, ago).FillWidth().MinWidth(0).FontSize(12).FixedLineHeight(16).TextColor(secondary)
		}
	}
	keepSchedule := publicSidebarBefore(c, m, "sidebar.session.row.hover", a.item.SessionID, "schedule-tasks")
	ordered := orderSchedulePreview(a.tasks, sampledAt)
	if keepSchedule && len(ordered) > 0 {
		timingInk, relativeInk := ui.RGB(207, 211, 214), ui.RGB(173, 178, 184)
		ui.Column(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
			for _, task := range ordered[:min(2, len(ordered))] {
				ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Start).Children(func() {
					ui.Box(c).Size(12, 16).Shrink(0).Center().Role(ui.RoleNone).Children(func() {
						ui.Icon(c, sessionHoverTaskClockArtwork).Size(12, 12).TextColor(timingInk).Role(ui.RoleNone)
					})
					ui.Column(c).MinWidth(0).Grow(1).Gap(2).Children(func() {
						ui.Text(c, task.Title).FillWidth().MinWidth(0).FontSize(12).FixedLineHeight(16).TextColor(white)
						absolute, relative := schedulePreviewParts(task, sampledAt)
						ui.RichText(c, ui.Span{Text: scheduleHoverFrequency(task, sampledAt) + " · " + absolute + " ", Color: timingInk},
							ui.Span{Text: relative, Color: relativeInk}).FillWidth().MinWidth(0).FontSize(12).FixedLineHeight(16).TextColor(timingInk)
					})
				})
			}
			if len(ordered) > 2 {
				ui.Text(c, fmt.Sprintf(workspaceText("另有 %d 个任务", "%d more"), len(ordered)-2)).FillWidth().FontSize(12).FixedLineHeight(16).TextColor(relativeInk)
			}
		})
	}
	publicSidebarBefore(c, m, "sidebar.session.row.hover", a.item.SessionID, "")
	for _, status := range a.statuses {
		ui.Row(c).FillWidth().MinWidth(0).Gap(6).AlignItems(ui.Center).Children(func() {
			widgets.StateDot(c, status.state)
			ui.Text(c, status.label).MinWidth(0).Grow(1).FontSize(12).FixedLineHeight(16).TextColor(secondary)
		})
	}
	if a.archived {
		ui.Row(c).FillWidth().MinWidth(0).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, menuArchiveArtwork).Size(14, 14).TextColor(secondary)
			ui.Text(c, workspaceText("已归档", "Archived")).MinWidth(0).Grow(1).FontSize(12).FixedLineHeight(16).TextColor(secondary)
		})
	}
}
