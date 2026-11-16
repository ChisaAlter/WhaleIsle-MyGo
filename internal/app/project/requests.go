package projectctl

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/projects"
)

// QueueSubmission binds a rendered request queue action to the Host, navigation, and request generation.
type QueueSubmission struct {
	Change                                    projectflow.RequestQueueChange
	Client                                    *dsh.Client
	Generation, Navigation, RequestGeneration uint64
}

// Matches reports whether the queue submission still belongs to f and client.
func (o QueueSubmission) Matches(f *flowState, client *dsh.Client) bool {
	return o.Client != nil && client == o.Client && f.client == o.Client && f.generation == o.Generation && f.navigation == o.Navigation && f.requestGeneration == o.RequestGeneration
}

// QueueChange constructs an atomic queue move/enqueue/cancel/resume command from displayed revisions and queued IDs.
func QueueChange(page projectflow.RequestPage, detail *projectflow.RequestCard, queue *projectflow.RequestQueue, id, action string) (projectflow.RequestQueueChange, error) {
	change := projectflow.RequestQueueChange{RequestID: id, Action: action}
	if queue == nil {
		return change, fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectQueueUnavailable))
	}
	if detail != nil && detail.ID == id {
		change.RequestRevision = detail.Revision
	}
	for _, item := range page.Items {
		if item.ID == id && change.RequestRevision == 0 {
			change.RequestRevision = item.Revision
			break
		}
	}
	index := -1
	for i, item := range queue.Queued {
		if item.ID == id {
			index = i
			if change.RequestRevision == 0 {
				change.RequestRevision = item.Revision
			}
			break
		}
	}
	if action == "moveUp" || action == "moveDown" {
		if index < 0 {
			return change, fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectRequestNotQueued))
		}
		change.Action = "move"
		if action == "moveUp" {
			if index == 0 {
				return change, fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectRequestFirst))
			}
			change.BeforeRequestID = queue.Queued[index-1].ID
		} else {
			if index+1 >= len(queue.Queued) {
				return change, fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectRequestLast))
			}
			if index+2 < len(queue.Queued) {
				change.BeforeRequestID = queue.Queued[index+2].ID
			}
		}
	}
	if id == "" || change.RequestRevision <= 0 {
		return change, fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectRequestRevision))
	}
	if change.Action != "enqueue" && change.Action != "move" && change.Action != "cancel" && change.Action != "resume" {
		return change, fmt.Errorf("Unknown request queue action")
	}
	return change, nil
}

func requestCardKey(projectID, requestID string) string {
	return projectID + "\x00" + requestID
}

// rememberRequestCardLocked accepts only a Host result already admitted
// by the caller's controller/request ownership checks. Caller holds mgr.mu.
func (c *Controller) rememberRequestCardLocked(card *projectflow.RequestCard) {
	f := &c.flow
	if card == nil || card.ID == "" || card.ProjectID == "" || card.ProjectID != f.requestProjectID || f.client == nil {
		return
	}
	copy := *card
	copy.Work = append([]projectflow.RequestWork(nil), card.Work...)
	copy.WorkPreview = append([]projectflow.RequestWork(nil), card.WorkPreview...)
	if card.Changes != nil {
		changes := *card.Changes
		if changes.RevertedAt != nil {
			revertedAt := *changes.RevertedAt
			changes.RevertedAt = &revertedAt
		}
		copy.Changes = &changes
	}
	if card.Delivery != nil {
		delivery := *card.Delivery
		delivery.Artifacts = append([]projectflow.Artifact(nil), delivery.Artifacts...)
		// Page references are values; freeze the slice and delivery container
		// so another card read or a retained UI snapshot cannot change grants.
		delivery.Pages = append([]projectflow.DeliveryPageReference(nil), delivery.Pages...)
		copy.Delivery = &delivery
	}
	if f.requestCards == nil {
		f.requestCards = make(map[string]projectflow.RequestCard)
		f.requestReverts = nil
	}
	f.requestCards[requestCardKey(card.ProjectID, card.ID)] = copy
}

