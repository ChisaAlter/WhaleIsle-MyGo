// Package im loads dsh-im RPC data when im/* endpoints respond; otherwise it
// keeps a local channel list and message history under userData/im/ and
// surfaces harness pluginInventory diagnostics for @xmanrui/dsh-im.
package im

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/state"
	"whaleisle/internal/store/imonboard"
	"whaleisle/internal/store/internal/storefs"
	officestore "whaleisle/internal/store/office"
	"whaleisle/internal/store/storelocale"
)

const (
	channelsFile = "channels.json"
	draftsFile   = "drafts.json"
	historyFile  = "history.json"

	defaultHistoryLimit = 50
)

// Dir returns userData/im.
func Dir(userData string) string {
	return filepath.Join(userData, "im")
}

// Entry is one row in the channel / conversation list.
type Entry struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Detail      string   `json:"detail,omitempty"`
	Channel     string   `json:"-"`
	BotID       string   `json:"-"`
	State       string   `json:"-"`
	Workspace   string   `json:"-"`
	AgentPreset string   `json:"-"`
	Presets     []Preset `json:"-"`
}

// Message is one history line for a channel (local and/or remote).
type Message struct {
	ID        string `json:"id"`
	ChannelID string `json:"channelId"`
	Text      string `json:"text"`
	At        string `json:"at"` // RFC3339
	Outbound  bool   `json:"outbound"`
	Remote    bool   `json:"remote,omitempty"` // true when merged from server history
	// State records what happened to an outbound message: pending while the
	// im/send RPC is in flight, sent/failed from its real result, local when no
	// send RPC exists, and unknown when the process ended mid-send. Empty means
	// the legacy history predates this field.
	State string `json:"state,omitempty"`
}

const (
	MessageStatePending = "pending"
	MessageStateSent    = "sent"
	MessageStateFailed  = "failed"
	MessageStateLocal   = "local"
	MessageStateUnknown = "unknown"
)

// ViewState is a UI-safe snapshot of the store. It is published immutable:
// readers share one version until the next write publishes. Fault fields are
// classified render-ready errors, never raw error text.
type ViewState struct {
	ConnectionBusy bool
	ConnectionNote string
	WorkspaceDraft string
	ClientReady    bool
	Loading        bool
	Err            *state.Fault
	HasIMRPC       bool
	ListMethod     string
	HistoryMethod  string
	Methods        []string
	Rows           []Entry
	RawListPreview string
	SelectedID     string
	LastSend       *state.Fault
	Messages       []Message
	Draft          string
	LocalMode      bool         // true when no list RPC answered this session
	BridgeMode     string       // "rpc" | "cordis" | "inventory" | "local"
	PluginDiag     *state.Fault // pluginInventory / dsh-im connection diagnostics
	SyncedRemote   bool         // true when at least one remote history merge ran
}

// Store fetches im/* list/history/send when available and persists local
// channels / drafts / history under root (userData/im). The mutable document
// stays under mu; readers get a published immutable version. Disk writes run
// outside mu, serialized by persistMu so the last write wins. conn owns the
// attach lifetime, refresh the probe lifetime: async completions holding a
// retired lease are dropped.
type Store struct {
	Office       officestore.Store
	Onboarding   imonboard.Store
	mu           sync.Mutex
	persistMu    sync.Mutex
	root         string
	client       *dsh.Client
	plugin       *pluginrpc.Client
	notify       func()
	pub          state.Published[ViewState]
	conn         state.LeaseOwner
	refresh      state.LeaseOwner
	connLease    state.Lease
	refreshLease state.Lease

	loading         bool
	fault           *state.Fault
	diagText        string
	hasIMRPC        bool
	listMethod      string
	historyMethod   string
	methods         []string
	rows            []Entry
	rawListPreview  string
	selectedID      string
	lastSend        *state.Fault
	drafts          map[string]string
	draftBuf        string
	history         map[string][]Message
	bridgeMode      string
	pluginDiag      *state.Fault
	syncedRemote    bool
	connectionBusy  bool
	connectionNote  string
	workspaceDrafts map[string]string
	// persistHook, when a test sets it, runs inside persistMu before disk
	// writes with a snapshot of the history map being written.
	persistHook func(what int, hist map[string][]Message)
}

// New returns an in-memory IM store with no persistence root.
func New() *Store {
	s := &Store{
		drafts:  map[string]string{},
		history: map[string][]Message{},
	}
	s.connLease = s.conn.Renew(context.Background())
	s.refreshLease = s.refresh.Renew(context.Background())
	return s
}

// Open loads (or seeds) persisted IM state under userData/im.
func Open(userData string) (*Store, error) {
	root := Dir(userData)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	s := &Store{
		root:    root,
		drafts:  map[string]string{},
		history: map[string][]Message{},
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	if len(s.rows) == 0 {
		s.rows = defaultChannels()
		if err := writeJSON(filepath.Join(s.root, channelsFile), s.rows); err != nil {
			return nil, err
		}
	}
	if s.selectedID == "" && len(s.rows) > 0 {
		s.selectedID = channelKey(s.rows[0])
	}
	s.draftBuf = s.drafts[s.selectedID]
	s.connLease = s.conn.Renew(context.Background())
	s.refreshLease = s.refresh.Renew(context.Background())
	s.publishLocked()
	return s, nil
}

// Root returns the persistence directory (may be empty for New()).
func (s *Store) Root() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.root
}

