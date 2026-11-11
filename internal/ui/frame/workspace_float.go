package frame

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"sort"
	termui "whaleisle/internal/ui/terminal"
	"whaleisle/internal/ui/theme"
)

const workspaceFloatTop float32 = 48

// Float is an in-app presentation of the same resource, never an OS window.
// The owning dock retains the resource in its stable registry so existing open,
// async-read, save, terminal and browser callbacks keep their original identity.
// dockedLeaf excludes floated resources from dock layout and tab strips.
type WorkspaceFloat struct {
	X         float32 `json:"x"`
	Y         float32 `json:"y"`
	Width     float32 `json:"width"`
	Height    float32 `json:"height"`
	Minimized bool    `json:"minimized,omitempty"`
	Order     int     `json:"order,omitempty"`
}

// boundedFloat is presentation-only: a smaller desktop does not overwrite the
// user's preferred frame, so returning to a larger viewport restores it.
func boundedFloat(f WorkspaceFloat, width, height float32) WorkspaceFloat {
	f.Width = min(max(float32(220), f.Width), max(float32(0), width))
	f.Height = max(float32(140), f.Height)
	if f.Minimized {
		f.Height = 38
	}
	f.Height = min(f.Height, max(float32(0), height-workspaceFloatTop))
	f.X = max(float32(0), min(f.X, width-f.Width))
	f.Y = max(workspaceFloatTop, min(f.Y, height-f.Height))
	return f
}

func (n *WorkspacePane) dockedLeaf() *WorkspacePane {
	if n == nil {
		return nil
	}
	if n.First != nil {
		if p := n.First.dockedLeaf(); p != nil {
			return p
		}
		return n.Second.dockedLeaf()
	}
	for _, tab := range append([]*WorkspacePane{n}, n.DockTabs...) {
		if tab.Float == nil {
			return tab
		}
	}
	return nil
}

func (w *WorkspacePanes) raiseFloat(n *WorkspacePane) {
	if n == nil || n.Float == nil {
		return
	}
	highest := 0
	w.Root.eachResource(func(p *WorkspacePane) {
		if p.Float != nil {
			highest = max(highest, p.Float.Order)
		}
	})
	if w.Active == n.ID && n.Float.Order == highest {
		return
	}
	n.Float.Order = highest + 1
	w.Active, w.route = n.ID, n.Kind
}

func (w *WorkspacePanes) FloatWorkspaceTab(id string) bool {
	n := w.Root.find(id)
	if n == nil || n.First != nil {
		return false
	}
	if n.Float == nil {
		n.Float = &WorkspaceFloat{X: 80, Y: 80, Width: 380, Height: 300}
	}
	n.Float.Minimized = false
	if group := w.Root.dockGroup(id); group == n {
		if docked := group.dockedLeaf(); docked != nil {
			w.activateDockTab(docked.ID)
		}
	}
	w.raiseFloat(n)
	return true
}
func (w *WorkspacePanes) MoveWorkspaceFloat(id string, x, y float32) {
	n := w.Root.find(id)
	if n == nil || n.Float == nil {
		return
	}
	n.Float.X, n.Float.Y = max(float32(0), x), max(workspaceFloatTop, y)
	w.raiseFloat(n)
}
func (w *WorkspacePanes) ResizeWorkspaceFloat(id string, width, height float32) {
	n := w.Root.find(id)
	if n == nil || n.Float == nil {
		return
	}
	n.Float.Width, n.Float.Height = max(float32(220), width), max(float32(140), height)
	w.raiseFloat(n)
}
func (w *WorkspacePanes) MinimizeWorkspaceFloat(id string, minimized bool) {
	n := w.Root.find(id)
	if n == nil || n.Float == nil {
		return
	}
	n.Float.Minimized = minimized
	if !minimized {
		w.raiseFloat(n)
	}
}
func (w *WorkspacePanes) RedockWorkspaceTab(id, destination string) bool {
	n := w.Root.find(id)
	if n == nil || n.Float == nil || n.Kind == "project-observer" {
		return false
	}
	target := w.Root.find(destination)
	if target == nil || target.Float != nil {
		target = w.Root.dockedLeaf()
	}
	n.Float = nil
	if target != nil && target != n {
		group := w.Root.dockGroup(target.ID)
		return w.MoveWorkspaceTab(id, target.ID, len(group.dockTabs()))
	}
	w.activateDockTab(id)
	return true
}