func (c *Controller) runRequests(win any, id, action, arg string, queueChanges ...QueueSubmission) {
	if win == nil || id == "" {
		return
	}
	snap := c.mgr.hostSnapshot()
	lc := c.mgr.lifecycleContext()
	c.mgr.mu.Lock()
	f := &c.flow
	client := snap.Client
	if action == "queue" && (len(queueChanges) != 1 || !queueChanges[0].Matches(f, client)) {
		c.mgr.mu.Unlock()
		return
	}
	if f.busy {
		c.mgr.mu.Unlock()
		return
	}
	if client == nil || f.client != client || !f.listing.Available {
		f.err = i18n.T(i18n.KeyCmdProjectHostIsUnavailableRefreshToRetry)
		c.mgr.mu.Unlock()
		return
	}
	var project projectflow.Project
	for _, p := range f.listing.Projects {
		if p.ID == id {
			project = p
			break
		}
	}
	if project.ID == "" {
		f.err = i18n.T(i18n.KeyCmdProjectIsNoLongerInTheCurrentListRef)
		c.mgr.mu.Unlock()
		return
	}
	if (action == "archive" || action == "restore" || action == "queue") && (project.Lifecycle != "ready" || project.Archiving) {
		f.err = i18n.T(i18n.KeyCmdRestoreTheProjectBeforeChangingRequest)
		c.mgr.mu.Unlock()
		return
	}
	if f.requestProjectID != id {
		c.closeVisualResultsLocked()
		f.requestPage = projectflow.RequestPage{}
		f.requestCard = nil
		f.requestLoaded = false
		f.requestCursors = nil
		f.requestKind = "current"
	}
	f.requestProjectID = id
	kind := f.requestKind
	cursors := append([]string(nil), f.requestCursors...)
	switch action {
	case "list":
		kind = arg
		cursors = []string{""}
	case "next":
		if f.requestPage.NextCursor == nil {
			c.mgr.mu.Unlock()
			return
		}
		cursors = append(cursors, *f.requestPage.NextCursor)
	case "previous":
		if len(cursors) <= 1 {
			c.mgr.mu.Unlock()
			return
		}
		cursors = cursors[:len(cursors)-1]
	case "archive", "restore":
		cursors = []string{""}
	case "get":
	case "queue":
		if project.Queue == nil || len(queueChanges) != 1 || queueChanges[0].Change.RequestID != arg {
			c.mgr.mu.Unlock()
			return
		}
	default:
		c.mgr.mu.Unlock()
		return
	}
	if len(cursors) == 0 {
		cursors = []string{""}
	}
	f.busy = true
	f.err = ""
	f.requestRevertError = ""
	f.generation++
	generation := f.generation
	f.requestGeneration++
	requestGeneration := f.requestGeneration
	selectedCard := f.requestCard
	c.mgr.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(lc, 30*time.Second)
		defer cancel()
		rpc := projectRPC(client)
		var page projectflow.RequestPage
		var card projectflow.RequestCard
		var err error
		mutated := false
		var queueResult *projectflow.RequestQueueResult
		if action == "get" {
			card, err = rpc.Request(ctx, id, arg)
		} else {
			if action == "queue" {
				result, queueErr := rpc.ChangeRequestQueue(ctx, id, queueChanges[0].Change)
				err = queueErr
				if err == nil {
					queueResult = &result
					mutated = true
				}
			}
			if action == "archive" || action == "restore" {
				err = rpc.SetRequestArchived(ctx, id, arg, action == "archive")
				mutated = err == nil
			}
			if err == nil {
				page, err = rpc.Requests(ctx, id, kind, cursors[len(cursors)-1])
				if err == nil && action == "queue" && selectedCard != nil {
					card, err = rpc.Request(ctx, id, selectedCard.ID)
				}
			}
		}
		c.mgr.updateWindow(win, func() {
			nowSnap := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			defer c.mgr.mu.Unlock()
			f := &c.flow
			if nowSnap.Client != client || f.client != client || f.generation != generation {
				return
			}
			f.busy = false
			if f.requestProjectID != id || f.requestGeneration != requestGeneration {
				return
			}
			if queueResult != nil {
				for i := range f.listing.Projects {
					if f.listing.Projects[i].ID == id {
						f.listing.Projects[i].Queue = &queueResult.ProjectQueue
					}
				}
			}
			if err != nil {
				f.err = err.Error()
				if mutated {
					f.requestCard = nil
					f.requestPage = projectflow.RequestPage{}
					f.requestLoaded = false
					f.err = fmt.Sprintf("%s: %s", i18n.T(i18n.KeyCmdCardUpdatedButListRefreshFailedRefresh), err)
				}
				return
			}
			f.assetDiff = nil
			f.assetNote = ""
			if action == "get" {
				f.requestCard = &card
				c.rememberRequestCardLocked(f.requestCard)
				return
			}
			f.requestCard = nil
			if action == "queue" && selectedCard != nil {
				f.requestCard = &card
				c.rememberRequestCardLocked(f.requestCard)
			}
			f.requestKind = kind
			f.requestPage = page
			f.requestCursors = cursors
			f.requestLoaded = true
		})
	}()
}

