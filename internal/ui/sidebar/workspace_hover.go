package sidebar

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

const (
	workspaceHoverDelay   = 800 * time.Millisecond
	workspaceHoverLeave   = 200 * time.Millisecond
	workspaceCopyFeedback = time.Second
)

// One browser-local seat owns the workspace hover group. Like the installed
// HoverCardGroup, it is warm while in the list or its still-mounted card. There
// is no second, arbitrary warm timeout. A Host change retires the entire seat.
type workspaceHoverState struct {
	HostGeneration              uint64
	candidate, shown, dismissed string
	since, leaveAt, copiedUntil time.Time
	warm                        bool
	panelHeight                 float32
}

type workspaceHoverAnchor struct {
	row       ui.Element
	workspace session.WorkspaceRow
	label     string
	pressed   bool
}

func workspaceHoverGroup(c *ui.Context, list ui.Element, m Model) *workspaceHoverState {
	if m.sessionHover != nil {
		return &m.sessionHover.workspace
	}
	s := ui.Local(list, "workspace-hover-group", func() workspaceHoverState { return workspaceHoverState{} })
	if s.HostGeneration != m.Snapshot.WorkspaceLayout.HostGeneration {
		*s = workspaceHoverState{HostGeneration: m.Snapshot.WorkspaceLayout.HostGeneration}
	}
	return s
}

// Workspace.createdAt follows the app's dictionary, with a local clock. It must
// not follow the operating-system language after the app locale changes.
func workspaceCreatedLabel(raw string) string {
	created, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return ""
	}
	created = created.In(time.Local)
	clock := created.Format("15:04")
	if i18n.Locale() == i18n.LocaleZH {
		return fmt.Sprintf("创建于 %d年%d月%d日 %s", created.Year(), created.Month(), created.Day(), clock)
	}
	return fmt.Sprintf("Created %d-%d-%d %s", created.Year(), created.Month(), created.Day(), clock)
}

func (s *workspaceHoverState) close() {
	s.shown, s.leaveAt, s.copiedUntil, s.panelHeight = "", time.Time{}, time.Time{}, 0
}

func (s *workspaceHoverState) reset() {
	s.close()
	s.candidate, s.since, s.warm = "", time.Time{}, false
}

func (s *workspaceHoverState) dismiss(id string) {
	s.close()
	s.candidate, s.since, s.dismissed = "", time.Time{}, id
}

