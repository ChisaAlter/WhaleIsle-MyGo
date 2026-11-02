package main

import (
	"whaleisle/internal/harness"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

// Match the selected canonical session against the current Host's catalog.
// A Bot's personal session and a room member's worker session are not rooms.
func (s *mainState) assistantGroupMembers(sessionID string) (bool, []session.GroupMember) {
	s.mu.Lock()
	roster := s.dshbotRoster
	ep := s.bootStatus.Endpoint
	ready := s.bootStatus.State == harness.Ready
	cfg := s.cfgStore
	s.mu.Unlock()
	if roster == nil || cfg == nil || !cfg.Get().DshbotEnabled || !ready || sessionID == "" {
		return false, nil
	}
	snap := roster.Snapshot()
	if snap.Origin != ep.Origin {
		return false, nil
	}
	for _, room := range snap.Rows {
		if room.Kind != "room" || room.SessionID != sessionID || room.Hidden {
			continue
		}
		members := make([]session.GroupMember, 0, len(room.MemberBotIDs))
		for _, id := range room.MemberBotIDs {
			for _, bot := range snap.Rows {
				if bot.ID == id && bot.Kind != "room" {
					members = append(members, session.GroupMember{ID: bot.ID, Name: bot.Name, Title: bot.Title})
					break
				}
			}
		}
		return true, s.refreshAssistantGroupRuntime(ep, room.ID, sessionID, members)
	}
	return false, nil
}
func (s *mainState) beginAssistantGroupReply(sessions *session.Store, draft *composer.DraftState, id, threadID string) {
	if sessions == nil || sessions.SelectedSessionID() != id {
		return
	}
	room, members := s.assistantGroupMembers(id)
	if !room {
		return
	}
	for _, thread := range session.ProjectGroupThreads(sessions.Snapshot().Messages, members) {
		if thread.ID == threadID {
			draft.BeginGroupReply(threadID)
			return
		}
	}
}
