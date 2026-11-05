// Package session holds workspace session list state backed by dsh RPC.
package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/sessioncost"
	"whaleisle/internal/workflowrun"
)

// Item is one row in the session tree.
type Item struct {
	Presentation          *SessionPresentation
	presentationKnown     bool
	presentationKind      string
	presentationSeq       float64
	SubagentIdentity      *SessionSubagentIdentity
	subagentIdentityKnown bool
	subagentIdentityKind  string
	subagentIdentitySeq   float64
	CompletionUnread      bool
	AgentMetrics          AgentMetrics
	BilledUsage           *sessioncost.Projection
	BilledUsageSeq        float64
	BilledUsageKind       string

	SessionID    string
	ParentID     string
	Title        string
	Cwd          string
	Running      bool
	ContactError string
	Blank        bool
	UpdatedAt    float64
	Pinned       bool
	// Origin identifies durable subagents; a removed live Agent does not delete them.
	Origin         string
	AgentAvailable bool
	FormatStatus   string
}

// MessageKind classifies how the chat UI should render a line.
type MessageKind string

const (
	MessageKindText         MessageKind = "text"
	MessageKindImage        MessageKind = "image"
	MessageKindUnknown      MessageKind = "unknown"
	MessageKindTool         MessageKind = "tool"
	MessageKindThinking     MessageKind = "thinking"
	MessageKindDeliverables MessageKind = "deliverables"
	MessageKindCompaction   MessageKind = "compaction"
)

// Message is one chat line from session/follow persistence or streaming.
type Message struct {
	// ContextWaking is a drawing fact derived from the loaded Inbox claims.
	ContextWaking       bool
	AutomaticCompaction *ManualCompactionView
	Retry               *RetryStatus
	TurnDelivery        *TurnDelivery
	TrajectoryTime      *TrajectoryTiming
	Workflow            *workflowrun.Run
	Submission          *PendingSubmission
	// Tool is populated only by the read-only chat lifecycle projection.
	Tool    *ToolCallView
	Command *CommandView
	ID      string
	// SourceMessageID is the durable identity shared by projected assistant rows.
	SourceMessageID string
	// Conversation coordinates are retained for process-local Assistant rows,
	// whose raw durable event does not exist until settlement.
	ConversationTurn *float64 `json:",omitempty"`
	ConversationStep *float64 `json:",omitempty"`
	ImageRef         string
	ContentBlock     json.RawMessage
	Kind             MessageKind
	Role             string
	Body             string
	Seq              float64         // persistence seq when known (editMessageSeq)
	EventType        string          // persistence type when Kind is tool / deliverables
	Raw              json.RawMessage // full persistence event JSON
	TurnSeq          float64         // owning turn/start seq for edit projection
	TurnOpening      bool
	Streaming        bool
	Files            []string // deliverables/presented paths
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

// SubagentCatalogEntry is one projected subagent entry for a parent session.
type SubagentCatalogEntry struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Mode  string `json:"mode"`
}

// ViewState is a UI-safe snapshot of the store.
type ViewState struct {
	SubagentCatalog          []SubagentCatalogEntry
	SidebarSubagentActivity  map[string]SidebarSubagentActivity
	TrajectoryRequests       []TrajectoryRequest
	TrajectoryRequestHeaders []TrajectoryRequestHeader
	// Independent loaded journal: retain records hidden by later chat edits.
	TrajectoryMessages []Message
	ContextOccupancy   *ContextOccupancy

	StopSequenceMS int64
	// Stop identity is the existing live turn and selected follow binding.
	StopTurnSeq    float64
	StopGeneration uint64
	TurnUsage      map[float64]TurnUsage
	TurnFiles      TurnFilesSnapshot
	WorkflowRuns   []workflowrun.Run
	// PlanHistoryBinding owns the last authoritative follow snapshot.
	PlanHistoryBinding            string
	WorkspaceLayout               WorkspaceLayoutState
	Workspaces                    []WorkspaceRow
	WorkspaceReady                bool
	WorkspaceError                string
	WorkspaceNavigation           *WorkspaceNavigationState
	QuestionProjection            QuestionProjectionState
	ArchiveConfirmation           *ArchiveConfirmation
	SlashCommands                 []api.ListResultItem2
	SlashCommandsLoading          bool
	SlashCommandsKnown            bool
	SlashCommandsErr              string
	NativeContributions           dsh.NativeContributions
	NativeContributionsGeneration uint64
	NativeModelLabelGeneration    uint64
	nativeCommandGenerations      [4]uint64
	projection                    ProjectionKey
	historyProjection             ProjectionKey
	rosterProjection              ProjectionKey
	PermissionCatalogRevision     uint64
	QueueEditor                   QueueEditorState

	CostSources      []sessioncost.Source
	SelectedProvider string

	ScratchCwd       string
	SelectedCwd      string
	WorkspaceMembers map[string]bool
	ClientReady      bool
	ListLoading      bool
	ListErr          string
	Items            []Item
	SelectedID       string
	Messages         []Message
	// HistoryReady records an authoritative opening snapshot for this selection.
	// Reopening the same selection retains that baseline while HistoryLoading.
	HistoryLoading bool
	HistoryReady   bool
	HistoryError   string
	// SelectedSummaryBlank is Host list authority, not inferred from Messages.
	SelectedSummaryBlank bool
	MessageEditBusy      bool
	PromptBusy           bool
	// PromptAttempted preserves the original Session's blank-to-engaging edge.
	PromptAttempted bool
	LastAction      string
	// MessageRatings maps assistant message id → "up"|"down" for the current session.
	MessageRatings map[string]string
	// FeedbackBusyID is the message id with an in-flight messageFeedback/put (or "").
	FeedbackBusyID          string
	ModelChoices            map[string]api.ModelCatalogDefault
	ModelCatalogChoices     []ModelCatalogChoice
	ModelEffortChoices      map[string]api.ModelCatalogDefault
	ModelSelection          *api.ModelCatalogDefault
	ModelSelectionKnown     bool
	ModelSelectionBusy      bool
	ModelEffortLabels       []string
	SelectedModelEffort     string
	ModelLabels             []string
	SelectedModel           string
	ModelsErr               string
	PermissionLabels        []string
	SelectedPermission      string
	SelectedPermissionValue string
	PermissionSelectionBusy bool
	PermissionsErr          string
	PlanModeControl         PlanModeControl
	AgentPresetLabels       []string
	SelectedAgentPreset     string
	AgentPresetsErr         string
	SkillCatalog            SkillCatalog
	SkillOptions            []SkillOption
	SkillsErr               string
	SkillHubHits            []SkillHubHit

	SearchLoading  bool
	SearchUseLocal bool
	SearchErr      string
	SearchRPCQuery string
	SearchHitIDs   map[string]bool
	SearchResults  []api.SessionSearchValueItemsItem
	SearchHasMore  bool

	// PlanBoard is the folded plan/goal/queue/todo strip from follow events.
	PlanBoard PlanBoard
	// GoalReadError is a selected-follow diagnostic; background goal reads
	// never replace the conversation's LastAction or open an empty goal panel.
	GoalReadError string

	HistoryNavigation HistoryNavigationView
	HistoryBranchBusy bool
	HistoryHasMore    bool
	HistoryPageBusy   bool
	HistoryPageErr    string

	// PromptMode is the session/prompt wire mode: "queue" (排队) or "steer" (插话).
	PromptMode                PromptMode
	PromptModeExplicit        bool
	ComposerHostGeneration    uint64
	ComposerSessionGeneration uint64
	// PinnedSessionIDs is workspace pin order (first = top).
	PinnedSessionIDs []string
	// ArchivedSessionIDs is the last workspace archive roster (for unarchive UI).
	ArchivedSessionIDs []string

	// BandActions/BandGeneration come from claudeCodeMods/watchBand when present.
	BandActions    []BandAction
	BandGeneration float64
	// SpeechSelection is the latest speech/follow catalog selection summary.
	SpeechSelection                                            string
	SpeechReadinessKnown, SpeechReady, SpeechNeedsInstallation bool
	// ControlNote is the latest session/control frame type (or short preview).
	ControlNote string
}

