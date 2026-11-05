package main

import (
	"context"
	"time"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/settings"
	"whaleisle/internal/ui/theme"
)

// confirmWaitTimeout bounds a background caller that blocks on an in-app
// confirmation. A closed window or a replaced request must never park it
// forever.
const confirmWaitTimeout = 5 * time.Minute

// confirmAskRequest is one shell-level confirmation rendered inside the
// application chrome. Callers that would otherwise open an operating-system
// message box queue one of these and continue the work from Run, or block on
// Result when they already run on a background goroutine.
type confirmAskRequest struct {
	Context      context.Context
	Owner        *mygo.Window
	Title        string
	Body         string
	Detail       string
	Width        float32
	ConfirmLabel string
	CancelLabel  string
	OtherLabel   string
	Other        func()
	// TechnicalDetails is the optional collapsed evidence block (port of the
	// reference shell update dialog's technicalDetails).
	TechnicalDetails string
	Danger           bool
	Run              func()
	// Cancel, when non-nil, runs after the user dismisses the request.
	Cancel func()
	// Result, when non-nil, receives the answer instead of Run being called.
	Result         chan bool
	settled        bool
	closed         func()
	temporaryOwner bool
}

// askConfirm queues a confirmation and repaints. Only the most recent request
// survives: a second request replaces the first without running it.
func (s *mainState) askConfirm(req confirmAskRequest) *confirmAskRequest {
	req.Owner = s.lifecycleDialogOwner(req.Owner)
	s.mu.Lock()
	previous := s.confirmAsk
	s.confirmAsk = &req
	// A replaced request starts with the evidence block closed, so the previous
	// question's expansion never leaks into an unrelated one.
	s.confirmAskTechnicalOpen = false
	s.mu.Unlock()
	if previous != nil {
		s.settleConfirmation(previous, false)
	}
	if req.Owner == nil {
		s.settleConfirmation(&req, false)
		return &req
	}
	mygo.RunOnMain(func() {
		if req.Owner.IsDestroyed() {
			s.settleConfirmation(&req, false)
			return
		}
		// A URL window cannot paint native Content overlays. Keep the same
		// visible parent and use the shared shell dialog in an owned modal seat.
		parent := req.Owner
		if parent.Page() != nil {
			var dialog *mygo.Window
			width := req.Width
			if width <= 0 {
				width = 380
			}
			dialog = mygo.NewWindow(mygo.WindowOptions{Parent: parent, Modal: true, Title: req.Title,
				Width: int(width) + 48, Height: 380, MinWidth: 300, MinHeight: 220, AutoHideMenuBar: true, Frameless: true, Transparent: true, DisableResize: true, BackgroundColor: "#00000000",
				Content: ui.View(func(c *ui.Context) { ui.Box(c).Fill().Background(ui.Transparent); s.syncConfirmAskFor(c, dialog) })})
			s.mu.Lock()
			if req.settled {
				s.mu.Unlock()
				if dialog != nil {
					dialog.Destroy()
				}
				return
			}
			req.Owner, req.temporaryOwner = dialog, true
			s.mu.Unlock()
		}
		if req.Owner == nil || req.Owner.IsDestroyed() {
			s.settleConfirmation(&req, false)
			return
		}
		offClosed := req.Owner.OnClosed(func() { s.settleConfirmation(&req, false) })
		offHide := parent.OnHide(func() { s.settleConfirmation(&req, false) })
		offParent := parent.OnClosed(func() { s.settleConfirmation(&req, false) })
		off := func() { offClosed(); offHide(); offParent() }
		s.mu.Lock()
		settled := req.settled
		if !settled {
			req.closed = off
		}
		s.mu.Unlock()
		if settled {
			off()
			return
		}
		req.Owner.Invalidate()
	})
	return &req
}

// The originating visible surface owns its decision. A hidden main window is
// never substituted for the launcher or another foreground native window.
func (s *mainState) lifecycleDialogOwner(preferred *mygo.Window) *mygo.Window {
	s.mu.Lock()
	main, lw := s.win, s.launcherWin
	s.mu.Unlock()
	var launcher *mygo.Window
	if !lw.destroyed() {
		launcher = lw.win
	}
	focused := mygo.FocusedWindow()
	s.mu.Lock()
	if !s.lifecycleWindowOwnedLocked(focused) {
		focused = nil
	}
	s.mu.Unlock()
	for _, win := range []*mygo.Window{preferred, focused, launcher, main} {
		if win != nil && !win.IsDestroyed() && win.IsVisible() && !win.IsMinimized() {
			return win
		}
	}
	return nil
}

