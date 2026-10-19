package frame

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// SidebarRight configures DockKit with two horizontal seats and a .2 minimum
// fraction. A chip needs 100 DIP after the measured, non-shrinking chrome.
func (w *WorkspacePanes) canSplit(n *WorkspacePane) bool {
	if w == nil || n == nil || n.Float != nil || w.Fullscreen != "" {
		return false
	}
	count := 0
	w.Root.eachLeaf(func(p *WorkspacePane) {
		if p.dockedLeaf() != nil {
			count++
		}
	})
	if count >= 2 {
		return false
	}
	if n.paneWidth <= 0 || n.stripWidth <= 0 || n.tabRunWidth <= 0 {
		return true
	}
	borders := max(float32(0), n.paneWidth-n.stripWidth)
	fixed := max(float32(0), n.stripWidth-n.tabRunWidth)
	return n.paneWidth/2-borders >= fixed+100
}

func validateWorkspaceRestore(w *WorkspacePanes) error {
	seen := map[string]bool{}
	var groups []*WorkspacePane
	var visit func(*WorkspacePane, bool) error
	visit = func(n *WorkspacePane, tab bool) error {
		if n == nil || n.ID == "" || seen[n.ID] {
			return fmt.Errorf("invalid workspace pane identity")
		}
		seen[n.ID] = true
		if suffix, ok := strings.CutPrefix(n.ID, "workspace-pane-"); ok {
			if number, err := strconv.Atoi(suffix); err == nil {
				w.Next = max(w.Next, number)
			}
		}
		if n.First != nil || n.Second != nil {
			if tab || n.First == nil || n.Second == nil || n.Axis != "horizontal" && n.Axis != "vertical" || n.Kind != "" || len(n.DockTabs) != 0 || n.Float != nil || math.IsNaN(float64(n.Ratio)) || math.IsInf(float64(n.Ratio), 0) || n.Ratio <= 0 || n.Ratio >= 1 {
				return fmt.Errorf("invalid workspace split")
			}
			n.Axis = "horizontal"
			n.Ratio = min(float32(.8), max(float32(.2), n.Ratio))
			if err := visit(n.First, false); err != nil {
				return err
			}
			return visit(n.Second, false)
		}
		if n.Kind == "" || n.Axis != "" || tab && len(n.DockTabs) > 0 || n.DockIndex < 0 || n.DockIndex > len(n.DockTabs) {
			return fmt.Errorf("invalid workspace resource")
		}
		if !tab {
			groups = append(groups, n)
		}
		for _, other := range n.DockTabs {
			if err := visit(other, true); err != nil {
				return err
			}
		}
		return nil
	}
	if w.Root == nil {
		if w.Active != "" || w.Fullscreen != "" {
			return fmt.Errorf("workspace selection has no resource")
		}
		return nil
	}
	if err := visit(w.Root, false); err != nil {
		return err
	}
	// Older Go layouts allowed arbitrary split trees. Keep every resource and
	// draft, folding surplus seats into tabs instead of rejecting their recovery.
	var docked, floating []*WorkspacePane
	for _, group := range groups {
		if group.dockedLeaf() != nil {
			docked = append(docked, group)
		} else {
			floating = append(floating, group)
		}
	}
	if len(docked) > 2 || len(floating) > 0 && len(groups) > 1 {
		keep := docked
		if len(keep) == 0 {
			keep = []*WorkspacePane{floating[0]}
			floating = floating[1:]
		}
		if len(keep) > 2 {
			keep = keep[:2]
		}
		target := keep[len(keep)-1]
		tabs := target.dockTabs()
		for _, group := range groups {
			if group != keep[0] && group != target {
				tabs = append(tabs, group.dockTabs()...)
			}
		}
		setDockTabs(target, tabs)
		w.Root = keep[0]
		if len(keep) == 2 {
			w.Root = &WorkspacePane{ID: w.leaf("").ID, Axis: "horizontal", Ratio: .5, First: keep[0], Second: keep[1]}
		}
	}
	if w.Fullscreen != "" && !seen[w.Fullscreen] {
		w.Fullscreen = ""
	}
	return nil
}