// workspaceHoverCard uses an interactive overlay, not TooltipBase: a tooltip
// passes events through and cannot own the installed card's copy interaction.
// The complete compact card is the copy button, including Enter and Space;
// successful copying replaces its contents for one second without resizing it.
func workspaceHoverCard(c *ui.Context, m Model, list ui.Element, state *workspaceHoverState, anchors []workspaceHoverAnchor) {
	if list.MenuOpen() {
		state.reset()
		return
	}
	var hovered, shown *workspaceHoverAnchor
	for index := range anchors {
		a := &anchors[index]
		if a.pressed {
			state.dismiss(a.workspace.WorkspaceID)
			return
		}
		if a.row.Hovered() {
			hovered = a
		}
		if a.workspace.WorkspaceID == state.shown {
			shown = a
		}
	}
	if hovered == nil || hovered.workspace.WorkspaceID != state.dismissed {
		state.dismissed = ""
	}
	now := c.Now()
	if hovered != nil && hovered.workspace.WorkspaceID != state.dismissed {
		id := hovered.workspace.WorkspaceID
		if state.candidate != id {
			state.candidate, state.since = id, now
		}
		if state.shown == id {
			state.leaveAt = time.Time{}
		} else if state.warm || now.Sub(state.since) >= workspaceHoverDelay {
			state.shown, state.leaveAt, state.copiedUntil, state.panelHeight = id, time.Time{}, time.Time{}, 0
			shown, state.warm = hovered, true
		} else {
			c.After(workspaceHoverDelay - now.Sub(state.since))
		}
	} else {
		state.candidate, state.since = "", time.Time{}
	}
	if state.shown == "" {
		if !list.Hovered() {
			state.warm = false
		}
		return
	}
	if shown == nil {
		// A folded ancestor, filter, removed registry row or Host replacement
		// cannot leave a detached card alive.
		state.reset()
		return
	}
	ui.Overlay(c.Key("workspace-hover-overlay:"+state.shown), func() {
		width, height := m.viewport(c)
		// The installed compact variant sits 8 DIP right of the header, at
		// its top; the native attachment flips it to keep it in the window.
		panel := ui.ButtonBase(c.Key("workspace-hover-panel:"+state.shown)).Selectable().Column().
			Label("Copy workspace path "+state.shown).Description(workspaceText("复制：", "Copy: ")+shown.workspace.Path).
			AttachTo(shown.row, ui.AnchorTopRight, ui.AnchorTopLeft).Margin(0, 0, 0, 8).
			Width(min(float32(244), max(float32(0), width-16))).MaxHeight(max(float32(0), height-16)).
			AlignItems(ui.Start).Justify(ui.Start).Padding(12, 16).Gap(8).Radius(12).Clip().
			Background(ui.RGB(44, 44, 46)).TextColor(ui.RGB(255, 255, 255)).
			BorderWidth(0).Shadow(0, 0, 1, 0, ui.RGBA(0, 0, 0, .2)).
			Shadow(0, 0, 4, 0, ui.RGBA(0, 0, 0, .02)).Shadow(0, 12, 32, 0, ui.RGBA(0, 0, 0, .08))
		copying := now.Before(state.copiedUntil)
		if copying {
			panel.MinHeight(state.panelHeight).Justify(ui.Center).AlignItems(ui.Center)
			c.After(state.copiedUntil.Sub(now))
		} else {
			panel.ObserveLayout(func(r ui.Rect) { state.panelHeight = r.H })
		}
		panel.Children(func() {
			if copying {
				ui.Text(c, workspaceText("已复制", "Copied")).FontSize(14).FixedLineHeight(20).TextColor(ui.RGB(255, 255, 255))
				return
			}
			ui.Text(c, shown.label).Label("Workspace preview: " + state.shown).FillWidth().MinWidth(0).FontSize(14).FixedLineHeight(20).TextColor(ui.RGB(255, 255, 255))
			if shown.workspace.Path != "" {
				ui.Text(c, shown.workspace.Path).FillWidth().MinWidth(0).FontSize(12).FixedLineHeight(16).TextColor(ui.RGB(207, 211, 214))
			}
			if created := workspaceCreatedLabel(shown.workspace.CreatedAt); created != "" {
				ui.Text(c, created).FillWidth().MinWidth(0).FontSize(12).FixedLineHeight(16).TextColor(ui.RGB(207, 211, 214))
			}
		})
		if panel.OverlayShortcut(0, ui.KeyEscape) || panel.PressedOutside() {
			state.dismiss(state.shown)
			panel.Invisible()
			c.After(0)
			return
		}
		clicked := panel.Clicked()
		enter := panel.Shortcut(0, ui.KeyEnter)
		space := panel.Shortcut(0, ui.KeySpace)
		if (clicked || enter || space) && !copying && shown.workspace.Path != "" {
			// This is a synchronous clipboard operation during the UI frame;
			// no Context or borrowed Element escapes into an async callback.
			c.WriteClipboard(shown.workspace.Path)
			state.copiedUntil = now.Add(workspaceCopyFeedback)
			c.After(0)
			c.After(workspaceCopyFeedback)
		}
		if shown.row.Hovered() || panel.Hovered() {
			state.leaveAt, state.warm = time.Time{}, true
			return
		}
		// The installed 200ms close timer permits a pointer to cross the gap
		// into the interactive card. Entering another row replaces this card.
		if state.leaveAt.IsZero() {
			state.leaveAt = now
		}
		if wait := workspaceHoverLeave - now.Sub(state.leaveAt); wait > 0 {
			c.After(wait)
		} else {
			state.close()
			if !list.Hovered() {
				state.warm = false
			}
			panel.Invisible()
			c.After(0)
		}
	})
}