// Attach binds a dsh client and an optional UI refresh hook.
func (s *Store) Attach(client *dsh.Client, notify func()) {
	s.mu.Lock()
	dirtyHistory := s.invalidateRemoteLocked()
	s.connLease = s.conn.Renew(context.Background())
	s.connectionBusy = false
	s.connectionNote = ""
	s.client = client
	s.plugin = nil
	s.notify = notify
	s.fault = nil
	if client != nil && strings.TrimSpace(client.Endpoint.Origin) != "" {
		s.plugin = &pluginrpc.Client{BaseURL: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTPClient: client.HTTP}
	}
	s.Office.Bind(s.plugin, s.connLease.Generation(), s.fire)
	s.Onboarding.Bind(s.plugin, s.connLease.Generation(), s.fire)
	s.publishLocked()
	s.mu.Unlock()
	if dirtyHistory {
		s.persist(persistHistory)
	}
}

// AttachPlugin binds a Cordis-style plugin HTTP client (POST /{channel}/connection.status).
func (s *Store) AttachPlugin(plugin *pluginrpc.Client) {
	s.mu.Lock()
	dirtyHistory := s.invalidateRemoteLocked()
	s.connLease = s.conn.Renew(context.Background())
	s.connectionBusy = false
	s.connectionNote = ""
	s.plugin = plugin
	s.Office.Bind(s.plugin, s.connLease.Generation(), s.fire)
	s.Onboarding.Bind(s.plugin, s.connLease.Generation(), s.fire)
	s.publishLocked()
	s.mu.Unlock()
	if dirtyHistory {
		s.persist(persistHistory)
	}
}

// Detach clears the client when harness stops; local channels/history stay.
func (s *Store) Detach() {
	s.mu.Lock()
	dirtyHistory := s.invalidateRemoteLocked()
	s.connLease = s.conn.Renew(context.Background())
	s.connectionBusy = false
	s.connectionNote = ""
	s.client = nil
	s.plugin = nil
	s.Office.Bind(nil, s.connLease.Generation(), s.fire)
	s.Onboarding.Bind(nil, s.connLease.Generation(), s.fire)
	s.loading = false
	s.hasIMRPC = false
	s.listMethod = ""
	s.historyMethod = ""
	s.methods = nil
	s.rawListPreview = ""
	s.lastSend = nil
	s.bridgeMode = ""
	s.pluginDiag = nil
	s.diagText = ""
	s.syncedRemote = false
	s.fault = nil
	// Keep rows, selectedID, drafts, history for offline UX.
	s.publishLocked()
	s.mu.Unlock()
	if dirtyHistory {
		s.persist(persistHistory)
	}
	s.fire()
}

// publishLocked builds the immutable view of the current mutable state; the
// caller holds s.mu. SetDraft publishes only on an actual change so the
// per-frame binding call does not churn versions.
func (s *Store) publishLocked() {
	st := ViewState{
		ConnectionBusy: s.connectionBusy,
		ConnectionNote: s.connectionNote,
		WorkspaceDraft: s.workspaceDraftLocked(s.selectedID),
		ClientReady:    s.client != nil,
		Loading:        s.loading,
		Err:            s.fault,
		HasIMRPC:       s.hasIMRPC,
		ListMethod:     s.listMethod,
		HistoryMethod:  s.historyMethod,
		RawListPreview: s.rawListPreview,
		SelectedID:     s.selectedID,
		LastSend:       s.lastSend,
		LocalMode:      !s.hasIMRPC,
		BridgeMode:     s.bridgeMode,
		PluginDiag:     s.pluginDiag,
		SyncedRemote:   s.syncedRemote,
		Draft:          s.draftBuf,
	}
	st.Methods = append([]string(nil), s.methods...)
	st.Rows = append([]Entry(nil), s.rows...)
	for i := range st.Rows {
		st.Rows[i].Presets = append([]Preset(nil), st.Rows[i].Presets...)
	}
	if msgs := s.history[s.selectedID]; len(msgs) > 0 {
		st.Messages = append([]Message(nil), msgs...)
	}
	s.pub.Publish(st)
}

