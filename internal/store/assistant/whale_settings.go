package assistant

import (
	"errors"
	"strings"
	"sync"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/store/storelocale"
)

// WhaleSettingsEditor holds only this native editor's in-memory drafts. It
// never reads local MEMORY.md, persists drafts, or starts model/session work.
type WhaleSettingsEditor struct {
	Preferences                     WhalePreferencesEditor
	mu                              sync.Mutex
	host                            string
	epoch, serial, revision         uint64
	pending                         uint64
	mutationSerial, mutationPending uint64
	loaded, seen, busy, confirm     bool
	server                          pluginrpc.WhaleSettings
	draft                           pluginrpc.WhaleSettings
	notice, lastErr                 string
	completedAction                 string
	completedSerial                 uint64
}
type WhaleSettingsSnapshot struct {
	HostKey                              string
	Loaded, Busy, ConfirmClear           bool
	Name, UserTitle, PersonaText, Memory string
	Server                               pluginrpc.WhaleSettings
	Notice, Error                        string
	Revision                             uint64
	CompletedAction                      string
	CompletedSerial                      uint64
}
type WhaleSettingsTicket struct {
	host                    string
	epoch, serial, revision uint64
}
type WhaleSettingsCommand struct {
	Action      string
	Base, Value pluginrpc.WhaleSettings
}

func (s *WhaleSettingsEditor) Bind(host string) {
	s.Preferences.Bind(host)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.host == host {
		return
	}
	s.epoch++
	s.host = host
	s.pending = 0
	s.mutationPending = 0
	s.loaded = false
	s.seen = false
	s.busy = false
	s.confirm = false
	s.server = pluginrpc.WhaleSettings{}
	s.draft = pluginrpc.WhaleSettings{}
	s.notice = ""
	s.lastErr = ""
	s.completedAction, s.completedSerial = "", 0
	s.revision++
}
func (s *WhaleSettingsEditor) Suspend() {
	s.Preferences.Suspend()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.epoch++
	s.pending = 0
	s.mutationPending = 0
	if s.busy {
		s.loaded = false
		s.notice = storelocale.T(storelocale.KeyStoreAssistantInterruptedRefreshToVerify)
	}
	s.busy = false
	s.confirm = false
}
func (s *WhaleSettingsEditor) Snapshot() WhaleSettingsSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return WhaleSettingsSnapshot{HostKey: s.host, Loaded: s.loaded, Busy: s.busy || s.mutationPending != 0, ConfirmClear: s.confirm, Name: s.draft.Name, UserTitle: s.draft.UserTitle, PersonaText: s.draft.PersonaText, Memory: s.draft.Memory, Server: s.server, Notice: s.notice, Error: s.lastErr, Revision: s.revision, CompletedAction: s.completedAction, CompletedSerial: s.completedSerial}
}

