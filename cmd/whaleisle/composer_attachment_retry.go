package main

import (
	"context"
	"github.com/egoist/mygo"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

func (s *mainState) prepareComposerFile(win *mygo.Window, sessions *session.Store, id string, d *composer.DraftState, index int, retry bool) {
	if id == "" || win == nil || sessions == nil || d == nil {
		return
	}
	// Capture the Host before asynchronous work begins; navigation never retargets
	// the upload, and replacing the Host invalidates its receipt through check().
	owner, err := sessions.CaptureComposerTarget(id)
	if err != nil {
		return
	}
	a, ok := d.BeginAttachmentPreparation(index, retry)
	if !ok {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	d.SetAttachmentCancel(a, cancel)
	go func() {
		defer cancel()
		prepared, err := owner.PrepareAttachment(ctx, a)
		win.Update(func() {
			d.CompleteAttachmentPreparation(index, a, prepared, err)
		})
	}()
}