// Snapshot returns the published view version; unchanged reads share it.
func (s *Store) Snapshot() *state.Version[ViewState] {
	if v := s.pub.Load(); v != nil {
		return v
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pub.Load() == nil {
		s.publishLocked()
	}
	return s.pub.Load()
}

// Select marks a channel/conversation as active for compose and history.
// The outgoing channel's draft buffer is folded into the drafts map first.
func (s *Store) Select(id string) {
	id = strings.TrimSpace(id)
	s.mu.Lock()
	if id == "" || id == s.selectedID {
		s.mu.Unlock()
		return
	}
	s.syncDraftLocked()
	s.selectedID = id
	s.draftBuf = s.drafts[id]
	s.publishLocked()
	s.mu.Unlock()
	if err := s.persist(persistDrafts); err != nil {
		s.mu.Lock()
		s.fault = state.FaultOf(err)
		s.publishLocked()
		s.mu.Unlock()
	}
	s.fire()
}

// syncDraftLocked folds the live draft buffer into the drafts map for the
// selected channel. The caller holds s.mu.
func (s *Store) syncDraftLocked() {
	if s.selectedID == "" {
		return
	}
	text := strings.TrimRight(s.draftBuf, "\r\n")
	if s.drafts == nil {
		s.drafts = map[string]string{}
	}
	if strings.TrimSpace(text) == "" {
		delete(s.drafts, s.selectedID)
	} else {
		s.drafts[s.selectedID] = text
	}
}

// invalidateRemoteLocked retires in-flight probes and resets remote state.
// It reports whether pending history entries were settled and must persist.
func (s *Store) invalidateRemoteLocked() bool {
	s.refreshLease = s.refresh.Renew(context.Background())
	s.loading = false
	s.hasIMRPC = false
	s.listMethod = ""
	s.historyMethod = ""
	s.methods = nil
	s.rawListPreview = ""
	s.pluginDiag = nil
	s.diagText = ""
	s.bridgeMode = "local"
	s.syncedRemote = false
	changed := false
	for channel, messages := range s.history {
		for i := range messages {
			if messages[i].State == MessageStatePending {
				messages[i].State = MessageStateUnknown
				changed = true
			}
		}
		s.history[channel] = messages
	}
	return changed
}

// DraftBuffer is the live text binding for the selected channel's draft:
// the UI writes it directly, the store folds it into the drafts map at
// Select, Persist, SetDraft and SendDraft boundaries. It is the mutable
// exception to the published immutable view.
func (s *Store) DraftBuffer() *string {
	return &s.draftBuf
}

// SetDraft updates the in-memory draft for channelID (or the selected channel
// when channelID is empty). Call Persist to flush drafts to disk.
func (s *Store) SetDraft(channelID, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := strings.TrimSpace(channelID)
	if id == "" {
		id = s.selectedID
	}
	if id == "" {
		return
	}
	if s.drafts == nil {
		s.drafts = map[string]string{}
	}
	text = strings.TrimRight(text, "\r\n")
	changed := false
	if strings.TrimSpace(text) == "" {
		if _, ok := s.drafts[id]; ok {
			delete(s.drafts, id)
			changed = true
		}
	} else if s.drafts[id] != text {
		s.drafts[id] = text
		changed = true
	}
	if id == s.selectedID && s.draftBuf != text {
		s.draftBuf = text
		changed = true
	}
	if changed {
		s.publishLocked()
	}
}

// Persist flushes channels, drafts, and history to userData/im when rooted.
func (s *Store) Persist() error {
	s.mu.Lock()
	s.syncDraftLocked()
	s.mu.Unlock()
	return s.persist(persistChannels | persistDrafts | persistHistory)
}

const (
	persistChannels = 1 << iota
	persistDrafts
	persistHistory
)

// setPersistHook installs or clears the test persist hook under persistMu.
func (s *Store) setPersistHook(fn func(what int, hist map[string][]Message)) {
	s.persistMu.Lock()
	s.persistHook = fn
	s.persistMu.Unlock()
}

func cloneHistoryMap(src map[string][]Message) map[string][]Message {
	hist := make(map[string][]Message, len(src))
	for k, v := range src {
		hist[k] = append([]Message(nil), v...)
	}
	return hist
}

func setHistoryMessageState(hist map[string][]Message, channelID, id, msgState string) {
	for index := range hist[channelID] {
		if hist[channelID][index].ID == id {
			hist[channelID][index].State = msgState
			return
		}
	}
}

// persist serializes disk writes outside s.mu so writes stay ordered and the
// newest state wins: it re-reads the current document under a brief lock
// rather than persisting what an earlier caller captured.
func (s *Store) persist(what int) error {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	hook := s.persistHook
	s.mu.Lock()
	root := s.root
	var rows []Entry
	var drafts map[string]string
	var hist map[string][]Message
	if what&persistChannels != 0 {
		rows = append([]Entry(nil), s.rows...)
		for i := range rows {
			rows[i].Presets = append([]Preset(nil), rows[i].Presets...)
		}
	}
	if what&persistDrafts != 0 {
		drafts = map[string]string{}
		for k, v := range s.drafts {
			drafts[k] = v
		}
	}
	if what&persistHistory != 0 {
		hist = cloneHistoryMap(s.history)
	}
	s.mu.Unlock()
	if hook != nil {
		hook(what, hist)
	}
	if root == "" {
		return nil
	}
	if rows != nil {
		if err := writeJSON(filepath.Join(root, channelsFile), rows); err != nil {
			return err
		}
	}
	if drafts != nil {
		if err := writeJSON(filepath.Join(root, draftsFile), drafts); err != nil {
			return err
		}
	}
	if hist != nil {
		if err := writeJSON(filepath.Join(root, historyFile), hist); err != nil {
			return err
		}
	}
	return nil
}

// commitMessageState persists targetState for one message before updating
// s.history and publishing, keeping lock order persistMu -> s.mu. If the disk
// write fails, s.history stays pre-terminal, s.fault records saveErr, and
// subscribers never observe an unpersisted terminal state.
func (s *Store) commitMessageState(channelID, msgID, targetState string, valid func() bool, onCommitLocked func(), onFailLocked func(saveErr error)) (bool, error) {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	hook := s.persistHook
	s.mu.Lock()
	if valid != nil && !valid() {
		s.mu.Unlock()
		return false, nil
	}
	root := s.root
	hist := cloneHistoryMap(s.history)
	setHistoryMessageState(hist, channelID, msgID, targetState)
	s.mu.Unlock()
	if hook != nil {
		hook(persistHistory, hist)
	}
	if root != "" {
		if err := writeJSON(filepath.Join(root, historyFile), hist); err != nil {
			s.mu.Lock()
			if valid != nil && !valid() {
				s.mu.Unlock()
				return false, nil
			}
			s.fault = state.FaultOf(err)
			if onFailLocked != nil {
				onFailLocked(err)
			}
			s.publishLocked()
			s.mu.Unlock()
			return false, err
		}
	}
	s.mu.Lock()
	if valid != nil && !valid() {
		s.mu.Unlock()
		return false, nil
	}
	s.setMessageStateLocked(channelID, msgID, targetState)
	if onCommitLocked != nil {
		onCommitLocked()
	}
	s.publishLocked()
	s.mu.Unlock()
	return true, nil
}

// SendDraft appends one outbound message and calls the resolved im/send RPC.
// The message starts pending and only becomes sent after the RPC answers;
// without a send RPC it stays local. A failed call is marked failed, so the
// history never presents an undelivered message as delivered.
func (s *Store) SendDraft(ctx context.Context, channelID, text string) {
	channelID = strings.TrimSpace(channelID)
	text = strings.TrimSpace(text)
	if channelID == "" || text == "" {
		return
	}

	s.mu.Lock()
	client := s.client
	lease := s.connLease
	sendMethod := s.pickSendMethodLocked()
	msgState := MessageStateLocal
	if client != nil && sendMethod != "" {
		msgState = MessageStatePending
	}
	msg := Message{
		ID:        fmt.Sprintf("m-%d", time.Now().UnixNano()),
		ChannelID: channelID,
		Text:      text,
		At:        time.Now().UTC().Format(time.RFC3339),
		Outbound:  true,
		State:     msgState,
	}
	bridge := s.bridgeMode
	if s.history == nil {
		s.history = map[string][]Message{}
	}
	s.syncDraftLocked()
	s.history[channelID] = append(s.history[channelID], msg)
	delete(s.drafts, channelID)
	s.selectedID = channelID
	s.draftBuf = s.drafts[channelID]
	s.publishLocked()
	s.mu.Unlock()
	s.fire()

	// The pending entry persists before the RPC goes out: a crash after
	// dispatch never claims an undelivered message.
	if err := s.persist(persistHistory); err != nil {
		s.mu.Lock()
		msgs := s.history[channelID]
		if n := len(msgs); n > 0 && msgs[n-1].ID == msg.ID {
			s.history[channelID] = msgs[:n-1]
		}
		s.drafts[channelID] = text
		if channelID == s.selectedID {
			s.draftBuf = text
		}
		s.fault = state.FaultOf(err)
		s.lastSend = state.FaultOf(err)
		s.publishLocked()
		s.mu.Unlock()
		s.fire()
		return
	}
	if err := s.persist(persistDrafts); err != nil {
		draftFault := state.FaultOf(err)
		restoreDraft := func() {
			s.drafts[channelID] = text
			if channelID == s.selectedID {
				s.draftBuf = text
			}
			s.lastSend = draftFault
		}
		// The RPC is not dispatched when its local draft cannot be saved.
		// Retire the persisted pending entry before committing or publishing it.
		_, _ = s.commitMessageState(
			channelID,
			msg.ID,
			MessageStateFailed,
			nil,
			func() {
				restoreDraft()
				s.fault = draftFault
			},
			func(_ error) {
				restoreDraft()
			},
		)
		s.fire()
		return
	}
	if client == nil || sendMethod == "" {
		note := "saved locally (no im/send RPC): " + channelID
		if bridge == "cordis" {
			note += " — Cordis only supports bot.reconnect+sendTest (fixed test ping), not arbitrary text"
		}
		s.mu.Lock()
		s.lastSend = state.FaultText(note)
		s.publishLocked()
		s.mu.Unlock()
		s.fire()
		return
	}

	go func() {
		args := map[string]any{
			"channelId": channelID,
			"text":      text,
		}
		var raw json.RawMessage
		err := client.Call(ctx, sendMethod, args, &raw)
		targetState := MessageStateSent
		lastSend := state.FaultText("sent via " + sendMethod)
		if err != nil {
			targetState = MessageStateFailed
			lastSend = state.FaultOf(err)
		}
		committed, saveErr := s.commitMessageState(
			channelID,
			msg.ID,
			targetState,
			func() bool { return s.client == client && lease.Valid() },
			func() {
				s.lastSend = lastSend
			},
			func(saveErr error) {
				if err != nil {
					s.lastSend = lastSend
				} else {
					s.lastSend = state.FaultOf(saveErr)
				}
			},
		)
		if committed || saveErr != nil {
			s.fire()
		}
	}()
}

// setMessageStateLocked updates one message's delivery state in s.history.
// The caller holds s.mu; terminal state commits use commitMessageState so disk
// persistence completes before memory/view state advances.
func (s *Store) setMessageStateLocked(channelID, id, state string) {
	for index := range s.history[channelID] {
		if s.history[channelID][index].ID == id {
			s.history[channelID][index].State = state
			return
		}
	}
}

func (s *Store) pickSendMethodLocked() string {
	// An empty result means the current vendor has no registered im/send RPC;
	// SendDraft then remains in the auditable local queue/history path.
	return api.ResolveIMSendMethod()
}

// Refresh probes im/* list/history via Client.Call (candidates from api.im),
// merges remote history into local persistence, and when no im/* answers polls
// pluginInventory/list for @xmanrui/dsh-im connection diagnostics.
func (s *Store) Refresh(ctx context.Context) {
	regMethods := api.IMMethods()
	listCands := api.IMListMethodCandidates()
	histCands := api.IMHistoryMethodCandidates()

	s.mu.Lock()
	if s.connectionBusy {
		s.mu.Unlock()
		return
	}
	client := s.client
	plugin := s.plugin
	connLease := s.connLease
	s.refreshLease = s.refresh.Renew(ctx)
	refreshLease := s.refreshLease
	s.loading = true
	s.fault = nil
	s.methods = append([]string(nil), regMethods...)
	s.rawListPreview = ""
	s.pluginDiag = nil
	s.diagText = ""
	s.bridgeMode = "local"
	s.publishLocked()
	s.mu.Unlock()
	s.fire()
	finish := func(a finishArgs) {
		a.client, a.plugin, a.connLease, a.refreshLease = client, plugin, connLease, refreshLease
		s.finishRefresh(a)
	}

	if client == nil && plugin == nil {
		finish(finishArgs{
			fault:      state.FaultText("im: client not attached — local channels only"),
			bridgeMode: "local",
			pluginDiag: state.FaultText("no dsh client; cannot probe im/* or pluginInventory/list"),
		})
		return
	}

	go func() {
		if client != nil {
			defer s.refreshCordisInventory(ctx, client, connLease, refreshLease)
		}
		var listMethod string
		var rows []Entry
		var preview string
		var listErr error
		if client != nil {
			listMethod, rows, preview, listErr = probeList(ctx, client, listCands)
		}
		if listMethod != "" {
			s.mu.Lock()
			selected := s.selectedID
			s.mu.Unlock()
			// History follows a channel that will remain selected after row replace.
			histChannel := selected
			if len(rows) > 0 && (histChannel == "" || !hasChannel(rows, histChannel)) {
				histChannel = channelKey(rows[0])
			}

			histMethod, remoteMsgs, histErr := probeHistory(ctx, client, histCands, histChannel)
			fa := finishArgs{
				fault:         state.FaultOf(firstErr(listErr, histErr)),
				rows:          rows,
				preview:       preview,
				replaceRows:   true,
				hasRPC:        true,
				listMethod:    listMethod,
				historyMethod: histMethod,
				bridgeMode:    "rpc",
				remoteMsgs:    remoteMsgs,
				channelID:     histChannel,
			}
			if histMethod != "" {
				fa.syncedRemote = true
				fa.pluginDiag = state.FaultText(fmt.Sprintf("im RPC: list=%s history=%s", listMethod, histMethod))
			} else {
				fa.pluginDiag = state.FaultText(fmt.Sprintf("im RPC: list=%s (history candidates failed)", listMethod))
			}
			finish(fa)
			return
		}

		if plugin != nil {
			cordisRows, cordisPreview, cordisHits, cordisErr := probeCordisStatus(ctx, plugin)
			if len(cordisHits) > 0 {
				fa := finishArgs{
					fault:        state.FaultOf(cordisErr),
					rows:         cordisRows,
					preview:      cordisPreview,
					replaceRows:  true,
					hasRPC:       true,
					listMethod:   "connection.status",
					bridgeMode:   "cordis",
					pluginDiag:   state.FaultText(fmt.Sprintf("dsh-im Cordis: %s", strings.Join(cordisHits, "; "))),
					syncedRemote: true,
				}
				finish(fa)
				return
			}
		}

		if client == nil {
			finish(finishArgs{
				bridgeMode: "local",
				pluginDiag: state.FaultText("im/* empty; Cordis connection.status unanswered; no dsh client for pluginInventory"),
			})
			return
		}

		// No im/* list — plugin inventory bridge for dsh-im diagnostics.
		diag, invErr := probePluginInventory(ctx, client)
		fa := finishArgs{
			bridgeMode: "inventory",
			pluginDiag: state.FaultText(diag),
			hasRPC:     false,
			listMethod: "",
		}
		if invErr != nil {
			fa.fault = state.FaultOf(invErr)
			if fa.pluginDiag == nil {
				fa.pluginDiag = state.FaultOf(invErr)
			}
		} else {
			if fa.pluginDiag == nil {
				fa.pluginDiag = state.FaultText("pluginInventory/list: dsh-im entry not found")
			}
			// Keep list probe note in diag only — not as hard Err (local mode OK).
			if listErr != nil {
				if fa.pluginDiag != nil {
					fa.pluginDiag = state.FaultText(fa.pluginDiag.SentenceText() + " | im/* probe failed")
				} else {
					fa.pluginDiag = state.FaultOf(listErr)
				}
			}
		}
		finish(fa)
	}()
}

type finishArgs struct {
	client        *dsh.Client
	plugin        *pluginrpc.Client
	connLease     state.Lease
	refreshLease  state.Lease
	fault         *state.Fault
	rows          []Entry
	preview       string
	replaceRows   bool
	hasRPC        bool
	listMethod    string
	historyMethod string
	bridgeMode    string
	pluginDiag    *state.Fault
	remoteMsgs    []Message
	channelID     string
	syncedRemote  bool
}

func (s *Store) finishRefresh(a finishArgs) {
	s.mu.Lock()
	if s.client != a.client || s.plugin != a.plugin || !a.connLease.Valid() || !a.refreshLease.Valid() {
		s.mu.Unlock()
		return
	}
	s.loading = false
	s.fault = a.fault
	s.rawListPreview = a.preview
	s.hasIMRPC = a.hasRPC
	s.listMethod = a.listMethod
	s.historyMethod = a.historyMethod
	s.bridgeMode = a.bridgeMode
	s.pluginDiag = a.pluginDiag
	s.diagText = ""
	if a.pluginDiag != nil {
		s.diagText = a.pluginDiag.SentenceText()
	}
	if a.syncedRemote {
		s.syncedRemote = true
	}
	persistWhat := 0
	if a.replaceRows {
		s.rows = append([]Entry(nil), a.rows...)
		persistWhat |= persistChannels
		if len(s.rows) == 0 {
			s.selectedID = ""
		} else if s.selectedID == "" || !hasChannel(s.rows, s.selectedID) {
			s.selectedID = channelKey(s.rows[0])
		}
		s.draftBuf = s.drafts[s.selectedID]
	} else if len(s.rows) == 0 {
		s.rows = defaultChannels()
		persistWhat |= persistChannels
		if s.selectedID == "" {
			s.selectedID = channelKey(s.rows[0])
		}
	}
	if len(a.remoteMsgs) > 0 && a.channelID != "" {
		if s.history == nil {
			s.history = map[string][]Message{}
		}
		s.history[a.channelID] = mergeMessages(s.history[a.channelID], a.remoteMsgs)
		persistWhat |= persistHistory
	}
	s.publishLocked()
	s.mu.Unlock()
	if persistWhat != 0 {
		if err := s.persist(persistWhat); err != nil {
			s.mu.Lock()
			s.fault = firstFault(s.fault, state.FaultOf(err))
			s.publishLocked()
			s.mu.Unlock()
		}
	}
	s.fire()
}

func firstFault(faults ...*state.Fault) *state.Fault {
	for _, f := range faults {
		if f != nil {
			return f
		}
	}
	return nil
}

// diagNoteLocked appends a curated diagnostic note (IDs and status words
// only, never raw error text) and republishes PluginDiag.
func (s *Store) diagNoteLocked(note string) {
	if s.diagText != "" {
		s.diagText += " | "
	}
	s.diagText += note
	s.pluginDiag = state.FaultText(s.diagText)
}

// diagFaultLocked replaces PluginDiag with the classified form of a caught
// error; the raw text stays in logs, never in view state.
func (s *Store) diagFaultLocked(err error) {
	s.pluginDiag = state.FaultOf(err)
}

func firstErr(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

func probeList(ctx context.Context, client *dsh.Client, cands []string) (method string, rows []Entry, preview string, lastErr error) {
	for _, m := range cands {
		var raw json.RawMessage
		err := client.Call(ctx, m, nil, &raw)
		if err != nil {
			lastErr = err
			continue
		}
		rows, preview = parseIMList(raw)
		return m, rows, preview, nil
	}
	return "", nil, "", lastErr
}

func probeHistory(ctx context.Context, client *dsh.Client, cands []string, channelID string) (method string, msgs []Message, lastErr error) {
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		return "", nil, nil
	}
	args := map[string]any{
		"channelId": channelID,
		"limit":     defaultHistoryLimit,
	}
	for _, m := range cands {
		var raw json.RawMessage
		err := client.Call(ctx, m, args, &raw)
		if err != nil {
			lastErr = err
			continue
		}
		msgs = parseIMHistory(raw, channelID)
		return m, msgs, nil
	}
	return "", nil, lastErr
}

func probePluginInventory(ctx context.Context, client *dsh.Client) (string, error) {
	var snap api.PluginInventorySnapshot
	if err := client.Call(ctx, "pluginInventory/list", nil, &snap); err != nil {
		return "", err
	}
	var hits []string
	for _, e := range snap.Entries {
		if !api.IsDshIMInventoryEntry(e.ModuleName, e.EntryId) {
			continue
		}
		phase := e.FiberPhase
		if phase == "" {
			phase = "?"
		}
		en := "disabled"
		if e.Enabled {
			en = "enabled"
		}
		hits = append(hits, fmt.Sprintf("%s entryId=%s %s fiber=%s", e.ModuleName, e.EntryId, en, phase))
	}
	mgmt := "management=?"
	if snap.ManagementAvailable != nil {
		if *snap.ManagementAvailable {
			mgmt = "management=available"
		} else {
			mgmt = "management=unavailable"
		}
	}
	if len(hits) == 0 {
		return fmt.Sprintf("pluginInventory/list (%s): no %s entry among %d plugins — dsh-im may be unloaded or disabled",
			mgmt, api.DshIMModuleName, len(snap.Entries)), nil
	}
	return fmt.Sprintf("pluginInventory/list (%s): %s — channel RPC is plugin-internal (connection.status); no Typert im/history yet",
		mgmt, strings.Join(hits, "; ")), nil
}

func hasChannel(rows []Entry, id string) bool {
	for _, r := range rows {
		if channelKey(r) == id {
			return true
		}
	}
	return false
}

func channelKey(e Entry) string {
	if id := strings.TrimSpace(e.ID); id != "" {
		return id
	}
	return strings.TrimSpace(e.Label)
}

func defaultChannels() []Entry {
	return []Entry{
		{ID: "general", Label: storelocale.T(storelocale.KeyStoreImGeneral), Detail: storelocale.T(storelocale.KeyStoreImLocalChannels)},
		{ID: "notes", Label: storelocale.T(storelocale.KeyStoreImNotes), Detail: storelocale.T(storelocale.KeyStoreImLocalChannels)},
	}
}

func (s *Store) load() error {
	if s.root == "" {
		return nil
	}
	if rows, err := readJSON[[]Entry](filepath.Join(s.root, channelsFile)); err != nil {
		return err
	} else if rows != nil {
		s.rows = *rows
	}
	if drafts, err := readJSON[map[string]string](filepath.Join(s.root, draftsFile)); err != nil {
		return err
	} else if drafts != nil {
		s.drafts = *drafts
	}
	if hist, err := readJSON[map[string][]Message](filepath.Join(s.root, historyFile)); err != nil {
		return err
	} else if hist != nil {
		s.history = *hist
	}
	if s.drafts == nil {
		s.drafts = map[string]string{}
	}
	if s.history == nil {
		s.history = map[string][]Message{}
	}
	// A pending send cannot survive a restart: the RPC result was never
	// recorded, so say the outcome is unknown rather than claiming delivery.
	for channel, messages := range s.history {
		for index := range messages {
			if messages[index].State == MessageStatePending {
				messages[index].State = MessageStateUnknown
			}
		}
		s.history[channel] = messages
	}
	return nil
}

func (s *Store) saveChannelsLocked() error {
	if s.root == "" {
		return nil
	}
	return writeJSON(filepath.Join(s.root, channelsFile), s.rows)
}

func (s *Store) saveDraftsLocked() error {
	if s.root == "" {
		return nil
	}
	if s.drafts == nil {
		s.drafts = map[string]string{}
	}
	return writeJSON(filepath.Join(s.root, draftsFile), s.drafts)
}

func (s *Store) saveHistoryLocked() error {
	if s.root == "" {
		return nil
	}
	if s.history == nil {
		s.history = map[string][]Message{}
	}
	return writeJSON(filepath.Join(s.root, historyFile), s.history)
}

func readJSON[T any](path string) (*T, error) {
	var out T
	ok, err := storefs.ReadJSON("im", path, &out, true)
	if err != nil || !ok {
		return nil, err
	}
	return &out, nil
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return storefs.WriteFileAtomic(path, data, 0o644)
}

func parseIMList(raw json.RawMessage) ([]Entry, string) {
	preview := truncateJSON(string(raw), 360)
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil, preview
	}
	for _, key := range []string{"items", "conversations", "channels", "threads", "list"} {
		if chunk, ok := top[key]; ok {
			if rows := entriesFromArray(chunk); len(rows) > 0 {
				return rows, preview
			}
		}
	}
	if rows := entriesFromArray(raw); len(rows) > 0 {
		return rows, preview
	}
	return nil, preview
}

