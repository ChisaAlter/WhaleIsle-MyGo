package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/i18n"
)

// detachedSendTimeout bounds preparation and the admission RPC. Durable
// confirmation belongs to Session and has no page/staging deadline.
const detachedSendTimeout = 2 * time.Minute

func (s *mainState) submitDetachedComposer(win *mygo.Window, d *composer.DraftState, id string, owner *session.ComposerTarget) {
	flight, ok := d.BeginDetachedSend()
	if !ok {
		log.Printf("composer submit dropped: detach-refused id=%s sending=%v preparing=%d editSeq=%v att=%d textLen=%d", id, d.Sending, d.Preparing, d.EditSeq, len(d.Attachments), len(d.Text))
		return
	}
	requestID := owner.BeginSubmission(flight.Text, flight.Attachments)
	if requestID == "" {
		d.CompleteDetachedSend(flight.ID, flight.Attachments, errors.New(i18n.T(i18n.KeyCmdComposerHostChanged)))
		return
	}
	d.BindDetachedSubmission(flight.ID, requestID, owner.SessionID)
	draftSnapshot := d.SaveDraft()
	go func() {
		// Bound local preparation/admission, never the accepted business task.
		ctx, cancel := context.WithTimeout(s.lifecycleContext(), detachedSendTimeout)
		defer cancel()
		log.Printf("qa-flight %d: begin (attachments=%d)", flight.ID, len(flight.Attachments))
		attachments := append([]session.ComposerAttachment(nil), flight.Attachments...)
		err := owner.StageSubmissionDraft(requestID, flight.Text, id, draftSnapshot)
		for i := range attachments {
			if err != nil {
				break
			}
			attachments[i], err = owner.PrepareAttachment(ctx, attachments[i])
			log.Printf("qa-flight %d: prepare %d -> err=%v", flight.ID, i, err)
			if err != nil {
				break
			}
		}
		if err == nil {
			owner.UpdateSubmissionAttachments(requestID, attachments)
			err = owner.Submit(ctx, flight.Text, attachments, 0, requestID)
			log.Printf("qa-flight %d: submit -> err=%v", flight.ID, err)
			// The Session owns confirmation after dispatch. RPC acceptance frees
			// the input, but this snapshot remains until durable handoff; a page
			// switch or staging deadline is never a reason to restore it for retry.
		} else {
			stageErr := err
			err = owner.AbandonSubmission(requestID, stageErr)
			log.Printf("qa-flight %d: abandon -> err=%v", flight.ID, err)
			// Abandon succeeding must not mask the stage failure: a nil here
			// would complete the flight as sent and silently drop the draft.
			if err == nil {
				err = stageErr
			}
		}
		if errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, session.ErrSubmissionUnconfirmed) {
			err = errors.New(i18n.T(i18n.KeyCmdComposerSendTimeout))
		}
		win.Update(func() {
			s.mu.Lock()
			current := s.composerDrafts[id] == d
			s.mu.Unlock()
			if current {
				if err == nil {
					d.AwaitDetachedSubmission(flight.ID, attachments)
				} else {
					d.CompleteDetachedSend(flight.ID, attachments, err)
				}
			}
			s.mu.Lock()
			sessions := s.sessions
			s.mu.Unlock()
			s.reconcileComposerSubmissions(sessions)
		})
	}()
}
