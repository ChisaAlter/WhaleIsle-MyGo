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
}

// ViewState is a UI-safe snapshot of the store.
type ViewState struct {
	ConnectionBusy bool
	ConnectionNote string
	WorkspaceDraft string
	ClientReady    bool
	Loading        bool
	Err            string
	HasIMRPC       bool
	ListMethod     string
	HistoryMethod  string
	Methods        []string
	Rows           []Entry
	RawListPreview string
	SelectedID     string
	LastSend       string
	Messages       []Message
	Draft          string
	LocalMode      bool   // true when no list RPC answered this session
	BridgeMode     string // "rpc" | "cordis" | "inventory" | "local"
	PluginDiag     string // pluginInventory / dsh-im connection diagnostics
	SyncedRemote   bool   // true when at least one remote history merge ran
}

// Store fetches im/* list/history/send when available and persists local
// channels / drafts / history under root (userData/im).
type Store struct {
	mu     sync.Mutex
	root   string
	client *dsh.Client
	plugin *pluginrpc.Client
	notify func()

	loading         bool
	err             error
	hasIMRPC        bool
	listMethod      string
	historyMethod   string
	methods         []string
	rows            []Entry
	rawListPreview  string
	selectedID      string
	lastSend        string
	drafts          map[string]string
	history         map[string][]Message
	bridgeMode      string
	pluginDiag      string
	syncedRemote    bool
	connectionBusy  bool
	connectionNote  string
	workspaceDrafts map[string]string
	connectionEpoch uint64
}

// New returns an in-memory IM store with no persistence root.
func New() *Store {
	return &Store{
		drafts:  map[string]string{},
		history: map[string][]Message{},
	}
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
		if err := s.saveChannelsLocked(); err != nil {
			return nil, err
		}
	}
	if s.selectedID == "" && len(s.rows) > 0 {
		s.selectedID = channelKey(s.rows[0])
	}
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
	s.connectionEpoch++
	s.connectionBusy = false
	s.connectionNote = ""
	s.client = client
	s.plugin = nil
	s.notify = notify
	s.err = nil
	if client != nil && strings.TrimSpace(client.Endpoint.Origin) != "" {
		s.plugin = &pluginrpc.Client{BaseURL: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTPClient: client.HTTP}
	}
	s.mu.Unlock()
}

// AttachPlugin binds a Cordis-style plugin HTTP client (POST /{channel}/connection.status).
func (s *Store) AttachPlugin(plugin *pluginrpc.Client) {
	s.mu.Lock()
	s.connectionEpoch++
	s.connectionBusy = false
	s.connectionNote = ""
	s.plugin = plugin
	s.mu.Unlock()
}

// Detach clears the client when harness stops; local channels/history stay.
func (s *Store) Detach() {
	s.mu.Lock()
	s.connectionEpoch++
	s.connectionBusy = false
	s.connectionNote = ""
	s.client = nil
	s.plugin = nil
	s.loading = false
	s.hasIMRPC = false
	s.listMethod = ""
	s.historyMethod = ""
	s.methods = nil
	s.rawListPreview = ""
	s.lastSend = ""
	s.bridgeMode = ""
	s.pluginDiag = ""
	s.syncedRemote = false
	s.err = nil
	// Keep rows, selectedID, drafts, history for offline UX.
	s.mu.Unlock()
	s.fire()
}

// Snapshot copies observable state for one frame.
func (s *Store) Snapshot() ViewState {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := ViewState{
		ConnectionBusy: s.connectionBusy,
		ConnectionNote: s.connectionNote,
		WorkspaceDraft: s.workspaceDraftLocked(s.selectedID),
		ClientReady:    s.client != nil,
		Loading:        s.loading,
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
		Draft:          s.drafts[s.selectedID],
	}
	if s.err != nil {
		st.Err = s.err.Error()
	}
	st.Methods = append([]string(nil), s.methods...)
	st.Rows = append([]Entry(nil), s.rows...)
	for i := range st.Rows {
		st.Rows[i].Presets = append([]Preset(nil), st.Rows[i].Presets...)
	}
	if msgs := s.history[s.selectedID]; len(msgs) > 0 {
		st.Messages = append([]Message(nil), msgs...)
	}
	return st
}

