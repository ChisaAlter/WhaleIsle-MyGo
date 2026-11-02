package main

import (
	"context"
	"fmt"

	desktopgit "whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/i18n"
)

// reviewGitPush binds a titlebar Push click to the branch and destination shown.
// The confirmation is an in-app modal so the themed window is not interrupted by
// an operating-system message box.
func (s *mainState) reviewGitPush(root string) {
	plan, err := desktopgit.PreparePush(root)
	s.mu.Lock()
	win := s.win
	if err != nil {
		s.gitMsg = err.Error()
	}
	s.mu.Unlock()
	if err != nil {
		if win != nil {
			win.Update(func() {})
		}
		return
	}
	message := fmt.Sprintf("%s → %s/%s\n%s", plan.Branch, plan.Remote, plan.Target, plan.URL)
	message += "\n\n" + i18n.T(i18n.KeyCmdPushReviewCommitted)
	if plan.DefaultBranch {
		message += "\n" + i18n.T(i18n.KeyCmdPushReviewDefaultBranch)
	}
	s.askConfirm(confirmAskRequest{
		Title:        i18n.T(i18n.KeyCmdConfirmPushDestination),
		Body:         message,
		ConfirmLabel: i18n.T(i18n.KeyCmdConfirmPush),
		CancelLabel:  i18n.T(i18n.KeyCmdCancel),
		Run: func() {
			s.runGitRemote("push", root, func(ctx context.Context) (string, error) {
				return desktopgit.PushReviewedContext(ctx, root, plan)
			}, i18n.T(i18n.KeyGitPushOK), i18n.T(i18n.KeyGitActionFailed))
		},
	})
}