func workspaceFloating(c *ui.Context, m Model) {
	w := m.Sizes.Workspace
	viewportWidth, viewportHeight := viewportDIP(c)
	viewportWidth = max(float32(0), viewportWidth-m.RightOverlayReserve)
	var floats []*WorkspacePane
	w.Root.eachResource(func(n *WorkspacePane) {
		if n.Float != nil && (n.Kind != "project-observer" || projectObserverVisible(m, n)) {
			floats = append(floats, n)
		}
	})
	sort.SliceStable(floats, func(i, j int) bool { return floats[i].Float.Order < floats[j].Float.Order })
	defer func() {
		w.Root.eachResource(func(n *WorkspacePane) {
			if n.browser == nil {
				return
			}
			var holes []mygo.Rectangle
			w.Root.eachResource(func(other *WorkspacePane) {
				if n.Float == nil && other.browser != nil {
					if r, ok := other.browser.MiniPlayerBounds(); ok {
						holes = append(holes, r)
					}
				}
			})
			for _, above := range floats {
				if n == above || n.Float != nil && above.Float.Order <= n.Float.Order {
					continue
				}
				f := boundedFloat(*above.Float, viewportWidth, viewportHeight)
				holes = append(holes, mygo.Rectangle{X: int(f.X), Y: int(f.Y), Width: int(f.Width), Height: int(f.Height)})
			}
			n.browser.SetWorkspaceOcclusions(holes)
			// Mini-players sit below resource floats, including their own source.
			miniHoles := miniPlayerFloatOcclusions(floats, viewportWidth, viewportHeight)
			n.browser.SetMiniPlayerOcclusions(miniHoles)
		})
	}()
	for _, n := range floats {
		n := n
		f := n.Float
		ui.Overlay(c, func() {
			visible := boundedFloat(*f, viewportWidth, viewportHeight)
			panel := ui.Column(c).Key("workspace-float:"+n.ID).Absolute().Left(visible.X).Top(visible.Y).Width(visible.Width).Height(visible.Height).Background(c.Theme().Background).Border(1, c.Theme().Border).Clip().FocusBind(&w.focusedResource, n.ID)
			panel.Children(func() {
				ui.Row(c).FillWidth().Height(38).Gap(4).Padding(4).Children(func() {
					terminal, sid, title, editing := workspaceTerminalTitle(m, n)
					grip := ui.Box(c).Key("float-move").Grow(1).Height(28).Cursor(ui.CursorMove).Focusable().Label(paneLabel("移动浮动面板", "Move floating pane"))
					grip.Children(func() {
						if !editing || !termui.HostTitleEditor(c, terminal, sid) {
							ui.Text(c, title).FontSize(theme.InterfaceSize(11))
						}
					})
					if !editing {
						dx, dy, held := grip.Dragged()
						if held {
							w.MoveWorkspaceFloat(n.ID, visible.X+dx, visible.Y+dy)
						}
						if grip.DoubleClicked() && n.Kind == "host-terminal" {
							beginWorkspaceTerminalRename(m, n)
						} else if grip.Clicked() {
							w.raiseFloat(n)
						}
						for _, key := range []ui.Key{ui.KeyLeft, ui.KeyRight, ui.KeyUp, ui.KeyDown} {
							if grip.Shortcut(0, key) {
								x, y := f.X, f.Y
								switch key {
								case ui.KeyLeft:
									x -= 10
								case ui.KeyRight:
									x += 10
								case ui.KeyUp:
									y -= 10
								case ui.KeyDown:
									y += 10
								}
								w.MoveWorkspaceFloat(n.ID, x, y)
							}
						}
					}
					ui.Button(c, "⠿").Key("float-dock-drag").Label(paneLabel("拖到停靠落点", "Drag to dock target")).Drag(workspaceTabDrag{ID: n.ID})
					label := "−"
					if f.Minimized {
						label = "+"
					}
					if ui.Button(c, label).Label(paneLabel("最小化 / 恢复", "Minimize / restore")).Clicked() {
						w.MinimizeWorkspaceFloat(n.ID, !f.Minimized)
					}
					if n.Kind != "project-observer" && ui.Button(c, "↙").Label(paneLabel("停靠面板", "Redock pane")).Clicked() {
						w.pending = func() { w.RedockWorkspaceTab(n.ID, "") }
					}
					if n.Kind != "project-observer" && ui.Button(c, "×").Label(paneLabel("关闭标签", "Close tab")).Clicked() {
						w.pending = func() { w.close(n.ID) }
					}
				})
				if !f.Minimized {
					ui.Box(c).Grow(1).FillWidth().MinHeight(0).Children(func() { workspaceNode(c, m, n) })
					grip := ui.Box(c).Key("float-resize").FillWidth().Height(12).Cursor(ui.CursorResizeNWSE).Focusable().Label(paneLabel("调整浮动面板大小", "Resize floating pane"))
					dx, dy, held := grip.Dragged()
					if held {
						w.ResizeWorkspaceFloat(n.ID, visible.Width+dx, visible.Height+dy)
					}
					if grip.Shortcut(0, ui.KeyRight) {
						w.ResizeWorkspaceFloat(n.ID, f.Width+10, f.Height)
					}
					if grip.Shortcut(0, ui.KeyLeft) {
						w.ResizeWorkspaceFloat(n.ID, f.Width-10, f.Height)
					}
					if grip.Shortcut(0, ui.KeyDown) {
						w.ResizeWorkspaceFloat(n.ID, f.Width, f.Height+10)
					}
					if grip.Shortcut(0, ui.KeyUp) {
						w.ResizeWorkspaceFloat(n.ID, f.Width, f.Height-10)
					}
				}
			})
			if panel.FocusWithin() {
				w.raiseFloat(n)
			}
		})
	}
}