func parseIMHistory(raw json.RawMessage, channelID string) []Message {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err == nil {
		for _, key := range []string{"messages", "items", "history", "list"} {
			if chunk, ok := top[key]; ok {
				if msgs := messagesFromArray(chunk, channelID); len(msgs) > 0 {
					return msgs
				}
			}
		}
	}
	return messagesFromArray(raw, channelID)
}

func messagesFromArray(raw json.RawMessage, channelID string) []Message {
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil
	}
	out := make([]Message, 0, len(arr))
	for i, item := range arr {
		msg, ok := messageFromObject(item, channelID, i)
		if !ok {
			continue
		}
		out = append(out, msg)
	}
	return out
}

func messageFromObject(raw json.RawMessage, channelID string, index int) (Message, bool) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		text := truncateJSON(string(raw), 120)
		if text == "" {
			return Message{}, false
		}
		return Message{
			ID:        fmt.Sprintf("remote-%d", index+1),
			ChannelID: channelID,
			Text:      text,
			At:        time.Now().UTC().Format(time.RFC3339),
			Remote:    true,
		}, true
	}
	id := ""
	for _, k := range []string{"id", "messageId", "msgId"} {
		if v, ok := obj[k]; ok {
			id = jsonString(v)
			if id != "" {
				break
			}
		}
	}
	if id == "" {
		id = fmt.Sprintf("remote-%d", index+1)
	}
	text := ""
	for _, k := range []string{"text", "content", "body", "message"} {
		if v, ok := obj[k]; ok {
			text = jsonString(v)
			if text != "" {
				break
			}
		}
	}
	if text == "" {
		return Message{}, false
	}
	ch := channelID
	for _, k := range []string{"channelId", "conversationId"} {
		if v, ok := obj[k]; ok {
			if s := jsonString(v); s != "" {
				ch = s
				break
			}
		}
	}
	at := ""
	for _, k := range []string{"at", "createdAt", "timestamp", "time", "sentAt"} {
		if v, ok := obj[k]; ok {
			at = normalizeTime(jsonString(v), v)
			if at != "" {
				break
			}
		}
	}
	if at == "" {
		at = time.Now().UTC().Format(time.RFC3339)
	}
	outbound := false
	if v, ok := obj["outbound"]; ok {
		outbound = jsonBool(v)
	} else if v, ok := obj["fromMe"]; ok {
		outbound = jsonBool(v)
	} else if v, ok := obj["direction"]; ok {
		d := strings.ToLower(jsonString(v))
		outbound = d == "out" || d == "outbound" || d == "send" || d == "sent"
	}
	return Message{
		ID:        id,
		ChannelID: ch,
		Text:      text,
		At:        at,
		Outbound:  outbound,
		Remote:    true,
	}, true
}

