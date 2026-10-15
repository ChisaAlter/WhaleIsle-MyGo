package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/projectflow"
)

// AgentSessionAddress preserves the original discriminated SessionAddress.
// Child identity is never passed to Store.Select or ordinary session actions.
type AgentSessionAddress struct {
	Kind            string `json:"kind"`
	SessionID       string `json:"sessionId,omitempty"`
	ParentSessionID string `json:"parentSessionId,omitempty"`
	ChildSessionID  string `json:"childSessionId,omitempty"`
	Mode            string `json:"mode,omitempty"`
}

func (a AgentSessionAddress) validChild() bool {
	return a.Kind == "subagent" && a.SessionID == "" && a.ParentSessionID != "" && a.ChildSessionID != "" && a.ChildSessionID != a.ParentSessionID && (a.Mode == "one-shot" || a.Mode == "continuable" || a.Mode == "unknown")
}

type AgentNavigationRow struct {
	Address AgentSessionAddress
	Label   string
	Running bool
}
type AgentNavigationView struct {
	catalog          agentNavigationCatalog // Host-computed whole value and its watermark
	Messages         []Message              // isolated live presentation; never main chat messages
	ActionBusy       bool
	ActionStatus     string
	Token            uint64
	SessionID, Title string
	Address          AgentSessionAddress
	Depth            int
	Busy             bool
	Error            string
	Rows             []AgentNavigationRow
	Process          projectflow.ProcessSnapshot
	Body             string
}
type agentNavigationState struct {
	notifyPending bool // coalesce live repaint requests until the pane reads
	actionCancel  context.CancelFunc
	actionEpoch   uint64
	view          AgentNavigationView
	parents       []AgentNavigationView
	binding       string
	root          string
	client        *dsh.Client
	mux           *dsh.Mux
	generation    uint64
	cancel        context.CancelFunc
}

func (s *Store) invalidateAgentNavigationLocked() {
	if s.agentNavigation.actionCancel != nil {
		s.agentNavigation.actionCancel()
	}
	if s.agentNavigation.cancel != nil {
		s.agentNavigation.cancel()
	}
	token := s.agentNavigation.view.Token + 1
	s.agentNavigation = agentNavigationState{view: AgentNavigationView{Token: token}}
}
func (s *Store) agentNavigationOwnedLocked() bool {
	n := &s.agentNavigation
	return n.binding != "" && n.binding == s.planHistoryBinding && n.root == s.selectedID && n.client == s.client && n.mux == s.mux && n.generation == s.followGeneration
}
func cloneAgentNavigationView(v AgentNavigationView) AgentNavigationView {
	v.Rows = append([]AgentNavigationRow(nil), v.Rows...)
	v.Messages = append([]Message(nil), v.Messages...)
	v.Process.Records = append([]projectflow.ProcessRecord(nil), v.Process.Records...)
	return v
}
func (s *Store) AgentNavigationSnapshot() AgentNavigationView {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.agentNavigationOwnedLocked() {
		return AgentNavigationView{}
	}
	s.agentNavigation.notifyPending = false
	s.refreshAgentNavigationRowsLocked(&s.agentNavigation.view)
	return cloneAgentNavigationView(s.agentNavigation.view)
}

// SyncAgentNavigation may load only non-activating projections. The path and
// child history are runtime-only and never restored or prefetched at startup.
func (s *Store) SyncAgentNavigation(binding string) AgentNavigationView {
	s.mu.Lock()
	if binding == "" || binding != s.planHistoryBinding || binding != fmt.Sprintf("%p\x00%s", s.client, s.selectedID) || s.client == nil || s.mux == nil {
		s.invalidateAgentNavigationLocked()
		s.mu.Unlock()
		return AgentNavigationView{}
	}
	if s.agentNavigationOwnedLocked() {
		s.refreshAgentNavigationRowsLocked(&s.agentNavigation.view)
		s.agentNavigation.notifyPending = false
		v := cloneAgentNavigationView(s.agentNavigation.view)
		s.mu.Unlock()
		return v
	}
	s.invalidateAgentNavigationLocked()
	token := s.agentNavigation.view.Token
	title := s.selectedID
	for _, it := range s.items {
		if it.SessionID == s.selectedID && it.Title != "" {
			title = it.Title
			break
		}
	}
	s.agentNavigation = agentNavigationState{view: AgentNavigationView{Token: token, SessionID: s.selectedID, Title: title, Address: AgentSessionAddress{Kind: "session", SessionID: s.selectedID}}, binding: binding, root: s.selectedID, client: s.client, mux: s.mux, generation: s.followGeneration}
	s.startAgentNavigationReadLocked(false)
	v := cloneAgentNavigationView(s.agentNavigation.view)
	s.mu.Unlock()
	return v
}
func (s *Store) CloseAgentNavigation() {
	s.mu.Lock()
	if s.agentNavigation.binding != "" {
		s.invalidateAgentNavigationLocked()
	}
	s.mu.Unlock()
}

