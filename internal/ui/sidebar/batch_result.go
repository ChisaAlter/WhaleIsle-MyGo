package sidebar

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/widgets"
)

type batchResultOwner struct {
	Token, Host, Connection uint64
	Scope                   session.SessionBatchScope
}

type batchResultNotice struct {
	Owner                                 batchResultOwner
	Bound, Dismissed, Observed, Announced bool
}

func batchResultCurrentOwner(m Model) batchResultOwner {
	return batchResultOwner{Token: m.Batch.Token, Host: m.Snapshot.WorkspaceLayout.HostGeneration, Connection: m.ConnectionEpoch, Scope: BatchSelectionScope(m)}
}

// Repaint wakes never hold a callback into a later batch. A same-token scope
// transition retires an existing result instead of announcing it in a new tree.
func (n *batchResultNotice) observe(owner batchResultOwner, b session.SessionBatch) bool {
	if !n.Bound || n.Owner.Token != owner.Token {
		*n = batchResultNotice{Owner: owner, Bound: true}
	} else if n.Owner.Host != owner.Host || n.Owner.Connection != owner.Connection {
		// A Host transition must not replay a receipt whose operation token did
		// not advance. The Store normally replaces both in the same render.
		*n = batchResultNotice{Owner: owner, Bound: true, Dismissed: true}
	} else if n.Owner.Scope != owner.Scope {
		shown := n.Observed
		*n = batchResultNotice{Owner: owner, Bound: true, Dismissed: shown && !b.Busy}
	}
	if b.Busy || b.Confirming || len(b.Results) == 0 {
		return false
	}
	if n.Dismissed {
		return false
	}
	n.Observed = true
	return true
}

func batchResultSummary(b session.SessionBatch) (string, []session.SessionBatchResult) {
	succeeded := 0
	var failures []session.SessionBatchResult
	for _, result := range b.Results {
		if result.Error == "" {
			succeeded++
		} else {
			failures = append(failures, result)
		}
	}
	text := i18n.WSf(i18n.KeyWSSelectionResultArchived, "n", strconv.Itoa(succeeded))
	if b.Kind == "delete" {
		text = i18n.WSf(i18n.KeyWSSelectionResultDeleted, "n", strconv.Itoa(succeeded))
	}
	if len(failures) > 0 {
		text += "；" + i18n.WSf(i18n.KeyWSSelectionResultFailed, "n", strconv.Itoa(len(failures)))
	}
	return text, failures
}

// batchResultReceipt reports a finished batch: a success is an acknowledgement
// and stays a toast, while a batch with failures draws its receipt in the
// sidebar's own flow above the footer, where it survives search, archive
// disclosure and the rail, and keeps the failed rows readable until the reader
// closes it.
func batchResultReceipt(c *ui.Context, m Model) {
	if m.Collapsed {
		return
	}
	anchor := ui.Box(c.Key("batch-result-owner")).Size(0, 0).Shrink(0).PassThrough()
	notice := ui.Local(anchor, "notice", func() batchResultNotice { return batchResultNotice{} })
	owner := batchResultCurrentOwner(m)
	if !notice.observe(owner, m.Batch) {
		return
	}
	summary, failures := batchResultSummary(m.Batch)
	if !notice.Announced {
		c.Announce(summary)
		notice.Announced = true
	}
	if len(failures) == 0 {
		widgets.Notice(c, "batch-result", true, func() string { return summary })
		return
	}
	dismiss := func() {
		if notice.Owner != owner || m.Batch.Busy || m.Batch.Confirming || batchResultCurrentOwner(m) != owner {
			return
		}
		notice.Dismissed = true
		if m.OnBatchEdit != nil {
			m.OnBatchEdit("dismiss", nil)
		}
	}
	widgets.Alert(c, widgets.AlertOptions{
		ID:        "batch-result",
		Message:   summary,
		Detail:    batchFailureDetail(failures),
		Dismiss:   workspaceText("关闭结果", "Dismiss results"),
		OnDismiss: dismiss,
	})
}

// batchFailureDetail is what the reader opens the receipt to see: which rows the
// Host refused and its own words for why, never the receipt's sentence.
func batchFailureDetail(failures []session.SessionBatchResult) string {
	lines := make([]string, 0, len(failures))
	for _, failure := range failures {
		title := strings.TrimSpace(failure.Title)
		if title == "" {
			title = failure.ID
		}
		lines = append(lines, title+": "+failure.Error)
	}
	return strings.Join(lines, "\n")
}
