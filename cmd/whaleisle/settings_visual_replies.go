package main

import (
	"context"
	"errors"
	"whaleisle/internal/ui/i18n"

	"whaleisle/internal/desktop/htmlpreview"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
)

var helperHTMLPreview = htmlpreview.NewRendererService(captureHTMLInHelper)

// Called only from a user-initiated visual-replies.enabled=true write. The
// settings store owns the background goroutine, disables Save while checking,
// and shows the exact failure without sending the enabling RPC.
func (s *mainState) verifyVisualReplyEnable(ctx context.Context, origin string) error {
	s.mu.Lock()
	sup := s.sup
	s.mu.Unlock()
	if sup == nil {
		return errors.New(i18n.T(i18n.KeyCmdVisualRepliesHostStopped))
	}
	current := sup.Status()
	if current.State != harness.Ready || current.Endpoint.Origin != origin {
		return errors.New(i18n.T(i18n.KeyCmdVisualRepliesHostMismatch))
	}
	caps := harness.ReadCurrentHostCapabilities(paths.VendorDir(), harness.CurrentHostBridges{})
	if !caps.ArtifactsReady {
		return errors.New(caps.Reason)
	}
	status := helperHTMLPreview.Probe(ctx)
	if !status.Available {
		return errors.New(status.Reason)
	}
	current = sup.Status()
	if current.State != harness.Ready || current.Endpoint.Origin != origin {
		return errors.New(i18n.T(i18n.KeyCmdVisualRepliesHostChanged))
	}
	// A real runtime result is required before publishing the capability. This
	// never changes the default-off preference or writes Host settings itself.
	caps = harness.ReadCurrentHostCapabilities(paths.VendorDir(), harness.CurrentHostBridges{HTMLPreview: status.Available, VisualReplyIsolation: status.Available})
	if !caps.VisualReplies {
		return errors.New(caps.Reason)
	}
	return nil
}