type requestRevision struct {
	id       string
	revision int
}

type requestRevertState struct {
	client     *dsh.Client
	navigation uint64
	projectID  string
	errors     map[requestRevision]string
}

func (c *Controller) revertStateLocked() *requestRevertState {
	f := &c.flow
	if f.client == nil || f.requestProjectID == "" || f.requestCards == nil {
		return nil
	}
	state := f.requestReverts
	if state == nil || state.client != f.client || state.navigation != f.navigation || state.projectID != f.requestProjectID {
		state = &requestRevertState{
			client:     f.client,
			navigation: f.navigation,
			projectID:  f.requestProjectID,
			errors:     make(map[requestRevision]string),
		}
		f.requestReverts = state
	}
	return state
}

func (c *Controller) revertCardLocked(state *requestRevertState, key requestRevision, currentClient *dsh.Client) (projectflow.RequestCard, bool) {
	f := &c.flow
	if state == nil || f.requestReverts != state || currentClient != state.client || f.client != state.client || f.navigation != state.navigation || f.requestProjectID != state.projectID || !f.visible() || !f.listing.Available {
		return projectflow.RequestCard{}, false
	}
	card, ok := f.requestCards[requestCardKey(state.projectID, key.id)]
	if !ok || card.ID != key.id || card.ProjectID != state.projectID || card.Revision != key.revision {
		return projectflow.RequestCard{}, false
	}
	for _, project := range f.listing.Projects {
		if project.ID == state.projectID {
			return card, true
		}
	}
	return projectflow.RequestCard{}, false
}

func (c *Controller) revertAllowedLocked(card projectflow.RequestCard) bool {
	if card.Changes == nil || !card.Changes.Revertible || card.Changes.RevertedAt != nil {
		return false
	}
	for _, project := range c.flow.listing.Projects {
		if project.ID == card.ProjectID {
			return project.Lifecycle == "ready" && !project.Archiving
		}
	}
	return false
}

func (c *Controller) bindRequestReverts(win any, m *projects.Model) {
	c.mgr.mu.Lock()
	state := c.revertStateLocked()
	c.mgr.mu.Unlock()
	m.RequestRevert = func(id string, revision int) projects.RequestRevertState {
		key := requestRevision{id, revision}
		snap := c.mgr.hostSnapshot()
		c.mgr.mu.Lock()
		card, known := c.revertCardLocked(state, key, snap.Client)
		view := projects.RequestRevertState{Known: known, Allowed: known && c.revertAllowedLocked(card)}
		if state != nil && c.flow.requestReverts == state {
			view.Error = state.errors[key]
		}
		c.mgr.mu.Unlock()
		if known && win != nil {
			view.Confirm = func() { c.revertRequestOwned(win, state, key) }
			view.Ask = func() {
				nowSnap := c.mgr.hostSnapshot()
				c.mgr.mu.Lock()
				current, allowed := c.revertCardLocked(state, key, nowSnap.Client)
				c.mgr.mu.Unlock()
				if allowed {
					c.runRecoveryDraft(win, current.ProjectID, "revert", current)
				}
			}
		}
		return view
	}
}

