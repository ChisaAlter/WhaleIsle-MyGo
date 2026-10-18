package main

import (
	"context"
	"errors"
	"github.com/egoist/mygo"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

func (s *mainState) submitDetachedComposer(win *mygo.Window, d *composer.DraftState, id string, owner *session.ComposerTarget) {
	flight, ok := d.BeginDetachedSend()
	if !ok {
		return
	}
	requestID := owner.BeginSubmission(flight.Text, flight.Attachments)
	if requestID == "" {
		d.CompleteDetachedSend(flight.ID, flight.Attachments, errors.New("Host changed; draft retained"))
		return
	}
	go func() {
		ctx := context.Background()
		attachments := append([]session.ComposerAttachment(nil), flight.Attachments...)
		var err error
		for i := range attachments {
			attachments[i], err = owner.PrepareAttachment(ctx, attachments[i])
			if err != nil {
				break
			}
		}
		if err == nil {
			owner.UpdateSubmissionAttachments(requestID, attachments)
			err = owner.Submit(ctx, flight.Text, attachments, 0, requestID)
		} else {
			err = owner.AbandonSubmission(requestID, err)
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