// BeginMutation serializes the identity/memory and model/skill writers for the
// same Host. Rebinding revokes only the old ticket, not a newer Host's writer.
func (s *WhaleSettingsEditor) BeginMutation(host string) (func(), bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if host == "" || host != s.host || s.busy || s.mutationPending != 0 {
		return nil, false
	}
	s.mutationSerial++
	s.mutationPending = s.mutationSerial
	epoch, serial := s.epoch, s.mutationPending
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.host == host && s.epoch == epoch && s.mutationPending == serial {
			s.mutationPending = 0
		}
	}, true
}
func (s *WhaleSettingsEditor) Edit(host, field, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if host == "" || s.host != host {
		return
	}
	switch field {
	case "memory":
		s.draft.Memory = value
	case "persona":
		s.draft.PersonaText = value
	case "name":
		s.draft.Name = value
	case "title":
		s.draft.UserTitle = value
	default:
		return
	}
	s.revision++
	s.confirm = false
	s.notice = ""
}
func (s *WhaleSettingsEditor) Restore(host, field string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if host != s.host || s.busy || !s.loaded {
		return
	}
	switch field {
	case "memory":
		s.draft.Memory = s.server.Memory
	case "persona":
		s.draft.PersonaText = s.server.PersonaText
	case "identity":
		s.draft.Name = s.server.Name
		s.draft.UserTitle = s.server.UserTitle
	default:
		return
	}
	s.revision++
	s.confirm = false
	s.lastErr = ""
	s.notice = storelocale.T(storelocale.KeyStoreAssistantDraftRestoredFromHost)
}
func (s *WhaleSettingsEditor) ticket() WhaleSettingsTicket {
	s.serial++
	s.pending = s.serial
	s.busy = true
	s.confirm = false
	s.lastErr = ""
	s.notice = ""
	return WhaleSettingsTicket{s.host, s.epoch, s.serial, s.revision}
}
func (s *WhaleSettingsEditor) matches(q WhaleSettingsTicket) bool {
	return q.host == s.host && q.epoch == s.epoch && q.serial == s.pending && s.busy
}
func (s *WhaleSettingsEditor) BeginLoad(host string) (WhaleSettingsTicket, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if host == "" || host != s.host || s.busy {
		return WhaleSettingsTicket{}, false
	}
	return s.ticket(), true
}
func (s *WhaleSettingsEditor) mergeRead(v pluginrpc.WhaleSettings) {
	if !s.seen || !s.server.SameIdentity(v) {
		s.draft = v
		s.revision++
		if s.seen {
			s.notice = storelocale.T(storelocale.KeyStoreAssistantWhaleIdentityChanged)
		}
	} else {
		if s.draft.Memory == s.server.Memory {
			s.draft.Memory = v.Memory
		}
		if s.draft.Name == s.server.Name {
			s.draft.Name = v.Name
		}
		if s.draft.UserTitle == s.server.UserTitle {
			s.draft.UserTitle = v.UserTitle
		}
		if s.draft.PersonaText == s.server.PersonaText {
			s.draft.PersonaText = v.PersonaText
		}
	}
	s.server = v
	s.loaded = true
	s.seen = true
}
func (s *WhaleSettingsEditor) FinishLoad(q WhaleSettingsTicket, v pluginrpc.WhaleSettings, err error) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.matches(q) {
		return false
	}
	s.busy = false
	s.pending = 0
	if err != nil {
		s.loaded = false
		s.lastErr = err.Error()
		return true
	}
	s.mergeRead(v)
	if s.notice == "" {
		s.notice = storelocale.T(storelocale.KeyStoreAssistantLoadedUnsavedDraftsPreserved)
	}
	return true
}
func (s *WhaleSettingsEditor) ReviewClear(host string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if host == s.host && s.loaded && !s.busy {
		s.confirm = true
		s.notice = ""
	}
}
func (s *WhaleSettingsEditor) CancelClear(host string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if host == s.host && !s.busy {
		s.confirm = false
	}
}
func (s *WhaleSettingsEditor) BeginWrite(host, action string) (WhaleSettingsTicket, WhaleSettingsCommand, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if host == "" || host != s.host || s.busy || !s.loaded {
		return WhaleSettingsTicket{}, WhaleSettingsCommand{}, false
	}
	cmd := WhaleSettingsCommand{action, s.server, s.draft}
	var err error
	switch action {
	case "clear":
		if !s.confirm {
			return WhaleSettingsTicket{}, WhaleSettingsCommand{}, false
		}
	case "memory":
		if pluginrpc.WhaleTextLength(cmd.Value.Memory) > 40000 {
			err = errors.New(storelocale.T(storelocale.KeyStoreAssistantMemoryExceeds40000Utf16Units))
		}
	case "identity":
		err = pluginrpc.ValidateWhaleSettingsPatch(pluginrpc.WhaleSettingsPatch{Name: &cmd.Value.Name, UserTitle: &cmd.Value.UserTitle})
	case "name":
		err = pluginrpc.ValidateWhaleSettingsPatch(pluginrpc.WhaleSettingsPatch{Name: &cmd.Value.Name})
	case "title":
		err = pluginrpc.ValidateWhaleSettingsPatch(pluginrpc.WhaleSettingsPatch{UserTitle: &cmd.Value.UserTitle})
	case "persona":
		err = pluginrpc.ValidateWhaleSettingsPatch(pluginrpc.WhaleSettingsPatch{PersonaText: &cmd.Value.PersonaText})
	default:
		return WhaleSettingsTicket{}, WhaleSettingsCommand{}, false
	}
	if err != nil {
		s.lastErr = err.Error()
		return WhaleSettingsTicket{}, WhaleSettingsCommand{}, false
	}
	return s.ticket(), cmd, true
}

