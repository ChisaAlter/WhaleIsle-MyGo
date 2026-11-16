package main

import (
	"context"
	"errors"
	"log"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/i18n"
)

// composerDraftLocked is called while mainState.mu is held. A draft's pointer
// stays valid across paints and navigation, including native IME callbacks.
func (s *mainState) composerDraftLocked(id string) *composer.DraftState {
	if s.composerDrafts == nil {
		s.composerDrafts = map[string]*composer.DraftState{}
	}
	d := s.composerDrafts[id]
	if d == nil {
		d = composer.NewDraftState("")
		s.composerDrafts[id] = d
	}
	return d
}

// Match the resident editor's unmount cleanup: navigation restores the draft
// that was borrowed for editing. An admitted send keeps its original owner
// until its completion callback; EndEdit deliberately refuses while sending.
func (s *mainState) observeComposerEdit(rendered bool, sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, d := range s.composerDrafts {
		// Addressed children have their own navigation lease and can be visible
		// beside the root composer. Their stable opaque keys are not root IDs.
		if strings.HasPrefix(id, "\x00child-draft:") {
			continue
		}
		if !rendered || id != sessionID {
			d.EndEdit()
			d.RetireBinding()
		}
	}
}

func (s *mainState) addComposerFiles(win *mygo.Window, sessions *session.Store, id string, d *composer.DraftState, paths []string) {
	if d == nil || d.Sending {
		return
	}
	if sessions != nil {
		if err := validateComposerImageIntake(d.Attachments, paths, sessions.ComposerImageLimits(id)); err != nil {
			d.Error = err.Error()
			return
		}
	}
	cwd, local := s.nativeComposerFileScope(sessions, id, d)
	uploads, err := insertNativeComposerFiles(d, cwd, paths, local)
	if err != nil {
		d.Error = err.Error()
		return
	}
	for _, path := range uploads {
		if strings.TrimSpace(path) == "" {
			continue
		}
		a := session.ComposerAttachment{Path: path, Name: filepath.Base(path)}
		index := len(d.Attachments)
		d.Attachments = append(d.Attachments, a)
		s.prepareComposerFile(win, sessions, id, d, index, false)
	}
	d.Error = ""
}

