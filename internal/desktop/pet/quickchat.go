package pet

import (
	"context"
	"errors"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/petdirect"
	"whaleisle/internal/store/storelocale"
)

// QuickChatOptions bind one compact card to one authenticated Host generation.
// Current and OpenMain run only on the UI thread. Neither is page IPC.
type QuickChatOptions struct {
	Origin      string
	Cookie      string
	Unavailable string
	Current     func() bool
	OpenMain    func(string)
	// Direct is set only when opening a card with Host/Whale already unavailable.
	// A shared-session request can never switch or fall back to this client.
	Direct *petdirect.Client
}

// QuickChat is UI-thread owned, including every request slot and draft. Worker
// goroutines capture immutable clients/arguments and return via Window.Update.
type QuickChat struct {
	owner       *Companion
	parent      *mygo.Window
	win         *mygo.Window
	options     QuickChatOptions
	client      *quickChatClient
	life        context.Context
	cancel      context.CancelFunc
	closed      bool
	stale       bool
	state       quickChatSnapshot
	direct      petdirect.Memory
	draft       string
	status      string
	readNote    string
	catalogNote string
	catalogAt   time.Time
	list        ui.ListState
	history     quickChatRequest
	catalog     quickChatRequest
	sending     quickChatRequest
	picking     quickChatRequest
	stopping    quickChatRequest
	jumping     quickChatRequest
	offAnchor   []func()
	positioning bool
}

// OpenQuickChat preserves the draft and focuses an already open live card.
// Reopening an invalidated card starts a fresh connection, retaining its draft.
func (p *Companion) OpenQuickChat(options QuickChatOptions) {
	if p == nil || p.win == nil || p.win.IsDestroyed() || !p.ChatAllowed() {
		return
	}
	draft := ""
	if old := p.QuickChat; old != nil {
		if old.check() && old.win != nil {
			old.win.Show()
			old.win.Focus()
			return
		}
		draft = petdirect.RetainDraft(old.options.Direct != nil, options.Direct != nil, old.draft)
		p.CloseQuickChat()
	}
	ctx, cancel := context.WithCancel(context.Background())
	q := &QuickChat{owner: p, parent: p.win, options: options, life: ctx, cancel: cancel, draft: draft, status: storelocale.T(storelocale.KeyDesktopPetLoadingSharedConversation)}
	q.list.FollowEnd = true
	q.list.Key = func(index int) any { return quickChatHistoryKey(q.state, index) }
	if options.Direct != nil {
		q.status = options.Unavailable + storelocale.T(storelocale.KeyDesktopPetSendUsesDesktopApiSettings)
	} else if options.Unavailable != "" {
		q.stale, q.status = true, options.Unavailable
	} else {
		client, err := newQuickChatClient(options.Origin, options.Cookie)
		if err != nil {
			q.stale, q.status = true, storelocale.T(storelocale.KeyDesktopPetHostUnavailable)
		} else {
			q.client = client
		}
	}
	p.QuickChat = q
	p.setQuickChatOpen(true)
	w := openQuickChatWindow(mygo.WindowOptions{
		Title: storelocale.T(storelocale.KeyDesktopPetWhaleQuickChat), Width: quickChatWidth, Height: quickChatHeight,
		MinWidth: quickChatMinWidth, MinHeight: quickChatMinHeight, Parent: p.win, AlwaysOnTop: true,
		Frameless: true, SkipTaskbar: true, DisableMinimize: true, DisableMaximize: true, DisableFullScreen: true,
		Content: ui.View(q.view),
	})
	q.win = w
	if w == nil || w.IsDestroyed() {
		p.Error = storelocale.T(storelocale.KeyDesktopPetQuickChatWindowCouldNotOpen)
		p.CloseQuickChat()
		return
	}
	w.OnClosed(func() {
		q.shutdown()
		if p.QuickChat == q {
			p.QuickChat = nil
			p.setQuickChatOpen(false)
		}
	})
	q.followPet()
	focusQuickChatWindow(w)
	if q.stale {
		return
	}
	q.refreshHistory()
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.Update(func() {
					if !q.check() {
						return
					}
					q.refreshHistory()
					if time.Since(q.catalogAt) >= time.Minute {
						q.refreshCatalog()
					}
				})
			}
		}
	}()
}

func (p *Companion) CloseQuickChat() {
	if p == nil || p.QuickChat == nil {
		return
	}
	q := p.QuickChat
	p.QuickChat = nil
	p.setQuickChatOpen(false)
	q.shutdown()
	if q.win != nil && !q.win.IsDestroyed() {
		q.win.Destroy()
	}
}

