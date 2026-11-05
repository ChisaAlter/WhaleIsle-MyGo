package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// WorkspaceSessionTarget owns a single sidebar creation gesture. Host and
// navigation generations are captured before any asynchronous work begins.
// The result can only be opened from that same navigation, even after an
// away-and-back selection or reconnection using the same client.
type WorkspaceSessionTarget struct {
	store                                                *Store
	client                                               *dsh.Client
	endpoint                                             dsh.Endpoint
	mux                                                  *dsh.Mux
	hostGeneration, composerGeneration, followGeneration uint64
	workspaceNavigationToken                             uint64
	selectedID, cwd, workspaceID, workspacePath          string
	scratch                                              bool
	stagedPreset                                         string
	excluded                                             map[string]bool
	result                                               string
	consumed, busy                                       bool
}

// CaptureWorkspaceSessionTarget expands the addressed group and captures the
// current owner. ::scratch delegates to the existing no-directory creation API.
// expectedHost is the rendered sidebar Host generation; excludeIDs have drafts.
func (s *Store) CaptureWorkspaceSessionTarget(expectedHost uint64, workspaceID string, excludeIDs ...string) (*WorkspaceSessionTarget, error) {
	if s == nil {
		return nil, ErrNoClient
	}
	if workspaceID == "" || workspaceID != strings.TrimSpace(workspaceID) {
		return nil, errors.New("invalid workspace identity")
	}
	s.mu.Lock()
	if s.client == nil {
		s.mu.Unlock()
		return nil, ErrNoClient
	}
	if s.eventsGeneration != expectedHost {
		s.mu.Unlock()
		return nil, errors.New("Host connection changed")
	}
	if s.workspaceErr != nil {
		err := s.workspaceErr
		s.mu.Unlock()
		return nil, err
	}
	scratch := workspaceID == "::scratch"
	ws, ok := s.workspaceMemberships[workspaceID]
	workspacePath := ws.Path
	if scratch {
		workspacePath, ok = s.scratchCwd, true
	}
	// A no-directory gesture may precede the first workspace baseline. Its
	// target waits for that same Host's scratch root; ordinary workspace rows
	// still require an authoritative membership before creation.
	if !ok || (!scratch && !s.workspaceReady) || (strings.TrimSpace(workspacePath) == "" && (s.workspaceReady || !scratch)) {
		s.mu.Unlock()
		return nil, errors.New("workspace is no longer available")
	}
	t := &WorkspaceSessionTarget{store: s, client: s.client, endpoint: s.client.Endpoint, mux: s.mux,
		hostGeneration: s.eventsGeneration, composerGeneration: s.composerHostGeneration,
		followGeneration: s.followGeneration, workspaceNavigationToken: s.workspaceNavigationToken, selectedID: s.selectedID, cwd: s.cwd,
		workspaceID: workspaceID, workspacePath: workspacePath, scratch: scratch, excluded: map[string]bool{}}
	t.stagedPreset = s.stagedPresetLocked(workspacePath)
	for _, id := range excludeIDs {
		t.excluded[id] = true
	}
	changed := s.expandWorkspaceGroupLocked(workspaceID)
	s.mu.Unlock()
	if changed {
		s.fire()
	}
	return t, nil
}

// SetWorkspaceSessionErrorForHost reports capture failures without letting a
// stale sidebar action write into a replacement Host or later navigation.
func (s *Store) SetWorkspaceSessionErrorForHost(expectedHost, expectedFollow uint64, selectedID string, err error) bool {
	if s == nil || err == nil {
		return false
	}
	s.mu.Lock()
	if s.client == nil || s.eventsGeneration != expectedHost || s.followGeneration != expectedFollow || s.selectedID != selectedID {
		s.mu.Unlock()
		return false
	}
	s.lastAction = "create workspace session: " + err.Error()
	s.mu.Unlock()
	s.fire()
	return true
}

// ExpandWorkspaceGroupForHost opens the addressed row only for the Host that
// rendered it. Repeated calls keep an already open group open.
func (s *Store) ExpandWorkspaceGroupForHost(expectedHost uint64, id string) bool {
	if s == nil || id == "" {
		return false
	}
	s.mu.Lock()
	valid := s.client != nil && s.eventsGeneration == expectedHost && s.workspaceReady
	if valid && id != "::scratch" {
		_, valid = s.workspaceMemberships[id]
	}
	changed := valid && s.expandWorkspaceGroupLocked(id)
	s.mu.Unlock()
	if changed {
		s.fire()
	}
	return valid
}

