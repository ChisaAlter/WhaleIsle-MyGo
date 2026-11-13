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

// detachedSendTimeout bounds a detached send's staging pipeline (attachment
// prepare, submit dispatch, submission retirement). Every stage is a local-RPC
// call that should complete in seconds; two minutes is already generous and
// converts a silently stuck send into a surfaced draft-preserving failure.
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
	go func() {
		// Bound the whole staging pipeline: prepare/submit/retire are local-RPC
		// calls that should finish in seconds. An unbounded Background here
		// turned a stuck RPC into a silent infinite in-flight send (qa19/qa22).
		ctx, cancel := context.WithTimeout(context.Background(), detachedSendTimeout)
		defer cancel()
		log.Printf("qa-flight %d: begin (attachments=%d)", flight.ID, len(flight.Attachments))
		attachments := append([]session.ComposerAttachment(nil), flight.Attachments...)
		var err error
		for i := range attachments {
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
			if err == nil && len(attachments) > 0 {
				err = owner.AwaitSubmissionRetirement(ctx, requestID)
				log.Printf("qa-flight %d: retire -> err=%v", flight.ID, err)
			}
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
		if errors.Is(err, context.DeadlineExceeded) {
			err = errors.New(i18n.T(i18n.KeyCmdComposerSendTimeout))
		}
		win.Update(func() {
			s.mu.Lock()
			current := s.composerDrafts[id] == d
			s.mu.Unlock()
			if current {
				d.CompleteDetachedSend(flight.ID, attachments, err)
			}
		})
	}()
}
