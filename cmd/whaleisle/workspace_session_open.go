package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

// workspaceSessionOpenState is local to one mainState. A later creation gesture
// invalidates the earlier one even before either Host request returns.
type workspaceSessionOpenState struct{ generation uint64 }

type workspaceSessionOpenTarget interface {
	Create(context.Context) (string, error)
	Select(string) bool
	Current() bool
	ReportFailure(error) bool
}

type workspaceSessionOpen struct {
	state           *mainState
	store           *session.Store
	win             *mygo.Window
	target          workspaceSessionOpenTarget
	generation      uint64
	bootGeneration  int
	panelGeneration uint64
	sourceID        string
	sourceDraft     *composer.DraftState
	sourceSnapshot  json.RawMessage
	sourceRevision  uint64
	sourceEditSeq   float64
	sourceSending   bool
	sourcePreparing int
	sourceReply     *composer.GroupReplyDraft
	carryDraft      bool
}

// An existing local draft belongs to its original session. It must not be
// borrowed for a new-session request even if the Host still calls it blank.
func workspaceSessionDraftOccupied(d *composer.DraftState) bool {
	return d != nil && (d.Text != "" || len(d.Attachments) != 0 || d.EditSeq != 0 || d.Sending || d.Preparing != 0 || d.GroupReply != nil || d.FailedSendCount() != 0)
}

func workspaceSessionDraftRevision(d *composer.DraftState) uint64 {
	if d == nil || d.Client == nil {
		return 0
	}
	return d.Client.Revision()
}

// captureWorkspaceSessionOpen is called on the UI owner. It neither clears the
// source draft nor changes selection, and excludes all locally occupied drafts.
func (s *mainState) captureWorkspaceSessionOpen(win *mygo.Window, sessions *session.Store, expectedHost uint64, workspaceID string) (*workspaceSessionOpen, error) {
	if s == nil || sessions == nil || win == nil {
		return nil, session.ErrNoClient
	}
	snap := sessions.SelectionSummary()
	s.mu.Lock()
	if s.sessions != sessions || s.win != win || s.quitting {
		s.mu.Unlock()
		return nil, session.ErrNoClient
	}
	s.workspaceSessionOpen.generation++
	generation, bootGeneration := s.workspaceSessionOpen.generation, s.bootStatus.Generation
	var excluded []string
	for id, draft := range s.composerDrafts {
		if workspaceSessionDraftOccupied(draft) {
			excluded = append(excluded, id)
		}
	}
	source := s.composerDraftLocked(snap.SelectedID)
	sourceSnap := append(json.RawMessage(nil), source.SaveDraft()...)
	sourceRev := workspaceSessionDraftRevision(source)
	sourceEditSeq, sourceSending, sourcePreparing, sourceReply := source.EditSeq, source.Sending, source.Preparing, source.GroupReply
	s.mu.Unlock()
	panelGen := s.assistantService().ConversationGeneration()
	op := &workspaceSessionOpen{state: s, store: sessions, win: win, generation: generation, bootGeneration: bootGeneration,
		panelGeneration: panelGen, sourceID: snap.SelectedID, sourceDraft: source, sourceSnapshot: sourceSnap,
		sourceRevision: sourceRev, sourceEditSeq: sourceEditSeq, sourceSending: sourceSending,
		sourcePreparing: sourcePreparing, sourceReply: sourceReply}
	target, err := sessions.CaptureWorkspaceSessionTarget(expectedHost, workspaceID, excluded...)
	if err != nil {
		sessions.SetWorkspaceSessionErrorForHost(expectedHost, snap.ComposerSessionGeneration, snap.SelectedID, err)
		return nil, err
	}
	if target.SourceHostGeneration() != snap.WorkspaceHostGeneration || target.SourceSessionID() != snap.SelectedID || target.SourceFollowGeneration() != snap.ComposerSessionGeneration {
		return nil, nil
	}
	op.target = target
	return op, nil
}

