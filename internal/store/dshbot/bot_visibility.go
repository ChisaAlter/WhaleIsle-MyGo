package dshbot

import (
	"fmt"
	"reflect"
	"sync"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/store/storelocale"
)

type BotVisibilityReview struct {
	Command      pluginrpc.DshbotVisibilityCommand
	Sequence     uint64
	Acknowledged bool
}
type BotVisibilitySnapshot struct {
	Scope                                         string
	Generation                                    uint64
	Loaded, Writable, Busy, Uncertain, ShowHidden bool
	Catalog                                       pluginrpc.DshbotBotCatalog
	Review                                        *BotVisibilityReview
	Pending                                       *pluginrpc.DshbotVisibilityCommand
	Candidate                                     *pluginrpc.DshbotBotItem
	Result                                        pluginrpc.DshbotBotItem
	FocusID, FocusSessionID, Error, Notice        string
}
type BotVisibilityTicket struct {
	Scope                string
	Generation, Sequence uint64
}
type botVisibilityAttempt struct {
	Command  pluginrpc.DshbotVisibilityCommand
	Sequence uint64
	Sent     bool
}
type botVisibilityEntry struct {
	state   BotVisibilitySnapshot
	attempt *botVisibilityAttempt
	loading uint64
}

// Separate from the visible roster: hiding a row never removes this management
// entry or its locked attempt. State is account-isolated and memory-only.
type BotVisibilityManager struct {
	mu                   sync.Mutex
	active, binding      string
	generation, sequence uint64
	entries              map[string]*botVisibilityEntry
}

func (c *BotVisibilityManager) entry(k string) *botVisibilityEntry {
	if c.entries == nil {
		c.entries = map[string]*botVisibilityEntry{}
	}
	if c.entries[k] == nil {
		c.entries[k] = &botVisibilityEntry{state: BotVisibilitySnapshot{Scope: k}}
	}
	return c.entries[k]
}
func (c *BotVisibilityManager) valid(k string) bool {
	return k != "" && c.active == k && c.binding != ""
}
func (c *BotVisibilityManager) Bind(k, b string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.active == k && c.binding == b {
		return
	}
	c.generation++
	c.active, c.binding = k, b
	if k != "" {
		e := c.entry(k)
		e.state.Loaded = false
		e.state.Review = nil
		e.state.Candidate = nil
		e.loading = 0
		if e.attempt == nil {
			e.state.Busy = false
		}
	}
}
func (c *BotVisibilityManager) Suspend() { c.Bind("", "") }
func (c *BotVisibilityManager) ViewCurrent(k string, g uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.valid(k) && g == c.generation
}
func (c *BotVisibilityManager) Snapshot(k string) BotVisibilitySnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[k]
	if e == nil {
		return BotVisibilitySnapshot{Scope: k, Generation: c.generation}
	}
	out := e.state
	out.Generation = c.generation
	out.Catalog = pluginrpc.CloneDshbotBotCatalog(out.Catalog)
	out.Result = pluginrpc.CloneDshbotBotItem(out.Result)
	if out.Review != nil {
		r := *out.Review
		r.Command = pluginrpc.CloneDshbotVisibilityCommand(r.Command)
		out.Review = &r
	}
	if e.attempt != nil {
		cmd := pluginrpc.CloneDshbotVisibilityCommand(e.attempt.Command)
		out.Pending = &cmd
	}
	if out.Candidate != nil {
		v := pluginrpc.CloneDshbotBotItem(*out.Candidate)
		out.Candidate = &v
	}
	return out
}
func (c *BotVisibilityManager) Show(k string, on bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(k) {
		c.entry(k).state.ShowHidden = on
	}
}