func (s *Store) expandWorkspaceGroupLocked(id string) bool {
	if collapsed, explicit := s.workspaceLayout.view.Collapsed[id]; explicit && !collapsed {
		return false
	}
	if s.workspaceLayout.view.Collapsed == nil {
		s.workspaceLayout.view.Collapsed = map[string]bool{}
	}
	s.workspaceLayout.view.Collapsed[id] = false
	s.workspacePreferencesChangedLocked("collapsed", id)
	return true
}

func (t *WorkspaceSessionTarget) hostCurrentLocked() bool {
	s := t.store
	return s.client == t.client && s.client != nil && s.client.Endpoint == t.endpoint &&
		s.mux == t.mux && s.eventsGeneration == t.hostGeneration && s.composerHostGeneration == t.composerGeneration
}
func (t *WorkspaceSessionTarget) workspaceCurrentLocked() bool {
	if t.store.workspaceErr != nil {
		return false
	}
	if t.scratch {
		if !t.store.workspaceReady {
			return t.workspacePath == ""
		}
		if t.workspacePath == "" {
			// Bind once to the first authoritative root, while the captured
			// Host and navigation generations remain owned by this target.
			t.workspacePath = t.store.scratchCwd
		}
		return t.workspacePath != "" && t.store.scratchCwd == t.workspacePath
	}
	if !t.store.workspaceReady {
		return false
	}
	ws, ok := t.store.workspaceMemberships[t.workspaceID]
	return ok && ws.Path == t.workspacePath
}
func (t *WorkspaceSessionTarget) currentLocked() bool {
	return !t.consumed && t.hostCurrentLocked() && t.workspaceCurrentLocked() &&
		t.store.selectedID == t.selectedID && t.store.followGeneration == t.followGeneration && t.store.cwd == t.cwd &&
		t.store.workspaceNavigationToken == t.workspaceNavigationToken
}

// SourceSessionID is the selected identity captured for the gesture.
func (t *WorkspaceSessionTarget) SourceSessionID() string {
	if t == nil {
		return ""
	}
	return t.selectedID
}

// SourceHostGeneration identifies the rendered Host that owned the gesture.
func (t *WorkspaceSessionTarget) SourceHostGeneration() uint64 {
	if t == nil {
		return 0
	}
	return t.hostGeneration
}

// SourceFollowGeneration detects away-and-back selection during capture.
func (t *WorkspaceSessionTarget) SourceFollowGeneration() uint64 {
	if t == nil {
		return 0
	}
	return t.followGeneration
}

func (t *WorkspaceSessionTarget) Current() bool {
	if t == nil || t.store == nil {
		return false
	}
	t.store.mu.Lock()
	defer t.store.mu.Unlock()
	return t.currentLocked()
}

func (t *WorkspaceSessionTarget) candidateLocked(id string) bool {
	if t.excluded[id] {
		return false
	}
	s := t.store
	ws, ok := s.workspaceMemberships[t.workspaceID]
	if !ok || ws.Path != t.workspacePath {
		return false
	}
	member := false
	for _, value := range ws.SessionIDs {
		if value == id {
			member = true
			break
		}
	}
	if !member {
		return false
	}
	for _, value := range s.archivedSessionIDs {
		if value == id {
			return false
		}
	}
	for _, item := range s.items {
		if item.SessionID == id {
			return item.Blank && !item.Running && !item.Pinned && item.ParentID == "" && item.Origin == "" && item.Title == "" && item.Presentation == nil &&
				hostPathKey(item.Cwd) != "" && hostPathKey(item.Cwd) == hostPathKey(ws.Path)
		}
	}
	return false
}

