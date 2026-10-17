// Package session holds workspace session list state backed by dsh RPC.
package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// Item is one row in the session tree.
type Item struct {
	SessionID string
	ParentID  string
	Title     string
	Cwd       string
	Running   bool
	Blank     bool
	UpdatedAt float64
	Pinned    bool
}

// MessageKind classifies how the chat UI should render a line.
type MessageKind string

const (
	MessageKindText         MessageKind = "text"
	MessageKindTool         MessageKind = "tool"
	MessageKindThinking     MessageKind = "thinking"
	MessageKindDeliverables MessageKind = "deliverables"
	MessageKindCompaction   MessageKind = "compaction"
)

// Message is one chat line from session/follow persistence or streaming.
type Message struct {
	ID        string
	Kind      MessageKind
	Role      string
	Body      string
	Seq       float64         // persistence seq when known (editMessageSeq)
	EventType string          // persistence type when Kind is tool / deliverables
	Raw       json.RawMessage // full persistence event JSON
	Streaming bool
	Files     []string // deliverables/presented paths
	// Model is the assistant source model when known (trajectory 模型 column).
	Model string
	// Args is tool-call arguments / structured payload for the inspector 参数 tab.
	Args string
	// Web is the typed presentation metadata carried by a web tool/result.
	// The persistence schema keeps tool metadata opaque; this projection is
	// populated only for the protocol-defined web search/fetch shapes.
	Web *WebResult
	// EventTime is harness event time (unix ms) when present.
	EventTime float64
}

// ViewState is a UI-safe snapshot of the store.
type ViewState struct {
	ClientReady bool
	ListLoading bool
	ListErr     string
	Items       []Item
	SelectedID  string
	Messages    []Message
	PromptBusy  bool
	LastAction  string
	// MessageRatings maps assistant message id → "up"|"down" for the current session.
	MessageRatings map[string]string
	// FeedbackBusyID is the message id with an in-flight messageFeedback/put (or "").
	FeedbackBusyID      string
	ModelLabels         []string
	SelectedModel       string
	ModelsErr           string
	PermissionLabels    []string
	SelectedPermission  string
	PermissionsErr      string
	AgentPresetLabels   []string
	SelectedAgentPreset string
	AgentPresetsErr     string
	SkillOptions        []SkillOption
	SkillsErr           string
	SkillHubHits        []SkillHubHit

	SearchLoading  bool
	SearchUseLocal bool
	SearchErr      string
	SearchRPCQuery string
	SearchHitIDs   map[string]bool

	// PlanBoard is the folded plan/goal/queue/todo strip from follow events.
	PlanBoard PlanBoard

	HistoryHasMore  bool
	HistoryPageBusy bool
	HistoryPageErr  string

	// PromptMode is the session/prompt wire mode: "queue" (排队) or "steer" (插话).
	PromptMode PromptMode
	// PinnedSessionIDs is workspace pin order (first = top).
	PinnedSessionIDs []string
	// ArchivedSessionIDs is the last workspace archive roster (for unarchive UI).
	ArchivedSessionIDs []string

	// BandActions/BandGeneration come from claudeCodeMods/watchBand when present.
	BandActions    []BandAction
	BandGeneration float64
	// SpeechSelection is the latest speech/follow catalog selection summary.
	SpeechSelection string
	// ControlNote is the latest session/control frame type (or short preview).
	ControlNote string
}

