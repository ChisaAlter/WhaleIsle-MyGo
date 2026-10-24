package main

import (
	"context"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/dsh"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/i18n"
)

type projectRefreshLoop struct {
	cancel context.CancelFunc
	client *dsh.Client
}

func (s *projectController) projectRefreshVisibleLocked() bool {
	f := s.projectFlow
	return !s.quitting && !s.projectWindowHidden && !s.settingsOpen && !s.usageOpen && !s.imOpen && !s.remoteOpen && !s.webviewOpen && !s.browserPreviewOpen && f.visible() && s.dshClient != nil && f.client == s.dshClient
}
func (s *projectController) stopProjectRefreshLocked() {
	if r := s.projectFlow.refreshLoop; r != nil {
		r.cancel()
		s.projectFlow.refreshLoop = nil
	}
}
func (s *projectController) ensureProjectRefreshLocked(win *mygo.Window) {
	f := s.projectFlow
	if win == nil || !s.projectRefreshVisibleLocked() {
		s.stopProjectRefreshLocked()
		return
	}
	if f.refreshLoop != nil && f.refreshLoop.client == s.dshClient {
		return
	}
	s.stopProjectRefreshLocked()
	ctx, cancel := context.WithCancel(context.Background())
	r := &projectRefreshLoop{cancel: cancel, client: s.dshClient}
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
			if !s.refreshProjectSnapshot(ctx, win, r) {
				return
			}
		}
	}()
}

// refreshProjectSnapshot reads the original list/detail/projection APIs, never a
// control or Agent tool. It leaves explicit operations' generations and drafts
// alone and publishes only within the captured Host/navigation/page identity.
func (s *projectController) refreshProjectSnapshot(ctx context.Context, win *mygo.Window, r *projectRefreshLoop) bool {
	s.mu.Lock()
	f := s.projectFlow
	if f.refreshLoop != r || !s.projectRefreshVisibleLocked() {
		if f.refreshLoop == r {
			f.refreshLoop = nil
		}
		s.mu.Unlock()
		return false
	}
	if f.busy || !f.loaded {
		s.mu.Unlock()
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
	s.mu.Unlock()
	callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	rpc := projectflow.Client{Origin: r.client.Endpoint.Origin, Cookie: r.client.Endpoint.Cookie, HTTP: r.client.HTTP}
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
	win.Update(func() {
		defer close(ack)
		s.mu.Lock()
		defer s.mu.Unlock()
		f := s.projectFlow
		if ctx.Err() != nil || f.refreshLoop != r || s.dshClient != r.client || f.client != r.client || !s.projectRefreshVisibleLocked() || f.busy || f.generation != generation || f.navigation != navigation || f.detailGeneration != detailGeneration || f.teamID != teamIDAtStart {
			return
		}
		// Keep separate live-refresh feedback so a read never clears an unsent
		// draft error or a completed mutation's follow-up notice.
		if listErr != nil {
			f.refreshNote = i18n.T(i18n.KeyCmdAutomaticRefreshFailedShowingPreviousS) + listErr.Error()
			return
		}
		f.listing = listing
		f.refreshNote = i18n.T(i18n.KeyCmdAutoRefreshEvery5s) + time.Now().Format("15:04:05")
		if !listing.Available {
			f.refreshNote = listing.Error
			s.stopProjectProcessLocked()
			return
		}
		if id == "" {
			return
		}
		if !found {
			s.stopProjectProcessLocked()
			f.team = nil
			f.refreshNote = i18n.T(i18n.KeyCmdThisProjectIsNoLongerInTheHostList)
			return
		}
		if teamErr != nil {
			f.refreshNote = i18n.T(i18n.KeyCmdTeamRefreshFailedShowingPreviousState) + teamErr.Error()
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
			f.refreshNote = i18n.T(i18n.KeyCmdMemberDetailRefreshFailed) + detailErr.Error()
			return
		}
		// A successor delegation is a new run. Release the previous observer rather
		// than silently presenting old-run records as the new run's live process.
		if work.Workstream.WorkerSessionID != detail.Workstream.WorkerSessionID || work.Workstream.CurrentDelegationRef != detail.Workstream.CurrentDelegationRef {
			s.stopProjectProcessLocked()
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

func projectRefreshStatus(enabled bool, note string) string {
	if note != "" {
		return note
	}
	if enabled {
		return i18n.T(i18n.KeyCmdProjectAndTeamStateRefreshAutomaticall)
	}
	return ""
}
