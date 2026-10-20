package dshbot

import (
	"fmt"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/store/storelocale"
)

type RoutineTriggerSnapshot struct {
	Open, Review, Acknowledged, Busy, Ready bool
	Sequence                                uint64
	RoutineID, RoutineName, SessionID       string
	Revision                                int
	Error                                   string
}
type RoutineTriggerTicket struct {
	Scope                RoutineScope
	Binding, Sequence    uint64
	RoutineID, SessionID string
}
type routineTriggerState struct {
	meta    RoutineTriggerSnapshot
	binding uint64
	token   []byte // Never copied into snapshots, logs, persistence or a Host/Bot cache.
}

func (c *RoutineControls) clearRoutineTrigger() {
	clear(c.trigger.token)
	if c.trigger.meta.Busy {
		c.state.Busy = false
		c.generation++
	}
	c.triggerSequence++
	c.trigger = routineTriggerState{}
}
func (c *RoutineControls) PrepareTrigger(s RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(s) || !c.state.Loaded || c.state.Busy {
		return
	}
	row, ok := c.row(c.state.SelectedID)
	if !ok {
		return
	}
	c.clearRoutineTrigger()
	c.trigger = routineTriggerState{binding: c.binding, meta: RoutineTriggerSnapshot{Open: true, Review: true, Sequence: c.triggerSequence, RoutineID: row.ID, RoutineName: row.Name, SessionID: c.state.SessionID, Revision: c.state.Revision}}
	c.state.Review = nil
}
func (c *RoutineControls) triggerBound(s RoutineScope, sequence uint64) bool {
	return c.valid(s) && c.trigger.meta.Open && c.trigger.binding == c.binding && c.trigger.meta.Sequence == sequence && c.trigger.meta.RoutineID == c.state.SelectedID && c.trigger.meta.SessionID == c.state.SessionID
}
func (c *RoutineControls) AcknowledgeTrigger(s RoutineScope, sequence uint64, ack bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.triggerBound(s, sequence) && c.trigger.meta.Review && !c.state.Busy {
		c.trigger.meta.Acknowledged = ack
	}
}
func (c *RoutineControls) CloseTrigger(s RoutineScope, sequence uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.triggerBound(s, sequence) {
		c.clearRoutineTrigger()
	}
}
func (c *RoutineControls) BeginTrigger(s RoutineScope, sequence uint64) (RoutineTriggerTicket, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	m := c.trigger.meta
	if !c.triggerBound(s, sequence) || !c.state.Loaded || c.state.Busy || !m.Review || !m.Acknowledged || m.Revision != c.state.Revision {
		return RoutineTriggerTicket{}, false
	}
	if _, ok := c.row(m.RoutineID); !ok {
		return RoutineTriggerTicket{}, false
	}
	c.trigger.meta.Review = false
	c.trigger.meta.Busy = true
	c.state.Busy = true
	c.state.Loaded = false
	c.state.Review = nil
	c.generation++
	return RoutineTriggerTicket{Scope: s, Binding: c.binding, Sequence: sequence, RoutineID: m.RoutineID, SessionID: m.SessionID}, true
}
func (c *RoutineControls) triggerCurrent(t RoutineTriggerTicket) bool {
	return c.triggerBound(t.Scope, t.Sequence) && c.binding == t.Binding && c.trigger.meta.RoutineID == t.RoutineID && c.trigger.meta.SessionID == t.SessionID && c.trigger.meta.Busy
}
func (c *RoutineControls) CurrentTrigger(t RoutineTriggerTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.triggerCurrent(t)
}
func (c *RoutineControls) FinishTrigger(t RoutineTriggerTicket, token []byte, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.triggerCurrent(t) {
		return false
	}
	c.trigger.meta.Busy = false
	c.state.Busy = false
	c.state.Loaded = false
	c.state.Review = nil
	// Get-or-create returns no catalog revision. Require a read-only catalog
	// refresh before further writes; that refresh also clears the displayed token.
	c.state.Notice = storelocale.T(storelocale.KeyStoreDshbotRefreshBeforeFurtherChangesRefreshC)
	if err != nil || len(token) == 0 {
		c.trigger.meta.Error = storelocale.T(storelocale.KeyStoreDshbotCredentialOutcomeUncertainNoAutomati)
		return true
	}
	c.trigger.token = append([]byte(nil), token...)
	c.trigger.meta.Ready = true
	return true
}

// Material is requested only while this exact view remains visible. No token is
// captured in the main/UI model; clear removes and overwrites the owned buffer.
// Rendering/clipboard may create transient strings; physical memory erasure is
// not claimed, and an explicitly copied OS clipboard is outside this state.
func (c *RoutineControls) TriggerMaterial(s RoutineScope, sequence uint64, origin string) (string, string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.triggerBound(s, sequence) || !c.trigger.meta.Ready || len(c.trigger.token) == 0 {
		return "", "", fmt.Errorf("trigger view is no longer current")
	}
	address, err := pluginrpc.DshbotRoutineTriggerURL(origin, c.trigger.meta.RoutineID)
	if err != nil {
		return "", "", err
	}
	return address, "X-Dshbot-Token: " + string(c.trigger.token), nil
}