// ProjectRowItem highlights its own current group only while expanded; every
// registered ancestor stays active, even while its subtree is collapsed.
func workspaceGroupHighlights(m Model, groups []session.WorkspaceGroup) map[string]bool {
	active, parents := map[string]bool{}, map[string]string{}
	current := ""
	for _, group := range groups {
		parents[group.Key] = group.ParentID
		if current != "" {
			continue
		}
		for _, id := range group.Workspace.SessionIDs {
			if id != "" && id == m.Snapshot.SelectedID {
				current = group.Key
				break
			}
		}
		if current == "" {
			for _, item := range group.Items {
				if item.SessionID != "" && item.SessionID == m.Snapshot.SelectedID {
					current = group.Key
					break
				}
			}
		}
	}
	if current == "" {
		return active
	}
	active[current] = current == "::scratch" || !session.WorkspaceGroupCollapsed(m.Snapshot, current)
	visited := map[string]bool{current: true}
	for parent := parents[current]; parent != "" && !visited[parent]; parent = parents[parent] {
		active[parent], visited[parent] = true, true
	}
	return active
}

// Hidden menu controls remain keyboard-addressable but take no width from the
// resting title. The visible action group is the installed 16px/gap15 layout.
func workspaceRowActions(c *ui.Context, m Model, header ui.Element, g session.WorkspaceGroup, muted ui.Color) {
	shown := header.Hovered() || header.FocusWithin() || header.MenuOpen()
	actions := ui.Row(c.Key("workspace-actions:" + g.Key)).Height(20).Shrink(0).AlignItems(ui.Center).Gap(15)
	if !shown {
		actions.Absolute().Right(8).Top(7).Opacity(0)
	}
	actions.Children(func() {
		if g.Kind == "workspace" && (m.OnBeginWorkspaceRename != nil || m.OnBeginWorkspaceRemoval != nil) {
			more := ui.Button(c.Key("workspace-menu:"+g.Key), "").Label(workspaceText("工作区操作 ", "Workspace actions ")+g.Key).
				Size(16, 16).Padding(0).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Center()
			more.Children(func() { ui.Icon(c, menuEllipsisArtwork).Size(16, 16).TextColor(muted) })
			more.Menu(func(menu *ui.Menu) {
				menu.CloseOnPointerLeave(true)
				if m.OnBeginWorkspaceRename != nil && menu.Item(workspaceText("重命名", "Rename")).Icon(menuEditArtwork).Chosen() {
					m.OnBeginWorkspaceRename(g.Key)
				}
				if m.OnBeginWorkspaceRemoval != nil && menu.Item(workspaceText("删除工作区", "Delete workspace")).Icon(menuTrashArtwork).Danger(true).Chosen() {
					m.OnBeginWorkspaceRemoval(g.Key)
				}
			})
		}
		var label string
		var run func()
		if g.Kind == "workspace" && m.OnCreateWorkspaceSession != nil {
			label = workspaceText("新建会话 ", "New session ") + g.Key
			run = func() { m.OnCreateWorkspaceSession(m.Snapshot.WorkspaceLayout.HostGeneration, g.Key) }
		} else if g.Kind == "scratch" && (m.OnCreateWorkspaceSession != nil || m.OnNoDirectory != nil) {
			label = workspaceText("无目录会话", "No directory session")
			run = func() {
				if m.OnCreateWorkspaceSession != nil {
					m.OnCreateWorkspaceSession(m.Snapshot.WorkspaceLayout.HostGeneration, g.Key)
				} else {
					if session.WorkspaceGroupCollapsed(m.Snapshot, g.Key) && m.OnToggleWorkspaceGroup != nil {
						m.OnToggleWorkspaceGroup(g.Key)
					}
					m.OnNoDirectory()
				}
			}
		}
		if run != nil {
			create := ui.Button(c.Key("workspace-create:"+g.Key), "").Label(label).Tooltip(label).Size(16, 16).Padding(0).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Center()
			art := menuNewChatArtwork
			if g.Kind == "scratch" {
				art = workspaceScratchCreateArtwork
			}
			create.Children(func() { ui.Icon(c, art).Size(16, 16).TextColor(muted) })
			if create.Clicked() {
				run()
			}
		}
	})
}

var workspaceScratchCreateArtwork = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor"><path d="M8 2v12M2 8h12"/></svg>`))