func mergeMessages(local, remote []Message) []Message {
	byID := map[string]Message{}
	order := make([]string, 0, len(local)+len(remote))
	add := func(m Message) {
		id := strings.TrimSpace(m.ID)
		if id == "" {
			id = fmt.Sprintf("anon-%d-%s", len(order), m.At)
			m.ID = id
		}
		if prev, ok := byID[id]; ok {
			// Prefer remote text/flags when merging same id; keep earlier At if local older.
			merged := m
			if prev.At != "" && (merged.At == "" || prev.At < merged.At) {
				// keep chronological identity; still overwrite content from newer remote when Remote
			}
			if m.Remote {
				merged = m
				if merged.Text == "" {
					merged.Text = prev.Text
				}
			} else if !prev.Remote {
				merged = m
			} else {
				merged = prev
				if m.Text != "" {
					merged.Text = m.Text
				}
			}
			byID[id] = merged
			return
		}
		byID[id] = m
		order = append(order, id)
	}
	for _, m := range local {
		add(m)
	}
	for _, m := range remote {
		add(m)
	}
	out := make([]Message, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].At < out[j].At
	})
	return out
}

func normalizeTime(s string, raw json.RawMessage) string {
	s = strings.TrimSpace(s)
	if s == "" {
		var n float64
		if err := json.Unmarshal(raw, &n); err == nil && n > 0 {
			// ms vs s heuristic
			sec := int64(n)
			if n > 1e12 {
				sec = int64(n / 1000)
			}
			return time.Unix(sec, 0).UTC().Format(time.RFC3339)
		}
		return ""
	}
	if _, err := time.Parse(time.RFC3339, s); err == nil {
		return s
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC().Format(time.RFC3339)
	}
	return s
}