// askConfirmBlocking queues a confirmation and waits for the answer. Call it
// only from a goroutine, never from the UI thread: painting the dialog needs a
// frame.
func (s *mainState) askConfirmBlocking(req confirmAskRequest) bool {
	answer := make(chan bool, 1)
	req.Result = answer
	pending := s.askConfirm(req)
	timer := time.NewTimer(confirmWaitTimeout)
	defer timer.Stop()
	select {
	case ok := <-answer:
		return ok
	case <-confirmContextDone(req.Context):
		s.settleConfirmation(pending, false)
		return false
	case <-timer.C:
		s.settleConfirmation(pending, false)
		return false
	}
}

// syncConfirmAsk paints the queued confirmation. It runs from mainView's
// deferred notice pass, the same place the account and schedule notices paint,
// so a click handler can queue a question without owning a view.
func (s *mainState) syncConfirmAsk(c *ui.Context, palette ...theme.Colors) {
	s.syncConfirmAskFor(c, s.win, palette...)
}

func (s *mainState) syncConfirmAskFor(c *ui.Context, owner *mygo.Window, palette ...theme.Colors) {
	s.mu.Lock()
	req := s.confirmAsk
	themeID := s.themeID
	s.mu.Unlock()
	if req == nil || req.Owner != owner {
		return
	}
	colors := theme.Resolve(themeID).Active(mygo.Theme.IsDark())
	if len(palette) > 0 {
		colors = palette[0]
	}
	settings.ConfirmDialog(c, settings.ConfirmOptions{
		Width:        req.Width,
		Title:        req.Title,
		Body:         req.Body,
		Detail:       req.Detail,
		ConfirmLabel: req.ConfirmLabel,
		CancelLabel:  req.CancelLabel,
		OtherLabel:   req.OtherLabel, OnOther: func() { s.settleConfirmationChoice(req, 2) },
		Colors:           colors,
		TechnicalDetails: req.TechnicalDetails,
		TechnicalLabel:   i18n.T(i18n.KeyCmdTechnicalDetails),
		TechnicalOpen:    &s.confirmAskTechnicalOpen,
		Danger:           req.Danger,
		OnConfirm:        func() { s.settleConfirmation(req, true) },
		OnCancel:         func() { s.settleConfirmation(req, false) },
	})
}

func (s *mainState) settleConfirmation(req *confirmAskRequest, run bool) {
	choice := 0
	if run {
		choice = 1
	}
	s.settleConfirmationChoice(req, choice)
}
func (s *mainState) settleConfirmationChoice(req *confirmAskRequest, choice int) {
	s.mu.Lock()
	if req == nil || req.settled {
		s.mu.Unlock()
		return
	}
	// A replaced request can only cancel; a delayed click cannot authorize it.
	if s.confirmAsk != req {
		choice = 0
	} else {
		s.confirmAsk = nil
	}
	req.settled = true
	off := req.closed
	req.closed = nil
	win := req.Owner
	temporary := req.temporaryOwner
	s.mu.Unlock()
	if off != nil {
		off()
	}
	if temporary && win != nil && !win.IsDestroyed() {
		win.Destroy()
	}
	if req.Result != nil {
		select {
		case req.Result <- (choice == 1):
		default:
		}
	} else if choice == 2 {
		if req.Other != nil {
			req.Other()
		}
	} else if choice == 1 {
		if req.Run != nil {
			req.Run()
		}
	} else if req.Cancel != nil {
		req.Cancel()
	}
	if win != nil {
		win.Update(func() {})
	}
}

func confirmContextDone(ctx context.Context) <-chan struct{} {
	if ctx == nil {
		return nil
	}
	return ctx.Done()
}

func firstConfirmOwner(owners []*mygo.Window) *mygo.Window {
	if len(owners) > 0 {
		return owners[0]
	}
	return nil
}
