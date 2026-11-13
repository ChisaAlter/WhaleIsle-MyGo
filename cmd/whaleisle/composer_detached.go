package main

import (
	"context"
	"errors"
	"log"

	"github.com/egoist/mygo"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/i18n"
)

func (s *mainState) submitDetachedComposer(win *mygo.Window, d *composer.DraftState, id string, owner *session.ComposerTarget) {
	flight, ok := d.BeginDetachedSend()
	if !ok {
		return
	}
	requestID := owner.BeginSubmission(flight.Text, flight.Attachments)
	if requestID == "" {
		d.CompleteDetachedSend(flight.ID, flight.Attachments, errors.New(i18n.T(i18n.KeyCmdComposerHostChanged)))
		return
	}
	go func() {
		ctx := context.Background()
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
			err = owner.AbandonSubmission(requestID, err)
			log.Printf("qa-flight %d: abandon -> err=%v", flight.ID, err)
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
