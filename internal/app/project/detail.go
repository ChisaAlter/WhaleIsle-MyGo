package projectctl

import (
	"context"
	"errors"
	"fmt"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/i18n"
)

type refreshLoop struct {
	cancel context.CancelFunc
	client *dsh.Client
}

func (c *Controller) runDetail(win any, id, action, workID, memberID string) {
	if win == nil || id == "" {
		return
	}
	snap := c.mgr.hostSnapshot()
	sessions := c.mgr.sessionsStore()
	lc := c.mgr.lifecycleContext()

	c.mgr.mu.Lock()
	f := &c.flow
	client := snap.Client
	if f.busy {
		c.mgr.mu.Unlock()
		return
	}
	if client == nil || f.client != client || !f.listing.Available || f.teamID != id || f.team == nil || f.team.Team == nil {
		f.err = i18n.T(i18n.KeyCmdRefreshTheProjectTeamFirst)
		c.mgr.mu.Unlock()
		return
	}
	c.stopProcessLocked()
	p := f.team.Project
	prior := f.detail
	oldProcess := f.process
	mux := snap.Mux
	if action != "detail" && prior == nil {
		c.mgr.mu.Unlock()
		return
	}
	f.busy = true
	f.err = ""
	f.generation++
	generation := f.generation
	f.detailGeneration++
	detailGeneration := f.detailGeneration
	navigation := f.navigation
	c.mgr.mu.Unlock()

	selected := ""
	if sessions != nil {
		selected = sessions.SelectedSessionID()
	}
	go func() {
		ctx, cancel := context.WithTimeout(lc, 30*time.Second)
		defer cancel()
		rpc := projectRPC(client)
		var detail projectflow.WorkDetail
		var diff projectflow.WorkDiff
		var process projectflow.ProcessSnapshot
		var draft string
		var err error
		switch action {
		case "detail":
			detail, err = rpc.WorkDetail(ctx, id, workID, memberID)
		case "diff":
			detail, err = rpc.WorkDetail(ctx, id, prior.Workstream.ID, "")
			if err == nil {
				if detail.Worker == nil || detail.Worker.Role == "readonly" || workID != detail.Workstream.ID {
					err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectRefreshWritableMember))
				} else {
					diff, err = rpc.WorkDiff(ctx, id, detail.Workstream.ID)
				}
			}
		case "process":
			detail = *prior
			process, err = projectflow.ReadProcess(ctx, mux, p, detail)
		case "earlier":
			detail = *prior
			if oldProcess == nil {
				err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectOpenMemberHistory))
			} else {
				process, err = rpc.EarlierProcess(ctx, p, detail, *oldProcess)
			}
		case "retry-draft":
			detail, err = rpc.WorkDetail(ctx, id, prior.Workstream.ID, "")
			if err == nil {
				draft, err = projectflow.RetryDraft(p, detail)
			}
		default:
			err = fmt.Errorf("unsupported Project detail action")
		}
		c.mgr.updateWindow(win, func() {
			nowSnap := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			f := &c.flow
			if nowSnap.Client != client || f.client != client || f.generation != generation {
				c.mgr.mu.Unlock()
				return
			}
			f.busy = false
			if f.teamID != id || f.detailGeneration != detailGeneration {
				c.mgr.mu.Unlock()
				return
			}
			if err != nil {
				f.err = err.Error()
				c.mgr.mu.Unlock()
				return
			}
			f.detail = &detail
			if action == "detail" {
				f.assetDiff = nil
				f.process = nil
			}
			if action == "diff" {
				f.assetDiff = &diff
			}
			if action == "process" || action == "earlier" {
				f.process = &process
			}
			if action != "retry-draft" {
				c.mgr.mu.Unlock()
				return
			}
			if !f.visible() || f.navigation != navigation || sessions == nil || sessions.SelectedSessionID() != selected {
				f.err = i18n.T(i18n.KeyCmdNavigationChangedTheRetryDraftWasNotI)
				c.mgr.mu.Unlock()
				return
			}
			c.mgr.mu.Unlock()

			var placeErr error
			if c.mgr.deps.PlaceLeadDraft != nil {
				placeErr = c.mgr.deps.PlaceLeadDraft(p.CoordinatorSessionID, draft)
			} else {
				placeErr = errors.New(i18n.T(i18n.KeyCmdProjectRecoveryUnavailable))
			}
			c.mgr.mu.Lock()
			if c.flow.client == client && c.flow.teamID == id && c.flow.detailGeneration == detailGeneration {
				if placeErr != nil {
					c.flow.err = placeErr.Error()
				} else {
					c.flow.teamNote = i18n.T(i18n.KeyCmdRetryRequestPlacedInTheOriginalLeadDr)
				}
			}
			c.mgr.mu.Unlock()
			if placeErr == nil {
				sessions.Select(p.CoordinatorSessionID)
			}
		})
	}()
}