// Create uses the existing no-directory operation for scratch and the real
// session/create API with workspaceId for a registered workspace. A named blank
// is proposed only after blankReuse succeeds; the Host makes the final decision.
// Creation never changes the selected session or any UI draft.
func (t *WorkspaceSessionTarget) Create(ctx context.Context) (string, error) {
	if t == nil || t.store == nil {
		return "", errors.New("workspace creation has no owner")
	}
	if ctx == nil {
		return "", errors.New("workspace creation has no context")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s := t.store
	s.mu.Lock()
	if !t.currentLocked() || t.result != "" || t.busy {
		s.mu.Unlock()
		return "", errors.New("workspace creation owner changed")
	}
	t.busy = true
	defer func() {
		s.mu.Lock()
		t.busy = false
		if ctx.Err() != nil {
			t.consumed = true
		}
		s.mu.Unlock()
	}()
	var candidates []string
	if !t.scratch {
		for _, item := range s.items {
			if t.candidateLocked(item.SessionID) {
				candidates = append(candidates, item.SessionID)
			}
		}
	}
	s.mu.Unlock()
	if t.scratch {
		excluded := make([]string, 0, len(t.excluded))
		for id := range t.excluded {
			excluded = append(excluded, id)
		}
		id, err := s.createNoDirectorySessionForTarget(ctx, t.client, t.hostGeneration, t, excluded...)
		if err != nil {
			return "", fmt.Errorf("create no-directory session: %w", err)
		}
		s.mu.Lock()
		if !t.currentLocked() || ctx.Err() != nil {
			s.mu.Unlock()
			return "", errors.New("workspace creation owner changed")
		}
		t.result = id
		s.mu.Unlock()
		return id, nil
	}
	var candidate string
	for _, id := range candidates {
		var out api.SessionBlankReuseValue
		err := t.client.Call(ctx, "session/blankReuse", map[string]any{"request": map[string]any{"sessionId": id}}, &out)
		if err != nil {
			return "", fmt.Errorf("check blank workspace session: %w", err)
		}
		s.mu.Lock()
		current, valid := t.currentLocked(), t.candidateLocked(id)
		s.mu.Unlock()
		if !current || ctx.Err() != nil {
			return "", errors.New("workspace creation owner changed")
		}
		if out.Reusable && valid {
			candidate = id
			break
		}
	}
	s.mu.Lock()
	current := t.currentLocked() && t.result == ""
	if candidate != "" && !t.candidateLocked(candidate) {
		candidate = ""
	}
	s.mu.Unlock()
	if !current || ctx.Err() != nil {
		return "", errors.New("workspace creation owner changed")
	}
	req := api.SessionCreateRequest{WorkspaceId: &t.workspaceID}
	if t.stagedPreset != "" {
		req.AgentPreset = &t.stagedPreset
	}
	if candidate != "" {
		req.SessionId = &candidate
	}
	var out api.SessionCreateValue
	err := t.client.Call(ctx, "session/create", map[string]any{"request": req}, &out)
	if err != nil {
		// This Host error names a real session already published before attach
		// failed. Keep it visible as ungrouped; do not invent a membership.
		var failure *dsh.Error
		if errors.As(err, &failure) && failure.Code == "session/workspace-attach-failed" {
			var detail struct {
				SessionID string `json:"sessionId"`
			}
			if json.Unmarshal(failure.Details, &detail) == nil && validCreatedSessionID(detail.SessionID) {
				t.recordReceipt(detail.SessionID, false)
			}
		}
		return "", fmt.Errorf("create workspace session: %w", err)
	}
	if !validCreatedSessionID(out.SessionId) {
		return "", errors.New("session/create returned no valid session")
	}
	if !t.recordReceipt(out.SessionId, true) {
		return "", errors.New("Host workspace changed after session creation")
	}
	s.mu.Lock()
	if !t.currentLocked() || ctx.Err() != nil {
		s.mu.Unlock()
		return "", errors.New("workspace creation owner changed")
	}
	t.result = out.SessionId
	s.mu.Unlock()
	return out.SessionId, nil
}

func validCreatedSessionID(id string) bool { return id != "" && strings.TrimSpace(id) == id }

// recordReceipt projects only the actual Host receipt; it is safe even if the
// user navigated elsewhere while the Host was executing the request.
func (t *WorkspaceSessionTarget) recordReceipt(id string, attached bool) bool {
	s := t.store
	s.mu.Lock()
	if !t.hostCurrentLocked() || (attached && !t.workspaceCurrentLocked()) {
		s.mu.Unlock()
		return false
	}
	found := false
	for _, item := range s.items {
		if item.SessionID == id {
			found = true
			break
		}
	}
	if !found {
		item := Item{SessionID: id, Blank: true, AgentAvailable: true}
		if attached {
			item.Cwd = t.workspacePath
		}
		s.items = append(s.items, item)
		if s.listLoading {
			s.listMutations = append(s.listMutations, lifecycleMutation{kind: "api-session/added", sessionID: id, item: item})
		}
	}
	if attached {
		ws := s.workspaceMemberships[t.workspaceID]
		present := false
		for _, member := range ws.SessionIDs {
			if member == id {
				present = true
				break
			}
		}
		if !present {
			ws.SessionIDs = append(append([]string(nil), ws.SessionIDs...), id)
			s.workspaceMemberships[t.workspaceID] = ws
			s.signalWorkspaceLocked()
		}
	}
	s.invalidateViewLocked()
	s.mu.Unlock()
	s.fire()
	return true
}

// ReportFailure only publishes an error while the originating navigation is
// still current. Late errors cannot replace another Host's status.
func (t *WorkspaceSessionTarget) ReportFailure(err error) bool {
	if t == nil || t.store == nil || err == nil {
		return false
	}
	s := t.store
	s.mu.Lock()
	if !t.currentLocked() {
		s.mu.Unlock()
		return false
	}
	s.lastAction = err.Error()
	s.mu.Unlock()
	s.fire()
	return true
}

// workspaceSelectedOwner pins refresh admission to the selected result and
// Host epoch. Each adapter checks it while holding the Store lock.
type workspaceSelectedOwner struct {
	client                                               *dsh.Client
	hostGeneration, composerGeneration, followGeneration uint64
	sessionID, cwd                                       string
}

func (o *workspaceSelectedOwner) currentLocked(s *Store) bool {
	return o != nil && o.client != nil && s.client == o.client &&
		s.eventsGeneration == o.hostGeneration && s.composerHostGeneration == o.composerGeneration &&
		s.followGeneration == o.followGeneration && s.selectedID == o.sessionID && s.cwd == o.cwd
}

// Select consumes exactly the result owned by this gesture, atomically with
// its navigation fence. It never clears or migrates a composer draft.
func (t *WorkspaceSessionTarget) Select(id string) bool {
	if t == nil || t.store == nil || !validCreatedSessionID(id) {
		return false
	}
	s := t.store
	s.mu.Lock()
	if !t.currentLocked() || t.result != id {
		s.mu.Unlock()
		return false
	}
	t.consumed = true
	if t.stagedPreset != "" && s.stagedPresetLocked(t.workspacePath) == t.stagedPreset {
		delete(s.presetDraft.staged, filepathClean(t.workspacePath))
	}
	if s.selectedID == id {
		control, changed := s.beginControlWatchLocked(context.Background())
		s.mu.Unlock()
		if changed {
			s.fire()
		}
		if control != nil {
			go control()
		}
		return true
	}
	s.selectSessionLocked(id)
	generation, client, mux := s.followGeneration, s.client, s.mux
	owner := workspaceSelectedOwner{client: client, hostGeneration: s.eventsGeneration, composerGeneration: s.composerHostGeneration,
		followGeneration: generation, sessionID: id, cwd: s.cwd}
	var followCtx, bandCtx context.Context
	if mux != nil {
		followCtx, s.followCancel = context.WithCancel(context.Background())
		if s.bandCancel != nil {
			s.bandCancel()
		}
		bandCtx, s.bandCancel = context.WithCancel(context.Background())
		s.bandActions, s.bandGeneration = nil, 0
	}
	s.mu.Unlock()
	s.fire()
	ctx := context.Background()
	if mux != nil {
		go s.runFollow(followCtx, mux, id, generation)
		s.WatchControl(ctx)
		go s.runBandWatch(bandCtx, mux, id)
	}
	go s.runRefreshGoalForOwner(ctx, client, id, generation, &owner)
	s.refreshProjectionsForOwner(ctx, &owner)
	s.refreshSkillsForOwner(ctx, &owner)
	s.refreshFeedbackForOwner(ctx, &owner)
	return true
}

// NewSessionWorkspaceID resolves Host membership before path or recency. The
// scratch group remains a separate identity even when another group has its cwd.
func (s *Store) NewSessionWorkspaceID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.newSessionWorkspaceIDLocked()
}