// Store tracks sessions for the active dsh client.
type Store struct {
	subagentCatalogs       map[string][]SubagentCatalogEntry
	sidebarCatalog         sidebarCatalogState
	subagentIdentities     map[string]sessionSubagentIdentityState
	presetDraft            agentPresetDraftState
	imageLimits            composerImageLimitState
	planModeControl        planModeState
	archiveUndo            *archiveUndoState
	archiveUndoToken       uint64
	archiveUndoActive      atomic.Bool
	completionUnread       map[string]bool
	sidebarRunning         map[string]bool
	mainConversationReadID string
	contextProjection      contextProjectionState

	stopSequenceConfig         stopSequenceConfig
	quotaNotice                quotaNoticeState
	inspector                  sessionInspectorState
	nextPrompt                 nextPromptState
	systemPromptAnchors        map[float64]float64
	trajectoryBoundaries       map[float64]json.RawMessage
	usageCacheRev              uint64
	usageEventsCache           map[float64]usageCacheEntry
	trajCacheRev               uint64
	trajJournalCache           map[float64]json.RawMessage
	trajEventsCache            map[float64]trajectoryRequestCacheEntry
	trajHeadersCache           []TrajectoryRequestHeader
	trajRequestsCache          []TrajectoryRequest
	trajHadLiveAttempt         bool
	trajTimingCache            map[float64]trajectoryTimingCacheEntry
	trajTimingRev              uint64
	trajTimingParsed           map[float64]trajectoryParsedEvent
	trajTimingOps              map[string]*TrajectoryTiming
	trajTimingAnchors          map[string]json.RawMessage
	assistantMsgCache          map[float64]assistantMsgCacheEntry
	systemPromptRowsCache      []Message
	retryRowsCache             []Message
	durableTimedCache          []Message
	durableUnmappedCompactions []Message
	durableStepIndices         map[string][]int
	durableHasEdits            bool
	durableCacheValid          bool
	boundariesVerifiedRev      uint64
	historyVerifiedRev         uint64
	historyValid               bool
	historyMessages            []historyMessage
	historyVolatiles           []historyMessage
	historyBoundaries          map[float64]string
	historyWorkflows           map[float64]string
	historySelectedID          string
	historyEditTurnSeq         float64
	historyHasMorePrev         bool
	historyHadStream           bool
	historyHadRetained         bool
	historyInlinePending       []PendingSubmission
	historyInspActive          bool
	msgProjCache               []Message
	workflowRunsCache          []workflowrun.Run
	// Cache hits require the same revision as the last projected view. Writers
	// must invalidate under the same lock that publishes their state changes.
	viewRev                   uint64
	projectionToken           *projectionToken
	projectionTokenRev        uint64
	historyProjectionToken    *projectionToken
	rosterProjectionToken     *projectionToken
	rosterDirty               bool
	renderFrame               *FrameSnapshot // lazily made by RenderSnapshot
	viewCacheRev              uint64
	viewCache                 ViewState
	viewCacheValid            bool
	snapshotBuilds            uint64
	snapshotHits              uint64
	workflowBoundaries        map[float64]json.RawMessage
	workflowMember            workflowMemberState
	agentNavigation           agentNavigationState
	agentNavigationPaneID     string
	agentNavigationOwner      *Store
	agentNavigationPanes      map[string]*Store
	agentOccurrenceSequence   uint64
	agentRetainedReads        map[string]*agentRetainedRead
	agentRetainedRead         *agentRetainedRead
	agentRetainEnabled        bool
	planHistoryBinding        string
	jobs                      nativeJobsState
	composerHostGeneration    uint64
	composerOrder             uint64
	promptAttempted           map[string]bool
	composerPending           map[string]*composerPending
	composerObserved          map[string]composerObserved
	composerAdmissions        map[string]*ComposerTarget
	workspacePreferences      workspacePreferenceBinding
	rename                    SessionRenameView
	renameEpoch               uint64
	sessionFeedback           SessionFeedbackView
	sessionFeedbackEpoch      uint64
	workspaceLayout           workspaceLayoutState
	workspaceNavigation       workspaceNavigationOperation
	workspaceNavigationToken  uint64
	initialWorkspaceSession   initialWorkspaceSessionState
	workspaceReadyHost        uint64
	workspaceOrder            []string
	workspaceFollowGeneration uint64
	questionProjections       map[string]QuestionProjectionState
	permissionRisk            *permissionRiskOperation
	permissionRiskToken       uint64
	assistantStream           assistantStreamState
	batch                     SessionBatch
	batchToken                uint64
	archive                   archiveOperation
	deletion                  sessionDeletion
	archiveToken              uint64
	commandCatalog            commandCatalog
	nativeContributions       nativeContributionState
	feedback                  feedbackState
	visualReply               visualReplyRead
	visualReplyToken          uint64
	queueEditor               queueEditor
	queueMutationScope        string
	queueMutationID           string
	queueEditorToken          uint64
	queueProjectionSessionID  string
	queueProjectionGeneration uint64
	queueProjectionSeq        float64

	costProjections map[string]costProjection

	noDirectoryMu        sync.Mutex
	scratchCwd           string
	workspaceReady       bool
	workspaceErr         error
	workspaceChanged     chan struct{}
	workspaceMemberships map[string]workspaceMembership
	mu                   sync.Mutex
	client               *dsh.Client
	mux                  *dsh.Mux
	notify               func()
	viewChanged          chan struct{}

	listLoading                          bool
	listAuthoritativeHost                uint64
	listGeneration                       uint64
	listFollowGeneration                 uint64
	eventsGeneration                     uint64
	listMutations                        []lifecycleMutation
	listErr                              error
	items                                []Item
	selectedID                           string
	messages                             []Message
	historyLoading                       bool
	historyReady                         bool
	historyError                         string
	promptBusy                           bool
	historyBranchBusy                    bool
	lastAction                           string
	goalReadError                        goalReadErrorState
	messageRatings                       map[string]string
	feedbackBusyID                       string
	modelOptions                         []ModelOption
	modelState                           modelSelectionState
	modelsErr                            error
	permissionOptions                    []PermissionOption
	permissionCatalogRevision            uint64
	permissionProjection                 permissionProjectionState
	permissionsErr                       error
	agentPresetOptions                   []AgentPresetOption
	selectedAgentPresetLabel             string
	agentPresetsErr                      error
	agentPresetsRequest                  uint64
	agentPresetsLoaded                   bool
	agentPresetsLoading                  bool
	skillCatalogClient                   *dsh.Client
	skillCatalogLoading                  bool
	skillCatalog                         SkillCatalog
	skillCatalogScope                    api.SkillInventoryScope
	skillCatalogVersion                  uint64
	skillRemoval                         SkillRemovalReview
	skillRemovalSequence                 uint64
	skillCatalogSession, skillCatalogCwd string
	skillRefreshOnEventsReady            bool
	skillOptionsEpoch                    uint64
	skillOptionsOwner                    skillOptionsScope
	skillOptions                         []SkillOption
	skillsErr                            error
	skillHubHits                         []SkillHubHit
	searchInput                          string
	searchLoading                        bool
	searchUseLocal                       bool
	searchErr                            error
	searchRPCQuery                       string
	searchHits                           map[string]struct{}
	searchResults                        []api.SessionSearchValueItemsItem
	searchHasMore                        bool
	searchCancel                         context.CancelFunc
	searchSeq                            uint64
	searchDebounce                       *time.Timer
	followCancel                         context.CancelFunc
	followGeneration                     uint64
	turnFiles                            turnFilesState
	turnFilesHistoryBinding              string
	workspaceFollowCancel                context.CancelFunc
	bandCancel                           context.CancelFunc
	bandActions                          []BandAction
	bandGeneration                       float64
	speechFollowCancel                   context.CancelFunc
	speechSelection                      string
	speechReadiness                      speechReadinessState
	controlCancel                        context.CancelFunc
	controlWatchStreamCtx                context.Context
	controlWatchParentCtx                context.Context
	controlWatchClient                   *dsh.Client
	controlWatchMux                      *dsh.Mux
	controlWatchHost                     uint64
	controlNote                          string
	cwd                                  string
	workspaceID                          string
	planBoard                            PlanBoard
	historyNavigation                    historyNavigationState
	historyHasMore                       bool
	historyPageBusy                      bool
	historyPageRequest                   uint64
	historyPageErr                       error
	historyThroughSeq                    float64
	promptMode                           PromptMode
	promptModeExplicit                   bool
	composerRosterClient                 *dsh.Client
	pinnedSessionIDs                     []string
	archivedSessionIDs                   []string
	editTurnSeq                          float64
	editTurnOpening                      bool
	turnStartMs                          float64
}