func (c *Controller) runRequestChanges(win any, projectID, requestID string, owner QueueSubmission) {
	if win == nil {
		return
	}
	snap := c.mgr.hostSnapshot()
	lc := c.mgr.lifecycleContext()
	c.mgr.mu.Lock()
	f, client := &c.flow, snap.Client
	if !owner.Matches(f, client) || f.busy || !f.visible() || !f.listing.Available || f.requestProjectID != projectID {
		c.mgr.mu.Unlock()
		return
	}
	_, known := f.requestCards[requestCardKey(projectID, requestID)]
	known = known || f.requestCard != nil && f.requestCard.ID == requestID && f.requestCard.ProjectID == projectID
	for _, card := range f.requestPage.Items {
		known = known || card.ID == requestID && card.ProjectID == projectID
	}
	for _, project := range f.listing.Projects {
		if project.ID == projectID && project.Queue != nil && project.Queue.Active != nil {
			known = known || project.Queue.Active.ID == requestID
		}
	}
	if !known {
		c.mgr.mu.Unlock()
		return
	}
	f.busy, f.err, f.requestRevertError = true, "", ""
	f.generation++
	f.requestGeneration++
	generation, navigation, requestGeneration := f.generation, f.navigation, f.requestGeneration
	c.mgr.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(lc, 30*time.Second)
		defer cancel()
		rpc := projectRPC(client)
		card, err := rpc.Request(ctx, projectID, requestID)
		var changes projectflow.RequestChanges
		if err == nil {
			changes, err = rpc.RequestChanges(ctx, projectID, requestID)
		}
		c.mgr.updateWindow(win, func() {
			nowSnap := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			defer c.mgr.mu.Unlock()
			f := &c.flow
			if nowSnap.Client != client || f.client != client || f.generation != generation {
				return
			}
			f.busy = false
			if f.navigation != navigation || f.requestGeneration != requestGeneration || f.requestProjectID != projectID {
				return
			}
			if err != nil {
				f.err = err.Error()
				return
			}
			f.requestCard = &card
			c.rememberRequestCardLocked(f.requestCard)
			f.assetDiff = &projectflow.WorkDiff{Requirement: &changes}
			f.assetNote = ""
		})
	}()
	c.mgr.updateWindow(win, func() {})
}

func (c *Controller) revertRequest(win any, projectID, requestID string, revision int, owner QueueSubmission) {
	c.mgr.mu.Lock()
	state := c.revertStateLocked()
	valid := state != nil && owner.Client == state.client && owner.Navigation == state.navigation && projectID == state.projectID
	c.mgr.mu.Unlock()
	if valid {
		c.revertRequestOwned(win, state, requestRevision{requestID, revision})
	}
}