// Store tracks sessions for the active dsh client.
type Store struct {
	mu     sync.Mutex
	client *dsh.Client
	mux    *dsh.Mux
	notify func()

	listLoading              bool
	listErr                  error
	items                    []Item
	selectedID               string
	messages                 []Message
	promptBusy               bool
	lastAction               string
	messageRatings           map[string]string
	feedbackBusyID           string
	modelOptions             []ModelOption
	selectedModelLabel       string
	modelsErr                error
	permissionOptions        []PermissionOption
	selectedPermissionLabel  string
	permissionsErr           error
	agentPresetOptions       []AgentPresetOption
	selectedAgentPresetLabel string
	agentPresetsErr          error
	skillOptions             []SkillOption
	skillsErr                error
	skillHubHits             []SkillHubHit
	searchInput              string
	searchLoading            bool
	searchUseLocal           bool
	searchErr                error
	searchRPCQuery           string
	searchHits               map[string]struct{}
	searchSeq                uint64
	searchDebounce           *time.Timer
	permSettingsNS           string
	permSettingsRevision     float64
	permSettingsWritable     bool
	followCancel             context.CancelFunc
	workspaceFollowCancel    context.CancelFunc
	bandCancel               context.CancelFunc
	bandActions              []BandAction
	bandGeneration           float64
	speechFollowCancel       context.CancelFunc
	speechSelection          string
	controlCancel            context.CancelFunc
	controlNote              string
	cwd                      string
	workspaceID              string
	planBoard                PlanBoard
	historyHasMore           bool
	historyPageBusy          bool
	historyPageErr           error
	historyThroughSeq        float64
	promptMode               PromptMode
	pinnedSessionIDs         []string
	archivedSessionIDs       []string
	turnStartMs              float64
}

// New returns an empty store; call Attach when the harness endpoint is ready.
func New() *Store {
	return &Store{promptMode: PromptModeQueue}
}

// Attach binds a dsh client and an optional UI refresh hook.
func (s *Store) Attach(client *dsh.Client, notify func()) {
	s.mu.Lock()
	s.client = client
	s.notify = notify
	s.listErr = nil
	s.mu.Unlock()
}

// SetWorkspaceCwd sets the cwd used for session/create.
func (s *Store) SetWorkspaceCwd(cwd string) {
	s.mu.Lock()
	s.cwd = cwd
	s.mu.Unlock()
}