func workspaceDockNode(c *ui.Context, m Model, n *WorkspacePane) {
	if n == nil || n.dockedLeaf() == nil {
		return
	}
	if n.First == nil {
		workspaceNode(c, m, n.dockedLeaf())
		return
	}
	first, second := n.First.dockedLeaf(), n.Second.dockedLeaf()
	if first == nil {
		workspaceDockNode(c, m, n.Second)
		return
	}
	if second == nil {
		workspaceDockNode(c, m, n.First)
		return
	}
	box := ui.Box(c).Key(n.ID).Fill().MinWidth(0).MinHeight(0).Clip()
	box.Children(func() {
		total := box.Bounds().W
		if n.Axis == "vertical" {
			total = box.Bounds().H
		}
		if total <= 1 {
			total = 400
		}
		if n.Ratio == 0 {
			n.Ratio = .5
		}
		size := (total - 1) * n.Ratio
		a := func() { workspaceDockNode(c, m, n.First) }
		b := func() { workspaceDockNode(c, m, n.Second) }
		if n.Axis == "vertical" {
			ui.SplitVertical(c, &size, a, b).Fill()
		} else {
			ui.Split(c, &size, a, b).Fill()
		}
		n.Ratio = size / (total - 1)
	})
}

// Unlike dock clipping, mini-player clipping must include the source's own
// floating resource: all resource floats paint above the conversation overlay.
func miniPlayerFloatOcclusions(floats []*WorkspacePane, width, height float32) []mygo.Rectangle {
	var holes []mygo.Rectangle
	for _, n := range floats {
		f := boundedFloat(*n.Float, width, height)
		holes = append(holes, mygo.Rectangle{X: int(f.X), Y: int(f.Y), Width: int(f.Width), Height: int(f.Height)})
	}
	return holes
}