func (c *Controller) revertRequestOwned(win any, state *requestRevertState, key requestRevision) {
	if win == nil {
		return
	}
	snap := c.mgr.hostSnapshot()
	lc := c.mgr.lifecycleContext()
	c.mgr.mu.Lock()
	f, client := &c.flow, snap.Client
	card, known := c.revertCardLocked(state, key, client)
	if !known || f.busy || !c.revertAllowedLocked(card) {
		c.mgr.mu.Unlock()
		return
	}
	projectID, requestID, revision := card.ProjectID, key.id, key.revision
	f.busy, f.requestRevertError = true, ""
	delete(state.errors, key)
	f.generation++
	generation := f.generation
	c.mgr.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(lc, 30*time.Second)
		defer cancel()
		rpc := projectRPC(client)
		err := rpc.RevertRequest(ctx, projectID, requestID, revision)
		var fresh projectflow.RequestCard
		committed := err == nil
		if committed {
			fresh, err = rpc.Request(ctx, projectID, requestID)
		}
		c.mgr.updateWindow(win, func() {
			nowSnap := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			defer c.mgr.mu.Unlock()
			f := &c.flow
			if nowSnap.Client != client || f.client != client || f.generation != generation {
				return
			}
			f.busy = false
			if _, current := c.revertCardLocked(state, key, nowSnap.Client); !current {
				return
			}
			if err != nil {
				f.requestRevertError = err.Error()
				state.errors[key] = err.Error()
				if committed {
					delete(f.requestCards, requestCardKey(projectID, requestID))
					if f.requestCard != nil && f.requestCard.ID == requestID {
						f.requestCard = nil
					}
					f.err = i18n.T(i18n.KeyCmdChangesWereRevertedButRefreshingFailed) + " " + err.Error()
				}
				return
			}
			f.requestCard = &fresh
			delete(state.errors, key)
			c.rememberRequestCardLocked(f.requestCard)
			for i := range f.requestPage.Items {
				if f.requestPage.Items[i].ID == requestID {
					f.requestPage.Items[i] = fresh
				}
			}
			f.assetDiff = nil
			f.assetNote = i18n.T(i18n.KeyCmdThisRequirementSChangesWereReverted)
		})
	}()
	c.mgr.updateWindow(win, func() {})
}

func (c *Controller) runRequestWorkDetail(win any, projectID, requestID, workID string, owner QueueSubmission) {
	c.mgr.mu.Lock()
	state := c.revertStateLocked()
	card, known := c.flow.requestCards[requestCardKey(projectID, requestID)]
	valid := known && state != nil && owner.Client == state.client && owner.Navigation == state.navigation && state.projectID == projectID
	c.mgr.mu.Unlock()
	if valid {
		c.runRequestWorkDetailOwned(win, state, requestRevision{requestID, card.Revision}, workID)
	}
}

func (c *Controller) runRequestWorkDetailOwned(win any, state *requestRevertState, key requestRevision, workID string) {
	if state == nil {
		return
	}
	projectID, requestID := state.projectID, key.id
	if win == nil || projectID == "" || requestID == "" || workID == "" {
		return
	}
	snap := c.mgr.hostSnapshot()
	lc := c.mgr.lifecycleContext()
	c.mgr.mu.Lock()
	f := &c.flow
	client := snap.Client
	card, known := c.revertCardLocked(state, key, client)
	if !known || f.busy {
		c.mgr.mu.Unlock()
		return
	}
	linked := false
	for _, work := range card.Work {
		linked = linked || work.WorkstreamID == workID
	}
	if !linked {
		c.mgr.mu.Unlock()
		return
	}
	revision := key.revision
	c.stopProcessLocked()
	f.busy = true
	f.err = ""
	f.generation++
	generation := f.generation
	f.detailGeneration++
	detailGeneration := f.detailGeneration
	c.mgr.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(lc, 30*time.Second)
		defer cancel()
		rpc := projectRPC(client)
		team, detail, err := ReadRequestWork(ctx, rpc, projectID, requestID, revision, workID)
		c.mgr.updateWindow(win, func() {
			nowSnap := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			defer c.mgr.mu.Unlock()
			f := &c.flow
			if nowSnap.Client != client || f.client != client || f.generation != generation {
				return
			}
			f.busy = false
			if _, current := c.revertCardLocked(state, key, nowSnap.Client); !current || f.detailGeneration != detailGeneration {
				return
			}
			if err != nil {
				f.err = err.Error()
				return
			}
			f.teamID = projectID
			f.team = &team
			f.detail = &detail
			f.process = nil
			f.assetDiff = nil
		})
	}()
}

