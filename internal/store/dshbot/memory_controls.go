package dshbot

import (
	"fmt"
	"sync"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/store/storelocale"
)

type MemoryTicket struct {
	Scope      RoutineScope
	generation uint64
}
type MemoryDraft struct {
	Track, Input, BaseRevision, BaseText string
	Index                                int
	Remove                               bool
	Entries                              []string
	Version                              uint64
}
type MemoryReview struct {
	Command                pluginrpc.DshbotMemoryCommand
	Before                 string
	Sequence, DraftVersion uint64
	Remove                 bool
}
type MemorySnapshot struct {
	Generation                uint64
	Scope                     RoutineScope
	Loaded, Busy, Writable    bool
	BotName, SessionID, Track string
	Server                    pluginrpc.DshbotMemoryResult
	Draft                     *MemoryDraft
	Review                    *MemoryReview
	Error, Notice             string
}

// A separate memory state machine shares only the parent's Host/Bot lifecycle.
// Plaintext is transient local UI state and is never written to the roster cache.
type MemoryControls struct {
	mu                   sync.Mutex
	state                MemorySnapshot
	generation, sequence uint64
	attempt              *pluginrpc.DshbotMemoryCommand
}

func (c *MemoryControls) Bind(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Scope != k {
		c.generation++
		c.sequence++
		c.state = MemorySnapshot{Scope: k, Track: "bot"}
		c.attempt = nil
	}
}
func (c *MemoryControls) valid(k RoutineScope) bool {
	return k == c.state.Scope && k.HostKey != "" && k.BotID != ""
}
func (c *MemoryControls) Snapshot(k RoutineScope) MemorySnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Scope != k {
		return MemorySnapshot{Scope: k, Track: "bot"}
	}
	out := c.state
	out.Generation = c.generation
	out.Server = pluginrpc.CloneDshbotMemory(out.Server)
	if out.Draft != nil {
		d := *out.Draft
		d.Entries = append([]string(nil), d.Entries...)
		out.Draft = &d
	}
	if out.Review != nil {
		r := *out.Review
		out.Review = &r
	}
	return out
}
func (c *MemoryControls) current(t MemoryTicket) bool {
	return c.valid(t.Scope) && c.generation == t.generation && c.state.Busy
}
func (c *MemoryControls) ViewCurrent(k RoutineScope, generation uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.valid(k) && c.generation == generation
}
func (c *MemoryControls) Current(t MemoryTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current(t)
}
func (c *MemoryControls) BeginLoad(k RoutineScope) (MemoryTicket, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) || c.state.Busy {
		return MemoryTicket{}, false
	}
	c.generation++
	c.state.Busy = true
	c.state.Loaded = false
	c.state.Review = nil
	c.state.Error = ""
	return MemoryTicket{k, c.generation}, true
}
func (c *MemoryControls) FinishLoad(t MemoryTicket, bot pluginrpc.DshbotCatalogItem, v pluginrpc.DshbotMemoryResult, writable bool, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.current(t) {
		return false
	}
	c.generation++
	c.state.Busy = false
	if err == nil {
		err = pluginrpc.ValidateDshbotMemory(t.Scope.BotID, v)
	}
	if err == nil && (bot.ID != t.Scope.BotID || bot.Kind == "room" || bot.SessionID == "") {
		err = fmt.Errorf("select an available individual Bot")
	}
	if err != nil {
		c.state.Error = storelocale.T(storelocale.KeyStoreDshbotMemoryUnavailableDraftRetained)
		return true
	}
	if c.state.SessionID != "" && c.state.SessionID != bot.SessionID {
		c.state.Draft = nil
		c.attempt = nil
		c.state.Notice = storelocale.T(storelocale.KeyStoreDshbotBotIdentityChanged)
	}
	c.state.Server = pluginrpc.CloneDshbotMemory(v)
	c.state.SessionID = bot.SessionID
	c.state.BotName = bot.Name
	c.state.Loaded = true
	c.state.Writable = writable
	c.state.Error = ""
	if c.attempt != nil {
		actual := v.Track(c.attempt.Track)
		if actual.Text == c.attempt.Text {
			c.state.Draft = nil
			c.state.Notice = storelocale.T(storelocale.KeyStoreDshbotReadbackMatchesSubmittedContentNoRe)
		} else {
			c.state.Notice = storelocale.T(storelocale.KeyStoreDshbotRefreshedWithoutRetryingOrRebasingD)
		}
		c.attempt = nil
	}
	if d := c.state.Draft; d != nil && d.BaseRevision != v.Track(d.Track).Revision {
		c.state.Error = storelocale.T(storelocale.KeyStoreDshbotMemoryConflictNoAutomaticRebase)
	}
	return true
}
func (c *MemoryControls) SetTrack(k RoutineScope, track string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(k) && !c.state.Busy && c.state.Draft == nil && (track == "bot" || track == "user") {
		c.generation++
		c.state.Track = track
		c.state.Review = nil
	}
}
func (c *MemoryControls) start(k RoutineScope, index int, remove bool) bool {
	if !c.valid(k) || !c.state.Loaded || !c.state.Writable || c.state.Busy || c.state.Draft != nil {
		return false
	}
	v := c.state.Server.Track(c.state.Track)
	if index < -1 || index >= len(v.Entries) || remove && index < 0 {
		return false
	}
	c.sequence++
	d := &MemoryDraft{Track: c.state.Track, BaseRevision: v.Revision, BaseText: v.Text, Entries: append([]string(nil), v.Entries...), Index: index, Remove: remove, Version: c.sequence}
	if index >= 0 {
		d.Input = v.Entries[index]
	}
	c.generation++
	c.state.Draft = d
	c.state.Review = nil
	c.state.Error = ""
	return true
}
func (c *MemoryControls) StartEdit(k RoutineScope, index int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.start(k, index, false)
}
func (c *MemoryControls) Change(k RoutineScope, text string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(k) && !c.state.Busy && c.state.Draft != nil && !c.state.Draft.Remove {
		c.sequence++
		c.generation++
		c.state.Draft.Input = text
		c.state.Draft.Version = c.sequence
		c.state.Review = nil
	}
}
func (c *MemoryControls) Discard(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(k) && !c.state.Busy {
		c.generation++
		c.state.Draft = nil
		c.state.Review = nil
		c.state.Error = ""
	}
}
func (c *MemoryControls) Dismiss(k RoutineScope, expected *MemoryReview) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(k) && !c.state.Busy && expected != nil && c.state.Review != nil && expected.Sequence == c.state.Review.Sequence {
		c.generation++
		c.state.Review = nil
	}
}
func (c *MemoryControls) prepare(k RoutineScope) {
	d := c.state.Draft
	if !c.valid(k) || !c.state.Loaded || !c.state.Writable || c.state.Busy || d == nil {
		return
	}
	if d.BaseRevision != c.state.Server.Track(d.Track).Revision {
		c.state.Error = storelocale.T(storelocale.KeyStoreDshbotMemoryRevisionConflict)
		return
	}
	entries := append([]string(nil), d.Entries...)
	if d.Remove {
		entries = append(entries[:d.Index], entries[d.Index+1:]...)
	} else {
		if pluginrpc.DshbotMemoryText([]string{d.Input}) == "" {
			c.state.Error = storelocale.T(storelocale.KeyStoreDshbotEntryMustNotBeEmpty)
			return
		}
		if d.Index < 0 {
			entries = append(entries, d.Input)
		} else {
			entries[d.Index] = d.Input
		}
	}
	cmd := pluginrpc.DshbotMemoryCommand{BotID: k.BotID, SessionID: c.state.SessionID, Track: d.Track, MemoryRevision: d.BaseRevision, Text: pluginrpc.DshbotMemoryText(entries), Limit: c.state.Server.Limit(d.Track)}
	if err := pluginrpc.ValidateDshbotMemoryCommand(cmd); err != nil {
		c.state.Error = err.Error()
		return
	}
	if cmd.Text == d.BaseText {
		c.state.Error = storelocale.T(storelocale.KeyStoreDshbotNoMemoryChanges)
		return
	}
	c.sequence++
	c.generation++
	c.state.Review = &MemoryReview{Command: cmd, Before: d.BaseText, Sequence: c.sequence, DraftVersion: d.Version, Remove: d.Remove}
	c.state.Error = ""
}
func (c *MemoryControls) Prepare(k RoutineScope) { c.mu.Lock(); defer c.mu.Unlock(); c.prepare(k) }
func (c *MemoryControls) PrepareDelete(k RoutineScope, index int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.start(k, index, true) {
		c.prepare(k)
	}
}
func (c *MemoryControls) BeginSave(k RoutineScope, expected *MemoryReview) (MemoryTicket, pluginrpc.DshbotMemoryCommand, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var zero pluginrpc.DshbotMemoryCommand
	r := c.state.Review
	d := c.state.Draft
	if !c.valid(k) || !c.state.Loaded || !c.state.Writable || c.state.Busy || r == nil || expected == nil || d == nil || expected.Sequence != r.Sequence || r.DraftVersion != d.Version || r.Command.MemoryRevision != c.state.Server.Track(r.Command.Track).Revision || r.Command.SessionID != c.state.SessionID {
		return MemoryTicket{}, zero, false
	}
	c.generation++
	c.state.Busy = true
	c.state.Loaded = false
	cmd := r.Command
	c.attempt = &cmd
	return MemoryTicket{k, c.generation}, cmd, true
}
func (c *MemoryControls) FinishSave(t MemoryTicket, out pluginrpc.DshbotMemoryReceipt, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.current(t) || c.attempt == nil {
		return false
	}
	c.generation++
	c.state.Busy = false
	c.state.Review = nil
	cmd := *c.attempt
	if err == nil {
		err = pluginrpc.ValidateDshbotMemoryReceipt(cmd, out)
	}
	if err != nil {
		c.state.Error = storelocale.T(storelocale.KeyStoreDshbotSaveUncertainRefreshToVerify)
		return true
	}
	track := out.DshbotMemoryTrack
	track.Entries = append([]string(nil), track.Entries...)
	if cmd.Track == "user" {
		c.state.Server.User = track
	} else {
		c.state.Server.Bot = track
	}
	c.state.Loaded = true
	c.state.Draft = nil
	c.attempt = nil
	c.state.Error = ""
	c.state.Notice = storelocale.T(storelocale.KeyStoreDshbotHostConfirmedMemoryReplacement)
	return true
}
