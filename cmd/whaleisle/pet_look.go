package main

import (
	"context"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/pet"
	"whaleisle/internal/desktop/screenglance"
	"whaleisle/internal/harness"
	"whaleisle/internal/petlook"
	"whaleisle/internal/ui/i18n"
)

// petLookRoute uses only the committed pet-specific pair and image catalog.
// Global fallback and the assistant's current model never select a destination.
func petLookRoute(b config.PetBehavior, catalog petlook.Catalog) (string, string, bool) {
	if !petlook.Resolve(b.LookProvider, b.LookModel, catalog) {
		return "", "", false
	}
	return b.LookProvider, b.LookModel, true
}

// lookFromPet is reached only through the exact manual pet action. Permission
// is per screenshot; neither polling status nor opening the pet captures.
func (s *mainState) lookFromPet() {
	s.mu.Lock()
	p, cfg, settings, sessions, win, host := s.desktopPet, s.cfgStore, s.settings, s.sessions, s.win, s.bootStatus
	ready := !s.quitting && s.dshMuxReady && host.State == harness.Ready
	s.mu.Unlock()
	if p == nil {
		return
	}
	attempt := p.StartLook()
	if attempt == nil {
		return
	}
	if !ready || cfg == nil || !cfg.Get().WhaleAssistantEnabled || settings == nil || sessions == nil || win == nil {
		attempt.Finish(i18n.T(i18n.KeyCmdConnectHostAndEnableWhaleFirst))
		return
	}
	provider, model, available := petLookRoute(cfg.DesktopPet().Behavior, settings.ImageCatalogSnapshot())
	if !available {
		attempt.Finish(i18n.T(i18n.KeyCmdSelectThePetLookModelAndRefreshIt))
		return
	}
	selected := sessions.Snapshot().SelectedID
	current := func() bool {
		if !attempt.Current() {
			return false
		}
		s.mu.Lock()
		now := s.bootStatus
		same := !s.quitting && s.desktopPet == p && s.cfgStore == cfg && s.settings == settings && s.sessions == sessions && s.win == win && s.dshMuxReady && now.State == harness.Ready &&
			now.Generation == host.Generation && now.Endpoint.Origin == host.Endpoint.Origin && now.Endpoint.Cookie == host.Endpoint.Cookie
		s.mu.Unlock()
		if !same || !cfg.Get().WhaleAssistantEnabled || sessions.Snapshot().SelectedID != selected {
			return false
		}
		p, m, ok := petLookRoute(cfg.DesktopPet().Behavior, settings.ImageCatalogSnapshot())
		return ok && p == provider && m == model
	}
	answer, err := mygo.Dialog.Message(mygo.MessageOptions{
		Parent: attempt.Window, Type: mygo.MessageQuestion,
		Title:   i18n.T(i18n.KeyCmdLookAtScreen),
		Message: screenglance.ApprovalMessage(),
		Detail: "Provider: " + provider + "\nModel: " + model + "\nHost: " + host.Endpoint.Origin +
			"\n" + screenglance.StorageDetail(),
		Buttons: []string{i18n.T(i18n.KeyCmdCancel), i18n.T(i18n.KeyCmdCaptureAndSend)}, DefaultButton: 0, CancelButton: 0,
	})
	if err != nil || answer.Button != 1 {
		attempt.Finish(i18n.T(i18n.KeyCmdCancelledNoScreenshotTaken))
		return
	}
	if !current() || !attempt.CaptureStarted() {
		attempt.Finish(i18n.T(i18n.KeyCmdWindowConversationOrVisionRouteChan))
		return
	}
	handle := attempt.Window.NativeHandle()
	go func() {
		ctx, cancel := context.WithTimeout(attempt.Context, 60*time.Second)
		// A closed/recreated window cancels capture; stale bytes never reach Host.
		frame, err := screenglance.Capture(ctx, handle)
		if err != nil {
			cancel()
			win.Update(func() { attempt.Finish(screenglance.FailureMessage()) })
			return
		}
		win.Update(func() {
			if ctx.Err() != nil || !current() {
				clear(frame)
				cancel()
				attempt.Finish(i18n.T(i18n.KeyCmdContextChangedScreenshotWasNotSent))
				return
			}
			go func() {
				reply, err := pet.SendGlance(ctx, host.Endpoint.Origin, host.Endpoint.Cookie, provider, model, frame)
				clear(frame)
				win.Update(func() {
					defer cancel()
					if !current() {
						attempt.Finish(i18n.T(i18n.KeyCmdContextChanged))
						return
					}
					if err != nil {
						// Server error bodies can contain URLs or private content.
						attempt.Finish(i18n.T(i18n.KeyCmdLookFailedCheckTheVisionModel))
						return
					}
					attempt.Finish(reply)
				})
			}()
		})
	}()
}
