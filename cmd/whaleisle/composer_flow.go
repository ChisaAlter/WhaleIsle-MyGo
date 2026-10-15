package main

import (
	"context"
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

func (s *mainState) addComposerFiles(win *mygo.Window, sessions *session.Store, id string, d *composer.DraftState, paths []string) {
	if d.Sending {
		return
	}
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		a := session.ComposerAttachment{Path: path, Name: filepath.Base(path)}
		index := len(d.Attachments)
		d.Attachments = append(d.Attachments, a)
		if id != "" && win != nil && sessions != nil {
			d.Preparing++
			go func() {
				prepared, err := sessions.PrepareComposerAttachment(context.Background(), id, a)
				win.Update(func() {
					d.Preparing--
					if index >= len(d.Attachments) || d.Attachments[index].Path != a.Path {
						return
					}
					if err != nil {
						d.Error = err.Error()
						return
					}
					d.Attachments[index] = prepared
				})
			}()
		}
	}
	d.Error = ""
}

func (s *mainState) submitComposer(win *mygo.Window, sessions *session.Store, d *composer.DraftState, id, cwd string, editSeq float64) {
	if d.Sending || d.Preparing > 0 || (strings.TrimSpace(d.Text) == "" && len(d.Attachments) == 0) {
		return
	}
	if win == nil || sessions == nil {
		return
	}
	editSeq = d.EditSeq
	if editSeq > 0 && !d.EditAllowed(sessions.Snapshot()) {
		d.Error = i18n.T(i18n.KeyChatEditStale)
		return
	}
	text := d.Text
	// Bare feedback is local; text-bearing feedback stays on the Host command path.
	if editSeq == 0 && len(d.Attachments) == 0 && strings.TrimSpace(text) == "/feedback" {
		feedbackID := id
		if d.TargetID != "" {
			feedbackID = d.TargetID
		}
		// The dialog belongs to the visibly selected session. A failed home
		// draft may hold a created target without selecting it; keep that draft.
		if feedbackID != "" && sessions.Snapshot().SelectedID == feedbackID && sessions.OpenSessionFeedback(feedbackID) {
			d.Text = ""
			d.Error = ""
			d.Client.Sync()
		} else {
			d.Error = "Select a session before opening feedback"
		}
		return
	}
	// Local commands never consume attachments or dispatch to the Host.
	if editSeq == 0 && len(d.Attachments) == 0 && strings.TrimSpace(text) == "/clear" {
		d.Text = ""
		d.Client.Sync()
		return
	}
	if editSeq == 0 && len(d.Attachments) == 0 && strings.TrimSpace(text) == "/new" {
		sessions.CreateSession(context.Background())
		return
	}
	if editSeq == 0 && id != "" && !strings.HasPrefix(strings.TrimSpace(text), "/") {
		s.submitDetachedComposer(win, sessions, d, id)
		return
	}
	attachments := append([]session.ComposerAttachment(nil), d.Attachments...)
	if d.TargetID != "" {
		id = d.TargetID
	}
	owner, captureErr := sessions.CaptureComposerTarget(id)
	if captureErr != nil {
		d.Error = captureErr.Error()
		return
	}
	d.Sending = true
	d.Error = ""
	go func() {
		ctx := context.Background()
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
			err = owner.Submit(ctx, text, attachments, editSeq, "")
		}
		win.Update(func() {
			d.Sending = false
			d.TargetID = target
			if err != nil {
				d.Error = err.Error()
				d.Attachments = attachments
				return
			}
			if editSeq > 0 {
				d.EndEdit()
				return
			}
			// Typing while sending is allowed. Never erase newer edits.
			if d.Text == text {
				d.Text = ""
			}
			d.Attachments = nil
			d.Error = ""
			d.TargetID = ""
			d.Client.Sync()
			if id == "" && sessions.Snapshot().SelectedID == "" {
				s.mu.Lock()
				s.composerDrafts[target] = d
				delete(s.composerDrafts, "")
				s.mu.Unlock()
				sessions.Select(target)
				sessions.RefreshList(context.Background())
				sessions.FollowSelected(context.Background())
			}
		})
	}()
}
