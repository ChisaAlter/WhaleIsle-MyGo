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
	prepareComposerAttachmentContext(s.lifecycleContext(), owner, d, index, retry, func(fn func()) { win.Update(fn) })
}

// Both native and plugin retries retain the target captured at admission.
// post owns UI progress/completion; no worker mutates a resident draft directly.
func prepareComposerAttachment(owner *session.ComposerTarget, d *composer.DraftState, index int, retry bool, post func(func())) bool {
	return prepareComposerAttachmentContext(context.Background(), owner, d, index, retry, post)
}

func prepareComposerAttachmentContext(parent context.Context, owner *session.ComposerTarget, d *composer.DraftState, index int, retry bool, post func(func())) bool {
	if owner == nil || d == nil || post == nil || !owner.Current() {
		return false
	}
	a, ok := d.BeginAttachmentPreparation(index, retry)
	if !ok {
		return false
	}
	ctx, cancel := context.WithCancel(parent)
	d.SetAttachmentCancel(a, cancel)
	go func() {
		defer cancel()
		var prepared session.ComposerAttachment
		err := withComposerFileWorker(ctx, func() error {
			var err error
			prepared, err = owner.PrepareAttachmentProgress(ctx, a, func(sent, total int64) {
				post(func() { d.UpdateAttachmentProgress(a, sent, total) })
			})
			return err
		})
		post(func() {
			d.CompleteAttachmentPreparation(index, a, prepared, err)
		})
	}()
	return true
}