// OpenAgentNavigation is called only by an explicit click on a current catalog
// row. Addressed subagent follow only observes history; the Host promotes
// prepared Sessions only for ordinary kind:"session" addresses.
func (s *Store) OpenAgentNavigation(token uint64, address AgentSessionAddress) {
	s.mu.Lock()
	n := &s.agentNavigation
	if !s.agentNavigationOwnedLocked() || n.view.Token != token || n.view.Busy || !address.validChild() || address.ParentSessionID != n.view.SessionID || len(n.parents) >= 64 {
		s.mu.Unlock()
		return
	}
	var row AgentNavigationRow
	found := false
	for _, r := range n.view.Rows {
		if r.Address == address {
			row = r
			found = true
			break
		}
	}
	if !found {
		s.mu.Unlock()
		return
	}
	n.parents = append(n.parents, freezeAgentNavigationView(n.view))
	n.view = AgentNavigationView{Token: token, SessionID: address.ChildSessionID, Title: row.Label, Address: address, Depth: len(n.parents)}
	s.startAgentNavigationReadLocked(false)
	s.mu.Unlock()
	s.fire()
}

// Back restores the cached parent location/page. It never follows or executes.
func (s *Store) BackAgentNavigation(token uint64) {
	s.mu.Lock()
	n := &s.agentNavigation
	if !s.agentNavigationOwnedLocked() || n.view.Token != token || len(n.parents) == 0 {
		s.mu.Unlock()
		return
	}
	if n.cancel != nil {
		n.cancel()
		n.cancel = nil
	}
	v := freezeAgentNavigationView(n.parents[len(n.parents)-1])
	n.notifyPending = false
	n.parents = n.parents[:len(n.parents)-1]
	v.Token = n.view.Token + 1
	v.Busy = false
	v.ActionBusy = false
	if n.actionCancel != nil {
		n.actionCancel()
		n.actionCancel = nil
	}
	n.view = v
	s.mu.Unlock()
	s.fire()
}

// RefreshAgentNavigation is also explicit: for a child it opens a fresh cut.
func (s *Store) RefreshAgentNavigation(token uint64) {
	s.mu.Lock()
	n := &s.agentNavigation
	if !s.agentNavigationOwnedLocked() || n.view.Token != token || n.view.Busy {
		s.mu.Unlock()
		return
	}
	s.startAgentNavigationReadLocked(false)
	s.mu.Unlock()
	s.fire()
}
func (s *Store) EarlierAgentNavigation(token uint64) {
	s.mu.Lock()
	n := &s.agentNavigation
	if !s.agentNavigationOwnedLocked() || n.view.Token != token || n.view.Busy || !n.view.Address.validChild() || !n.view.Process.HasMore || agentNavigationBefore(n.view.Process) <= 0 {
		s.mu.Unlock()
		return
	}
	s.startAgentNavigationReadLocked(true)
	s.mu.Unlock()
	s.fire()
}
func agentNavigationBefore(p projectflow.ProcessSnapshot) float64 {
	before := float64(0)
	for _, r := range p.Records {
		if r.Type == "event" && r.Event.Seq > 0 && (before == 0 || r.Event.Seq < before) {
			before = r.Event.Seq
		}
	}
	return before
}