// ReadRequestWork loads the read-only Team overview and WorkDetail for an admitted requirement assignment.
func ReadRequestWork(ctx context.Context, rpc projectflow.Client, projectID, requestID string, revision int, workID string) (projectflow.Overview, projectflow.WorkDetail, error) {
	var team projectflow.Overview
	var detail projectflow.WorkDetail
	card, err := rpc.Request(ctx, projectID, requestID)
	if err != nil {
		return team, detail, err
	}
	if card.Revision != revision {
		return team, detail, fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectRequirementChanged))
	}
	linked := false
	for _, work := range append(append([]projectflow.RequestWork{}, card.Work...), card.WorkPreview...) {
		if work.WorkstreamID == workID {
			linked = true
			break
		}
	}
	if !linked {
		return team, detail, fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectWorkRequirementChanged))
	}
	team, err = rpc.TeamOverviewPage(ctx, projectID, "")
	if err != nil {
		return team, detail, err
	}
	detail, err = rpc.WorkDetail(ctx, projectID, workID, "")
	if err == nil && detail.Workstream.Request.ID != requestID {
		err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectWorkRequirementMismatch))
	}
	return team, detail, err
}

func exactRequestWork(card projectflow.RequestCard, selected projectflow.RequestWork) (projectflow.RequestWork, bool) {
	for _, work := range card.Work {
		if work.WorkstreamID == selected.WorkstreamID && work.DelegationRef == selected.DelegationRef && work.WorkerSessionID == selected.WorkerSessionID {
			return work, true
		}
	}
	return projectflow.RequestWork{}, false
}

// The click carries its exact assignment, not just a mutable workstream ID.
func (c *Controller) runRequestWorkAction(win any, state *requestRevertState, key requestRevision, selected projectflow.RequestWork, action string) {
	snap := c.mgr.hostSnapshot()
	lc := c.mgr.lifecycleContext()
	c.mgr.mu.Lock()
	f := &c.flow
	client, mux := snap.Client, snap.Mux
	card, known := c.revertCardLocked(state, key, client)
	_, linked := exactRequestWork(card, selected)
	if win == nil || !known || !linked || f.busy || client == nil || f.client != client || !f.listing.Available {
		c.mgr.mu.Unlock()
		return
	}
	f.busy = true
	f.err = ""
	generation, navigation := f.generation, f.navigation
	c.mgr.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(lc, 60*time.Second)
		defer cancel()
		rpc := projectRPC(client)
		fresh, err := rpc.Request(ctx, card.ProjectID, card.ID)
		var work projectflow.RequestWork
		if err == nil {
			var ok bool
			work, ok = exactRequestWork(fresh, selected)
			if fresh.Revision != key.revision || !ok {
				err = fmt.Errorf("The requirement assignment changed. Refresh before continuing.")
			}
		}
		var detail projectflow.WorkDetail
		var report projectflow.Report
		var binding projectflow.HistoricalBinding
		var window projectflow.HistoricalWindow
		var overview projectflow.Overview
		if err == nil {
			switch action {
			case "process":
				target := projectflow.HistoricalTarget{
					Assignment:      true,
					ProjectID:       card.ProjectID,
					WorkstreamID:    work.WorkstreamID,
					DelegationRef:   work.DelegationRef,
					RequestID:       card.ID,
					WorkerSessionID: work.WorkerSessionID,
				}
				binding, err = rpc.HistoricalAssignment(ctx, target)
				if err == nil {
					window, err = rpc.ReadHistoricalProcess(ctx, mux, binding)
				}
			case "report":
				var history projectflow.ReportHistory
				history, err = rpc.ReportHistory(ctx, card.ProjectID, work.WorkstreamID, "", work.DelegationRef)
				if err == nil {
					report = history.Items[0]
				}
			case "stop", "summary":
				if work.Current != nil && !*work.Current {
					err = fmt.Errorf("Historical assignments are read-only.")
					break
				}
				detail, err = rpc.WorkDetail(ctx, card.ProjectID, work.WorkstreamID, "")
				if err == nil && (detail.Workstream.CurrentDelegationRef != work.DelegationRef || detail.Workstream.Request.ID != card.ID) {
					err = fmt.Errorf("The current assignment changed. Refresh before continuing.")
				}
				if err == nil && action == "stop" {
					_, err = rpc.Control(ctx, card.ProjectID, "stop", work.WorkstreamID)
					if err == nil {
						overview, err = rpc.TeamOverviewPage(ctx, card.ProjectID, "")
					}
				}
			default:
				err = fmt.Errorf("Unknown requirement work action.")
			}
		}
		c.mgr.updateWindow(win, func() {
			nowSnap := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			f := &c.flow
			if f.client != client || nowSnap.Client != client || f.generation != generation || f.navigation != navigation {
				c.mgr.mu.Unlock()
				return
			}
			f.busy = false
			sourceValid := func(f *flowState, currentClient *dsh.Client) bool {
				cached, ok := c.revertCardLocked(state, key, currentClient)
				if !ok {
					return false
				}
				_, ok = exactRequestWork(cached, selected)
				return ok
			}
			if !sourceValid(f, nowSnap.Client) {
				c.mgr.mu.Unlock()
				return
			}
			if err != nil {
				f.err = err.Error()
				c.mgr.mu.Unlock()
				return
			}
			switch action {
			case "process":
				c.closeHistoricalLocked()
				browser := projectflow.NewHistoricalBrowser(fmt.Sprintf("%p", client), generation, binding.Target)
				req, _ := browser.Begin("record")
				browser.CompleteRecord(req, binding, nil)
				req, _ = browser.Begin("process")
				browser.CompleteProcess(req, window, nil)
				f.historical = &historicalState{browser: browser, sourceValid: sourceValid, generation: generation, navigation: navigation}
				f.historicalGeneration++
			case "report":
				f.reports = &projectflow.ReportBrowser{
					ProjectID:   card.ProjectID,
					WorkID:      work.WorkstreamID,
					Title:       work.Title,
					Selected:    &report,
					History:     projectflow.ReportHistory{Items: []projectflow.Report{report}, Total: 1},
					ListCursors: []string{""},
					WorkCursors: []string{""},
				}
			case "stop":
				f.team = &overview
				for i := range f.listing.Projects {
					if f.listing.Projects[i].ID == card.ProjectID {
						f.listing.Projects[i] = overview.Project
					}
				}
			}
			c.mgr.mu.Unlock()
			if action == "summary" {
				c.runRecoveryDraftOwned(win, card.ProjectID, "summary", &detail, sourceValid)
			}
		})
	}()
}

