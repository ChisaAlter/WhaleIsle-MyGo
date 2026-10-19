package main

import (
	"context"
	"time"

	"github.com/egoist/mygo"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/settings"
)

func (s *mainState) agentPresetReaderAccess(source *session.Store) *settings.AgentPresetReaderAccess {
	if source == nil {
		return nil
	}
	s.mu.Lock()
	var previous *session.AgentPresetReader
	if s.agentPresetReader == nil || s.agentPresetReaderSource != source {
		previous = s.agentPresetReader
		s.agentPresetReader = session.NewAgentPresetReader(source, func() bool { return s.creatorTaskNavigationCurrent(source) })
		s.agentPresetReaderSource = source
	}
	reader, win := s.agentPresetReader, s.win
	s.mu.Unlock()
	if previous != nil {
		previous.ObserveVisible(false)
	}
	reader.ObserveVisible(s.creatorTaskVisible(source))
	refresh := func() {
		if win != nil {
			win.Update(func() {})
		}
	}
	read := func(ticket uint64) {
		if !s.creatorTaskVisible(source) {
			reader.ObserveVisible(false)
			return
		}
		next := reader.Begin(ticket)
		if next == 0 {
			return
		}
		refresh()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			reader.Load(ctx, next)
			refresh()
		}()
	}
	return &settings.AgentPresetReaderAccess{
		State: reader.Snapshot(),
		OnSelect: func(ticket uint64, id string) {
			before := reader.Snapshot()
			if !s.creatorTaskVisible(source) || before.Busy || before.Ticket != ticket {
				return
			}
			reader.Select(ticket, id)
			selected := reader.Snapshot()
			if selected.SelectedID == id {
				read(selected.Ticket)
			}
		},
		OnClose: func(ticket uint64) { reader.Close(ticket); refresh() },
		OnCopy: func(ticket uint64) {
			if !s.creatorTaskVisible(source) {
				reader.ObserveVisible(false)
				return
			}
			if text, ok := reader.CopyText(ticket); ok {
				mygo.Clipboard.WriteText(text)
			}
		},
		OnRead: read,
	}
}

// Invalidate on the same render that closes, changes tabs, or hides the page.
func (s *mainState) observeAgentPresetReader(rendered bool) {
	s.mu.Lock()
	reader, source := s.agentPresetReader, s.agentPresetReaderSource
	s.mu.Unlock()
	if reader != nil {
		reader.ObserveVisible(rendered && s.creatorTaskVisible(source))
	}
}

func (s *mainState) suspendAgentPresetReader() {
	s.mu.Lock()
	reader := s.agentPresetReader
	s.mu.Unlock()
	if reader != nil {
		reader.ObserveVisible(false)
	}
}
