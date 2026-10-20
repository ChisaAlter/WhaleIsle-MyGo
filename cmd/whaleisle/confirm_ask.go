package main

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/settings"
)

// confirmAskRequest is one shell-level confirmation rendered inside the
// application chrome. Callers that would otherwise open an operating-system
// message box queue one of these and continue the work from Run.
type confirmAskRequest struct {
	Title        string
	Body         string
	Detail       string
	ConfirmLabel string
	CancelLabel  string
	Danger       bool
	Run          func()
}

// askConfirm queues a confirmation and repaints. Only the most recent request
// survives: a second request replaces the first without running it.
func (s *mainState) askConfirm(req confirmAskRequest) {
	s.mu.Lock()
	s.confirmAsk = &req
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
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
		if run && req.Run != nil {
			req.Run()
		}
		if win != nil {
			win.Update(func() {})
		}
	}
	settings.ConfirmDialog(c, settings.ConfirmOptions{
		Title:        req.Title,
		Body:         req.Body,
		Detail:       req.Detail,
		ConfirmLabel: req.ConfirmLabel,
		CancelLabel:  req.CancelLabel,
		Danger:       req.Danger,
		OnConfirm:    func() { settle(true) },
		OnCancel:     func() { settle(false) },
	})
}