func (s *Store) startAgentNavigationReadLocked(earlier bool) {
	n := &s.agentNavigation
	if n.actionCancel != nil {
		n.actionCancel()
		n.actionCancel = nil
	}
	n.view.ActionBusy = false
	n.view.ActionStatus = ""
	if n.cancel != nil {
		n.cancel()
	}
	n.view = freezeAgentNavigationView(n.view)
	n.notifyPending = false
	if !earlier && n.view.Address.validChild() {
		s.startAgentNavigationLiveLocked()
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	n.cancel = cancel
	n.view.Token++
	n.view.Busy = true
	n.view.Error = ""
	v := cloneAgentNavigationView(n.view)
	client := n.client
	go func() {
		defer cancel()
		current := func() bool {
			s.mu.Lock()
			defer s.mu.Unlock()
			return ctx.Err() == nil && s.agentNavigationOwnedLocked() && s.agentNavigation.view.Token == v.Token
		}
		var err error
		var projection agentNavigationProjection
		if v.Address.validChild() && earlier {
			v.Process, v.Messages, v.Body, err = readAgentNavigationPage(ctx, client, v.Address, v.Process)
		}
		if err == nil && !earlier && current() {
			err = client.Call(ctx, "session/projections", map[string]any{"request": map[string]any{"sessionId": v.SessionID}}, &projection)
		}
		s.mu.Lock()
		n := &s.agentNavigation
		if !s.agentNavigationOwnedLocked() || n.view.Token != v.Token {
			s.mu.Unlock()
			return
		}
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		n.view.Busy = false
		n.cancel = nil
		if err == nil {
			// A control push may have landed while this RPC/page was in flight.
			v.catalog = n.view.catalog
			if !earlier {
				err = seedAgentNavigationCatalog(&v, projection.Values, projection.AsOfSeq, s.items)
			}
			s.refreshAgentNavigationRowsLocked(&v)
		}
		if err != nil {
			n.view.Error = err.Error()
		} else {
			v.Busy = false
			n.view = v
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// Mirrors original agents.ts: an existing catalog (including []) wins over
// ordinary list materialization; lineage fallback has explicitly unknown mode.
func agentNavigationRows(parent string, values json.RawMessage, items []Item) ([]AgentNavigationRow, error) {
	var projection struct {
		Catalog json.RawMessage `json:"subagentCatalog"`
	}
	if len(values) > 0 && string(values) != "null" {
		if err := json.Unmarshal(values, &projection); err != nil {
			return nil, err
		}
	}
	rows := []AgentNavigationRow{}
	if len(projection.Catalog) > 0 && string(projection.Catalog) != "null" {
		var entries []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
			Mode  string `json:"mode"`
		}
		if err := json.Unmarshal(projection.Catalog, &entries); err != nil {
			return nil, err
		}
		// Original catalog is ordered facts, not a map keyed by child id.
		for _, entry := range entries {
			if entry.ID == "" || entry.ID == parent {
				continue
			}
			mode := entry.Mode
			if mode != "continuable" && mode != "one-shot" {
				mode = "unknown"
			}
			label := strings.TrimSpace(entry.Label)
			running := false
			for _, it := range items {
				if it.SessionID == entry.ID {
					running = it.Running
					if label == "" {
						label = it.Title
					}
					break
				}
			}
			if label == "" {
				label = entry.ID
			}
			rows = append(rows, AgentNavigationRow{Address: AgentSessionAddress{Kind: "subagent", ParentSessionID: parent, ChildSessionID: entry.ID, Mode: mode}, Label: label, Running: running})
		}
		return rows, nil
	}
	for _, it := range items {
		if it.ParentID == parent && it.SessionID != "" && it.SessionID != parent {
			label := it.Title
			if label == "" {
				label = it.SessionID
			}
			rows = append(rows, AgentNavigationRow{Address: AgentSessionAddress{Kind: "subagent", ParentSessionID: parent, ChildSessionID: it.SessionID, Mode: "unknown"}, Label: label, Running: it.Running})
		}
	}
	return rows, nil
}
func readAgentNavigationPage(ctx context.Context, client *dsh.Client, address AgentSessionAddress, previous projectflow.ProcessSnapshot) (projectflow.ProcessSnapshot, []Message, string, error) {
	before := agentNavigationBefore(previous)
	var page struct {
		Records []json.RawMessage `json:"records"`
		HasMore bool              `json:"hasMore"`
	}
	err := client.Call(ctx, "session/page", map[string]any{"request": map[string]any{"address": address, "throughSeq": previous.Cursor, "beforeSeq": before, "maxMessages": 30}}, &page)
	if err != nil {
		return projectflow.ProcessSnapshot{}, nil, "", err
	}
	raw, _ := json.Marshal(map[string]any{"type": "snapshot", "cursor": previous.Cursor, "records": page.Records, "hasMore": page.HasMore})
	return ProjectAgentNavigationHistory(raw, previous.Cursor, before)
}

// Agent actions are bound to the exact rendered navigation token and complete
// child address. Never look up a newer Host client after admitting a click.
func (s *Store) agentNavigationActionAllowedLocked(token uint64, address AgentSessionAddress) bool {
	if !s.agentNavigationOwnedLocked() || s.agentNavigation.view.Token != token || !address.validChild() {
		return false
	}
	v := s.agentNavigation.view
	if v.Address == address {
		return true
	}
	for _, row := range v.Rows {
		if row.Address == address {
			return true
		}
	}
	return false
}
func (s *Store) PromptAgentNavigation(token uint64, address AgentSessionAddress, text string) {
	text = trimPrompt(text)
	if text == "" || address.Mode != "continuable" {
		return
	}
	s.agentNavigationAction(token, []AgentSessionAddress{address}, text)
}
func (s *Store) InterruptAgentNavigation(token uint64, addresses []AgentSessionAddress) {
	s.agentNavigationAction(token, append([]AgentSessionAddress(nil), addresses...), "")
}
func (s *Store) agentNavigationAction(token uint64, addresses []AgentSessionAddress, text string) {
	s.mu.Lock()
	n := &s.agentNavigation
	if len(addresses) == 0 || n.view.ActionBusy || n.view.Busy {
		s.mu.Unlock()
		return
	}
	for _, address := range addresses {
		if !s.agentNavigationActionAllowedLocked(token, address) {
			s.mu.Unlock()
			return
		}
	}
	client, generation := n.client, n.generation
	delivery := string(NormalizePromptMode(string(s.promptMode)))
	n.actionEpoch++
	epoch := n.actionEpoch
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	n.actionCancel = cancel
	n.view.ActionBusy = true
	n.view.ActionStatus = ""
	s.mu.Unlock()
	s.fire()
	go func() {
		defer cancel()
		validLocked := func() bool {
			if ctx.Err() != nil || s.agentNavigation.client != client || s.agentNavigation.generation != generation || s.agentNavigation.actionEpoch != epoch {
				return false
			}
			for _, address := range addresses {
				if !s.agentNavigationActionAllowedLocked(token, address) {
					return false
				}
			}
			return true
		}
		var err error
		for _, address := range addresses {
			s.mu.Lock()
			valid := validLocked()
			s.mu.Unlock()
			if !valid {
				err = ctx.Err()
				if err == nil {
					err = fmt.Errorf("Child target is no longer in the current directory; refresh before retrying")
				}
				break
			}
			if text != "" {
				req := api.SubagentPromptRequest{RequestId: newRequestID(), ParentSessionId: address.ParentSessionID, ChildSessionId: address.ChildSessionID, Mode: "continuable", Delivery: delivery, Content: PromptContentBlocks(text)}
				var receipt api.SubagentPromptReceipt
				err = client.Call(ctx, "subagents/prompt", map[string]any{"request": req}, &receipt)
				if err == nil && strings.TrimSpace(receipt.MessageId) == "" {
					err = fmt.Errorf("Invalid child prompt acknowledgement; refresh history before retrying")
				}
			} else {
				var receipt json.RawMessage
				err = client.Call(ctx, "subagents/interruptByParent", map[string]any{"parentSessionId": address.ParentSessionID, "childSessionId": address.ChildSessionID, "mode": "cancel"}, &receipt)
			}
			if err != nil {
				break
			}
		}
		s.mu.Lock()
		// A timeout is reportable only while the same navigation owner is present.
		if !s.agentNavigationOwnedLocked() || s.agentNavigation.view.Token != token || s.agentNavigation.client != client || s.agentNavigation.generation != generation || s.agentNavigation.actionEpoch != epoch {
			s.mu.Unlock()
			return
		}
		s.agentNavigation.view.ActionBusy = false
		s.agentNavigation.actionCancel = nil
		if err != nil {
			s.agentNavigation.view.ActionStatus = err.Error()
		} else if text != "" {
			s.agentNavigation.view.ActionStatus = "子代理输入已提交 / Child prompt submitted"
		} else {
			s.agentNavigation.view.ActionStatus = "中断请求已提交 / Interrupt requested"
		}
		s.mu.Unlock()
		s.fire()
	}()
}
