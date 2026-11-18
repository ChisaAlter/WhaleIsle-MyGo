package main

import (
	"context"
	"strings"

	"whaleisle/internal/harness/installctrl"
	"whaleisle/internal/ui/i18n"
)

// confirmPluginSideEffect is the desktop half of a plugin tool's confirmation:
// a plugin uploads extracted text or a document only after a person allows it.
// Everything that is not an explicit allow — no window, a dismissal, a timeout,
// an empty request — answers false, because the plugin fails closed on it.
func (s *mainState) confirmPluginSideEffect(ctx context.Context, ask installctrl.ConfirmRequest) (any, error) {
	summary := strings.TrimSpace(ask.Summary)
	if summary == "" {
		return map[string]any{"ok": false}, nil
	}
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
	if win == nil {
		return map[string]any{"ok": false}, nil
	}
	body := i18n.T(i18n.KeyDeepSeekConfirmBody) + "\n\n" + summary
	approved := s.askConfirmBlocking(confirmAskRequest{
		Context:          ctx,
		Owner:            win,
		Title:            i18n.T(i18n.KeyDeepSeekConfirmTitle),
		Body:             body,
		TechnicalDetails: strings.TrimSpace(ask.Details),
		ConfirmLabel:     i18n.T(i18n.KeyDeepSeekConfirmAllow),
		CancelLabel:      i18n.T(i18n.KeyDeepSeekConfirmDeny),
	})
	return map[string]any{"ok": approved}, nil
}