func (c *Controller) stopProcessLocked() {
	f := &c.flow
	if f.processCancel != nil {
		f.processCancel()
		f.processCancel = nil
	}
	f.processMux = nil
	f.processWatchGeneration++
	if f.process != nil && f.process.Live {
		next := *f.process
		next.Live = false
		f.process = &next
	}
}

func (c *Controller) syncProcessLocked(mux *dsh.Mux) {
	f := &c.flow
	if f.processCancel != nil && (f.processMux != mux || !f.visible() || f.detail == nil || f.teamID == "" || !(f.observerOpen && f.observerSection == "team") && (f.reports != nil || f.materialKey != "" || f.requestProjectID != "") || f.archiveCandidate != nil) {
		c.stopProcessLocked()
	}
}

func (c *Controller) watchProcess(win any, id string) {
	if win == nil || id == "" {
		return
	}
	snap := c.mgr.hostSnapshot()
	lc := c.mgr.lifecycleContext()
	c.mgr.mu.Lock()
	f := &c.flow
	client, mux := snap.Client, snap.Mux
	if f.busy || client == nil || mux == nil || f.client != client || !f.visible() || f.teamID != id || f.team == nil || f.detail == nil {
		c.mgr.mu.Unlock()
		return
	}
	p, d := f.team.Project, *f.detail
	if _, err := d.ProcessAddress(p); err != nil {
		f.err = err.Error()
		c.mgr.mu.Unlock()
		return
	}
	c.stopProcessLocked()
	ctx, cancel := context.WithCancel(lc)
	f.processCancel = cancel
	f.processMux = mux
	watch, detailGeneration := f.processWatchGeneration, f.detailGeneration
	f.process = &projectflow.ProcessSnapshot{Live: true}
	f.err = ""
	c.mgr.mu.Unlock()

	currentLocked := func(nowSnap HostSnapshot) bool {
		f := &c.flow
		return nowSnap.Client == client &&
			nowSnap.Mux == mux &&
			f.client == client &&
			f.visible() &&
			f.teamID == id &&
			f.detailGeneration == detailGeneration &&
			f.processWatchGeneration == watch &&
			f.detail != nil &&
			f.detail.Workstream.ID == d.Workstream.ID &&
			f.detail.Workstream.WorkerSessionID == d.Workstream.WorkerSessionID &&
			f.detail.Workstream.CurrentDelegationRef == d.Workstream.CurrentDelegationRef
	}
	go func() {
		defer cancel()
		err := projectflow.FollowProcess(ctx, mux, p, d, func(next projectflow.ProcessSnapshot) bool {
			nowSnap := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			valid := ctx.Err() == nil && currentLocked(nowSnap)
			c.mgr.mu.Unlock()
			if !valid {
				return false
			}
			// At most one immutable snapshot waits on the UI queue. The shared
			// mux's bounded stream inbox supplies backpressure while it paints.
			ack := make(chan struct{})
			c.mgr.updateWindow(win, func() {
				defer close(ack)
				uiSnap := c.mgr.hostSnapshot()
				c.mgr.mu.Lock()
				defer c.mgr.mu.Unlock()
				if currentLocked(uiSnap) {
					c.flow.process = &next
				}
			})
			select {
			case <-ack:
				return true
			case <-ctx.Done():
				return false
			}
		})
		c.mgr.updateWindow(win, func() {
			uiSnap := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			defer c.mgr.mu.Unlock()
			if !currentLocked(uiSnap) {
				return
			}
			f := &c.flow
			f.processCancel = nil
			next := projectflow.ProcessSnapshot{}
			if f.process != nil {
				next = *f.process
			}
			next.Live = false
			if err != nil {
				next.Error = errcopy.Sentence(err)
			}
			f.process = &next
		})
	}()
}

func (c *Controller) refreshVisibleLocked(snap HostSnapshot) bool {
	f := &c.flow
	return !snap.Quitting && !c.mgr.windowHidden && !snap.OverlayOpen && f.visible() && snap.Client != nil && f.client == snap.Client
}

func (c *Controller) stopRefreshLocked() {
	if r := c.flow.refreshLoop; r != nil {
		r.cancel()
		c.flow.refreshLoop = nil
	}
}

func (c *Controller) ensureRefreshLocked(win any, snap HostSnapshot, lc context.Context) {
	f := &c.flow
	if win == nil || !c.refreshVisibleLocked(snap) {
		c.stopRefreshLocked()
		return
	}
	if f.refreshLoop != nil && f.refreshLoop.client == snap.Client {
		return
	}
	c.stopRefreshLocked()
	ctx, cancel := context.WithCancel(lc)
	r := &refreshLoop{cancel: cancel, client: snap.Client}
	f.refreshLoop = r
	go func() {
		defer cancel()
		timer := time.NewTicker(5 * time.Second)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
			}
			if !c.refreshSnapshot(ctx, win, r) {
				return
			}
		}
	}()
}