// New returns an empty store; call Attach when the harness endpoint is ready.
func New() *Store {
	return &Store{promptMode: PromptModeQueue}
}

// Attach binds a dsh client and an optional UI refresh hook.
func (s *Store) Attach(client *dsh.Client, notify func()) {
	s.mu.Lock()
	if s.client != client {
		s.initialWorkspaceSession = initialWorkspaceSessionState{binding: s.initialWorkspaceSession.binding + 1}
		s.listAuthoritativeHost, s.workspaceReadyHost = 0, 0
		s.resetNativeContributionsLocked()
		s.resetSessionPresentationsLocked()
		s.resetSessionSubagentIdentitiesLocked()
		s.subagentCatalogs = nil
		s.goalReadError = goalReadErrorState{}
		s.cancelRetainedAgentReadsLocked("")
		s.resetSidebarStatusesLocked()
		s.clearStopSequenceConfigLocked()
		s.quotaNotice.current = QuotaNotice{}
		s.quotaNotice.held = false
		s.clearNextPromptLocked()
		s.nextPrompt.accepted = nil
		s.clearSearchLocked()
		s.resetComposerHostLocked()
		s.invalidateSkillOptionsLocked()
		s.invalidateRenameLocked()
		s.invalidateSessionFeedbackLocked()
		s.questionProjections = nil
		s.permissionOptions = nil
		s.resetModelCatalogLocked()
		s.agentPresetOptions = nil
		s.selectedAgentPresetLabel = ""
		s.agentPresetsErr = nil
		s.agentPresetsLoaded, s.agentPresetsLoading = false, false
		s.invalidateArchiveLocked()
		s.invalidateCommandsLocked()
		s.resetWorkspaceLocked()
		s.eventsGeneration++
		s.invalidateFollowLocked()
		s.listGeneration++
		s.listLoading = false
		s.listMutations = nil
	}
	s.costProjections = nil
	s.client = client
	s.notify = notify
	s.listErr = nil
	s.invalidateViewLocked()
	s.mu.Unlock()
}

// SetWorkspaceCwd sets the cwd used for session/create.
func (s *Store) SetWorkspaceCwd(cwd string) {
	s.mu.Lock()
	if s.cwd != cwd {
		s.invalidateWorkspaceNavigationLocked()
		s.workspaceID = ""
	}
	s.cwd = cwd
	s.invalidateViewLocked()
	s.mu.Unlock()
}

// CreateSession preserves the current Host workspace membership and navigation.
// UI callers use WorkspaceSessionTarget to reuse only drafts known to be empty;
// offline callers retain the existing local blank reuse contract.
func (s *Store) CreateSession(ctx context.Context) {
	s.mu.Lock()
	// Registered workspaces require an explicit Host identity, not merely cwd.
	// Non-UI callers cannot inspect unsent drafts, so only the UI-owned path
	// may propose an existing blank for reuse.
	workspaceID, host := s.newSessionWorkspaceIDLocked(), s.eventsGeneration
	if s.client != nil && workspaceID != "" {
		var excluded []string
		for _, it := range s.items {
			excluded = append(excluded, it.SessionID)
		}
		s.mu.Unlock()
		target, err := s.CaptureWorkspaceSessionTarget(host, workspaceID, excluded...)
		if err != nil {
			s.SetLastAction("create workspace session: " + err.Error())
			return
		}
		go func() {
			id, err := target.Create(ctx)
			if err != nil {
				target.ReportFailure(err)
				return
			}
			if target.Select(id) {
				s.InitializeDefaultModel(ctx)
			}
		}()
		return
	}
	selectedID := s.selectedID
	scratchCwd := s.scratchCwd
	members := s.workspaceMembersLocked()
	for _, it := range s.items {
		if it.SessionID == selectedID && scratchCwd != "" && filepathClean(it.Cwd) == filepathClean(scratchCwd) && !members[it.SessionID] {
			// Generic new-session actions cannot inspect UI-owned unsent drafts.
			// Never reuse another identity without that knowledge.
			ids := make([]string, 0, len(s.items))
			for _, candidate := range s.items {
				ids = append(ids, candidate.SessionID)
			}
			client, hostGeneration := s.client, s.eventsGeneration
			s.mu.Unlock()
			go s.runCreateNoDirectorySession(ctx, client, hostGeneration, selectedID, ids)
			return
		}
	}
	client := s.client
	cwd := s.cwd
	hostGeneration := s.eventsGeneration
	navigation, follow := s.workspaceNavigationToken, s.followGeneration
	stagedPreset := s.stagedPresetLocked(cwd)
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
			s.invalidateViewLocked()
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
		s.mu.Lock()
		if s.client != client || s.eventsGeneration != hostGeneration || s.workspaceNavigationToken != navigation || s.followGeneration != follow || s.selectedID != selectedID {
			s.mu.Unlock()
			return
		}
		if err == nil && out.Reusable {
			changed := s.selectedID != id
			if changed {
				s.selectSessionLocked(id)
			}
			s.lastAction = "blank session reused"
			s.invalidateViewLocked()
			s.mu.Unlock()
			if changed {
				s.fire()
				s.refreshSelectedViews()
			}
			if stagedPreset != "" {
				s.SelectAgentPresetIDForSession(ctx, id, stagedPreset)
			}
			s.fire()
			return
		}
		if err != nil {
			// A failed authoritative check is not permission to reuse an identity.
			s.lastAction = "blank session check: " + err.Error()
			s.invalidateViewLocked()
			s.mu.Unlock()
			s.fire()
			return
		}
		s.mu.Unlock()
	}
	go s.runCreateSessionOwned(ctx, client, hostGeneration, cwd, stagedPreset, navigation, follow, selectedID)
}

