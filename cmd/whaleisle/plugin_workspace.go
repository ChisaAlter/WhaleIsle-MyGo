package main

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo"

	"whaleisle/internal/desktop/pluginui"
	"whaleisle/internal/ui/workspacepicker"
)

func (s *mainState) pluginNavigationOwner() *pluginui.Owner {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p := s.nativePlugins.navigation; p != nil {
		return p.owner
	}
	return nil
}
func (s *mainState) pluginWorkspaceNavigation() []pluginui.WorkspaceNavigation {
	if owner := s.pluginNavigationOwner(); owner != nil {
		snap := owner.Snapshot()
		if snap.Connected && snap.Subject != nil {
			return snap.WorkspaceNavigation
		}
	}
	return nil
}
func (s *mainState) runPluginWorkspace(owner *pluginui.Owner, snap pluginui.Snapshot, op, id, path, presentationOwner string) {
	s.mu.Lock()
	win, sessions := s.win, s.sessions
	s.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		err := owner.InvokeWorkspace(ctx, snap, op, id, path, presentationOwner)
		if win == nil || win.IsDestroyed() {
			return
		}
		win.Update(func() {
			if s.pluginNavigationOwner() != owner {
				return
			}
			if err != nil {
				s.notifyUser(win, err.Error())
				return
			}
			if op == "open" && sessions != nil {
				s.mu.Lock()
				current := s.sessions == sessions
				s.mu.Unlock()
				if current {
					sessions.Select(id)
					s.revealMainConversation()
				}
			}
		})
	}()
}
func (s *mainState) pluginWorkspaceSessionAction(id, ownerID, action string) {
	if owner := s.pluginNavigationOwner(); owner != nil {
		s.runPluginWorkspace(owner, owner.Snapshot(), action, id, "", ownerID)
	}
}
func (s *mainState) workspaceDirectoryActions(win *mygo.Window) []workspacepicker.DirectoryAction {
	result := s.projectDirectoryActions(win)
	owner := s.pluginNavigationOwner()
	if owner == nil {
		return result
	}
	snap := owner.Snapshot()
	if !snap.Connected || snap.Subject == nil {
		return result
	}
	entries := append([]pluginui.WorkspaceDirectoryAction(nil), snap.DirectoryActions...)
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].Order < entries[j].Order })
	for _, entry := range entries {
		row := entry
		picker := row.Picker
		result = append(result, workspacepicker.DirectoryAction{ID: row.ID, Order: row.Order, Label: row.Label, Picker: &picker, Pick: func() {
			s.mu.Lock()
			p := s.nativePlugins.navigation
			sessions := s.sessions
			current := p != nil && p.owner == owner
			if current {
				p.directoryPickID = row.Handle
				p.directoryPickOwner = owner
			}
			s.mu.Unlock()
			if current {
				s.openDirectoryPicker(sessions, win, "plugin-directory:"+row.Handle)
			}
		}, Run: func(path string) { s.runPluginWorkspace(owner, owner.Snapshot(), "directory", row.Handle, path, "") }, Visible: func(path string) bool { _, ok := row.RowLabels[path]; return ok }, LabelForPath: func(path string) string { return row.RowLabels[path] }})
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Order < result[j].Order })
	return result
}

func (s *mainState) completePluginDirectoryAction(mode, path string) bool {
	if !strings.HasPrefix(mode, "plugin-directory:") {
		return false
	}
	id := strings.TrimPrefix(mode, "plugin-directory:")
	s.mu.Lock()
	p := s.nativePlugins.navigation
	var owner *pluginui.Owner
	if p != nil && p.directoryPickID == id && p.directoryPickOwner == p.owner {
		owner = p.owner
	}
	if p != nil {
		p.directoryPickID = ""
		p.directoryPickOwner = nil
	}
	win := s.win
	s.mu.Unlock()
	if owner == nil {
		if win != nil {
			s.notifyUser(win, pluginCopy("目录动作已撤销，请重新选择。", "The directory action was withdrawn. Select it again."))
		}
		return true
	}
	s.runPluginWorkspace(owner, owner.Snapshot(), "directory", id, path, "")
	return true
}