// setQuickChatOpen publishes only card visibility to non-UI readers. The
// QuickChat pointer itself remains UI-thread owned.
func (p *Companion) setQuickChatOpen(open bool) {
	p.assetMu.Lock()
	p.quickChatOpen = open
	p.assetMu.Unlock()
}

// CheckQuickChat is called on Host/task or behavior-setting changes, and on
// every paint/action/result. A disabled pet-chat setting dismisses the card.
func (p *Companion) CheckQuickChat() {
	if p == nil || p.QuickChat == nil {
		return
	}
	if !p.ChatAllowed() {
		p.CloseQuickChat()
		return
	}
	p.QuickChat.check()
}
func (q *QuickChat) shutdown() {
	if q.closed {
		return
	}
	q.closed = true
	for _, off := range q.offAnchor {
		off()
	}
	q.offAnchor = nil
	q.direct.Reset()
	if q.options.Direct != nil {
		q.state.History = nil
		q.draft = ""
	}
	q.cancel()
	q.abortRequests()
}
func (q *QuickChat) abortRequests() {
	q.direct.Cancel()
	q.history.abort()
	q.catalog.abort()
	q.sending.abort()
	q.picking.abort()
	q.stopping.abort()
	q.jumping.abort()
}
func (q *QuickChat) check() bool {
	if q.closed || q.stale {
		return false
	}
	current := q.owner.QuickChat == q && q.owner.win == q.parent && q.parent != nil && !q.parent.IsDestroyed() && q.win != nil && !q.win.IsDestroyed() && q.owner.ChatAllowed() && q.options.Current != nil && q.options.Current()
	if current {
		return true
	}
	q.stale = true
	q.cancel()
	q.abortRequests()
	q.status = storelocale.T(storelocale.KeyDesktopPetConnectionChangedReopenQuickChat)
	if q.win != nil {
		q.win.Invalidate()
	}
	return false
}
func (q *QuickChat) refreshHistory() {
	if q.options.Direct != nil {
		return
	}
	if !q.check() || q.picking.busy() {
		return
	}
	ctx, serial := q.history.begin(q.life, 30*time.Second)
	if ctx == nil {
		return
	}
	client, win, expected := q.client, q.win, q.state.SessionID
	go func() {
		next, err := client.state(ctx, false)
		win.Update(func() {
			if !q.check() || !q.history.finish(serial) || q.state.SessionID != expected {
				return
			}
			if err != nil {
				q.readNote = storelocale.T(storelocale.KeyDesktopPetSharedHistoryUnavailable)
				return
			}
			q.readNote = ""
			changed := q.state.replaceHistory(next)
			if changed {
				q.sending.abort()
				q.picking.abort()
				q.stopping.abort()
				q.jumping.abort()
				q.catalog.abort()
				q.catalogAt = time.Time{}
				q.list.ScrollToEnd()
				if expected == "" {
					q.status = storelocale.T(storelocale.KeyDesktopPetSharedWithMainConversation)
				} else {
					q.status = storelocale.T(storelocale.KeyDesktopPetPetSessionChanged)
				}
			}
			if changed || q.catalogAt.IsZero() {
				q.refreshCatalog()
			}
		})
	}()
}
func (q *QuickChat) refreshCatalog() {
	if q.options.Direct != nil {
		return
	}
	if !q.check() || q.state.SessionID == "" || q.picking.busy() {
		return
	}
	ctx, serial := q.catalog.begin(q.life, 30*time.Second)
	if ctx == nil {
		return
	}
	client, win, expected := q.client, q.win, q.state.SessionID
	go func() {
		next, err := client.state(ctx, true)
		win.Update(func() {
			if !q.check() || !q.catalog.finish(serial) || q.state.SessionID != expected {
				return
			}
			if err != nil || next.SessionID != expected {
				q.catalogNote = storelocale.T(storelocale.KeyDesktopPetModelCatalogUnavailable)
				if err == nil {
					q.refreshHistory()
				}
				return
			}
			// A slow catalog never replaces fresher history or model selection.
			q.state.Groups = next.Groups
			q.catalogAt = time.Now()
			q.catalogNote = ""
			if next.Groups == nil {
				q.catalogNote = storelocale.T(storelocale.KeyDesktopPetHostModelCatalogUnavailable)
			}
		})
	}()
}
func (q *QuickChat) send() {
	if q.options.Direct != nil {
		q.sendDirect()
		return
	}
	if !q.check() || q.state.SessionID == "" || q.picking.busy() || q.stopping.busy() {
		return
	}
	draft := q.draft
	text, err := quickChatText(draft)
	if err != nil {
		q.status = storelocale.T(storelocale.KeyDesktopPetEnterUpTo2000Utf16Units)
		return
	}
	ctx, serial := q.sending.begin(q.life, 140*time.Second)
	if ctx == nil {
		return
	}
	q.status = storelocale.T(storelocale.KeyDesktopPetWaitingForSharedSession)
	client, win, expected := q.client, q.win, q.state.SessionID
	go func() {
		err := client.send(ctx, expected, text)
		win.Update(func() {
			if !q.check() || !q.sending.finish(serial) || q.state.SessionID != expected {
				return
			}
			if err == nil {
				if q.draft == draft {
					q.draft = ""
				}
				q.status = storelocale.T(storelocale.KeyDesktopPetTurnCompletedSyncingHistory)
			} else if errors.Is(err, errQuickChatSessionChanged) {
				q.status = storelocale.T(storelocale.KeyDesktopPetPetSessionChangedBeforeSend)
			} else {
				q.status = storelocale.T(storelocale.KeyDesktopPetCompletionUnconfirmedCheckHistory)
			}
			q.history.abort()
			q.refreshHistory()
		})
	}()
}
func (q *QuickChat) cancelWait() {
	if !q.check() || !q.sending.busy() {
		return
	}
	q.sending.abort()
	q.direct.Cancel()
	if q.options.Direct != nil {
		q.status = storelocale.T(storelocale.KeyDesktopPetWaitCancelledApiMayContinue)
		return
	}
	q.status = storelocale.T(storelocale.KeyDesktopPetWaitCancelledHostMayContinue)
	q.refreshHistory()
}
func (q *QuickChat) choose(choice quickChatModel) {
	if !q.check() || q.state.SessionID == "" || q.sending.busy() || q.stopping.busy() || !q.state.allows(choice) {
		return
	}
	if q.state.Model != nil && *q.state.Model == choice {
		return
	}
	ctx, serial := q.picking.begin(q.life, 30*time.Second)
	if ctx == nil {
		return
	}
	q.history.abort()
	q.catalog.abort()
	q.status = storelocale.T(storelocale.KeyDesktopPetChangingSharedSessionModel)
	client, win, expected := q.client, q.win, q.state.SessionID
	go func() {
		selected, err := client.selectModel(ctx, expected, choice)
		win.Update(func() {
			if !q.check() || !q.picking.finish(serial) || q.state.SessionID != expected {
				return
			}
			if err == nil {
				q.state.Model = selected
				q.status = storelocale.T(storelocale.KeyDesktopPetModelUpdated)
			} else {
				q.status = storelocale.T(storelocale.KeyDesktopPetModelChangeUnconfirmed)
			}
			q.history.abort()
			q.refreshHistory()
			q.catalogAt = time.Time{}
		})
	}()
}
func (q *QuickChat) stopTurn() {
	if !q.check() || q.state.SessionID == "" || q.picking.busy() {
		return
	}
	ctx, serial := q.stopping.begin(q.life, 30*time.Second)
	if ctx == nil {
		return
	}
	q.status = storelocale.T(storelocale.KeyDesktopPetStoppingTheCurrentPetTurn)
	client, win, expected := q.client, q.win, q.state.SessionID
	go func() {
		err := client.stop(ctx, expected)
		win.Update(func() {
			if !q.check() || !q.stopping.finish(serial) || q.state.SessionID != expected {
				return
			}
			if err == nil {
				q.sending.abort()
				q.status = storelocale.T(storelocale.KeyDesktopPetStopAcceptedQueuedMessagesRemain)
			} else {
				q.status = storelocale.T(storelocale.KeyDesktopPetStopUnconfirmedCheckMainConversation)
			}
			q.history.abort()
			q.refreshHistory()
		})
	}()
}
func (q *QuickChat) openMain() {
	if !q.check() || q.state.SessionID == "" || q.options.OpenMain == nil {
		return
	}
	ctx, serial := q.jumping.begin(q.life, 30*time.Second)
	if ctx == nil {
		return
	}
	client, win, expected := q.client, q.win, q.state.SessionID
	go func() {
		err := client.sameSession(ctx, expected)
		win.Update(func() {
			if !q.check() || !q.jumping.finish(serial) || q.state.SessionID != expected {
				return
			}
			if err != nil {
				q.status = storelocale.T(storelocale.KeyDesktopPetSessionChangedOrUnavailable)
				q.refreshHistory()
				return
			}
			q.options.OpenMain(expected)
		})
	}()
}