// CreateSession calls session/create then refreshes the list and selects the new id.
// Prefers session/blankReuse when a blank session already exists for the cwd
// (falls back to local Blank matching when the RPC rejects).
func (s *Store) CreateSession(ctx context.Context) {
	s.mu.Lock()
	client := s.client
	cwd := s.cwd
	var blankIDs []string
	for _, it := range s.items {
		if !it.Blank {
			continue
		}
		if cwd != "" && it.Cwd != "" && filepathClean(it.Cwd) != filepathClean(cwd) {
			continue
		}
		blankIDs = append(blankIDs, it.SessionID)
	}
	s.mu.Unlock()

	if client == nil {
		// Offline: Electron-compatible local blank reuse.
		if len(blankIDs) > 0 {
			s.Select(blankIDs[0])
			s.mu.Lock()
			s.lastAction = "blank session reused"
			s.mu.Unlock()
			s.fire()
		}
		return
	}
	for _, id := range blankIDs {
		var out api.SessionBlankReuseValue
		err := client.Call(ctx, "session/blankReuse", map[string]any{
			"request": map[string]any{"sessionId": id},
		}, &out)
		if err == nil && out.Reusable {
			s.Select(id)
			s.mu.Lock()
			s.lastAction = "blank session reused"
			s.mu.Unlock()
			s.fire()
			return
		}
		if err != nil {
			// RPC missing/unavailable: fall back to local blank reuse.
			s.Select(id)
			s.mu.Lock()
			s.lastAction = "blank session reused"
			s.mu.Unlock()
			s.fire()
			return
		}
	}
	go func() {
		req := api.SessionCreateRequest{}
		if cwd != "" {
			req.Cwd = &cwd
		}
		var out api.SessionCreateValue
		err := client.Call(ctx, "session/create", map[string]any{"request": req}, &out)
		s.mu.Lock()
		if err != nil {
			s.lastAction = err.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		s.selectedID = out.SessionId
		s.lastAction = "session created"
		s.mu.Unlock()
		s.fire()
		s.RefreshList(ctx)
		s.FollowSelected(ctx)
		s.InitializeDefaultModel(ctx)
	}()
}

func filepathClean(p string) string {
	return strings.TrimSpace(strings.ReplaceAll(p, "\\", "/"))
}

// Detach clears the client (harness stopped or lost).
func (s *Store) Detach() {
	s.mu.Lock()
	if s.followCancel != nil {
		s.followCancel()
		s.followCancel = nil
	}
	if s.workspaceFollowCancel != nil {
		s.workspaceFollowCancel()
		s.workspaceFollowCancel = nil
	}
	if s.bandCancel != nil {
		s.bandCancel()
		s.bandCancel = nil
	}
	s.bandActions = nil
	s.bandGeneration = 0
	if s.speechFollowCancel != nil {
		s.speechFollowCancel()
		s.speechFollowCancel = nil
	}
	s.speechSelection = ""
	if s.controlCancel != nil {
		s.controlCancel()
		s.controlCancel = nil
	}
	s.controlNote = ""
	if s.searchDebounce != nil {
		s.searchDebounce.Stop()
		s.searchDebounce = nil
	}
	s.clearSearchLocked()
	s.clearFeedbackLocked()
	s.client = nil
	s.mux = nil
	s.listLoading = false
	s.mu.Unlock()
	s.fire()
}

// Snapshot copies observable state for one frame.
func (s *Store) Snapshot() ViewState {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := ViewState{
		ClientReady:    s.client != nil,
		LastAction:     s.lastAction,
		ListLoading:    s.listLoading,
		SelectedID:     s.selectedID,
		PromptBusy:     s.promptBusy,
		FeedbackBusyID: s.feedbackBusyID,
	}
	if len(s.messageRatings) > 0 {
		st.MessageRatings = make(map[string]string, len(s.messageRatings))
		for k, v := range s.messageRatings {
			st.MessageRatings[k] = v
		}
	}
	if s.listErr != nil {
		st.ListErr = s.listErr.Error()
	}
	if s.modelsErr != nil {
		st.ModelsErr = s.modelsErr.Error()
	}
	if s.permissionsErr != nil {
		st.PermissionsErr = s.permissionsErr.Error()
	}
	if s.agentPresetsErr != nil {
		st.AgentPresetsErr = s.agentPresetsErr.Error()
	}
	if s.skillsErr != nil {
		st.SkillsErr = s.skillsErr.Error()
	}
	st.Items = append([]Item(nil), s.items...)
	st.Messages = append([]Message(nil), s.messages...)
	st.SelectedModel = s.selectedModelLabel
	for _, o := range s.modelOptions {
		st.ModelLabels = append(st.ModelLabels, o.Label)
	}
	st.SelectedPermission = s.selectedPermissionLabel
	for _, o := range s.permissionOptions {
		st.PermissionLabels = append(st.PermissionLabels, o.Label)
	}
	st.SelectedAgentPreset = s.selectedAgentPresetLabel
	for _, o := range s.agentPresetOptions {
		st.AgentPresetLabels = append(st.AgentPresetLabels, o.Label)
	}
	st.SkillOptions = append([]SkillOption(nil), s.skillOptions...)
	st.SkillHubHits = append([]SkillHubHit(nil), s.skillHubHits...)
	st.SearchLoading = s.searchLoading
	st.SearchUseLocal = s.searchUseLocal
	st.SearchRPCQuery = s.searchRPCQuery
	if s.searchErr != nil {
		st.SearchErr = s.searchErr.Error()
	}
	if len(s.searchHits) > 0 {
		st.SearchHitIDs = make(map[string]bool, len(s.searchHits))
		for id := range s.searchHits {
			st.SearchHitIDs[id] = true
		}
	}
	st.PlanBoard = clonePlanBoard(s.planBoard)
	st.HistoryHasMore = s.historyHasMore
	st.HistoryPageBusy = s.historyPageBusy
	if s.historyPageErr != nil {
		st.HistoryPageErr = s.historyPageErr.Error()
	}
	st.PromptMode = NormalizePromptMode(string(s.promptMode))
	if len(s.pinnedSessionIDs) > 0 {
		st.PinnedSessionIDs = append([]string(nil), s.pinnedSessionIDs...)
	}
	if len(s.archivedSessionIDs) > 0 {
		st.ArchivedSessionIDs = append([]string(nil), s.archivedSessionIDs...)
	}
	st.BandGeneration = s.bandGeneration
	st.BandActions = append([]BandAction(nil), s.bandActions...)
	st.SpeechSelection = s.speechSelection
	st.ControlNote = s.controlNote
	return st
}

func clonePlanBoard(b PlanBoard) PlanBoard {
	out := PlanBoard{PlanMode: b.PlanMode}
	if b.Goal != nil {
		g := *b.Goal
		out.Goal = &g
	}
	if len(b.Todos) > 0 {
		out.Todos = append([]PlanTodo(nil), b.Todos...)
	}
	if len(b.Queue) > 0 {
		out.Queue = append([]PlanQueueItem(nil), b.Queue...)
	}
	return out
}

// RunningCount returns how many workspace sessions are marked running.
func (s *Store) RunningCount() int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, it := range s.items {
		if it.Running {
			n++
		}
	}
	return n
}

