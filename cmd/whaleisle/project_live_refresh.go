package main

import (
	"context"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/dsh"
	"whaleisle/internal/projectflow"
)

type projectRefreshLoop struct {
	cancel context.CancelFunc
	client *dsh.Client
}

func (s *mainState) projectRefreshVisibleLocked() bool {
	f := &s.projectFlow
	return !s.quitting && !s.projectWindowHidden && !s.settingsOpen && !s.usageOpen && !s.imOpen && !s.remoteOpen && !s.webviewOpen && !s.browserPreviewOpen && f.expanded && s.dshClient != nil && f.client == s.dshClient
}
func (s *mainState) stopProjectRefreshLocked() {
	if r := s.projectFlow.refreshLoop; r != nil {
		r.cancel()
		s.projectFlow.refreshLoop = nil
	}
}
func (s *mainState) ensureProjectRefreshLocked(win *mygo.Window) {
	f := &s.projectFlow
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
func (s *mainState) refreshProjectSnapshot(ctx context.Context, win *mygo.Window, r *projectRefreshLoop) bool {
	s.mu.Lock()
	f := &s.projectFlow
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
	if f.reports != nil || f.materialKey != "" || f.requestProjectID != "" || f.archiveCandidate != nil {
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
		f := &s.projectFlow
		if ctx.Err() != nil || f.refreshLoop != r || s.dshClient != r.client || f.client != r.client || !s.projectRefreshVisibleLocked() || f.busy || f.generation != generation || f.navigation != navigation || f.detailGeneration != detailGeneration || f.teamID != teamIDAtStart {
			return
		}
		// Keep separate live-refresh feedback so a read never clears an unsent
		// draft error or a completed mutation's follow-up notice.
		if listErr != nil {
			f.refreshNote = resumeText("自动刷新失败，保留上次状态：", "Automatic refresh failed; showing previous state: ") + listErr.Error()
			return
		}
		f.listing = listing
		f.refreshNote = resumeText("每5秒自动刷新 · ", "Auto-refresh every 5s · ") + time.Now().Format("15:04:05")
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
			f.refreshNote = resumeText("此Project已不在Host列表中。", "This Project is no longer in the Host list.")
			return
		}
		if teamErr != nil {
			f.refreshNote = resumeText("Team刷新失败，保留上次状态：", "Team refresh failed; showing previous state: ") + teamErr.Error()
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
			f.refreshNote = resumeText("成员详情刷新失败：", "Member detail refresh failed: ") + detailErr.Error()
			return
		}
		// A successor delegation is a new run. Release the previous observer rather
		// than silently presenting old-run records as the new run's live process.
		if work.Workstream.WorkerSessionID != detail.Workstream.WorkerSessionID || work.Workstream.CurrentDelegationRef != detail.Workstream.CurrentDelegationRef {
			s.stopProjectProcessLocked()
			if f.process != nil {
				previous := *f.process
				previous.Error = resumeText("成员执行轮次已变化；保留旧过程，可重新选择实时跟进。", "The member run changed; previous records are preserved. Select live follow again for the new run.")
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
		return resumeText("每5秒自动刷新项目与Team状态。", "Project and Team state refresh automatically every 5 seconds.")
	}
	return ""
}