// Reveal is local navigation only. The target remains explicitly unverified
// until a full fresh catalog includes this exact ID and canonical session.
func (c *BotVisibilityManager) Reveal(k, id, session string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) || id == "" || session == "" {
		return false
	}
	e := c.entry(k)
	e.state.ShowHidden = true
	e.state.FocusID = id
	e.state.FocusSessionID = session
	e.state.Review = nil
	return true
}
func (c *BotVisibilityManager) BeginLoad(k string) (BotVisibilityTicket, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return BotVisibilityTicket{}, false
	}
	e := c.entry(k)
	if e.state.Busy {
		return BotVisibilityTicket{}, false
	}
	c.sequence++
	e.loading = c.sequence
	e.state.Busy = true
	e.state.Loaded = false
	e.state.Review = nil
	e.state.Candidate = nil
	e.state.Error = ""
	return BotVisibilityTicket{k, c.generation, c.sequence}, true
}
func (c *BotVisibilityManager) FinishLoad(t BotVisibilityTicket, v pluginrpc.DshbotBotCatalog, w bool, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
	if !c.valid(t.Scope) || t.Generation != c.generation || e == nil || e.loading != t.Sequence {
		return false
	}
	e.loading = 0
	e.state.Busy = false
	e.state.Writable = w
	if err != nil {
		e.state.Error = err.Error()
		return true
	}
	e.state.Catalog = pluginrpc.CloneDshbotBotCatalog(v)
	e.state.Loaded = true
	e.state.Error = ""
	if e.state.Uncertain && e.attempt != nil {
		row, matchErr := pluginrpc.DshbotVisibilityApplied(e.attempt.Command, v)
		if matchErr == nil {
			e.state.Candidate = &row
			e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotExactReadbackInspectAndAdopt)
		} else {
			e.state.Error = matchErr.Error()
			e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotStillUnknownNoRetry)
		}
	}
	return true
}
func (c *BotVisibilityManager) Prepare(k, id, session string, stamp int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return
	}
	e := c.entry(k)
	if e.state.Busy || !e.state.Loaded || !e.state.Writable || e.attempt != nil {
		return
	}
	e.state.Review = nil
	cmd := pluginrpc.DshbotVisibilityCommand{Expected: pluginrpc.CloneDshbotBotCatalog(e.state.Catalog), ID: id, SessionID: session, UpdatedAt: stamp}
	row, err := pluginrpc.DshbotVisibilityTarget(cmd)
	if err == nil {
		cmd.Hidden = !row.Hidden
		err = pluginrpc.ValidateDshbotVisibilityCommand(cmd)
	}
	if err != nil {
		e.state.Error = err.Error()
		return
	}
	c.sequence++
	e.state.Review = &BotVisibilityReview{Command: cmd, Sequence: c.sequence}
	e.state.Error = ""
}
func sameVisibilityReview(a, b *BotVisibilityReview) bool {
	return a != nil && b != nil && a.Sequence == b.Sequence && reflect.DeepEqual(a.Command, b.Command)
}
func (c *BotVisibilityManager) Acknowledge(k string, r *BotVisibilityReview, on bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(k) {
		e := c.entry(k)
		if !e.state.Busy && sameVisibilityReview(e.state.Review, r) {
			e.state.Review.Acknowledged = on
		}
	}
}
func (c *BotVisibilityManager) Cancel(k string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(k) {
		e := c.entry(k)
		if !e.state.Busy {
			e.state.Review = nil
		}
	}
}
func (c *BotVisibilityManager) Begin(k string, r *BotVisibilityReview) (BotVisibilityTicket, pluginrpc.DshbotVisibilityCommand, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var zero pluginrpc.DshbotVisibilityCommand
	if !c.valid(k) {
		return BotVisibilityTicket{}, zero, false
	}
	e := c.entry(k)
	if e.state.Busy || !e.state.Loaded || !e.state.Writable || e.attempt != nil || !sameVisibilityReview(e.state.Review, r) || !e.state.Review.Acknowledged || !pluginrpc.SameDshbotBotCatalog(e.state.Catalog, r.Command.Expected) {
		return BotVisibilityTicket{}, zero, false
	}
	cmd := pluginrpc.CloneDshbotVisibilityCommand(r.Command)
	c.sequence++
	e.attempt = &botVisibilityAttempt{Command: cmd, Sequence: c.sequence}
	e.state.Busy = true
	e.state.Review = nil
	e.state.Error = ""
	return BotVisibilityTicket{k, c.generation, c.sequence}, cmd, true
}
func (c *BotVisibilityManager) current(t BotVisibilityTicket) bool {
	e := c.entries[t.Scope]
	return c.valid(t.Scope) && c.generation == t.Generation && e != nil && e.attempt != nil && e.attempt.Sequence == t.Sequence && e.state.Busy
}
func (c *BotVisibilityManager) Current(t BotVisibilityTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current(t)
}
func (c *BotVisibilityManager) MarkSent(t BotVisibilityTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.current(t) || c.entries[t.Scope].attempt.Sent {
		return false
	}
	c.entries[t.Scope].attempt.Sent = true
	return true
}
func (c *BotVisibilityManager) AbortUnsent(t BotVisibilityTicket, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
	if e == nil || e.attempt == nil || e.attempt.Sequence != t.Sequence || e.attempt.Sent {
		return
	}
	e.attempt = nil
	e.state.Busy = false
	e.state.Loaded = false
	e.state.Review = nil
	e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotNotSentRefreshAndReview2)
	if err != nil {
		e.state.Error = err.Error()
	}
}
func (c *BotVisibilityManager) Finish(t BotVisibilityTicket, result pluginrpc.DshbotVisibilityResult, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
	if e == nil || e.attempt == nil || e.attempt.Sequence != t.Sequence || !e.attempt.Sent {
		return
	}
	e.state.Busy = false
	e.state.Loaded = false
	e.state.Review = nil
	if err != nil && result.Rejected {
		e.attempt = nil
		e.state.Uncertain = false
		e.state.Error = err.Error()
		e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotRejectedBeforeWrite)
		return
	}
	if err == nil {
		row, check := pluginrpc.DshbotVisibilityApplied(e.attempt.Command, result.Catalog)
		err = check
		if err == nil && !reflect.DeepEqual(row, result.Item) {
			err = fmt.Errorf("visibility receipt item differs")
		}
	}
	if err != nil {
		e.state.Uncertain = true
		e.state.Error = err.Error()
		e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotUnknownOutcomeNoRetry)
		return
	}
	e.state.Result = pluginrpc.CloneDshbotBotItem(result.Item)
	e.state.Catalog = pluginrpc.CloneDshbotBotCatalog(result.Catalog)
	e.state.Uncertain = false
	e.state.Error = ""
	e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotConfirmedVisibilityHiddenItemsRemain)
	e.state.ShowHidden = true
	e.attempt = nil
}
func (c *BotVisibilityManager) Adopt(k, id, session string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return false
	}
	e := c.entry(k)
	if e.state.Busy || !e.state.Uncertain || e.attempt == nil || e.state.Candidate == nil || e.state.Candidate.ID != id || e.state.Candidate.SessionID != session {
		return false
	}
	e.state.Result = pluginrpc.CloneDshbotBotItem(*e.state.Candidate)
	e.state.Candidate = nil
	e.state.Uncertain = false
	e.state.Error = ""
	e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotAdoptedExactReadbackNoResend)
	e.state.ShowHidden = true
	e.attempt = nil
	return true
}