// Select chooses the active session for prompt/cancel/follow.
func (s *Store) Select(sessionID string) {
	s.mu.Lock()
	if s.selectedID == sessionID {
		s.mu.Unlock()
		return
	}
	s.selectedID = sessionID
	s.clearFeedbackLocked()
	s.mu.Unlock()
	s.fire()
	s.FollowSelected(context.Background())
	s.WatchBand(context.Background())
	s.WatchControl(context.Background())
	s.RefreshProjections(context.Background())
	s.RefreshGoal(context.Background())
	s.RefreshSkills(context.Background())
	s.RefreshFeedback(context.Background())
}

// RefreshList loads session/list on a background goroutine when a client is set.
func (s *Store) RefreshList(ctx context.Context) {
	s.mu.Lock()
	client := s.client
	if client == nil {
		s.mu.Unlock()
		return
	}
	s.listLoading = true
	s.listErr = nil
	s.mu.Unlock()
	s.fire()

	go func() {
		var out api.SessionListValue
		err := client.Call(ctx, "session/list", map[string]any{
			"_request": api.SessionListRequest{},
		}, &out)

		s.mu.Lock()
		s.listLoading = false
		if err != nil {
			s.listErr = err
			s.mu.Unlock()
			s.fire()
			return
		}
		items := make([]Item, 0, len(out.Items))
		for _, it := range out.Items {
			items = append(items, itemFromListRow(it))
		}
		s.items = items
		s.applyPinnedFlagsLocked()
		startFollow := false
		if s.selectedID == "" && len(items) > 0 {
			s.selectedID = items[0].SessionID
			startFollow = true
		}
		s.mu.Unlock()
		s.fire()
		if startFollow {
			s.FollowSelected(context.Background())
		}
	}()
}

// SendPrompt posts session/prompt for the selected session (mode queue).
func (s *Store) SendPrompt(ctx context.Context, text string) {
	s.SendPromptEdit(ctx, text, 0)
}

