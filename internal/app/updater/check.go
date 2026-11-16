package updater

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	desktopupdate "whaleisle/internal/desktop/update"
	"whaleisle/internal/ui/i18n"
)

const (
	// EnvDisableAutoUpdate disables scheduled background checks and auto-download when set truthy.
	EnvDisableAutoUpdate  = "WHALEISLE_DISABLE_AUTO_UPDATE"
	envDisableAutoUpdate  = EnvDisableAutoUpdate
	updateCheckTimeout    = 30 * time.Second
	updateDownloadTimeout = 10 * time.Minute
)

// AutoUpdateDisabled reports whether WHALEISLE_DISABLE_AUTO_UPDATE suppresses scheduled checks.
func AutoUpdateDisabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(envDisableAutoUpdate))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// ShowUpdateCheck opens the native update center or executes a legacy update feed check.
func (c *Center) ShowUpdateCheck(win WindowRef) {
	if c == nil {
		return
	}
	env := c.env()
	if desktopupdate.FeedURLForRoute(env.DownloadRoute) == "" {
		c.OpenCenter(win)
		c.Refresh(win)
		return
	}
	c.mu.Lock()
	if c.checkPending || env.Quitting {
		c.mu.Unlock()
		return
	}
	c.checkPending = true
	c.mu.Unlock()

	c.RecordAction(desktopupdate.JournalActionCheckRequested)
	c.RecordState(desktopupdate.JournalState{Phase: desktopupdate.JournalPhaseChecking})
	c.SetProgress(win, &ProgressState{
		Phase:   ProgressPhaseChecking,
		Title:   i18n.T(i18n.KeyCmdCheckingForUpdates),
		Percent: -1,
	})
	if c.bridge != nil && win != nil {
		c.bridge.UpdateWindow(win, func() {})
	}
	go func() {
		defer func() {
			c.mu.Lock()
			c.checkPending = false
			c.mu.Unlock()
			c.ClearProgress(win)
			if c.bridge != nil && win != nil {
				c.bridge.UpdateWindow(win, func() {})
			}
		}()
		c.runUpdateCheck(win)
	}()
}

