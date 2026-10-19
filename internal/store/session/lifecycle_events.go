package session

import (
	"context"
	"encoding/json"
	"strings"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// lifecycleMutation is one validated Host event, retained only while a list
// request is in flight so its older snapshot cannot undo newer live state.
type lifecycleMutation struct {
	kind      string
	sessionID string
	item      Item
	hasTitle  bool
	running   bool
	updatedAt float64
	message   string
}

// EventsOwner binds one $events stream to its Store, client and generation.
// Call BeginEvents before opening each stream; callbacks retain that owner.
type EventsOwner struct {
	store      *Store
	client     *dsh.Client
	ctx        context.Context
	generation uint64
}

// BeginEvents fences prior streams even when reconnect reuses the same client
// and outer context. A stale/canceled caller cannot acquire a new generation.
func (s *Store) BeginEvents(ctx context.Context, client *dsh.Client) *EventsOwner {
	if s == nil || ctx == nil || ctx.Err() != nil || client == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client != client || ctx.Err() != nil {
		return nil
	}
	s.cancelRetainedAgentReadsLocked("") // Event authority is being re-established; never assume missed revocations.
	s.eventsGeneration++
	s.invalidateCommandsLocked()
	s.listGeneration++
	s.listLoading = false
	s.listMutations = nil
	return &EventsOwner{store: s, client: client, ctx: ctx, generation: s.eventsGeneration}
}

func (s *Store) eventsOwnerCurrentLocked(owner *EventsOwner) bool {
	return owner != nil && owner.store == s && owner.ctx != nil && owner.ctx.Err() == nil &&
		s.client == owner.client && s.eventsGeneration == owner.generation
}

// OnEventsReady atomically validates the stream owner before starting the
// existing session/list baseline request for this $events generation.
func (s *Store) OnEventsReady(owner *EventsOwner) {
	if s == nil {
		return
	}
	s.mu.Lock()
	if !s.eventsOwnerCurrentLocked(owner) {
		s.mu.Unlock()
		return
	}
	client, generation := s.beginListRefreshLocked()
	s.mu.Unlock()
	if client == nil {
		return
	}
	s.fire()
	go s.runRefreshList(owner.ctx, client, generation)
}

// OnEmit folds actual api-session wire arguments only for the active stream.
// Unknown/malformed events and superseded/canceled owners are safely ignored.
func (s *Store) OnEmit(owner *EventsOwner, emit dsh.EventsEmit) {
	if s == nil || emit.Type != dsh.EventEmit {
		return
	}
	mutation, ok := parseLifecycleEmit(emit)
	s.mu.Lock()
	if !s.eventsOwnerCurrentLocked(owner) {
		s.mu.Unlock()
		return
	}
	if s.applyCommandCatalogEmitLocked(emit) {
		s.mu.Unlock()
		s.EnsureSlashCommands(owner.ctx)
		s.fire()
		return
	}
	if !ok {
		s.mu.Unlock()
		return
	}
	if mutation.kind == "api-session/deleted" || mutation.kind == "api-session/removed" {
		s.cancelRetainedAgentReadsLocked(mutation.sessionID)
	}
	if s.listLoading && mutation.kind != "api-session/error" {
		s.listMutations = append(s.listMutations, mutation)
	}
	refreshGoal := false
	if mutation.kind == "api-session/status" && mutation.sessionID == s.selectedID {
		for _, item := range s.items {
			if item.SessionID == mutation.sessionID && item.Running != mutation.running {
				refreshGoal = true
				break
			}
		}
	}
	sessionID, generation := s.selectedID, s.followGeneration
	changed := s.applyLifecycleMutationLocked(mutation, true)
	s.mu.Unlock()
	if changed {
		s.fire()
	}
	if refreshGoal {
		go s.runRefreshGoal(owner.ctx, owner.client, sessionID, generation)
	}
}

func parseLifecycleEmit(emit dsh.EventsEmit) (lifecycleMutation, bool) {
	m := lifecycleMutation{kind: emit.Event}
	if emit.Event == "api-session/added" {
		if len(emit.Args) != 1 {
			return m, false
		}
		// Required booleans/numbers must be present; encoding/json otherwise
		// silently treats null or omitted values as their zero values.
		var required struct {
			SessionID      string   `json:"sessionId"`
			Running        *bool    `json:"running"`
			Blank          *bool    `json:"blank"`
			AgentAvailable *bool    `json:"agentAvailable"`
			UpdatedAt      *float64 `json:"updatedAt"`
		}
		var summary api.SessionListValueItemsItem
		if json.Unmarshal(emit.Args[0], &required) != nil || strings.TrimSpace(required.SessionID) == "" ||
			required.Running == nil || required.Blank == nil || required.AgentAvailable == nil || required.UpdatedAt == nil ||
			json.Unmarshal(emit.Args[0], &summary) != nil {
			return m, false
		}
		m.sessionID = summary.SessionId
		m.item = itemFromListRow(summary)
		m.hasTitle = summary.Projections != nil && titleFromProjections(summary.Projections.Values) != ""
		return m, true
	}
	count := 2
	switch emit.Event {
	case "api-session/removed", "api-session/deleted":
		count = 1
	case "api-session/status", "api-session/activity", "api-session/error":
	default:
		return m, false
	}
	if len(emit.Args) != count || json.Unmarshal(emit.Args[0], &m.sessionID) != nil || strings.TrimSpace(m.sessionID) == "" {
		return m, false
	}
	if count == 1 {
		return m, true
	}
	if strings.TrimSpace(string(emit.Args[1])) == "null" {
		return m, false
	}
	var err error
	switch emit.Event {
	case "api-session/status":
		err = json.Unmarshal(emit.Args[1], &m.running)
	case "api-session/activity":
		err = json.Unmarshal(emit.Args[1], &m.updatedAt)
	case "api-session/error":
		err = json.Unmarshal(emit.Args[1], &m.message)
	}
	return m, err == nil
}

func (s *Store) applyLifecycleMutationLocked(m lifecycleMutation, live bool) bool {
	retiredSubmission := false
	if m.kind == "api-session/removed" || m.kind == "api-session/deleted" {
		delete(s.promptAttempted, m.sessionID)
		if live && m.kind == "api-session/deleted" {
			// Delete receipts use this same live path. A later list replay only
			// repairs roster state; it must not retire a newer same-ID send.
			retiredSubmission = s.retireDeletedSessionSubmissionsLocked(m.sessionID)
		}
	}
	if m.kind == "api-session/status" {
		s.observeSidebarRunningLocked(m.sessionID, m.running, live && s.listLoading)
	}
	if m.kind == "api-session/added" {
		s.observeSidebarRunningLocked(m.sessionID, m.item.Running, false)
	}
	index := -1
	for i := range s.items {
		if s.items[i].SessionID == m.sessionID {
			index = i
			break
		}
	}
	if m.kind == "api-session/added" {
		if index < 0 {
			s.items = append(s.items, m.item)
		} else {
			// Availability-only summaries may omit projection hints. The source
			// client retains its known title projection independently of summary.
			if !m.hasTitle && s.items[index].Title != "" {
				m.item.Title = s.items[index].Title
			}
			m.item.AgentMetrics = mergeAgentMetrics(m.item.AgentMetrics, s.items[index].AgentMetrics)
			mergeSessionPresentation(&m.item, s.items[index])
			if m.item.BilledUsage == nil {
				m.item.BilledUsage = s.items[index].BilledUsage
				m.item.BilledUsageSeq = s.items[index].BilledUsageSeq
				m.item.BilledUsageKind = s.items[index].BilledUsageKind
			}
			s.items[index] = m.item
		}
		s.applyPinnedFlagsLocked()
		return true
	}
	if index < 0 {
		if live && s.selectedID == m.sessionID && (m.kind == "api-session/removed" || m.kind == "api-session/deleted") {
			s.selectSessionLocked("")
			return true
		}
		return retiredSubmission
	}
	switch m.kind {
	case "api-session/status":
		s.items[index].Running = m.running
		if m.running {
			s.items[index].ContactError = ""
		}
		s.items[index].AgentAvailable = true
	case "api-session/activity":
		s.items[index].UpdatedAt = m.updatedAt
	case "api-session/removed", "api-session/deleted":
		// Removed is a live-registry departure, not durable destruction. As in
		// the source client, durable subagent summaries remain available.
		if m.kind == "api-session/removed" && s.items[index].Origin == "subagent" {
			s.items[index].Running = false
			s.items[index].AgentAvailable = false
			return true
		}
		s.items = append(s.items[:index], s.items[index+1:]...)
		if live && s.selectedID == m.sessionID {
			s.selectSessionLocked("")
		}
	case "api-session/error":
		s.items[index].ContactError = m.message
		if !live || s.selectedID != m.sessionID {
			return true
		}
		s.lastAction = m.message
	}
	return true
}

func sessionListItems(rows []api.SessionListValueItemsItem) []Item {
	items := make([]Item, 0, len(rows))
	seen := make(map[string]bool, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.SessionId) == "" || seen[row.SessionId] {
			continue
		}
		seen[row.SessionId] = true
		items = append(items, itemFromListRow(row))
	}
	return items
}

