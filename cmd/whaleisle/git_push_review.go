package main

import (
	"fmt"

	"github.com/egoist/mygo"
	desktopgit "whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/i18n"
)

func gitActionLabel(zh, en string) string {
	if i18n.Locale() == "en" {
		return en
	}
	return zh
}

// reviewGitPush binds a titlebar Push click to the branch and destination shown.
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
	message += "\n\n" + gitActionLabel("将普通推送已提交内容，不会暂存或提交编辑器草稿。", "This pushes committed history without staging files or committing editor drafts.")
	if plan.DefaultBranch {
		message += "\n" + gitActionLabel("目标是默认分支。仅在确认要更新它时继续。", "The destination is the default branch. Continue only if you intend to update it.")
	}
	result, err := mygo.Dialog.Message(mygo.MessageOptions{
		Parent: win, Type: mygo.MessageQuestion,
		Title:         gitActionLabel("确认推送目标", "Confirm push destination"),
		Message:       message,
		Buttons:       []string{gitActionLabel("取消", "Cancel"), gitActionLabel("确认推送", "Confirm push")},
		DefaultButton: 0, CancelButton: 0,
	})
	if err != nil || result.Button != 1 {
		return
	}
	s.runGitRemote("push", root, func() (string, error) {
		return desktopgit.PushReviewed(root, plan)
	}, i18n.T(i18n.KeyGitPushOK), i18n.T(i18n.KeyGitActionFailed))
}