// current is evaluated again on the UI owner after the RPC. It fences a newer
// creation, window/Host changes, source replacement and edits, even when text
// was subsequently changed back to the same value.
func (op *workspaceSessionOpen) current() bool {
	if op == nil || op.state == nil || op.store == nil || op.target == nil {
		return false
	}
	s := op.state
	asst := s.assistantService()
	panelGen := asst.ConversationGeneration()
	asstOpen := asst.Open()
	s.mu.Lock()
	current := !s.quitting && s.sessions == op.store && s.win == op.win &&
		s.bootStatus.Generation == op.bootGeneration && s.workspaceSessionOpen.generation == op.generation &&
		panelGen == op.panelGeneration && !s.marketOpen && !s.scheduleOpen && !asstOpen &&
		!s.settingsOpen && !s.maintenanceOpen && !s.usageOpen && !s.imOpen && !s.remoteOpen && !s.webviewOpen && !s.browserPreviewOpen &&
		s.composerDrafts[op.sourceID] == op.sourceDraft
	if current {
		d := op.sourceDraft
		current = d != nil && d.EditSeq == op.sourceEditSeq && d.Sending == op.sourceSending && d.Preparing == op.sourcePreparing && d.GroupReply == op.sourceReply &&
			workspaceSessionDraftRevision(d) == op.sourceRevision && bytes.Equal(d.SaveDraft(), op.sourceSnapshot)
	}
	s.mu.Unlock()
	return current && op.target.Current()
}

func (op *workspaceSessionOpen) finish(id string, err error) bool {
	if !op.current() {
		return false
	}
	if err != nil {
		op.target.ReportFailure(err)
		return false
	}
	// A draft can appear on an initially empty candidate while the Host is
	// responding. Do not activate it as a new blank session or overwrite it.
	s := op.state
	s.mu.Lock()
	destination := s.composerDrafts[id]
	occupied := id != op.sourceID && workspaceSessionDraftOccupied(destination)
	s.mu.Unlock()
	if occupied {
		return false
	}
	if !op.target.Select(id) {
		return false
	}
	if op.carryDraft && id != op.sourceID {
		s.mu.Lock()
		if s.composerDrafts[op.sourceID] == op.sourceDraft && !workspaceSessionDraftOccupied(s.composerDrafts[id]) {
			s.composerDrafts[id] = op.sourceDraft
			op.sourceDraft.TargetID = id
			delete(s.composerDrafts, op.sourceID)
		}
		s.mu.Unlock()
	}
	return true
}

// All global entry points use the same Host membership and UI draft owner as
// workspace row creation. A ready empty Host retains the no-directory flow.
func (s *mainState) openGlobalWorkspaceSession(win *mygo.Window, sessions *session.Store) {
	if sessions == nil {
		return
	}
	state := sessions.Published().Value
	if !state.WorkspaceReady || state.WorkspaceError != nil {
		// Two different states used to share one waiting sentence. Without a
		// workspace there is nothing to reconnect to: the next step is picking a
		// folder, and the sentence has to say so.
		if state.WorkspaceError == nil && strings.TrimSpace(sessions.WorkspaceCwd()) == "" {
			sessions.SetLastAction(directoryPickerText("还没有选择工作区。请先选择一个文件夹，再开始新会话。", "No workspace is chosen yet. Pick a folder first, then start a new conversation."))
			return
		}
		sessions.SetLastAction(directoryPickerText("工作区列表尚未就绪，请等待连接恢复后重试。", "The workspace list is not ready. Reconnect and try again."))
		return
	}
	id := sessions.NewSessionWorkspaceID()
	if id == "" {
		s.revealMainConversation()
		sessions.Select("")
		retireStaleInlineVisualPages()
		return
	}
	s.openWorkspaceSession(win, sessions, state.WorkspaceLayout.HostGeneration, id)
}

func (s *mainState) openWorkspaceSession(win *mygo.Window, sessions *session.Store, expectedHost uint64, workspaceID string) {
	if sessions == nil || sessions.WorkspaceHostGeneration() != expectedHost {
		return
	}
	s.revealMainConversation()
	op, err := s.captureWorkspaceSessionOpen(win, sessions, expectedHost, workspaceID)
	if err != nil {
		return
	}
	if op == nil {
		return
	}
	go func() {
		id, err := op.target.Create(context.Background())
		if win.IsDestroyed() {
			return
		}
		ui.FrameTraceCause("session-create")
		win.Update(func() { op.finish(id, err) })
	}()
}
