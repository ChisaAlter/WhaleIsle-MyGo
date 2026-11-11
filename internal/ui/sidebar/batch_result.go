package sidebar

import (
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// The installed Toast defaults to 3000ms; failed batches explicitly hold forever.
const batchResultHold = 3 * time.Second

type batchResultOwner struct {
	Token, Host, Connection uint64
	Scope                   session.SessionBatchScope
}

type batchResultNotice struct {
	Owner                                 batchResultOwner
	Bound, Dismissed, Observed, Announced bool
	Started                               time.Time
}

func batchResultCurrentOwner(m Model) batchResultOwner {
	return batchResultOwner{Token: m.Batch.Token, Host: m.Snapshot.WorkspaceLayout.HostGeneration, Connection: m.ConnectionEpoch, Scope: BatchSelectionScope(m)}
}

// Repaint wakes never hold a callback into a later batch. A same-token scope
// transition retires an existing result instead of announcing it in a new tree.
func (n *batchResultNotice) observe(owner batchResultOwner, b session.SessionBatch, now time.Time) bool {
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
	if !n.Observed {
		n.Observed = true
		n.Started = now
	}
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

// batchResultToast is mounted at the sidebar root so results survive search,
// archive disclosure and rail layout. Only a completed receipt becomes a toast.
func batchResultToast(c *ui.Context, m Model) {
	anchor := ui.Box(c.Key("batch-result-owner")).Absolute().Size(0, 0).PassThrough()
	notice := ui.Local(anchor, "notice", func() batchResultNotice { return batchResultNotice{} })
	owner := batchResultCurrentOwner(m)
	if !notice.observe(owner, m.Batch, c.Now()) {
		return
	}
	summary, failures := batchResultSummary(m.Batch)
	dismiss := func() {
		if notice.Owner != owner || m.Batch.Busy || m.Batch.Confirming || batchResultCurrentOwner(m) != owner {
			return
		}
		notice.Dismissed = true
		if m.OnBatchEdit != nil {
			m.OnBatchEdit("dismiss", nil)
		}
	}
	if len(failures) == 0 {
		remaining := batchResultHold - c.Now().Sub(notice.Started)
		if remaining <= 0 {
			dismiss()
			return
		}
		c.After(remaining)
	}
	if !notice.Announced {
		c.Announce(summary)
		notice.Announced = true
	}
	ui.Overlay(c.Key(struct {
		Name  string
		Owner batchResultOwner
	}{"batch-result-toast", owner}), func() {
		ui.Column(c).Absolute().Top(120).Left(24).Right(24).AlignItems(ui.Center).PassThrough().Children(func() {
			toast := ui.Column(c).Label("Batch operation results").Role(ui.RoleStatus).Width(560).MaxWidthPercent(100).MaxHeightPercent(80).Gap(8).Padding(12, 16).Radius(12).
				Background(c.Theme().Background).Border(.5, c.Theme().Border).Shadow(0, 6, 20, 0, ui.RGBA(0, 0, 0, .2))
			toast.Children(func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(10).Children(func() {
					ink := c.Theme().Success
					if len(failures) > 0 {
						ink = c.Theme().Text
					}
					ui.Text(c, summary).Grow(1).MinWidth(0).FontSize(theme.InterfaceSize(13)).FixedLineHeight(20).TextColor(ink)
					if ui.Button(c.Key("batch-result-dismiss"), workspaceText("关闭结果", "Dismiss results")).Height(28).Padding(4, 8).Clicked() {
						dismiss()
					}
				})
				if len(failures) > 0 {
					ui.Scroll(c).Label("Batch failure details").FillWidth().Height(168).MinHeight(0).Children(func() {
						for _, failure := range failures {
							title := strings.TrimSpace(failure.Title)
							if title == "" {
								title = failure.ID
							}
							ui.Box(c.Key("batch-failure-" + failure.ID)).Label("Batch failure " + failure.ID).FillWidth().Children(func() {
								ui.Text(c, title+": "+failure.Error).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(18).TextColor(c.Theme().Text)
							})
						}
					})
				}
			})
		})
	})
}