func filepathClean(p string) string {
	return strings.TrimSpace(strings.ReplaceAll(p, "\\", "/"))
}

// Detach clears the client (harness stopped or lost).
func (s *Store) Detach() {
	s.mu.Lock()
	s.initialWorkspaceSession = initialWorkspaceSessionState{binding: s.initialWorkspaceSession.binding + 1}
	s.listAuthoritativeHost, s.workspaceReadyHost = 0, 0
	s.resetNativeContributionsLocked()
	s.resetSessionPresentationsLocked()
	s.resetSessionSubagentIdentitiesLocked()
	s.subagentCatalogs = nil
	s.cancelRetainedAgentReadsLocked("")
	s.resetSidebarStatusesLocked()
	s.clearNextPromptLocked()
	s.nextPrompt.accepted = nil
	s.resetComposerHostLocked()
	s.invalidateRenameLocked()
	s.invalidateSessionFeedbackLocked()
	s.resetModelCatalogLocked()
	s.agentPresetOptions = nil
	s.selectedAgentPresetLabel = ""
	s.agentPresetsErr = nil
	s.agentPresetsLoaded, s.agentPresetsLoading = false, false
	s.questionProjections = nil
	s.invalidateArchiveLocked()
	s.invalidateCommandsLocked()
	s.resetWorkspaceLocked()
	s.eventsGeneration++
	s.invalidateFollowLocked()
	s.listGeneration++
	s.listFollowGeneration = s.followGeneration
	s.listMutations = nil
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
	s.speechReadiness = speechReadinessState{generation: s.speechReadiness.generation + 1}
	if s.controlCancel != nil {
		s.controlCancel()
		s.controlCancel = nil
	}
	s.controlWatchStreamCtx = nil
	s.controlNote = ""
	if s.searchDebounce != nil {
		s.searchDebounce.Stop()
		s.searchDebounce = nil
	}
	s.clearSearchLocked()
	s.clearFeedbackLocked()
	s.client = nil
	s.quotaNotice.current = QuotaNotice{}
	s.quotaNotice.held = false
	s.mux = nil
	s.listLoading = false
	s.mu.Unlock()
	s.fire()
}

// SelectedSessionID returns just the current selection without building a view.
func (s *Store) SelectedSessionID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.selectedID
}

// Snapshot returns an independent copy of the current observable state. The
// caller may mutate it, including nested fields, without affecting the store,
// the revision cache, or another caller, even during a render frame. The copy
// keeps the projection keys of its revision; after editing its Messages,
// WorkflowRuns, Items or Workspaces, call WithoutProjectionKeys on it.
func (s *Store) Snapshot() ViewState {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureViewCacheLocked()
	return cloneViewState(s.viewCache)
}

type historyMessage struct {
	seq     float64
	raw     string
	content string
	msg     Message
}

