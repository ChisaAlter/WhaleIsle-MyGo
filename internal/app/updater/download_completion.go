package updater

import (
	"context"
	"errors"
	"strings"

	"whaleisle/internal/desktop/update"
	"whaleisle/internal/ui/i18n"
)

// completeDownload commits one complete outcome before making the download idle.
// diskMu keeps other persistence calls from replacing the committed record with
// an older progress snapshot between the disk write and its publication. The
// UI mutex remains unlocked during I/O, so cancellation can still be accepted.
func (c *Center) completeDownload(userDataDir, buildVersion, route string, rows []update.NativeRelease, selected update.NativeRelease, staged update.ApplyResult, downloadErr error) update.JournalState {
	c.diskMu.Lock()
	defer c.diskMu.Unlock()
	for {
		c.mu.Lock()
		before := c.releases
		c.mu.Unlock()

		next, journal := completedDownload(before, buildVersion, route, rows, selected, staged, downloadErr)
		if before.CancelRequested && before.Staging != "" {
			if err := DiscardNativePartial(userDataDir, before.Staging); err != nil {
				next.Err, next.Phase = err.Error(), "error"
			}
		}
		if err := writeNativeUpdateRecord(userDataDir, nativeUpdateRecord(next)); err != nil {
			// Keep the verified bytes available for a later resume, but never
			// advertise an installer whose recovery identity was not committed.
			next.Staged, next.StagedTag, next.StagedTitle = update.ApplyResult{}, "", ""
			next.Paused = next.Active.Tag != ""
			next.Phase = "error"
			next.Status = i18n.T(i18n.KeyCmdInstallationPreparationFailedTheCurren)
			next.Err = lifecycleText("保存更新状态失败：", "Could not save update state: ") + err.Error()
			journal = update.JournalState{Phase: update.JournalPhaseError, FailedOperation: "download", Message: next.Err}
		}

		c.mu.Lock()
		if c.releases.CancelRequested != before.CancelRequested || c.releases.PauseRequested != before.PauseRequested {
			// A cancellation accepted during the write owns the outcome. Commit
			// it before exposing idle; these request flags only move toward true.
			c.mu.Unlock()
			continue
		}
		r := &c.releases
		// Publish only the fields owned by this download. Modal lifecycle,
		// background work and preferences may have changed while disk I/O ran.
		r.Rows, r.CatalogRoute = next.Rows, next.CatalogRoute
		r.Active, r.Staging = next.Active, next.Staging
		r.Staged, r.StagedTag, r.StagedTitle = next.Staged, next.StagedTag, next.StagedTitle
		r.Received, r.Total = next.Received, next.Total
		r.Err, r.Status, r.Phase, r.Paused = next.Err, next.Status, next.Phase, next.Paused
		r.Busy, r.Cancel = false, nil
		c.mu.Unlock()
		return journal
	}
}

func completedDownload(r ReleaseState, buildVersion, route string, rows []update.NativeRelease, selected update.NativeRelease, staged update.ApplyResult, err error) (ReleaseState, update.JournalState) {
	if len(rows) > 0 {
		r.Rows, r.CatalogRoute = rows, route
	}
	journal := update.JournalState{Phase: update.JournalPhaseDownloaded, Version: selected.Version}
	if r.CancelRequested || r.PauseRequested {
		err = context.Canceled
	}
	if err != nil {
		r.Err = err.Error()
		r.Status = i18n.T(i18n.KeyCmdInstallationPreparationFailedTheCurren)
		if errors.Is(err, context.Canceled) {
			if r.PauseRequested {
				r.Paused = true
				r.Status = lifecycleText("下载已暂停，可继续。", "Download paused. You can resume it.")
			} else {
				r.Status = i18n.T(i18n.KeyCmdDownloadCancelledTheCurrentInstallatio)
			}
			r.Err = ""
			journal = update.JournalState{}
		} else {
			if selected.Tag != "" {
				r.Paused = true
			}
			journal = update.JournalState{Phase: update.JournalPhaseError, FailedOperation: "download", Message: err.Error()}
		}
	} else {
		r.Staged, r.StagedTag, r.StagedTitle = staged, selected.Tag, nativeStagedTitle(selected)
		r.Status = i18n.T(i18n.KeyCmdInstallerDownloadedAndVerifiedChooseIn)
		if selected.Platform == "darwin" {
			r.Status = i18n.T(i18n.KeyCmdDiskImageDownloadedAndVerifiedChooseOp)
		}
		if selected.IsDelta() && !staged.UsedRange && staged.CopiedBlocks == 0 {
			r.Status += i18n.T(i18n.KeyCmdTheSourceDidNotSupportRangesTheComple)
		}
		if strings.Contains(staged.Message, "native base cache unavailable:") {
			r.Status += i18n.T(i18n.KeyCmdTheInstallerIsRetainedButCachingItFor)
		}
		r.Err = ""
	}
	if r.CancelRequested {
		r.Active, r.Staged = update.NativeRelease{}, update.ApplyResult{}
		r.Paused, r.Staging = false, ""
		r.Received, r.Total = 0, 0
		r.StagedTag, r.StagedTitle, r.Err = "", "", ""
	}
	r.Phase = settledNativeReleasePhase(r, buildVersion)
	return r, journal
}