// CheckMessage converts a legacy update feed Result into localized user copy.
func CheckMessage(res desktopupdate.Result, err error) string {
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

func (c *Center) runUpdateCheck(win WindowRef) {
	if c.bridge == nil {
		return
	}
	initialEnv := c.env()
	route := initialEnv.DownloadRoute
	canPresent := func() bool {
		curEnv := c.env()
		c.mu.Lock()
		pending := c.checkPending
		c.mu.Unlock()
		return !curEnv.Quitting &&
			pending &&
			c.bridge.OwnedWindow(win) &&
			win != nil &&
			c.bridge.WindowAlive(win) &&
			curEnv.DownloadRoute == route
	}
	if !canPresent() {
		return
	}
	checkCtx, cancel := context.WithTimeout(c.lifecycleContext(), updateCheckTimeout)
	res, err := desktopupdate.CheckForUpdatesVersionRoute(checkCtx, initialEnv.BuildVersion, route)
	cancel()
	c.RecordCheckResult(res, err)
	if !canPresent() {
		return
	}
	msg := CheckMessage(res, err)
	download := err == nil && res.Status == desktopupdate.StatusUpdateAvailable && res.ReleaseURL != ""
	confirmLabel := i18n.T(i18n.KeyCmdOk)
	technical := ""
	if download {
		confirmLabel = i18n.T(i18n.KeyUpdateDownload)
		technical = AvailableDetail(res, route)
	}
	approved := c.bridge.AskConfirmBlocking(ConfirmRequest{
		Owner:            win,
		Width:            640,
		Title:            i18n.T(i18n.KeySettingsDesktopCheckUpdatesTitle),
		Body:             msg,
		ConfirmLabel:     confirmLabel,
		CancelLabel:      i18n.T(i18n.KeyCmdClose),
		TechnicalDetails: technical,
	})
	if !canPresent() || !approved || !download {
		return
	}
	c.RecordAction(desktopupdate.JournalActionDownloadRequested)
	c.SetProgress(win, &ProgressState{
		Phase:   ProgressPhaseDownloading,
		Title:   i18n.T(i18n.KeyCmdPreparingTheUpdateDownload),
		Detail:  technical,
		Percent: -1,
	})
	c.setLauncherNotice(i18n.T(i18n.KeyCmdDownloadingTheUpdate))

	requireSig := strings.EqualFold(strings.TrimSpace(os.Getenv("WHALEISLE_UPDATE_REQUIRE_SIGNATURE")), "1") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("WHALEISLE_UPDATE_REQUIRE_SIGNATURE")), "true")
	expectPub := strings.TrimSpace(os.Getenv("WHALEISLE_UPDATE_PUBLISHER"))

	onProgress := func(received, total int64) {
		percent := DownloadPercent(received, total)
		title := i18n.T(i18n.KeyCmdDownloadingTheUpdate)
		detail := DownloadDetail(received, total)
		c.SetProgress(win, &ProgressState{
			Phase:   ProgressPhaseDownloading,
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
		c.setLauncherNotice(notice)
		pinned := percent
		c.RecordState(desktopupdate.JournalState{Phase: desktopupdate.JournalPhaseDownloading, Version: res.Version, Percent: &pinned})
	}

	var apply desktopupdate.ApplyResult
	var aerr error
	dlCtx, dlCancel := context.WithTimeout(c.lifecycleContext(), updateDownloadTimeout)
	if res.IncrementalPossible {
		apply, aerr = desktopupdate.ApplyIncremental(dlCtx, desktopupdate.IncrementalOptions{
			CurrentVersion:    initialEnv.BuildVersion,
			Feed:              res.Feed,
			Start:             false,
			OldPath:           strings.TrimSpace(os.Getenv("WHALEISLE_UPDATE_OLD_INSTALLER")),
			RequireSignature:  requireSig,
			ExpectedPublisher: expectPub,
			OnProgress:        onProgress,
		})
	} else {
		apply, aerr = desktopupdate.DownloadAndStageOpts(dlCtx, desktopupdate.StageOptions{
			ReleaseURL:        res.ReleaseURL,
			Start:             false,
			ExpectSHA256:      res.SHA256,
			ExpectSize:        res.Size,
			RequireSignature:  requireSig,
			ExpectedPublisher: expectPub,
			OnProgress:        onProgress,
		})
	}
	dlCancel()

	if aerr == nil {
		c.RecordState(desktopupdate.JournalState{Phase: desktopupdate.JournalPhaseDownloaded, Version: res.Version})
		c.SetProgress(win, &ProgressState{
			Phase:   ProgressPhaseStaged,
			Title:   i18n.T(i18n.KeyCmdTheUpdateIsDownloadedAndVerifiedInstal),
			Detail:  apply.Message,
			Percent: 100,
		})
		c.setLauncherNotice(i18n.T(i18n.KeyCmdTheUpdateIsDownloadedAndVerifiedInstal))
	} else {
		c.RecordState(desktopupdate.JournalState{Phase: desktopupdate.JournalPhaseError, FailedOperation: "download", Message: aerr.Error()})
		c.ClearProgress(win)
		c.setLauncherNotice(c.bridge.SaveFailedToast(i18n.T(i18n.KeyCmdDownloadingTheUpdate), aerr))
	}
	if !canPresent() {
		return
	}
	if aerr == nil && desktopupdate.CanStartStaged(apply) && c.bridge.InstallStagedWithConsent(win, apply, false) {
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
		_ = c.bridge.AskConfirmBlocking(ConfirmRequest{
			Owner:        win,
			Width:        640,
			Title:        i18n.T(i18n.KeyUpdateDownloadTitle),
			Body:         amsg,
			ConfirmLabel: i18n.T(i18n.KeyCmdOk),
			CancelLabel:  i18n.T(i18n.KeyCmdClose),
		})
	}
}