func equalFloat64Ptr(a, b *float64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func hasMessagePointers(m Message) bool {
	return m.AutomaticCompaction != nil || m.Retry != nil || m.TurnDelivery != nil || m.TrajectoryTime != nil || m.Workflow != nil || m.Submission != nil || m.Tool != nil || m.Command != nil || m.Web != nil
}

// volatileMessageRow reports a row that only exists while a reply streams: no
// journal sequence and no raw record. The journal, usage and trajectory builders
// read nothing else from a message, so such a row cannot change what they
// produce. volatileHistoryRow is the same test on a remembered row.
func volatileMessageRow(m Message) bool {
	return m.Streaming && m.Seq == 0 && len(m.Raw) == 0
}

func volatileHistoryRow(h historyMessage) bool {
	return h.msg.Streaming && h.seq == 0 && h.raw == ""
}

func sameHistoryMessage(h historyMessage, m Message) bool {
	// A row that is still streaming has no journal record yet (no Seq, no Raw): its
	// body grows with every chunk. Everything cached against history is derived
	// from journal records, so treating each chunk as a history change threw all
	// of it away and re-decoded every event for every frame of a stream.
	if volatileHistoryRow(h) && volatileMessageRow(m) && h.msg.ID == m.ID {
		return true
	}
	if h.seq != m.Seq || h.raw != string(m.Raw) || h.content != string(m.ContentBlock) ||
		h.msg.ID != m.ID || h.msg.Role != m.Role || h.msg.Kind != m.Kind || h.msg.Body != m.Body ||
		h.msg.Streaming != m.Streaming || h.msg.TurnSeq != m.TurnSeq || h.msg.TurnOpening != m.TurnOpening ||
		h.msg.EventType != m.EventType || h.msg.EventTime != m.EventTime || h.msg.ImageRef != m.ImageRef ||
		h.msg.SourceMessageID != m.SourceMessageID || h.msg.Model != m.Model || h.msg.Args != m.Args ||
		h.msg.ContextWaking != m.ContextWaking ||
		!equalFloat64Ptr(h.msg.ConversationTurn, m.ConversationTurn) ||
		!equalFloat64Ptr(h.msg.ConversationStep, m.ConversationStep) ||
		!equalStringSlice(h.msg.Files, m.Files) {
		return false
	}
	if !hasMessagePointers(h.msg) && !hasMessagePointers(m) {
		return true
	}
	mCopy := m
	mCopy.Raw = nil
	mCopy.ContentBlock = nil
	return reflect.DeepEqual(h.msg, mCopy)
}

func splitVolatileTail(messages []Message) (durable []Message, volatile []Message, ok bool) {
	split := len(messages)
	for split > 0 && volatileMessageRow(messages[split-1]) {
		split--
	}
	for i := 0; i < split; i++ {
		if volatileMessageRow(messages[i]) {
			return nil, nil, false
		}
	}
	return messages[:split], messages[split:], true
}

func (s *Store) boundariesVerifiedLocked() bool {
	if s.historyValid && s.boundariesVerifiedRev == s.viewRev {
		return true
	}
	return s.boundariesCacheMatchesLocked()
}

func (s *Store) historyVerifiedLocked() bool {
	if s.historyValid && s.historyVerifiedRev == s.viewRev {
		return true
	}
	return s.historyCacheMatchesLocked()
}

func (s *Store) boundariesCacheMatchesLocked() bool {
	if !s.historyValid || len(s.historyBoundaries) != len(s.trajectoryBoundaries) {
		return false
	}
	for seq, raw := range s.trajectoryBoundaries {
		previous, ok := s.historyBoundaries[seq]
		if !ok || previous != string(raw) {
			return false
		}
	}
	return true
}

func (s *Store) workflowBoundariesMatchesLocked() bool {
	if !s.historyValid || len(s.historyWorkflows) != len(s.workflowBoundaries) {
		return false
	}
	for seq, raw := range s.workflowBoundaries {
		previous, ok := s.historyWorkflows[seq]
		if !ok || previous != string(raw) {
			return false
		}
	}
	return true
}

// historyCacheMatchesLocked compares the inputs used by both history parsers,
// including the bytes at every position and boundary rather than a fingerprint.
// Stored strings own their bytes, so in-place changes are detected as well.
func (s *Store) historyCacheMatchesLocked() bool {
	if !s.boundariesVerifiedLocked() {
		return false
	}
	durable, _, ok := splitVolatileTail(s.messages)
	if !ok {
		durable = s.messages
	}
	if len(s.historyMessages) != len(durable) {
		return false
	}
	for i, message := range durable {
		if !sameHistoryMessage(s.historyMessages[i], message) {
			return false
		}
	}
	return true
}

func (s *Store) volatileMessagesMatchLocked() bool {
	if !s.historyValid {
		return false
	}
	_, volatile, ok := splitVolatileTail(s.messages)
	if !ok || len(s.historyVolatiles) != len(volatile) {
		return false
	}
	for i, message := range volatile {
		h := s.historyVolatiles[i]
		if h.msg.ID != message.ID || h.msg.Body != message.Body || h.msg.Role != message.Role || h.msg.Kind != message.Kind ||
			h.msg.TurnSeq != message.TurnSeq || h.msg.EventType != message.EventType ||
			!equalFloat64Ptr(h.msg.ConversationTurn, message.ConversationTurn) ||
			!equalFloat64Ptr(h.msg.ConversationStep, message.ConversationStep) {
			return false
		}
	}
	return true
}

func (s *Store) rememberVolatilesLocked(volatile []Message) {
	if cap(s.historyVolatiles) >= len(volatile) {
		s.historyVolatiles = s.historyVolatiles[:len(volatile)]
	} else {
		s.historyVolatiles = make([]historyMessage, len(volatile))
	}
	for i, message := range volatile {
		cloned := cloneMessage(message)
		cloned.Raw = nil
		cloned.ContentBlock = nil
		s.historyVolatiles[i] = historyMessage{
			seq:     message.Seq,
			raw:     string(message.Raw),
			content: string(message.ContentBlock),
			msg:     cloned,
		}
	}
}

func (s *Store) rememberHistoryLocked() {
	durable, volatile, ok := splitVolatileTail(s.messages)
	if !ok {
		durable = s.messages
		volatile = nil
	}
	s.historyMessages = make([]historyMessage, len(durable))
	for i, message := range durable {
		cloned := cloneMessage(message)
		cloned.Raw = nil
		cloned.ContentBlock = nil
		s.historyMessages[i] = historyMessage{
			seq:     message.Seq,
			raw:     string(message.Raw),
			content: string(message.ContentBlock),
			msg:     cloned,
		}
	}
	s.rememberVolatilesLocked(volatile)
	s.historyBoundaries = make(map[float64]string, len(s.trajectoryBoundaries))
	for seq, raw := range s.trajectoryBoundaries {
		s.historyBoundaries[seq] = string(raw)
	}
	s.historyWorkflows = make(map[float64]string, len(s.workflowBoundaries))
	for seq, raw := range s.workflowBoundaries {
		s.historyWorkflows[seq] = string(raw)
	}
	s.historyValid = true
	s.boundariesVerifiedRev = s.viewRev
	s.historyVerifiedRev = s.viewRev
}

func (s *Store) projectMessagesIncrementalLocked(durable, volatile []Message, rebuildDurable bool) ([]Message, bool) {
	if rebuildDurable {
		if hasMessageEdits(durable) {
			s.durableHasEdits = true
			s.durableCacheValid = false
			return nil, false
		}
		s.durableHasEdits = false
		base := projectCommandMessages(s.projectAssistantMessagesLocked(durable))
		timed, unmapped, stepIndices := s.projectDurableTimingLocked(base)
		timed = s.projectContextTriggersLocked(timed)
		s.durableTimedCache = cloneMessages(timed)
		s.durableUnmappedCompactions = cloneMessages(unmapped)
		s.durableStepIndices = stepIndices
		s.durableCacheValid = true
	} else if !s.durableCacheValid || s.durableHasEdits {
		return nil, false
	}
	out := make([]Message, len(s.durableTimedCache), len(s.durableTimedCache)+len(volatile)+8)
	copy(out, s.durableTimedCache)
	_, ops, _ := s.ensureTrajectoryTimingCacheLocked()
	liveOpID, liveOp, live := s.liveAttemptTimingLocked(ops)
	if liveOpID != "" && liveOp != nil {
		for _, idx := range s.durableStepIndices[liveOpID] {
			if idx >= 0 && idx < len(out) {
				out[idx].TrajectoryTime = cloneTrajectoryTiming(liveOp)
			}
		}
	}
	for _, vm := range volatile {
		for _, pm := range projectAssistantMessage(vm) {
			if t := live[pm.ID]; t != nil {
				pm.TrajectoryTime = cloneTrajectoryTiming(t)
			}
			out = append(out, cloneMessage(pm))
		}
	}
	beforePrep := len(out)
	out = s.projectPreparingToolsLocked(out)
	for i := beforePrep; i < len(out); i++ {
		out[i] = cloneMessage(out[i])
	}
	if len(s.durableUnmappedCompactions) > 0 {
		out = append(out, cloneMessages(s.durableUnmappedCompactions)...)
	}
	out = s.projectRetryMessagesLocked(out)
	out = s.projectSystemPromptsLocked(out)
	return out[:len(out):len(out)], true
}

// ensureViewCacheLocked builds an independent, immutable projection for the
// current revision. Projection helpers can retain references into the store, so
// the projected value must be cloned before it is saved or handed to a caller.
func (s *Store) ensureViewCacheLocked() {
	if s.viewCacheValid && s.viewCacheRev == s.viewRev {
		atomic.AddUint64(&s.snapshotHits, 1)
		return
	}
	// The history fingerprint is only a hint: equal lengths and tails do not
	// prove the journal is unchanged. Compare exact immutable source events
	// against the last projection before trusting either parsed-history cache.
	boundariesChanged := !s.boundariesCacheMatchesLocked()
	if !boundariesChanged {
		s.boundariesVerifiedRev = s.viewRev
	}
	historyChanged := boundariesChanged || !s.historyCacheMatchesLocked()
	if !historyChanged {
		s.historyVerifiedRev = s.viewRev
	}
	if boundariesChanged {
		s.trajHeadersCache = nil
		s.trajRequestsCache = nil
		s.systemPromptRowsCache = nil
		s.retryRowsCache = nil
	}
	if historyChanged {
		s.trajJournalCache = nil
		s.trajRequestsCache = nil
		s.trajTimingParsed = nil
		s.trajTimingOps = nil
		s.trajTimingAnchors = nil
		s.durableCacheValid = false
		s.historyNavigation.cachedValid = false
	}
	turnUsage := s.viewCache.TurnUsage
	if historyChanged || !s.viewCacheValid {
		turnUsage = cloneTurnUsageMap(s.turnUsageLocked())
	}
	st := ViewState{
		StopTurnSeq:         s.editTurnSeq,
		StopSequenceMS:      s.stopSequenceConfig.intervalMS,
		StopGeneration:      s.followGeneration,
		TurnUsage:           turnUsage,
		TurnFiles:           s.turnFilesSnapshotLocked(),
		PlanHistoryBinding:  s.planHistoryBinding,
		WorkspaceLayout:     s.workspaceLayoutLocked(),
		Workspaces:          s.workspaceRowsLocked(),
		WorkspaceReady:      s.workspaceReady,
		WorkspaceNavigation: s.workspaceNavigationSnapshotLocked(),
		QuestionProjection:  s.questionProjectionLocked(),
		ArchiveConfirmation: s.archiveConfirmationLocked(),
		QueueEditor:         s.queueEditorSnapshotLocked(),
		CostSources:         s.costSourcesLocked(),
		ClientReady:         s.client != nil,
		ScratchCwd:          s.scratchCwd,
		WorkspaceMembers:    s.workspaceMembersLocked(),
		LastAction:          s.lastAction,
		GoalReadError:       s.goalReadErrorLocked(),
		ListLoading:         s.listLoading,
		SelectedID:          s.selectedID,
		HistoryLoading:      s.historyLoading,
		HistoryReady:        s.historyReady,
		HistoryError:        s.historyError,
		PromptBusy:          s.promptBusy || s.composerPromptBusyLocked(),
		PromptAttempted:     s.promptAttempted[s.selectedID],
		MessageEditBusy:     s.editTurnSeq > 0 || s.composerPendingForSelectedLocked(),
		HistoryBranchBusy:   s.historyBranchBusy,
		FeedbackBusyID:      s.feedbackBusyID,
	}
	if s.workspaceErr != nil {
		st.WorkspaceError = s.workspaceErr.Error()
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
	st.SidebarSubagentActivity = s.sidebarSubagentActivityLocked()
	if s.selectedID != "" {
		if len(s.subagentCatalogs[s.selectedID]) > 0 {
			st.SubagentCatalog = append([]SubagentCatalogEntry(nil), s.subagentCatalogs[s.selectedID]...)
		} else if cut, ok := s.sidebarCatalog.ByParent[s.selectedID]; ok && cut.Known && len(cut.IDs) > 0 {
			entries := make([]SubagentCatalogEntry, 0, len(cut.IDs))
			for _, id := range cut.IDs {
				entries = append(entries, SubagentCatalogEntry{ID: id})
			}
			st.SubagentCatalog = entries
		}
	}
	st.Items = append([]Item(nil), s.items...)
	for i := range st.Items {
		st.Items[i].CompletionUnread = s.completionUnread[st.Items[i].SessionID]
	}
	for _, it := range s.items {
		if it.SessionID == s.selectedID {
			st.SelectedCwd = it.Cwd
			st.SelectedSummaryBlank = it.Blank
			break
		}
	}
	workflowsChanged := !s.workflowBoundariesMatchesLocked()
	volatileChanged := !s.volatileMessagesMatchLocked()
	hasRetained := s.assistantStream.retained != nil || s.assistantStream.rebaseline
	hasStream := s.assistantStream.active != nil || hasRetained
	inspActive := s.inspector.client == s.client && s.inspector.session == s.selectedID && s.inspector.generation == s.followGeneration
	inspDirty := (inspActive && s.inspector.contextWaking == nil) || inspActive != s.historyInspActive
	msgProjChanged := historyChanged || workflowsChanged || volatileChanged || !s.viewCacheValid ||
		s.selectedID != s.historySelectedID || s.editTurnSeq != s.historyEditTurnSeq || s.historyHasMore != s.historyHasMorePrev ||
		hasStream || s.historyHadStream || inspDirty
	if !msgProjChanged {
		st.Messages = s.msgProjCache
		st.WorkflowRuns = s.workflowRunsCache
	} else {
		durable, volatile, splitOK := splitVolatileTail(s.messages)
		durableProjChanged := !splitOK || historyChanged || workflowsChanged || !s.viewCacheValid ||
			s.selectedID != s.historySelectedID || s.editTurnSeq != s.historyEditTurnSeq || s.historyHasMore != s.historyHasMorePrev ||
			hasRetained || s.historyHadRetained || inspDirty ||
			!s.durableCacheValid
		var incrementalOK bool
		if splitOK && !hasRetained {
			st.Messages, incrementalOK = s.projectMessagesIncrementalLocked(durable, volatile, durableProjChanged)
		}
		if !incrementalOK {
			s.durableCacheValid = false
			st.Messages = projectCommandMessages(s.projectAssistantMessagesLocked(projectMessageEdits(s.messages)))
			st.Messages = s.projectPreparingToolsLocked(st.Messages)
			st.Messages = s.projectTrajectoryTimingLocked(st.Messages)
			st.Messages = s.projectRetryMessagesLocked(st.Messages)
			st.Messages = s.projectSystemPromptsLocked(st.Messages)
			st.Messages = s.projectContextTriggersLocked(st.Messages)
			st.Messages = cloneMessages(st.Messages)
		}
		if len(st.Messages) > 0 {
			st.Messages = st.Messages[:len(st.Messages):len(st.Messages)]
		}
		workflowProjChanged := historyChanged || workflowsChanged || !s.viewCacheValid ||
			s.selectedID != s.historySelectedID || s.editTurnSeq != s.historyEditTurnSeq
		if !workflowProjChanged {
			st.WorkflowRuns = s.workflowRunsCache
		} else {
			st.WorkflowRuns = cloneWorkflowRuns(s.workflowRunsLocked(st.Messages))
			s.workflowRunsCache = st.WorkflowRuns
		}
		s.msgProjCache = st.Messages
		s.historySelectedID = s.selectedID
		s.historyEditTurnSeq = s.editTurnSeq
		s.historyHasMorePrev = s.historyHasMore
		s.historyHadStream = hasStream
		s.historyHadRetained = hasRetained
		s.historyInspActive = inspActive
	}
	if !msgProjChanged && !boundariesChanged && s.viewCacheValid {
		st.TrajectoryMessages = s.viewCache.TrajectoryMessages
	} else {
		st.TrajectoryMessages = s.trajectoryJournalMessagesLocked()
	}
	if !boundariesChanged && s.viewCacheValid && s.trajHeadersCache != nil {
		st.TrajectoryRequestHeaders = s.viewCache.TrajectoryRequestHeaders
	} else {
		s.trajHeadersCache = cloneTrajectoryRequestHeaders(s.trajectoryRequestHeadersLocked(), false)
		st.TrajectoryRequestHeaders = s.trajHeadersCache
	}
	hasLiveAttempt := s.assistantStream.active != nil
	if !historyChanged && s.viewCacheValid && !hasLiveAttempt && !s.trajHadLiveAttempt {
		st.TrajectoryRequests = s.viewCache.TrajectoryRequests
	} else {
		st.TrajectoryRequests = s.trajectoryRequestsViewLocked()
	}
	s.trajHadLiveAttempt = hasLiveAttempt
	s.snapshotCommandsLocked(&st)
	st.NativeContributions = s.nativeContributions.snapshot
	st.NativeContributionsGeneration = s.nativeContributions.generation
	st.NativeModelLabelGeneration = s.nativeContributions.modelLabel
	st.nativeCommandGenerations = s.nativeContributions.ports
	st.PermissionCatalogRevision = s.permissionCatalogRevision
	st.ComposerSessionGeneration = s.followGeneration
	s.snapshotModelsLocked(&st)
	st.HistoryNavigation = s.historyNavigationSnapshotLocked()
	st.SelectedPermission = s.permissionLabelLocked()
	if s.permissionProjectionCurrentLocked() {
		st.SelectedPermissionValue = s.permissionProjection.Value
		st.PermissionSelectionBusy = s.permissionProjection.Pending || s.permissionRisk != nil && s.permissionRiskCurrentLocked(s.permissionRisk.view.Token)
	}
	st.PlanModeControl = s.planModeControlLocked()
	if s.permissionProjectionCurrentLocked() {
		for _, o := range s.permissionOptions {
			st.PermissionLabels = append(st.PermissionLabels, o.Label)
		}
	}
	st.SelectedAgentPreset = s.selectedAgentPresetLabel
	for _, o := range s.agentPresetOptions {
		st.AgentPresetLabels = append(st.AgentPresetLabels, o.Label)
	}
	st.SkillCatalog = s.skillCatalog.clone()
	if s.skillCatalogClient != s.client || s.skillCatalogSession != s.selectedID || s.skillCatalogCwd != s.cwd {
		st.SkillCatalog = SkillCatalog{}
	}
	st.SkillOptions = s.currentSkillOptionsLocked()
	st.SkillHubHits = append([]SkillHubHit(nil), s.skillHubHits...)
	st.SearchLoading = s.searchLoading
	st.SearchUseLocal = s.searchUseLocal
	st.SearchRPCQuery = s.searchRPCQuery
	st.SearchResults = append([]api.SessionSearchValueItemsItem(nil), s.searchResults...)
	st.SearchHasMore = s.searchHasMore
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
	st.PlanBoard.QueueActions = s.queueActionsLocked()
	st.HistoryHasMore = s.historyHasMore
	st.HistoryPageBusy = s.historyPageBusy
	if s.historyPageErr != nil {
		st.HistoryPageErr = s.historyPageErr.Error()
	}
	st.PromptMode = NormalizePromptMode(string(s.promptMode))
	st.PromptModeExplicit = s.promptModeExplicit
	st.ContextOccupancy = s.contextOccupancyLocked()
	st.ComposerHostGeneration = s.composerHostGeneration
	if len(s.pinnedSessionIDs) > 0 {
		st.PinnedSessionIDs = append([]string(nil), s.pinnedSessionIDs...)
	}
	if len(s.archivedSessionIDs) > 0 {
		st.ArchivedSessionIDs = append([]string(nil), s.archivedSessionIDs...)
	}
	st.BandGeneration = s.bandGeneration
	st.BandActions = append([]BandAction(nil), s.bandActions...)
	st.SpeechSelection = s.speechSelection
	st.SpeechReadinessKnown, st.SpeechReady, st.SpeechNeedsInstallation = s.speechReadinessFlagsLocked()
	st.ControlNote = s.controlNote
	inlinePendingChanged := s.projectComposerSubmissionsLocked(&st)
	if historyChanged || workflowsChanged {
		s.rememberHistoryLocked()
	} else if volatileChanged {
		if _, volatile, ok := splitVolatileTail(s.messages); ok {
			s.rememberVolatilesLocked(volatile)
		} else {
			s.rememberHistoryLocked()
		}
	}
	rosterChanged := s.rosterDirty || !s.viewCacheValid
	s.rosterDirty = false
	st.projection = s.projectionKeyLocked()
	st.historyProjection = s.historyProjectionKeyLocked(msgProjChanged || inlinePendingChanged)
	st.rosterProjection = s.rosterProjectionKeyLocked(rosterChanged)
	s.viewCache = cloneStoreViewCache(st)
	s.viewCacheRev = s.viewRev
	s.viewCacheValid = true
	atomic.AddUint64(&s.snapshotBuilds, 1)
}

// FrameSnapshot is a read-only view cache for render passes. The view it
// returns shares its backing data with the store's revision cache: neither the
// handle nor the view may be mutated or passed to asynchronous work, and a
// view must not be kept past the pass that asked for it. Use Snapshot for every
// other read.
//
// The handle itself may outlive a pass. Every Snapshot call compares the
// store's revision under the store mutex and rebuilds the view when it moved, so
// a render owner that keeps one handle serves each pass on an unchanged revision
// (hover, scroll) from one clone. A write invalidated under that mutex is visible
// on the next read, including during the frame.
type FrameSnapshot struct {
	store *Store
	view  ViewState
	rev   uint64
	valid bool
}

// BeginFrame creates a pass bound to this particular store. There is no global
// frame state: overlapping passes and ordinary Snapshot callers are independent.
func (s *Store) BeginFrame() *FrameSnapshot {
	return &FrameSnapshot{store: s}
}

// Snapshot returns the pass's read-only view, rebuilding it when the store's
// revision changes. Do not mutate any part of the returned value.
func (f *FrameSnapshot) Snapshot() ViewState {
	s := f.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if f.valid && f.rev == s.viewRev {
		atomic.AddUint64(&s.snapshotHits, 1)
		return f.view
	}
	s.ensureViewCacheLocked()
	f.view = cloneFramePassView(s.viewCache)
	f.rev = s.viewRev
	f.valid = true
	return f.view
}

// EndFrame releases the handle's retained view. The handle stays usable: its
// next Snapshot rebuilds the view. It does not affect other handles.
func (f *FrameSnapshot) EndFrame() {
	s := f.store
	s.mu.Lock()
	f.view = ViewState{}
	f.valid = false
	s.mu.Unlock()
}

// RenderSnapshot returns the read-only view for drawing this store in a render
// pass when the caller holds no FrameSnapshot of it, such as a child agent's
// store drawn beside the main one. It carries FrameSnapshot's contract: the
// view is shared with the revision cache, so it must not be mutated, kept past
// the pass or handed to asynchronous work (use Snapshot for those). It costs one
// shallow clone per store revision where Snapshot deep-copies on every call.
func (s *Store) RenderSnapshot() ViewState {
	s.mu.Lock()
	if s.renderFrame == nil {
		s.renderFrame = &FrameSnapshot{store: s}
	}
	frame := s.renderFrame
	s.mu.Unlock()
	return frame.Snapshot()
}

// SnapshotStats reports how many snapshots were projected versus served from
// the revision cache. It exists for benchmarks and contract tests.
func (s *Store) SnapshotStats() (builds, hits uint64) {
	return atomic.LoadUint64(&s.snapshotBuilds), atomic.LoadUint64(&s.snapshotHits)
}

func clonePlanBoard(b PlanBoard) PlanBoard {
	out := PlanBoard{PlanMode: b.PlanMode, QueueActions: b.QueueActions}
	if b.Goal != nil {
		g := *b.Goal
		out.Goal = &g
	}
	if len(b.Todos) > 0 {
		out.Todos = append([]PlanTodo(nil), b.Todos...)
	}
	if len(b.Queue) > 0 {
		out.Queue = append([]PlanQueueItem(nil), b.Queue...)
		for i := range out.Queue {
			out.Queue[i].Attachments = append([]QueueAttachment(nil), b.Queue[i].Attachments...)
		}
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
		control, changed := s.beginControlWatchLocked(context.Background())
		s.mu.Unlock()
		if changed {
			s.fire()
		}
		if control != nil {
			go control()
		}
		return
	}
	s.selectSessionLocked(sessionID)
	s.mu.Unlock()
	s.fire()
	s.refreshSelectedViews()
}

func (s *Store) refreshSelectedViews() {
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
	if ctx.Err() != nil {
		return
	}
	s.mu.Lock()
	client, generation := s.beginListRefreshLocked()
	s.mu.Unlock()
	if client == nil {
		return
	}
	s.fire()
	go s.runRefreshList(ctx, client, generation)
}

func (s *Store) beginListRefreshLocked() (*dsh.Client, uint64) {
	s.invalidateViewLocked()
	if s.client == nil {
		return nil, 0
	}
	s.listGeneration++
	s.listFollowGeneration = s.followGeneration
	s.listMutations = nil
	s.listLoading = true
	s.listAuthoritativeHost = 0
	s.listErr = nil
	return s.client, s.listGeneration
}

func (s *Store) runRefreshList(ctx context.Context, client *dsh.Client, generation uint64) {
	var out api.SessionListValue
	err := client.Call(ctx, "session/list", map[string]any{
		"_request": api.SessionListRequest{},
	}, &out)
	var parsedItems []Item
	if err == nil && ctx.Err() == nil {
		parsedItems = sessionListItems(out.Items)
	}

	s.mu.Lock()
	if s.client != client || s.listGeneration != generation {
		s.mu.Unlock()
		return
	}
	s.listLoading = false
	s.invalidateViewLocked()
	if ctx.Err() != nil {
		s.listMutations = nil
		s.mu.Unlock()
		s.fire()
		return
	}
	if err != nil {
		s.listMutations = nil
		s.listErr = err
		s.mu.Unlock()
		s.fire()
		return
	}
	s.listAuthoritativeHost = s.eventsGeneration
	preserveSelected := s.listFollowGeneration != s.followGeneration && s.selectedID != ""
	var priorSelected Item
	if preserveSelected {
		for _, prior := range s.items {
			if prior.SessionID == s.selectedID {
				priorSelected = prior
				break
			}
		}
	}
	priorItems := s.items
	s.items = parsedItems
	priorPresentations := make(map[string]Item, len(priorItems))
	for _, prior := range priorItems {
		priorPresentations[prior.SessionID] = prior
	}
	for i := range s.items {
		if prior, ok := priorPresentations[s.items[i].SessionID]; ok {
			mergeSessionPresentation(&s.items[i], prior)
			mergeSessionSubagentIdentity(&s.items[i], prior)
		}
		s.syncSessionSubagentIdentityLocked(&s.items[i])
	}
	// A list response can finish after the user changed sessions and returned
	// to the same id. Preserve that current foreground row; the follow stream
	// owns its running/error state and the stale list must not replay it.
	if preserveSelected && priorSelected.SessionID != "" {
		for i := range s.items {
			if s.items[i].SessionID == priorSelected.SessionID {
				s.items[i] = priorSelected
				break
			}
		}
	}
	s.reconcileSidebarStatusesLocked(s.items)
	for _, row := range out.Items {
		if row.Projections != nil {
			s.applySubagentCatalogValuesLocked(row.SessionId, row.Projections.Values)
			if row.Projections.Kind == "" || row.Projections.Kind == "sequenced" {
				s.applyQuestionValuesForSessionLocked(row.SessionId, row.Projections.Values, row.Projections.AsOfSeq)
			}
		}
	}
	s.composerRosterClient = client
	selectedRemoved := false
	for _, mutation := range s.listMutations {
		if mutation.sessionID == s.selectedID && (mutation.kind == "api-session/removed" || mutation.kind == "api-session/deleted") {
			selectedRemoved = true
		}
		// Replay roster data, not notifications or intermediate selection effects.
		s.applyLifecycleMutationLocked(mutation, false)
	}
	if selectedRemoved {
		present := false
		for _, item := range s.items {
			if item.SessionID == s.selectedID {
				present = true
				break
			}
		}
		if !present {
			s.selectSessionLocked("")
		}
	}
	s.listMutations = nil
	s.applyPinnedFlagsLocked()
	startFollow := false
	if s.selectedID == "" && len(s.items) > 0 {
		pick := s.items[0].SessionID
		for _, it := range s.items {
			if it.ParentID == "" && it.Origin != "subagent" && it.SubagentIdentity == nil {
				pick = it.SessionID
				break
			}
		}
		s.selectSessionLocked(pick)
		startFollow = true
	}
	s.mu.Unlock()
	s.fire()
	if startFollow {
		s.FollowSelected(context.Background())
	}
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
	s.invalidateViewLocked()
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
		if s.selectedID != sessionID {
			s.mu.Unlock()
			return
		}
		s.promptBusy = false
		s.invalidateViewLocked()
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
				s.invalidateViewLocked()
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
			s.invalidateViewLocked()
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
		s.invalidateViewLocked()
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
	client, hostGeneration := s.client, s.eventsGeneration
	navigation, follow, selected := s.workspaceNavigationToken, s.followGeneration, s.selectedID
	s.mu.Unlock()
	if client == nil || sessionID == "" {
		return
	}
	go s.runForkSessionOwned(ctx, client, hostGeneration, sessionID, navigation, follow, selected)
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
	client, hostGeneration := s.client, s.composerHostGeneration
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
		// Stop belongs to the addressed Session, just like the original
		// Session.promptError. A late result cannot become another Session's
		// composer notice or cross a Host replacement, even if its ID is reused.
		if s.client != client || s.composerHostGeneration != hostGeneration || s.selectedID != sessionID {
			s.mu.Unlock()
			return
		}
		if err != nil {
			s.lastAction = err.Error()
		} else if out.Accepted {
			s.lastAction = "cancel accepted"
		} else {
			s.lastAction = "cancel rejected"
		}
		s.invalidateViewLocked()
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
		s.invalidateViewLocked()
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
		s.invalidateViewLocked()
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
	s.appendStreamDeltaLocked(role, delta)
}

func (s *Store) appendStreamDeltaLocked(role, delta string) {
	s.invalidateViewLocked()
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
	s.invalidateViewLocked()
	notify := s.notify
	s.mu.Unlock()
	if notify != nil {
		notify()
	}
}

// invalidateViewLocked advances the Snapshot revision. fire() calls it for the
// ordinary notification path; the state funnels below call it directly because
// several of them intentionally publish nothing (drafts, test fixtures and
// projections folded by an owning reader), yet Snapshot must observe them.
func (s *Store) invalidateViewLocked() {
	noteViewInvalidation()
	s.viewRev++
	s.rosterDirty = true
	if s.viewChanged != nil {
		close(s.viewChanged)
		s.viewChanged = nil
	}
}

func (s *Store) invalidateStreamViewLocked() {
	noteViewInvalidation()
	s.viewRev++
	if s.viewChanged != nil {
		close(s.viewChanged)
		s.viewChanged = nil
	}
}

func (s *Store) fire() {
	s.mu.Lock()
	s.invalidateViewLocked()
	fn := s.notify
	s.mu.Unlock()
	if fn != nil {
		fn()
	}
}

// fireStream publishes a revision that only advanced the streaming message
// state: one chunk of the assistant attempt in flight. Unlike fire it does not
// mark the roster dirty, so memos keyed on RosterProjectionKey (sidebar groups,
// filters, cost) survive a stream instead of rebuilding for every chunk.
// Any change that does reach the roster (a durable event, a rebaseline, a
// retired attempt) invalidates the view itself, under the lock, before the
// caller learns that something changed.
func (s *Store) fireStream() {
	s.mu.Lock()
	s.invalidateStreamViewLocked()
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