func (s *Store) newSessionWorkspaceIDLocked() string {
	rows := s.workspaceRowsLocked()
	for _, ws := range rows {
		for _, id := range ws.SessionIDs {
			if id == s.selectedID && id != "" {
				return ws.WorkspaceID
			}
		}
	}
	for _, it := range s.items {
		if it.SessionID == s.selectedID && s.scratchCwd != "" && hostPathKey(it.Cwd) == hostPathKey(s.scratchCwd) {
			return "::scratch"
		}
	}
	if ws, ok := s.workspaceMemberships[s.workspaceID]; ok && hostPathKey(ws.Path) == hostPathKey(s.cwd) {
		return ws.WorkspaceID
	}
	for _, ws := range rows {
		if hostPathKey(ws.Path) == hostPathKey(s.cwd) {
			return ws.WorkspaceID
		}
	}
	var target string
	var latest float64
	for _, ws := range rows {
		stamp := float64(0)
		if created, err := time.Parse(time.RFC3339Nano, ws.CreatedAt); err == nil {
			stamp = float64(created.UnixMilli())
		}
		for _, id := range ws.SessionIDs {
			for _, it := range s.items {
				if it.SessionID == id && it.Presentation == nil && it.UpdatedAt > stamp {
					stamp = it.UpdatedAt
				}
			}
		}
		if target == "" || stamp > latest {
			target, latest = ws.WorkspaceID, stamp
		}
	}
	return target
}
