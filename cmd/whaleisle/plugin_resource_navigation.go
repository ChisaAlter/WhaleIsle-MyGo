package main

import (
	"context"
	"fmt"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/frame"
)

// Registered resource doors are distinct from already mounted tab occurrences.
func (s *mainState) pluginRightPaneKinds() []frame.PluginRightPaneKind {
	owner := s.currentNativePluginSessionOwner()
	if owner == nil {
		return nil
	}
	registered := owner.Snapshot().RightPaneKinds
	kinds := make([]frame.PluginRightPaneKind, 0, len(registered))
	for _, kind := range registered {
		kinds = append(kinds, frame.PluginRightPaneKind{
			ID: kind.ID, ProviderID: kind.ProviderID, Kind: kind.Kind,
			Label: kind.Label, Description: kind.Description, Icon: kind.Icon,
			Order: kind.Order, Disabled: !kind.Available,
		})
	}
	return kinds
}

func (s *mainState) pluginRightPaneKindOpener(win *mygo.Window, sessions *session.Store, id string, workspace *frame.WorkspacePanes) func(string, string) error {
	return func(kind, guideID string) error {
		owner := s.currentNativePluginSessionOwner()
		if owner == nil || win == nil || sessions == nil || workspace == nil || sessions.SelectedSessionID() != id {
			return fmt.Errorf("%s", resumeText("会话已更改，请重新选择面板。", "The conversation changed. Choose the panel again."))
		}
		snapshot := owner.Snapshot()
		available := false
		for _, entry := range snapshot.RightPaneKinds {
			if entry.Kind == kind && entry.Available {
				available = true
				break
			}
		}
		if !available {
			return fmt.Errorf("%s", resumeText("该面板当前不可用。", "This panel is currently unavailable."))
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			tab, err := owner.OpenRightPaneKind(ctx, snapshot, kind)
			mygo.RunOnMain(func() {
				if win.IsDestroyed() || s.currentNativePluginSessionOwner() != owner || sessions.SelectedSessionID() != id {
					return
				}
				s.mu.Lock()
				current := s.sessions == sessions && s.layout.Workspace == workspace && !s.quitting
				s.mu.Unlock()
				if !current {
					return
				}
				if err != nil {
					s.notifyUser(win, resumeText("无法打开面板：", "Could not open the panel: ")+err.Error())
					return
				}
				// Only a real mounted handle may replace the guide that admitted
				// this gesture; a later active page is not the replacement target.
				if workspace.ReplaceGuideAfterOpen(guideID, tab.Handle, tab.Label) == "" {
					return
				}
				s.mu.Lock()
				s.filesPaneOpen, s.rightPaneTab = true, "plugin-native"
				s.mu.Unlock()
				win.Invalidate()
			})
		}()
		return nil
	}
}