// SendPromptEdit is SendPrompt with optional editMessageSeq (0 = new prompt).
func (s *Store) SendPromptEdit(ctx context.Context, text string, editSeq float64) {
	text = trimPrompt(text)
	if text == "" {
		return
	}
	s.mu.Lock()
	client := s.client
	sessionID := s.selectedID
	mode := NormalizePromptMode(string(s.promptMode))
	if client == nil || sessionID == "" {
		s.mu.Unlock()
		return
	}
	s.promptBusy = true
	s.mu.Unlock()
	s.fire()

	go func() {
		req := api.SessionPromptRequest{
			RequestId: newRequestID(),
			SessionId: sessionID,
			Mode:      string(mode),
			Content:   PromptContentBlocks(text),
		}
		if editSeq > 0 {
			seq := editSeq
			req.EditMessageSeq = &seq
		}
		var promptResult json.RawMessage
		err := client.Call(ctx, "session/prompt", map[string]any{"request": req}, &promptResult)

		s.mu.Lock()
		s.promptBusy = false
		if err != nil {
			s.lastAction = err.Error()
		} else if editSeq > 0 {
			s.lastAction = "prompt edited"
			for i := range s.messages {
				if s.messages[i].Seq == editSeq && s.messages[i].Role == "user" {
					s.messages[i].Body = text
					break
				}
			}
		} else {
			s.lastAction = "prompt sent"
			s.messages = append(s.messages, Message{
				ID:   newRequestID(),
				Role: "user",
				Body: text,
			})
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// RenameSession sets session/rename and updates the local list title.
func (s *Store) RenameSession(ctx context.Context, sessionID, title string) {
	title = strings.TrimSpace(title)
	if title == "" || sessionID == "" {
		return
	}
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		return
	}
	go func() {
		req := api.SessionRenameRequest{SessionId: sessionID, Title: title}
		var out api.SessionRenameValue
		err := client.Call(ctx, "session/rename", map[string]any{"request": req}, &out)
		s.mu.Lock()
		if err != nil {
			s.lastAction = err.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		for i := range s.items {
			if s.items[i].SessionID == sessionID {
				s.items[i].Title = out.Title
				break
			}
		}
		s.lastAction = "session renamed"
		s.mu.Unlock()
		s.fire()
		s.SetPresentation(ctx, sessionID, out.Title)
	}()
}

// SetPresentation calls session/setPresentation (Electron chat chrome title owner).
func (s *Store) SetPresentation(ctx context.Context, sessionID, title string) {
	sessionID = strings.TrimSpace(sessionID)
	title = strings.TrimSpace(title)
	s.mu.Lock()
	client := s.client
	if sessionID == "" {
		sessionID = s.selectedID
	}
	if title == "" {
		for _, it := range s.items {
			if it.SessionID == sessionID {
				title = it.Title
				break
			}
		}
	}
	s.mu.Unlock()
	if client == nil || sessionID == "" {
		return
	}
	go func() {
		req := api.SessionPresentationRequest{
			SessionId: sessionID,
			Presentation: api.SessionPresentationRequestPresentation{
				Owner: "whaleisle",
				Title: title,
			},
		}
		var out api.SessionPresentationValue
		err := client.Call(ctx, "session/setPresentation", map[string]any{"request": req}, &out)
		s.mu.Lock()
		if err != nil {
			// Non-fatal: rename already applied; keep lastAction unless empty.
			if s.lastAction == "" {
				s.lastAction = err.Error()
			}
		}
		s.mu.Unlock()
		if err == nil {
			s.fire()
		}
	}()
}

// OpenWorkspacePath calls session/openWorkspacePath (Electron open-in-app).
// action is optional (e.g. "reveal"); application is an id from workspacePathApplications.
func (s *Store) OpenWorkspacePath(ctx context.Context, path, action, application string) {
	path = strings.TrimSpace(path)
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil || path == "" {
		return
	}
	go func() {
		var can api.CanOpenWorkspacePathResult
		if err := client.Call(ctx, "session/canOpenWorkspacePath", map[string]any{}, &can); err == nil && !can {
			s.mu.Lock()
			s.lastAction = "workspace path open unavailable"
			s.mu.Unlock()
			s.fire()
			return
		}
		req := api.SessionOpenWorkspacePathRequest{Path: path}
		if a := strings.TrimSpace(action); a != "" {
			req.Action = &a
		}
		if app := strings.TrimSpace(application); app != "" {
			req.Application = &app
		}
		var out api.SessionOpenWorkspacePathValue
		err := client.Call(ctx, "session/openWorkspacePath", map[string]any{"request": req}, &out)
		s.mu.Lock()
		if err != nil {
			s.lastAction = err.Error()
		} else if out.Opened {
			s.lastAction = "workspace path opened"
		} else {
			s.lastAction = "workspace path not opened"
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// ListWorkspacePathApplications calls session/workspacePathApplications.
func (s *Store) ListWorkspacePathApplications(ctx context.Context, path string) ([]api.WorkspacePathApplicationsResultItem, error) {
	path = strings.TrimSpace(path)
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil || path == "" {
		return nil, nil
	}
	var out api.WorkspacePathApplicationsResult
	err := client.Call(ctx, "session/workspacePathApplications", map[string]any{
		"request": api.WorkspacePathApplicationsRequest{Path: path},
	}, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ForkSession calls session/fork then selects the new session id.
func (s *Store) ForkSession(ctx context.Context, sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil || sessionID == "" {
		return
	}
	go func() {
		req := api.SessionForkRequest{SessionId: sessionID}
		var out struct {
			SessionId string `json:"sessionId"`
		}
		err := client.Call(ctx, "session/fork", map[string]any{"request": req}, &out)
		s.mu.Lock()
		if err != nil {
			s.lastAction = err.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		if out.SessionId != "" {
			s.selectedID = out.SessionId
		}
		s.lastAction = "session forked"
		s.mu.Unlock()
		s.fire()
		s.RefreshList(ctx)
		s.FollowSelected(ctx)
	}()
}

// CancelRunning calls session/cancel for the selected session.
func (s *Store) CancelRunning(ctx context.Context) {
	s.mu.Lock()
	sessionID := s.selectedID
	s.mu.Unlock()
	s.CancelSession(ctx, sessionID)
}

// CancelSession calls session/cancel for an explicit session id (agents panel stop).
func (s *Store) CancelSession(ctx context.Context, sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil || sessionID == "" {
		return
	}
	go func() {
		var out api.SessionCancelValue
		err := client.Call(ctx, "session/cancel", map[string]any{
			"request": map[string]string{"sessionId": sessionID},
		}, &out)
		s.mu.Lock()
		if err != nil {
			s.lastAction = err.Error()
		} else if out.Accepted {
			s.lastAction = "cancel accepted"
		} else {
			s.lastAction = "cancel rejected"
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// InterruptChild calls subagents/interruptByParent for a child under parent.
// mode is the harness InterruptByParentMode wire string (e.g. "cancel").
func (s *Store) InterruptChild(ctx context.Context, parentID, childID, mode string) {
	parentID = strings.TrimSpace(parentID)
	childID = strings.TrimSpace(childID)
	mode = strings.TrimSpace(mode)
	if mode == "" {
		mode = "cancel"
	}
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil || parentID == "" || childID == "" {
		return
	}
	go func() {
		var out json.RawMessage
		err := client.Call(ctx, "subagents/interruptByParent", map[string]any{
			"childSessionId":  childID,
			"parentSessionId": parentID,
			"mode":            mode,
		}, &out)
		s.mu.Lock()
		if err != nil {
			s.lastAction = err.Error()
		} else {
			s.lastAction = "interrupt accepted"
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// PromptChild posts subagents/prompt to a child under parent (agents panel).
func (s *Store) PromptChild(ctx context.Context, parentID, childID, text string) {
	parentID = strings.TrimSpace(parentID)
	childID = strings.TrimSpace(childID)
	text = trimPrompt(text)
	if parentID == "" || childID == "" || text == "" {
		return
	}
	s.mu.Lock()
	client := s.client
	mode := string(NormalizePromptMode(string(s.promptMode)))
	s.mu.Unlock()
	if client == nil {
		return
	}
	go func() {
		req := api.SubagentPromptRequest{
			RequestId:       newRequestID(),
			ParentSessionId: parentID,
			ChildSessionId:  childID,
			Mode:            mode,
			Delivery:        "session",
			Content:         PromptContentBlocks(text),
		}
		var out api.SubagentPromptReceipt
		err := client.Call(ctx, "subagents/prompt", map[string]any{"request": req}, &out)
		s.mu.Lock()
		if err != nil {
			s.lastAction = err.Error()
		} else {
			s.lastAction = "subagent prompt sent"
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// AppendStreamDelta is a hook for future session/follow streaming.
func (s *Store) AppendStreamDelta(role, delta string) {
	if delta == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if n := len(s.messages); n > 0 && s.messages[n-1].Streaming && s.messages[n-1].Role == role {
		s.messages[n-1].Body += delta
	} else {
		s.messages = append(s.messages, Message{
			ID:        newRequestID(),
			Role:      role,
			Body:      delta,
			Streaming: true,
		})
	}
}

// SetLastAction records a local-only status line for the chat/settings chrome.
func (s *Store) SetLastAction(msg string) {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return
	}
	s.mu.Lock()
	s.lastAction = msg
	notify := s.notify
	s.mu.Unlock()
	if notify != nil {
		notify()
	}
}

func (s *Store) fire() {
	s.mu.Lock()
	fn := s.notify
	s.mu.Unlock()
	if fn != nil {
		fn()
	}
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func trimPrompt(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == ' ') {
		s = s[:len(s)-1]
	}
	for len(s) > 0 && (s[0] == '\n' || s[0] == '\r' || s[0] == ' ') {
		s = s[1:]
	}
	return s
}
