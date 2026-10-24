package main

import (
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/settings"
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
	Title        string
	Body         string
	Detail       string
	ConfirmLabel string
	CancelLabel  string
	// TechnicalDetails is the optional collapsed evidence block (port of the
	// reference shell update dialog's technicalDetails).
	TechnicalDetails string
	Danger           bool
	Run              func()
	// Cancel, when non-nil, runs after the user dismisses the request.
	Cancel func()
	// Result, when non-nil, receives the answer instead of Run being called.
	Result chan bool
}

// askConfirm queues a confirmation and repaints. Only the most recent request
// survives: a second request replaces the first without running it.
func (s *mainState) askConfirm(req confirmAskRequest) {
	s.mu.Lock()
	s.confirmAsk = &req
	// A replaced request starts with the evidence block closed, so the previous
	// question's expansion never leaks into an unrelated one.
	s.confirmAskTechnicalOpen = false
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}

// askConfirmBlocking queues a confirmation and waits for the answer. Call it
// only from a goroutine, never from the UI thread: painting the dialog needs a
// frame.
func (s *mainState) askConfirmBlocking(req confirmAskRequest) bool {
	answer := make(chan bool, 1)
	req.Result = answer
	s.askConfirm(req)
	select {
	case ok := <-answer:
		return ok
	case <-time.After(confirmWaitTimeout):
		return false
	}
}

// syncConfirmAsk paints the queued confirmation. It runs from mainView's
// deferred notice pass, the same place the account and schedule notices paint,
// so a click handler can queue a question without owning a view.
func (s *mainState) syncConfirmAsk(c *ui.Context) {
	s.mu.Lock()
	req := s.confirmAsk
	s.mu.Unlock()
	if req == nil {
		return
	}
	settle := func(run bool) {
		s.mu.Lock()
		s.confirmAsk = nil
		win := s.win
		s.mu.Unlock()
		if req.Result != nil {
			select {
			case req.Result <- run:
			default:
			}
		} else if run {
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
	settings.ConfirmDialog(c, settings.ConfirmOptions{
		Title:            req.Title,
		Body:             req.Body,
		Detail:           req.Detail,
		ConfirmLabel:     req.ConfirmLabel,
		CancelLabel:      req.CancelLabel,
		TechnicalDetails: req.TechnicalDetails,
		TechnicalLabel:   resumeText("技术细节", "Technical details"),
		TechnicalOpen:    &s.confirmAskTechnicalOpen,
		Danger:           req.Danger,
		OnConfirm:        func() { settle(true) },
		OnCancel:         func() { settle(false) },
	})
}