// Select marks a channel/conversation as active for compose and history.
// Current in-memory drafts are flushed to disk before the switch.
func (s *Store) Select(id string) {
	id = strings.TrimSpace(id)
	s.mu.Lock()
	if id == "" || id == s.selectedID {
		s.mu.Unlock()
		return
	}
	_ = s.saveDraftsLocked()
	s.selectedID = id
	s.mu.Unlock()
	s.fire()
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
	if strings.TrimSpace(text) == "" {
		delete(s.drafts, id)
	} else {
		s.drafts[id] = text
	}
}

// Persist flushes channels, drafts, and history to userData/im when rooted.
func (s *Store) Persist() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.saveChannelsLocked(); err != nil {
		return err
	}
	if err := s.saveDraftsLocked(); err != nil {
		return err
	}
	return s.saveHistoryLocked()
}

// SendDraft appends to local history (and clears the draft). When an im/send
// candidate answers it also calls the harness; otherwise LastSend notes local queue.
func (s *Store) SendDraft(ctx context.Context, channelID, text string) {
	channelID = strings.TrimSpace(channelID)
	text = strings.TrimSpace(text)
	if channelID == "" || text == "" {
		return
	}

	msg := Message{
		ID:        fmt.Sprintf("m-%d", time.Now().UnixNano()),
		ChannelID: channelID,
		Text:      text,
		At:        time.Now().UTC().Format(time.RFC3339),
		Outbound:  true,
	}

	s.mu.Lock()
	client := s.client
	sendMethod := s.pickSendMethodLocked()
	bridge := s.bridgeMode
	if s.history == nil {
		s.history = map[string][]Message{}
	}
	s.history[channelID] = append(s.history[channelID], msg)
	if s.drafts != nil {
		delete(s.drafts, channelID)
	}
	_ = s.saveHistoryLocked()
	_ = s.saveDraftsLocked()
	s.selectedID = channelID
	if client == nil || sendMethod == "" {
		s.lastSend = "saved locally (no im/send RPC): " + channelID
		if bridge == "cordis" {
			s.lastSend += " — Cordis only supports bot.reconnect+sendTest (fixed test ping), not arbitrary text"
		}
		s.mu.Unlock()
		s.fire()
		return
	}
	s.mu.Unlock()
	s.fire()

	go func() {
		args := map[string]any{
			"channelId": channelID,
			"text":      text,
		}
		var raw json.RawMessage
		err := client.Call(ctx, sendMethod, args, &raw)
		s.mu.Lock()
		if err != nil {
			s.lastSend = err.Error()
		} else {
			s.lastSend = "sent via " + sendMethod
		}
		s.mu.Unlock()
		s.fire()
	}()
}