func jsonBool(raw json.RawMessage) bool {
	var b bool
	if err := json.Unmarshal(raw, &b); err == nil {
		return b
	}
	s := strings.ToLower(jsonString(raw))
	return s == "true" || s == "1" || s == "yes"
}

func entriesFromArray(raw json.RawMessage) []Entry {
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil
	}
	out := make([]Entry, 0, len(arr))
	for i, item := range arr {
		id, label, detail := idLabelDetailFromObject(item)
		if label == "" {
			label = fmt.Sprintf("#%d", i+1)
		}
		out = append(out, Entry{ID: id, Label: label, Detail: detail})
	}
	return out
}

func idLabelDetailFromObject(raw json.RawMessage) (id, label, detail string) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", truncateJSON(string(raw), 80), ""
	}
	for _, k := range []string{"id", "conversationId", "channelId"} {
		if v, ok := obj[k]; ok {
			id = jsonString(v)
			if id != "" {
				break
			}
		}
	}
	for _, k := range []string{"title", "name", "label"} {
		if v, ok := obj[k]; ok {
			label = jsonString(v)
			if label != "" {
				break
			}
		}
	}
	if label == "" {
		label = id
	}
	for _, k := range []string{"preview", "lastMessage", "status", "kind", "type"} {
		if v, ok := obj[k]; ok {
			detail = jsonString(v)
			if detail != "" {
				break
			}
		}
	}
	return id, label, detail
}

func jsonString(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return string(raw)
}

func truncateJSON(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func (s *Store) fire() {
	s.mu.Lock()
	fn := s.notify
	s.mu.Unlock()
	if fn != nil {
		fn()
	}
}