// The four inventories have independent replacement-page cursors. Their owner
// is the same retained requirements view, not the last card opened in a modal.
func (c *Controller) bindRequirements(win any, m *projects.Model, o *projects.ObserverState) {
	if o == nil {
		return
	}
	o.OnLoadCatalog = func(kind string, delta int) { c.loadRequirementCatalog(win, o, kind, delta) }
	m.OnAskAnswer = func(requestID, workstreamID, question, answer string) {
		c.submitAskAnswer(win, o, requestID, workstreamID, question, answer)
	}
	queue := m.OnRequestQueue
	m.OnRequestQueue = func(id, action string) {
		if queue != nil {
			o.RefreshCatalog()
			queue(id, action)
		}
	}
	archive := m.OnRequestArchive
	m.OnRequestArchive = func(id string, on bool) {
		if archive != nil {
			o.RefreshCatalog()
			archive(id, on)
		}
	}
	original := m.RequestCardActions
	if original != nil {
		m.RequestCardActions = func(id string, revision int) projects.RequestCardActions {
			actions := original(id, revision)
			dispatch := actions.WorkAction
			if dispatch != nil {
				actions.WorkAction = func(action string, work projectflow.RequestWork) {
					if action == "stop" {
						o.RefreshCatalog()
					}
					dispatch(action, work)
				}
			}
			return actions
		}
	}
}