func (s *Store) pickSendMethodLocked() string {
	if m := api.PreferredIMSendMethod(); m != "" {
		return m
	}
	set := map[string]bool{}
	for _, m := range s.methods {
		set[m] = true
	}
	for _, p := range api.IMSendMethodCandidates() {
		if set[p] {
			return p
		}
	}
	// Probe primary candidate even when not in registry (live Call).
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
	s.loading = true
	s.err = nil
	s.methods = append([]string(nil), regMethods...)
	s.rawListPreview = ""
	s.pluginDiag = ""
	s.bridgeMode = "local"
	s.mu.Unlock()
	s.fire()

	if client == nil && plugin == nil {
		s.finishRefresh(finishArgs{
			err:        fmt.Errorf("im: client not attached — local channels only"),
			bridgeMode: "local",
			pluginDiag: "no dsh client; cannot probe im/* or pluginInventory/list",
		})
		return
	}

	go func() {
		if client != nil {
			defer s.RefreshCordisInventory(ctx)
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
				err:           firstErr(listErr, histErr),
				rows:          rows,
				preview:       preview,
				replaceRows:   len(rows) > 0,
				hasRPC:        true,
				listMethod:    listMethod,
				historyMethod: histMethod,
				bridgeMode:    "rpc",
				remoteMsgs:    remoteMsgs,
				channelID:     histChannel,
			}
			if histMethod != "" {
				fa.syncedRemote = true
				fa.pluginDiag = fmt.Sprintf("im RPC: list=%s history=%s", listMethod, histMethod)
			} else {
				fa.pluginDiag = fmt.Sprintf("im RPC: list=%s (history candidates failed)", listMethod)
			}
			s.finishRefresh(fa)
			return
		}

		if plugin != nil {
			cordisRows, cordisPreview, cordisHits, cordisErr := probeCordisStatus(ctx, plugin)
			if len(cordisHits) > 0 {
				fa := finishArgs{
					err:          cordisErr,
					rows:         cordisRows,
					preview:      cordisPreview,
					replaceRows:  len(cordisRows) > 0,
					hasRPC:       true,
					listMethod:   "connection.status",
					bridgeMode:   "cordis",
					pluginDiag:   fmt.Sprintf("dsh-im Cordis: %s", strings.Join(cordisHits, "; ")),
					syncedRemote: true,
				}
				s.finishRefresh(fa)
				return
			}
		}

		if client == nil {
			s.finishRefresh(finishArgs{
				bridgeMode: "local",
				pluginDiag: "im/* empty; Cordis connection.status unanswered; no dsh client for pluginInventory",
			})
			return
		}

		// No im/* list — plugin inventory bridge for dsh-im diagnostics.
		diag, invErr := probePluginInventory(ctx, client)
		fa := finishArgs{
			bridgeMode: "inventory",
			pluginDiag: diag,
			hasRPC:     false,
			listMethod: "",
		}
		if invErr != nil {
			fa.err = invErr
			if fa.pluginDiag == "" {
				fa.pluginDiag = fmt.Sprintf("im/* unavailable (%v); pluginInventory/list failed: %v", listErr, invErr)
			}
		} else {
			if fa.pluginDiag == "" {
				fa.pluginDiag = "pluginInventory/list: dsh-im entry not found"
			}
			// Keep list probe note in diag only — not as hard Err (local mode OK).
			if listErr != nil {
				fa.pluginDiag = fa.pluginDiag + fmt.Sprintf(" | im/* probe: %v", listErr)
			}
		}
		s.finishRefresh(fa)
	}()
}

type finishArgs struct {
	err           error
	rows          []Entry
	preview       string
	replaceRows   bool
	hasRPC        bool
	listMethod    string
	historyMethod string
	bridgeMode    string
	pluginDiag    string
	remoteMsgs    []Message
	channelID     string
	syncedRemote  bool
}

func (s *Store) finishRefresh(a finishArgs) {
	s.mu.Lock()
	s.loading = false
	s.err = a.err
	s.rawListPreview = a.preview
	s.hasIMRPC = a.hasRPC
	s.listMethod = a.listMethod
	s.historyMethod = a.historyMethod
	s.bridgeMode = a.bridgeMode
	s.pluginDiag = a.pluginDiag
	if a.syncedRemote {
		s.syncedRemote = true
	}
	if a.replaceRows {
		if len(a.rows) > 0 {
			s.rows = a.rows
			_ = s.saveChannelsLocked()
			if s.selectedID == "" || !hasChannel(s.rows, s.selectedID) {
				s.selectedID = channelKey(s.rows[0])
			}
		}
	} else if len(s.rows) == 0 {
		s.rows = defaultChannels()
		_ = s.saveChannelsLocked()
		if s.selectedID == "" {
			s.selectedID = channelKey(s.rows[0])
		}
	}
	if len(a.remoteMsgs) > 0 && a.channelID != "" {
		if s.history == nil {
			s.history = map[string][]Message{}
		}
		s.history[a.channelID] = mergeMessages(s.history[a.channelID], a.remoteMsgs)
		_ = s.saveHistoryLocked()
	}
	s.mu.Unlock()
	s.fire()
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
		{ID: "general", Label: "通用", Detail: "本地频道"},
		{ID: "notes", Label: "笔记", Detail: "本地频道"},
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
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(bytesTrimSpace(data)) == 0 {
		return nil, nil
	}
	var out T
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("im: parse %s: %w", filepath.Base(path), err)
	}
	return &out, nil
}

func bytesTrimSpace(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		// Windows: rename over existing can fail; write directly as fallback.
		return os.WriteFile(path, data, 0o644)
	}
	return nil
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