// refreshSnapshot reads the original list/detail/projection APIs, never a
// control or Agent tool. It leaves explicit operations' generations and drafts
// alone and publishes only within the captured Host/navigation/page identity.
func (c *Controller) refreshSnapshot(ctx context.Context, win any, r *refreshLoop) bool {
	snap := c.mgr.hostSnapshot()
	c.mgr.mu.Lock()
	f := &c.flow
	if f.refreshLoop != r || !c.refreshVisibleLocked(snap) {
		if f.refreshLoop == r {
			f.refreshLoop = nil
		}
		c.mgr.mu.Unlock()
		return false
	}
	if f.busy || !f.loaded {
		c.mgr.mu.Unlock()
		return true
	}
	generation, navigation, detailGeneration := f.generation, f.navigation, f.detailGeneration
	id := f.teamID
	detail := f.detail
	// Other Project overlays keep their own editable/history state. Refresh the
	// list, but do not replace a hidden Team/detail behind one of those overlays.
	if !(f.observerOpen && f.observerSection == "team") && (f.reports != nil || f.materialKey != "" || f.requestProjectID != "") || f.archiveCandidate != nil {
		id = ""
		detail = nil
	}
	cursor := ""
	var cursors []string
	if id != "" && f.team != nil && len(f.team.CurrentCursors) > 0 {
		cursors = append([]string(nil), f.team.CurrentCursors...)
		cursor = cursors[len(cursors)-1]
	}
	teamIDAtStart := f.teamID
	c.mgr.mu.Unlock()

	callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	rpc := projectRPC(r.client)
	listing, listErr := rpc.List(callCtx)
	var overview projectflow.Overview
	var work projectflow.WorkDetail
	var teamErr, detailErr error
	found := false
	if listErr == nil && listing.Available && id != "" {
		for _, p := range listing.Projects {
			if p.ID == id {
				found = true
				break
			}
		}
		if found {
			overview, teamErr = rpc.TeamOverviewPage(callCtx, id, cursor)
			if teamErr == nil && detail != nil {
				work, detailErr = rpc.WorkDetail(callCtx, id, detail.Workstream.ID, "")
			}
		}
	}
	ack := make(chan struct{})
	c.mgr.updateWindow(win, func() {
		defer close(ack)
		uiSnap := c.mgr.hostSnapshot()
		c.mgr.mu.Lock()
		defer c.mgr.mu.Unlock()
		f := &c.flow
		if ctx.Err() != nil || f.refreshLoop != r || uiSnap.Client != r.client || f.client != r.client || !c.refreshVisibleLocked(uiSnap) || f.busy || f.generation != generation || f.navigation != navigation || f.detailGeneration != detailGeneration || f.teamID != teamIDAtStart {
			return
		}
		// Keep separate live-refresh feedback so a read never clears an unsent
		// draft error or a completed mutation's follow-up notice.
		if listErr != nil {
			f.refreshNote = errcopy.Under(i18n.T(i18n.KeyCmdAutomaticRefreshFailedShowingPreviousS), listErr)
			return
		}
		f.listing = listing
		f.refreshNote = i18n.T(i18n.KeyCmdAutoRefreshEvery5s) + time.Now().Format("15:04:05")
		if !listing.Available {
			f.refreshNote = listing.Error
			c.stopProcessLocked()
			return
		}
		if id == "" {
			return
		}
		if !found {
			c.stopProcessLocked()
			f.team = nil
			f.refreshNote = i18n.T(i18n.KeyCmdThisProjectIsNoLongerInTheHostList)
			return
		}
		if teamErr != nil {
			f.refreshNote = errcopy.Under(i18n.T(i18n.KeyCmdTeamRefreshFailedShowingPreviousState), teamErr)
			return
		}
		if overview.WorkPage.CurrentCursor != nil && *overview.WorkPage.CurrentCursor == cursor && len(cursors) > 0 {
			overview.CurrentCursors = cursors
		}
		f.team = &overview
		if detail == nil {
			return
		}
		if detailErr != nil {
			f.refreshNote = errcopy.Under(i18n.T(i18n.KeyCmdMemberDetailRefreshFailed), detailErr)
			return
		}
		// A successor delegation is a new run. Release the previous observer rather
		// than silently presenting old-run records as the new run's live process.
		if work.Workstream.WorkerSessionID != detail.Workstream.WorkerSessionID || work.Workstream.CurrentDelegationRef != detail.Workstream.CurrentDelegationRef {
			c.stopProcessLocked()
			if f.process != nil {
				previous := *f.process
				previous.Error = i18n.T(i18n.KeyCmdTheMemberRunChangedPreviousRecordsAre)
				f.process = &previous
			}
			f.detailGeneration++
		}
		f.detail = &work
	})
	select {
	case <-ctx.Done():
		return false
	case <-ack:
		return true
	}
}

func refreshStatus(enabled bool, note string) string {
	if note != "" {
		return note
	}
	if enabled {
		return i18n.T(i18n.KeyCmdProjectAndTeamStateRefreshAutomaticall)
	}
	return ""
}
