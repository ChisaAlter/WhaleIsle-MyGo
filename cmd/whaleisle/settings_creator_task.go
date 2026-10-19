package main

import (
	"context"
	"time"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/settings"
)

// Safe from a background admission while its controller lock is held. Native
// visibility getters dispatch to the UI thread and must not run under that lock.
func (s *mainState) creatorTaskNavigationCurrent(source *session.Store) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.win != nil && !s.win.IsDestroyed() && s.sessions == source && s.settingsOpen && s.settingsAgentPresetsActive && !s.quitting && !s.launcherOpen && !s.usageOpen && !s.imOpen && !s.remoteOpen && !s.webviewOpen && !s.browserPreviewOpen
}

// Called only by UI-thread entry points and the post-render observer.
func (s *mainState) creatorTaskVisible(source *session.Store) bool {
	if !s.creatorTaskNavigationCurrent(source) {
		return false
	}
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
	return win != nil && !win.IsDestroyed() && win.IsVisible() && !win.IsMinimized()
}
func (s *mainState) suspendCreatorTask() {
	s.mu.Lock()
	creator := s.creatorTask
	s.mu.Unlock()
	if creator != nil {
		creator.ObserveVisible(false)
	}
}
func (s *mainState) creatorTaskAccess(source *session.Store) *settings.CreatorTaskAccess {
	if source == nil {
		return nil
	}
	s.mu.Lock()
	if s.creatorTask == nil {
		s.creatorTask = session.NewCreatorTask(source, func() bool { return s.creatorTaskNavigationCurrent(source) })
	}
	creator, win := s.creatorTask, s.win
	s.mu.Unlock()
	creator.ObserveVisible(s.creatorTaskVisible(source))
	refresh := func() {
		if win != nil {
			win.Update(func() {})
		}
	}
	create := func(ticket uint64) {
		if !s.creatorTaskVisible(source) {
			s.suspendCreatorTask()
			return
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			ready := creator.Create(ctx, ticket)
			if win != nil {
				win.Update(func() {
					if ready {
						s.openCreatorTask(creator, source, ticket, true)
					}
				})
			}
		}()
	}
	return &settings.CreatorTaskAccess{
		State: creator.Snapshot(),
		OnPrepare: func() {
			if !s.creatorTaskVisible(source) {
				return
			}
			creator.Prepare()
			prepared := creator.Snapshot()
			if prepared.Review && !prepared.Busy {
				// This explicit settings action is the admission, just as the
				// Plugins Creator menu is. Keep its captured target and receipts.
				create(prepared.Ticket)
			}
			refresh()
		},
		OnCancel: func(ticket uint64) { creator.Cancel(ticket); refresh() },
		OnCreate: create,
		OnLocate: func(ticket uint64) {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()
				creator.Locate(ctx, ticket)
				refresh()
			}()
		},
		OnOpen: func(ticket uint64) { s.openCreatorTask(creator, source, ticket, false); refresh() },
	}
}
func (s *mainState) openCreatorTask(creator *session.CreatorTask, source *session.Store, ticket uint64, automatic bool) {
	if !s.creatorTaskVisible(source) || !creator.Open(ticket, automatic) {
		return
	}
	s.mu.Lock()
	s.settingsOpen = false
	s.settingsAgentPresetsActive = false
	s.marketOpen = false
	s.scheduleOpen = false
	s.assistantConversationGeneration++
	s.assistantOpen = false
	s.toolsMenuOpen = false
	s.mu.Unlock()
	creator.ObserveVisible(false)
}

// The same-frame post-render fence invalidates a stale confirmation on Close,
// another settings tab, a hidden window, or any newer navigation.
func (s *mainState) observeCreatorTask(rendered bool) {
	s.mu.Lock()
	creator, source, win := s.creatorTask, s.sessions, s.win
	s.mu.Unlock()
	if creator == nil {
		return
	}
	visible := rendered && s.creatorTaskVisible(source)
	if win == nil || !win.IsVisible() || win.IsMinimized() {
		visible = false
	}
	creator.ObserveVisible(visible)
}