// selectSessionLocked invalidates all work belonging to the old transcript,
// including an Open or page request that completes after a selection change.
func (s *Store) selectSessionLocked(sessionID string) {
	s.mainConversationReadID = ""
	delete(s.completionUnread, sessionID)
	if s.selectedID == sessionID {
		return
	}
	s.invalidateArchiveLocked()
	s.invalidateFollowLocked()
	s.invalidateCommandsLocked()
	s.selectedID = sessionID
	s.historyReady = false
	s.historyLoading = sessionID != ""
	s.invalidateSkillOptionsLocked()
	s.messages = nil
	s.turnStartMs = 0
	s.editTurnSeq, s.editTurnOpening = 0, false
	s.resetPlanBoardLocked()
	s.resetHistoryPagingLocked()
	s.historyThroughSeq = 0
	s.clearFeedbackLocked()
}

func (s *Store) invalidateFollowLocked() {
	s.historyLoading = false
	s.clearHistoryErrorLocked()
	s.resetTurnFilesLocked()
	s.turnFilesHistoryBinding = ""
	s.resetHistoryPagingLocked()
	s.clearHistoryOutlineLocked()
	s.trajectoryBoundaries = nil
	s.systemPromptAnchors = nil
	s.planHistoryBinding = ""
	s.invalidateJobsLocked()
	s.invalidateWorkflowLocked()
	s.invalidateAgentNavigationLocked()
	s.resetPermissionRiskLocked()
	s.resetAssistantStreamLocked()
	s.cancelVisualReplyLocked()
	s.followGeneration++
	s.resetModelSelectionLocked()
	s.permissionProjection = permissionProjectionState{}
	s.permissionsErr = nil
	s.historyBranchBusy = false
	if s.followCancel != nil {
		s.followCancel()
		s.followCancel = nil
	}
}

func (s *Store) sessionResultCurrentLocked(ctx context.Context, client *dsh.Client, sessionID string, generation uint64) bool {
	return s.client == client && s.followCurrentLocked(ctx, sessionID, generation)
}