func (c *Controller) submitAskAnswer(win any, o *projects.ObserverState, requestID, workstreamID, question, answer string) {
	answer = strings.TrimSpace(answer)
	if win == nil || o == nil || requestID == "" || answer == "" {
		return
	}
	sessions := c.mgr.sessionsStore()
	c.mgr.mu.Lock()
	f := &c.flow
	if f.busy || sessions == nil || o.CoordinatorSessionID == "" {
		c.mgr.mu.Unlock()
		return
	}
	c.mgr.mu.Unlock()

	title := requestID
	for _, card := range o.Attention.Items {
		if card.ID == requestID && card.Title != "" {
			title = card.Title
			break
		}
	}
	workLabel := ""
	if workstreamID != "" {
		workLabel = fmt.Sprintf("，workId=%s", workstreamID)
	}
	draft := fmt.Sprintf("[回答「%s」（requestId=%s%s）的问题]\n问题：%s\n回答：%s", title, requestID, workLabel, question, answer)
	var submitErr error
	if c.mgr.deps.SubmitLeadDraft != nil {
		submitErr = c.mgr.deps.SubmitLeadDraft(win, sessions, o.CoordinatorSessionID, draft)
	} else {
		submitErr = errors.New(i18n.T(i18n.KeyCmdProjectRecoveryUnavailable))
	}
	if submitErr != nil {
		c.mgr.mu.Lock()
		c.flow.err = submitErr.Error()
		c.mgr.mu.Unlock()
		return
	}
	o.RefreshCatalog()
}

func (c *Controller) loadRequirementCatalog(win any, o *projects.ObserverState, kind string, delta int) {
	snap := c.mgr.hostSnapshot()
	lc := c.mgr.lifecycleContext()
	c.mgr.mu.Lock()
	f := &c.flow
	client := snap.Client
	if win == nil || o == nil || client == nil || f.client != client || f.busy || !f.listing.Available || o.Host == "" {
		c.mgr.mu.Unlock()
		return
	}
	known := false
	for _, p := range f.listing.Projects {
		if p.ID == o.ProjectID && p.CoordinatorSessionID == o.CoordinatorSessionID {
			known = true
		}
	}
	if !known {
		c.mgr.mu.Unlock()
		return
	}
	cursors := append([]string(nil), o.Cursors[kind]...)
	if len(cursors) == 0 {
		cursors = []string{""}
	}
	if delta > 0 {
		page := o.Pages[kind]
		if page.NextCursor == nil {
			c.mgr.mu.Unlock()
			return
		}
		cursors = append(cursors, *page.NextCursor)
	}
	if delta < 0 {
		if len(cursors) < 2 {
			c.mgr.mu.Unlock()
			return
		}
		cursors = cursors[:len(cursors)-1]
	}
	if delta == 0 {
		cursors = []string{""}
	}
	f.requestProjectID = o.ProjectID
	f.busy = true
	f.err = ""
	generation, navigation := f.generation, f.navigation
	c.mgr.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(lc, 30*time.Second)
		defer cancel()
		rpc := projectRPC(client)
		page, err := rpc.Requests(ctx, o.ProjectID, kind, cursors[len(cursors)-1])
		var overview projectflow.Overview
		var attention projectflow.RequestPage
		var hasAttention bool
		if err == nil && kind == "current" {
			overview, err = rpc.TeamOverviewPage(ctx, o.ProjectID, "")
			if att, attErr := rpc.AttentionRequests(ctx, o.ProjectID); attErr == nil {
				attention = att
				hasAttention = true
			}
		}
		c.mgr.updateWindow(win, func() {
			nowSnap := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			defer c.mgr.mu.Unlock()
			f := &c.flow
			if f.client != client || nowSnap.Client != client || f.generation != generation || f.navigation != navigation || f.requestProjectID != o.ProjectID {
				return
			}
			f.busy = false
			if err != nil {
				o.CatalogErrors[kind] = err.Error()
				f.err = err.Error()
				return
			}
			o.Pages[kind], o.Cursors[kind], o.CatalogLoaded[kind] = page, cursors, true
			delete(o.CatalogErrors, kind)
			f.requestKind, f.requestPage, f.requestCursors, f.requestLoaded = kind, page, cursors, true
			if kind == "current" {
				if hasAttention {
					o.Attention = attention
				}
				f.team = &overview
				for i := range f.listing.Projects {
					if f.listing.Projects[i].ID == o.ProjectID {
						f.listing.Projects[i] = overview.Project
					}
				}
			}
		})
	}()
}