func (s *mainState) submitComposer(win *mygo.Window, sessions *session.Store, d *composer.DraftState, id, cwd string, editSeq float64, accelerated bool, hostGeneration uint64) {
	if d.Sending || d.Preparing > 0 {
		log.Printf("composer submit dropped: busy sending=%v preparing=%d id=%s", d.Sending, d.Preparing, id)
		return
	}
	if win == nil || sessions == nil {
		return
	}
	s.mu.Lock()
	current := s.sessions == sessions && s.composerDrafts[id] == d
	preferences := s.settings
	s.mu.Unlock()
	live := sessions.Snapshot()
	if !current || live.SelectedID != id || live.ComposerHostGeneration != hostGeneration {
		log.Printf("composer submit dropped: stale current=%v selected=%q want=%q hostGen=%d/%d", current, live.SelectedID, id, live.ComposerHostGeneration, hostGeneration)
		return
	}
	// Local contributions own their bare invocation before workspace/prompt
	// admission. File and feedback actions retain existing attachment state.
	isRoom, _ := s.assistantGroupMembers(id)
	if !isRoom && strings.HasPrefix(strings.TrimSpace(d.Text), "/") {
		sessions.EnsureSlashCommands(s.lifecycleContext())
		live = sessions.Snapshot()
	}
	if !isRoom && s.handleSessionLogExportCommand(win, sessions, d, live) {
		return
	}
	if !isRoom && d.HandleNativeCommand(live, func(owner composer.NativeCommandOwner, name string) bool {
		return s.nativeComposerCommandAction(win, sessions, d, id, cwd, owner, name)
	}) {
		return
	}
	if failure := d.FailedAttachmentError(); failure != "" {
		d.Error = failure
		return
	}
	hero := !live.HasConversationContent() && !live.PromptAttempted && !live.PromptBusy
	for _, item := range live.Items {
		if item.SessionID == id && item.Running {
			hero = false
		}
	}
	if composer.RequiresWorkspace(live, hero) {
		// Match the Hero card's keyboard/click route even for a retained
		// submit callback: choosing a workspace precedes creating a Session.
		log.Printf("composer submit dropped: requires-workspace hero=%v sel=%q cwd=%q scratch=%q wsReady=%v workspaces=%d", hero, live.SelectedID, live.SelectedCwd, live.ScratchCwd, live.WorkspaceReady, len(live.Workspaces))
		s.pickAndApplyWorkspace(win, sessions, cwd)
		return
	}
	if strings.TrimSpace(d.Text) == "" && len(d.Attachments) == 0 {
		if d.CanSteerPendingQueue(id, accelerated) {
			sessions.SteerComposerQueue(s.lifecycleContext(), id, hostGeneration)
		}
		return
	}
	editSeq = d.EditSeq
	if editSeq > 0 && !d.EditAllowed(sessions.Snapshot()) {
		d.Error = i18n.T(i18n.KeyChatEditStale)
		return
	}
	text := d.Text
	inputText := text
	groupReply := false
	if isRoom && editSeq == 0 {
		thread, body, claimed, err := session.ParseGroupThreadReply(text)
		if err != nil {
			d.Error = err.Error()
			return
		}
		if claimed {
			if body == "" && len(d.Attachments) == 0 {
				d.Error = i18n.T(i18n.KeyCmdEnterAReplyOrAddAnAttachment)
				return
			}
			text, err = session.EncodeGroupThreadReply(thread, body)
			if err != nil {
				d.Error = err.Error()
				return
			}
			groupReply = true
		}
	}
	if groupReply && d.TargetID != "" && d.TargetID != id {
		d.Error = i18n.T(i18n.KeyCmdComposerReplyTargetChanged)
		return
	}
	sourceID := id
	if d.TargetID != "" {
		id = d.TargetID
	}
	owner, captureErr := sessions.CaptureComposerGestureTarget(sourceID, id, hostGeneration, accelerated, preferences.BusyEnterFor)
	if captureErr != nil {
		d.Error = captureErr.Error()
		return
	}
	if groupReply {
		owner = owner.GroupReplyTarget()
	}
	if !groupReply && editSeq == 0 && id != "" && sourceID != "" && !strings.HasPrefix(strings.TrimSpace(text), "/") {
		s.submitDetachedComposer(win, d, sourceID, owner)
		return
	}
	attachments := append([]session.ComposerAttachment(nil), d.Attachments...)
	d.Sending = true
	d.Error = ""
	go func() {
		ctx, cancel := context.WithTimeout(s.lifecycleContext(), detachedSendTimeout)
		defer cancel()
		target := id
		var err error
		if target == "" {
			target, err = owner.CreateSession(ctx, cwd)
		}
		if err == nil {
			for i := range attachments {
				attachments[i], err = owner.PrepareAttachment(ctx, attachments[i])
				if err != nil {
					break
				}
			}
		}
		if err == nil {
			requestID := ""
			if groupReply {
				requestID = owner.BeginSubmission(text, attachments)
			}
			err = owner.Submit(ctx, text, attachments, editSeq, requestID)
		}
		win.Update(func() {
			s.mu.Lock()
			current := s.composerDrafts[id] == d
			quitting := s.quitting
			s.mu.Unlock()
			d.Sending = false
			if quitting || !current {
				return
			}
			d.TargetID = target
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					d.Error = i18n.T(i18n.KeyCmdComposerSendTimeout)
				} else {
					d.Error = err.Error()
				}
				d.Attachments = attachments
				return
			}
			if editSeq > 0 {
				d.EndEdit()
				return
			}
			// Typing while sending is allowed. Never erase newer edits.
			if d.Text == inputText {
				d.Text = ""
			}
			if groupReply {
				d.GroupReply = nil
			}
			d.Attachments = nil
			d.Error = ""
			d.TargetID = ""
			d.Client.ClearHistory()
			if id == "" && sessions.SelectedSessionID() == "" {
				s.mu.Lock()
				s.composerDrafts[target] = d
				delete(s.composerDrafts, "")
				s.mu.Unlock()
				sessions.Select(target)
				sessions.RefreshList(s.lifecycleContext())
				sessions.FollowSelected(s.lifecycleContext())
			}
		})
	}()
}
