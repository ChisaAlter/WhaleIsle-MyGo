package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/pet"
	"whaleisle/internal/desktop/screenglance"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/petdirect"
	"whaleisle/internal/petlook"
	"whaleisle/internal/store/storelocale"
	"whaleisle/internal/ui/i18n"
)

// petLookRoute preserves an explicit legacy desktop model or an exact catalog
// pair. Global fallback and the assistant's model never choose a destination.
func petLookRoute(b config.PetBehavior, catalog petlook.Catalog) (string, string, bool) {
	if petlook.Legacy(b.LookProvider, b.LookModel) {
		return "", b.LookModel, true
	}
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
	if cfg == nil || win == nil || win.IsDestroyed() {
		attempt.Finish(i18n.T(i18n.KeyCmdContextChanged))
		return
	}
	behavior := cfg.DesktopPet().Behavior
	legacy := petlook.Legacy(behavior.LookProvider, behavior.LookModel)
	if !legacy && (!ready || !cfg.Get().WhaleAssistantEnabled || settings == nil || sessions == nil) {
		attempt.Finish(i18n.T(i18n.KeyCmdConnectHostAndEnableWhaleFirst))
		return
	}
	catalog := petlook.Catalog{}
	if !legacy {
		catalog = settings.ImageCatalogSnapshot()
	}
	provider, model, available := petLookRoute(behavior, catalog)
	if !available {
		attempt.Finish(i18n.T(i18n.KeyCmdSelectThePetLookModelAndRefreshIt))
		return
	}
	selected := ""
	if !legacy {
		selected = sessions.SelectedSessionID()
	}
	userData, home := paths.UserDataDir(), paths.DshHome()
	var direct *petdirect.Client
	if legacy {
		direct = newPetDirectChat(cfg, userData, home)
	}
	current := func() bool {
		if !attempt.Current() {
			return false
		}
		s.mu.Lock()
		now := s.bootStatus
		same := !s.quitting && s.desktopPet == p && s.cfgStore == cfg && s.win == win && !win.IsDestroyed()
		if !legacy {
			same = same && s.settings == settings && s.sessions == sessions && s.dshMuxReady && now.State == harness.Ready &&
				now.Generation == host.Generation && now.Endpoint.Origin == host.Endpoint.Origin && now.Endpoint.Cookie == host.Endpoint.Cookie
		}
		s.mu.Unlock()
		if !same {
			return false
		}
		if legacy {
			b := cfg.DesktopPet().Behavior
			return paths.UserDataDir() == userData && paths.DshHome() == home && petlook.Legacy(b.LookProvider, b.LookModel) && b.LookModel == model
		}
		if !cfg.Get().WhaleAssistantEnabled || sessions.SelectedSessionID() != selected {
			return false
		}
		p, m, ok := petLookRoute(cfg.DesktopPet().Behavior, settings.ImageCatalogSnapshot())
		return ok && p == provider && m == model
	}
	displayProvider, destination := provider, host.Endpoint.Origin
	if legacy {
		displayProvider = "Desktop API"
		destination = "Desktop API settings"
		if i18n.Locale() != "en" {
			displayProvider, destination = "本机 API", "桌面 API 设置中的服务器"
		}
	}
	answer, err := mygo.Dialog.Message(mygo.MessageOptions{
		Parent: attempt.Window, Type: mygo.MessageQuestion,
		Title:   i18n.T(i18n.KeyCmdLookAtScreen),
		Message: screenglance.ApprovalMessage(),
		Detail: fmt.Sprintf(i18n.T(i18n.KeyCmdPetLookApprovalDetails), displayProvider, model, destination) +
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
				var reply string
				var err error
				if legacy {
					reply, err = direct.SendGlance(ctx, model, frame)
				} else {
					reply, err = pet.SendGlance(ctx, host.Endpoint.Origin, host.Endpoint.Cookie, provider, model, frame)
				}
				clear(frame)
				win.Update(func() {
					defer cancel()
					if !current() {
						attempt.Finish(i18n.T(i18n.KeyCmdContextChanged))
						return
					}
					if err != nil {
						// Server error bodies can contain URLs or private content.
						message := i18n.T(i18n.KeyCmdLookFailedCheckTheVisionModel)
						if errors.Is(err, petdirect.ErrCredentials) {
							message = storelocale.T(storelocale.KeyDesktopPetDesktopApiCredentialsUnavailable)
						} else if errors.Is(err, petdirect.ErrEndpoint) {
							message = storelocale.T(storelocale.KeyDesktopPetApiEndpointIsInvalidOrInsecure)
						}
						attempt.Finish(message)
						return
					}
					attempt.Reply(reply)
				})
			}()
		})
	}()
}
