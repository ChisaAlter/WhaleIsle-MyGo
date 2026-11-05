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
	// Explicit legacy feed users keep their existing entry point; ordinary
	// installations share the native catalog/update-center owner everywhere.
	if desktopupdate.FeedURLForRoute(s.updateDownloadRoute()) == "" {
		s.openNativeUpdateCenter(win)
		s.refreshNativeReleases(win)
		return
	}
	s.mu.Lock()
	if s.updateCheckPending || s.quitting {
		s.mu.Unlock()
		return
	}
	s.updateCheckPending = true
	s.mu.Unlock()
	// Evidence and the visible row start with the request: a check that never
	// finishes still leaves the milestone on disk (update-journal.js contract).
	s.recordUpdateAction(desktopupdate.JournalActionCheckRequested)
	s.recordUpdateState(desktopupdate.JournalState{Phase: desktopupdate.JournalPhaseChecking})
	s.setUpdateProgress(win, &updateProgressState{
		Phase:   updatePhaseChecking,
		Title:   i18n.T(i18n.KeyCmdCheckingForUpdates),
		Percent: -1,
	})
	if win != nil {
		win.Update(func() {})
	}
	go func() {
		defer func() {
			s.mu.Lock()
			s.updateCheckPending = false
			s.mu.Unlock()
			// The row belongs to this check only; the settled answer lives in the
			// confirmation dialog that follows it.
			s.clearUpdateProgress(win)
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
		return i18n.T(i18n.KeyCmdCouldNotCheckForUpdates) + err.Error()
	case res.Status == desktopupdate.StatusNotConfigured:
		return i18n.T(i18n.KeyCmdThisBuildHasNoUpdateFeedConfiguredDev)
	case res.Status == desktopupdate.StatusUpToDate:
		msg := i18n.T(i18n.KeyCmdYouAreOnTheLatestVersion)
		if res.Version != "" {
			msg += "（" + res.Version + "）"
		}
		return msg
	case res.Status == desktopupdate.StatusUpdateAvailable:
		msg := i18n.T(i18n.KeyCmdUpdateAvailable)
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
		return i18n.T(i18n.KeyCmdCheckComplete)
	}
}

func (s *mainState) runUpdateCheck(win *mygo.Window) {
	route := s.updateDownloadRoute()
	canPresent := func() bool {
		s.mu.Lock()
		current := !s.quitting && s.updateCheckPending && s.lifecycleWindowOwnedLocked(win)
		s.mu.Unlock()
		return current && win != nil && !win.IsDestroyed() && s.updateDownloadRoute() == route
	}
	if !canPresent() {
		return
	}
	res, err := desktopupdate.CheckForUpdatesVersionRoute(context.Background(), buildVersion, route)
	s.recordUpdateCheckResult(res, err)
	msg := updateCheckMessage(res, err)
	download := err == nil && res.Status == desktopupdate.StatusUpdateAvailable && res.ReleaseURL != ""
	confirmLabel := i18n.T(i18n.KeyCmdOk)
	technical := ""
	if download {
		confirmLabel = i18n.T(i18n.KeyUpdateDownload)
		technical = updateAvailableDetail(res, route)
	}
	approved := s.askConfirmBlocking(confirmAskRequest{
		Owner: win, Width: 640,
		Title:            i18n.T(i18n.KeySettingsDesktopCheckUpdatesTitle),
		Body:             msg,
		ConfirmLabel:     confirmLabel,
		CancelLabel:      i18n.T(i18n.KeyCmdClose),
		TechnicalDetails: technical,
	})
	if !canPresent() {
		return
	}
	if approved && download {
		s.recordUpdateAction(desktopupdate.JournalActionDownloadRequested)
		// The download row and the launcher notice bar report the same phase, so
		// the existing launcher prompt stays the product-level status.
		s.setUpdateProgress(win, &updateProgressState{
			Phase:   updatePhaseDownloading,
			Title:   i18n.T(i18n.KeyCmdPreparingTheUpdateDownload),
			Detail:  technical,
			Percent: -1,
		})
		s.setLauncherUpdate(i18n.T(i18n.KeyCmdDownloadingTheUpdate))
		requireSig := strings.EqualFold(strings.TrimSpace(os.Getenv("WHALEISLE_UPDATE_REQUIRE_SIGNATURE")), "1") ||
			strings.EqualFold(strings.TrimSpace(os.Getenv("WHALEISLE_UPDATE_REQUIRE_SIGNATURE")), "true")
		expectPub := strings.TrimSpace(os.Getenv("WHALEISLE_UPDATE_PUBLISHER"))
		// The download row reports bytes and keeps the journal at integer
		// percent, so a long download leaves bounded evidence.
		onProgress := func(received, total int64) {
			percent := updateDownloadPercent(received, total)
			title := i18n.T(i18n.KeyCmdDownloadingTheUpdate)
			detail := updateDownloadDetail(received, total)
			s.setUpdateProgress(win, &updateProgressState{
				Phase:   updatePhaseDownloading,
				Title:   title,
				Detail:  detail,
				Percent: percent,
			})
			notice := title
			if percent >= 0 {
				notice = fmt.Sprintf("%s %.0f%%", title, percent)
			}
			if detail != "" {
				notice += "（" + detail + "）"
			}
			s.setLauncherUpdate(notice)
			pinned := percent
			s.recordUpdateState(desktopupdate.JournalState{Phase: desktopupdate.JournalPhaseDownloading, Version: res.Version, Percent: &pinned})
		}
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
				OnProgress:        onProgress,
			})
		} else {
			apply, aerr = desktopupdate.DownloadAndStageOpts(context.Background(), desktopupdate.StageOptions{
				ReleaseURL:        res.ReleaseURL,
				Start:             false,
				ExpectSHA256:      res.SHA256,
				ExpectSize:        res.Size,
				RequireSignature:  requireSig,
				ExpectedPublisher: expectPub,
				OnProgress:        onProgress,
			})
		}
		if aerr == nil {
			s.recordUpdateState(desktopupdate.JournalState{Phase: desktopupdate.JournalPhaseDownloaded, Version: res.Version})
			s.setUpdateProgress(win, &updateProgressState{
				Phase:   updatePhaseStaged,
				Title:   i18n.T(i18n.KeyCmdTheUpdateIsDownloadedAndVerifiedInstal),
				Detail:  apply.Message,
				Percent: 100,
			})
			s.setLauncherUpdate(i18n.T(i18n.KeyCmdTheUpdateIsDownloadedAndVerifiedInstal))
		} else {
			s.recordUpdateState(desktopupdate.JournalState{Phase: desktopupdate.JournalPhaseError, FailedOperation: "download", Message: aerr.Error()})
			s.clearUpdateProgress(win)
			s.setLauncherUpdate(saveFailedToast(i18n.T(i18n.KeyCmdDownloadingTheUpdate), aerr))
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
		if technical != "" {
			amsg += "\n\n" + i18n.T(i18n.KeyCmdTechnicalDetails) + ":\n" + technical
		}
		if canPresent() {
			s.askConfirmBlocking(confirmAskRequest{Owner: win, Width: 640, Title: i18n.T(i18n.KeyUpdateDownloadTitle), Body: amsg, ConfirmLabel: i18n.T(i18n.KeyCmdOk), CancelLabel: i18n.T(i18n.KeyCmdClose)})
		}
	}
}
