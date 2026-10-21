package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/egoist/mygo"
	desktopupdate "whaleisle/internal/desktop/update"
	"whaleisle/internal/ui/i18n"
)

// Settings and launcher callers retain a void UI callback. The admission flag
// covers checking, dialogs, download/staging and the protected install decision.
func (s *mainState) showUpdateCheck(win *mygo.Window) {
	s.mu.Lock()
	if s.updateCheckPending || s.quitting {
		s.mu.Unlock()
		return
	}
	s.updateCheckPending = true
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
	go func() {
		defer func() {
			s.mu.Lock()
			s.updateCheckPending = false
			s.mu.Unlock()
			if win != nil {
				win.Update(func() {})
			}
		}()
		s.runUpdateCheck(win)
	}()
}

// updateCheckMessage turns the feed result into product copy. The raw status
// token (not_configured, up_to_date, ...) is an implementation detail and used
// to leak into the confirmation dialog.
func updateCheckMessage(res desktopupdate.Result, err error) string {
	switch {
	case err != nil:
		return resumeText("检查更新失败：", "Could not check for updates: ") + err.Error()
	case res.Status == desktopupdate.StatusNotConfigured:
		return resumeText("当前构建没有配置更新源（开发版）。发布版会通过官方更新源检查更新。", "This build has no update feed configured (development build). Release builds check the official feed.")
	case res.Status == desktopupdate.StatusUpToDate:
		msg := resumeText("当前版本已是最新", "You are on the latest version")
		if res.Version != "" {
			msg += "（" + res.Version + "）"
		}
		return msg
	case res.Status == desktopupdate.StatusUpdateAvailable:
		msg := resumeText("发现新版本", "Update available")
		if res.Version != "" {
			msg += " " + res.Version
		}
		if res.ReleaseURL != "" {
			msg += "\n" + res.ReleaseURL
		}
		return msg
	case strings.TrimSpace(res.Message) != "":
		return res.Message
	default:
		return resumeText("检查完成。", "Check complete.")
	}
}

func (s *mainState) runUpdateCheck(win *mygo.Window) {
	canPresent := func() bool {
		s.mu.Lock()
		current := !s.quitting && s.updateCheckPending && s.win == win
		s.mu.Unlock()
		return current && win != nil && !win.IsDestroyed()
	}
	if !canPresent() {
		return
	}
	res, err := desktopupdate.CheckForUpdatesVersionRoute(context.Background(), buildVersion, s.updateDownloadRoute())
	msg := updateCheckMessage(res, err)
	download := err == nil && res.Status == desktopupdate.StatusUpdateAvailable && res.ReleaseURL != ""
	confirmLabel := resumeText("好", "OK")
	if download {
		confirmLabel = i18n.T(i18n.KeyUpdateDownload)
	}
	approved := s.askConfirmBlocking(confirmAskRequest{
		Title:        i18n.T(i18n.KeySettingsDesktopCheckUpdatesTitle),
		Body:         msg,
		ConfirmLabel: confirmLabel,
		CancelLabel:  resumeText("关闭", "Close"),
	})
	if !canPresent() {
		return
	}
	if approved && download {
		requireSig := strings.EqualFold(strings.TrimSpace(os.Getenv("WHALEISLE_UPDATE_REQUIRE_SIGNATURE")), "1") ||
			strings.EqualFold(strings.TrimSpace(os.Getenv("WHALEISLE_UPDATE_REQUIRE_SIGNATURE")), "true")
		expectPub := strings.TrimSpace(os.Getenv("WHALEISLE_UPDATE_PUBLISHER"))
		var apply desktopupdate.ApplyResult
		var aerr error
		if res.IncrementalPossible {
			apply, aerr = desktopupdate.ApplyIncremental(context.Background(), desktopupdate.IncrementalOptions{
				CurrentVersion:    buildVersion,
				Feed:              res.Feed,
				Start:             false,
				OldPath:           strings.TrimSpace(os.Getenv("WHALEISLE_UPDATE_OLD_INSTALLER")),
				RequireSignature:  requireSig,
				ExpectedPublisher: expectPub,
			})
		} else {
			apply, aerr = desktopupdate.DownloadAndStageOpts(context.Background(), desktopupdate.StageOptions{
				ReleaseURL:        res.ReleaseURL,
				Start:             false,
				ExpectSHA256:      res.SHA256,
				ExpectSize:        res.Size,
				RequireSignature:  requireSig,
				ExpectedPublisher: expectPub,
			})
		}
		if !canPresent() {
			return
		}
		if aerr == nil && desktopupdate.CanStartStaged(apply) && s.installStagedUpdate(win, apply) {
			return
		}
		amsg := apply.Message
		if aerr != nil {
			amsg = aerr.Error()
		} else if apply.Signature != "" && apply.Signature != desktopupdate.SignatureSkipped {
			amsg = fmt.Sprintf("%s\nsig: %s", amsg, apply.Signature)
			if apply.Publisher != "" {
				amsg += "\n" + apply.Publisher
			}
		}
		_, _ = mygo.Dialog.Message(mygo.MessageOptions{
			Parent:     win,
			Type:       mygo.MessageInfo,
			Title:      i18n.T(i18n.KeyUpdateDownloadTitle),
			Message:    amsg,
			Buttons:    []string{"OK"},
			BeforeShow: canPresent,
		})
	}
}