// Check compares only the edited field, plus the canonical Whale identity.
// The upstream API has no CAS; this preflight detects known changes, not races.
func (cmd WhaleSettingsCommand) Check(current pluginrpc.WhaleSettings) error {
	if !cmd.Base.SameIdentity(current) {
		return errors.New(storelocale.T(storelocale.KeyStoreAssistantWhaleIdentityChangedRefresh))
	}
	same := false
	switch cmd.Action {
	case "memory", "clear":
		same = cmd.Base.Memory == current.Memory
	case "identity":
		same = cmd.Base.Name == current.Name && cmd.Base.UserTitle == current.UserTitle
	case "name":
		same = cmd.Base.Name == current.Name
	case "title":
		same = cmd.Base.UserTitle == current.UserTitle
	case "persona":
		same = cmd.Base.PersonaText == current.PersonaText
	}
	if !same {
		return errors.New(storelocale.T(storelocale.KeyStoreAssistantHostContentChangedRefreshAndCompare))
	}
	return nil
}
func (s *WhaleSettingsEditor) FinishWrite(q WhaleSettingsTicket, cmd WhaleSettingsCommand, actual *pluginrpc.WhaleSettings, accepted bool, err error) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.matches(q) {
		return false
	}
	s.busy = false
	s.pending = 0
	s.confirm = false
	if err != nil || actual == nil || !accepted {
		s.loaded = false
		if accepted {
			s.notice = storelocale.T(storelocale.KeyStoreAssistantHostConfirmedTheOperationRefreshTo)
		} else {
			s.notice = storelocale.T(storelocale.KeyStoreAssistantNotConfirmedDraftRetainedRefreshBef)
		}
		if err != nil {
			s.lastErr = err.Error()
		}
		return true
	}
	if !cmd.Base.SameIdentity(*actual) {
		s.loaded = false
		s.lastErr = storelocale.T(storelocale.KeyStoreAssistantWhaleIdentityChangedAfterSubmission)
		return true
	}
	matches := true
	switch cmd.Action {
	case "memory":
		matches = actual.Memory == cmd.Value.Memory
	case "persona":
		matches = actual.PersonaText == cmd.Value.PersonaText
	case "identity", "name", "title":
		name := strings.TrimSpace(cmd.Value.Name)
		if name == "" {
			name = storelocale.T(storelocale.KeyStoreAssistantWhale)
		}
		matches = (cmd.Action == "title" || actual.Name == name) && (cmd.Action == "name" || actual.UserTitle == strings.TrimSpace(cmd.Value.UserTitle))
	}
	if !matches {
		s.mergeWriteRead(*actual, cmd.Action, true)
		s.lastErr = storelocale.T(storelocale.KeyStoreAssistantHostAcceptedTheWriteButReadBackDi)
		return true
	}
	if q.revision == s.revision {
		switch cmd.Action {
		case "memory", "clear":
			s.draft.Memory = actual.Memory
		case "identity":
			s.draft.Name = actual.Name
			s.draft.UserTitle = actual.UserTitle
		case "name":
			s.draft.Name = actual.Name
		case "title":
			s.draft.UserTitle = actual.UserTitle
		case "persona":
			s.draft.PersonaText = actual.PersonaText
		}
	}
	s.mergeWriteRead(*actual, cmd.Action, q.revision != s.revision)
	s.lastErr = ""
	s.notice = storelocale.T(storelocale.KeyStoreAssistantSavedAndReadBackFromHost)
	s.completedAction = cmd.Action
	s.completedSerial++
	if cmd.Action == "clear" {
		s.notice = storelocale.T(storelocale.KeyStoreAssistantClearConfirmedByHostCurrentContents)
	}
	return true
}

// Current invalidates a captured write when the editor closes or owner changes.
func (s *WhaleSettingsEditor) Current(q WhaleSettingsTicket) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.matches(q)
}

// A newer edit can intentionally equal the previous server value. Do not let
// the usual clean-field refresh heuristic consume that edit after a write.
func (s *WhaleSettingsEditor) mergeWriteRead(actual pluginrpc.WhaleSettings, action string, preserve bool) {
	draft := s.draft
	s.mergeRead(actual)
	if !preserve {
		return
	}
	switch action {
	case "memory", "clear":
		s.draft.Memory = draft.Memory
	case "persona":
		s.draft.PersonaText = draft.PersonaText
	case "identity":
		s.draft.Name = draft.Name
		s.draft.UserTitle = draft.UserTitle
	case "name":
		s.draft.Name = draft.Name
	case "title":
		s.draft.UserTitle = draft.UserTitle
	}
}
