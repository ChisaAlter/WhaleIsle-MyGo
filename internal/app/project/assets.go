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

type requestAssetOwner struct {
	state *requestRevertState
	key   requestRevision
}

func (c *Controller) bindRequestCardActions(win any, m *projects.Model) {
	c.mgr.mu.Lock()
	state := c.revertStateLocked()
	c.mgr.mu.Unlock()
	m.RequestCardActions = func(id string, revision int) projects.RequestCardActions {
		key := requestRevision{id, revision}
		snap := c.mgr.hostSnapshot()
		c.mgr.mu.Lock()
		card, known := c.revertCardLocked(state, key, snap.Client)
		c.mgr.mu.Unlock()
		actions := projects.RequestCardActions{Known: known}
		if !known || win == nil {
			return actions
		}
		owner := &requestAssetOwner{state: state, key: key}
		actions.OpenAuthorizedFile = func(file projectflow.Artifact, open func(projectflow.FileTarget)) {
			c.runAssetForCard(win, card.ProjectID, "file", file, "", nil, owner, nil, open)
		}
		actions.OpenFile = func(file projectflow.Artifact, open func(string)) {
			c.runAssetForCard(win, card.ProjectID, "file", file, "", open, owner, nil)
		}
		actions.WorkDiff = func(workID string) {
			c.runAssetForCard(win, card.ProjectID, "diff", projectflow.Artifact{}, workID, nil, owner, nil)
		}
		actions.WorkDetail = func(workID string) { c.runRequestWorkDetailOwned(win, state, key, workID) }
		c.mgr.mu.Lock()
		if team := c.flow.team; team != nil && team.Project.ID == card.ProjectID {
			actions.Workstreams = append([]projectflow.Workstream(nil), team.Workstreams...)
		}
		c.mgr.mu.Unlock()
		actions.OpenAssignment = func(work projectflow.RequestWork) { c.runRequestWorkAction(win, state, key, work, "process") }
		actions.WorkAction = func(action string, work projectflow.RequestWork) {
			c.runRequestWorkAction(win, state, key, work, action)
		}
		actions.Activity = func() { c.openRequestActivity(win, state, key) }
		if card.HandoverError != "" && card.AuthorityError == "" {
			actions.HandoverRetry = func() {
				nowSnap := c.mgr.hostSnapshot()
				c.mgr.mu.Lock()
				current, valid := c.revertCardLocked(state, key, nowSnap.Client)
				c.mgr.mu.Unlock()
				if valid && current.HandoverError == card.HandoverError {
					c.runRecoveryDraft(win, card.ProjectID, "handover", card)
				}
			}
		}
		return actions
	}
}

func (c *Controller) runAsset(win any, id, action string, file projectflow.Artifact, workID string, open func(string), owners ...QueueSubmission) {
	c.runAssetForCard(win, id, action, file, workID, open, nil, owners)
}

func (c *Controller) runAssetForCard(win any, id, action string, file projectflow.Artifact, workID string, open func(string), cardOwner *requestAssetOwner, owners []QueueSubmission, targets ...func(projectflow.FileTarget)) {
	if win == nil || id == "" {
		return
	}
	snap := c.mgr.hostSnapshot()
	sessions := c.mgr.sessionsStore()
	lc := c.mgr.lifecycleContext()

	c.mgr.mu.Lock()
	f := &c.flow
	client := snap.Client
	if len(owners) > 0 && !owners[0].Matches(f, client) {
		c.mgr.mu.Unlock()
		return
	}
	if f.busy {
		c.mgr.mu.Unlock()
		return
	}
	if client == nil || f.client != client || !f.listing.Available {
		f.err = i18n.T(i18n.KeyCmdRefreshTheDeliveryDetailsFirst)
		c.mgr.mu.Unlock()
		return
	}
	var requestID string
	var admissionErr error
	if cardOwner != nil {
		card, known := c.revertCardLocked(cardOwner.state, cardOwner.key, client)
		if !known || card.ProjectID != id {
			c.mgr.mu.Unlock()
			return
		}
		requestID, admissionErr = AssetCardReference(card, action, file, workID, open != nil || len(targets) > 0 && targets[0] != nil)
	} else {
		requestID, admissionErr = assetReference(f, id, action, file, workID, open != nil || len(targets) > 0 && targets[0] != nil)
	}
	if admissionErr != nil {
		f.err = admissionErr.Error()
		c.mgr.mu.Unlock()
		return
	}
	requestGeneration := f.requestGeneration
	navigation := f.navigation
	f.busy = true
	f.err = ""
	f.assetNote = ""
	f.generation++
	generation := f.generation
	c.mgr.mu.Unlock()

	selected := ""
	if sessions != nil {
		selected = sessions.SelectedSessionID()
	}
	go func() {
		ctx, cancel := context.WithTimeout(lc, 30*time.Second)
		defer cancel()
		rpc := projectRPC(client)
		var resolved projectflow.ResolvedArtifact
		var diff projectflow.WorkDiff
		var err error
		switch action {
		case "file":
			resolved, err = rpc.ResolveArtifact(ctx, id, file)
		case "diff":
			diff, err = rpc.WorkDiff(ctx, id, workID)
		case "directory":
			err = rpc.OpenProjectDirectory(ctx, id)
		default:
			err = fmt.Errorf("unsupported Project artifact action")
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
			if cardOwner != nil {
				card, known := c.revertCardLocked(cardOwner.state, cardOwner.key, nowSnap.Client)
				if _, referenceErr := AssetCardReference(card, action, file, workID, open != nil || len(targets) > 0 && targets[0] != nil); !known || referenceErr != nil {
					c.mgr.mu.Unlock()
					return
				}
			} else if action != "directory" && (f.requestProjectID != id || f.requestGeneration != requestGeneration || f.requestCard == nil || f.requestCard.ID != requestID) {
				c.mgr.mu.Unlock()
				return
			}
			if err != nil {
				f.err = err.Error()
				c.mgr.mu.Unlock()
				return
			}
			switch action {
			case "directory":
				f.assetNote = i18n.T(i18n.KeyCmdRequestedOpeningTheProjectDirectory)
			case "diff":
				f.assetDiff = &diff
			case "file":
				if !f.visible() || f.navigation != navigation || sessions == nil || sessions.SelectedSessionID() != selected {
					f.err = i18n.T(i18n.KeyCmdNavigationChangedTheFileWasNotOpened)
					c.mgr.mu.Unlock()
					return
				}
				if resolved.ModifiedSinceReport {
					f.assetNote = i18n.T(i18n.KeyCmdThisFileChangedAfterTheReportOpeningI)
				}
				// Route through the same native presented-file entry as chat deliverables.
				// Its dirty-editor guard owns admission; never bypass it with an OS opener.
				c.mgr.mu.Unlock()
				if len(targets) > 0 && targets[0] != nil {
					targets[0](c.fileTarget(id, resolved, client))
				} else if open != nil {
					open(resolved.CanonicalPath)
				}
				return
			}
			c.mgr.mu.Unlock()
		})
	}()
}

func assetReference(f *flowState, id, action string, file projectflow.Artifact, workID string, canOpen bool) (string, error) {
	if action == "directory" {
		for _, project := range f.listing.Projects {
			if project.ID == id {
				return "", nil
			}
		}
		return "", fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectListChanged))
	}
	if f.requestProjectID != id || f.requestCard == nil {
		return "", fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectRefreshDelivery))
	}
	return AssetCardReference(*f.requestCard, action, file, workID, canOpen)
}

// AssetCardReference validates that file or workID is explicitly admitted by card.
func AssetCardReference(card projectflow.RequestCard, action string, file projectflow.Artifact, workID string, canOpen bool) (string, error) {
	if action == "file" && canOpen && card.Delivery != nil {
		for _, candidate := range card.Delivery.Artifacts {
			if candidate.Path == file.Path && candidate.WorkstreamID == file.WorkstreamID && candidate.DelegationRef == file.DelegationRef {
				return card.ID, nil
			}
		}
	}
	if action == "diff" {
		for _, work := range card.Work {
			if work.WorkstreamID == workID {
				return card.ID, nil
			}
		}
	}
	return "", fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectDeliveryReferenceMismatch))
}

func (c *Controller) fileTarget(id string, resolved projectflow.ResolvedArtifact, expected *dsh.Client) projectflow.FileTarget {
	snap := c.mgr.hostSnapshot()
	c.mgr.mu.Lock()
	defer c.mgr.mu.Unlock()
	if expected == nil || snap.Client != expected || c.flow.client != expected {
		return projectflow.FileTarget{}
	}
	target := projectflow.FileTarget{ProjectID: id, CanonicalPath: resolved.CanonicalPath, Root: resolved.Root}
	for _, p := range c.flow.listing.Projects {
		if p.ID == id {
			target.CoordinatorSessionID = p.CoordinatorSessionID
			break
		}
	}
	return target
}

// The explicit recovery button revalidates saved evidence, then submits through
// the original Lead composer so ordinary send failure/recovery remains visible.
func (c *Controller) runRecoveryDraft(win any, id, kind string, requestCards ...projectflow.RequestCard) {
	c.runRecoveryDraftOwned(win, id, kind, nil, nil, requestCards...)
}

func (c *Controller) runRecoveryDraftOwned(win any, id, kind string, work *projectflow.WorkDetail, sourceValid func(*flowState, *dsh.Client) bool, requestCards ...projectflow.RequestCard) {
	if win == nil || id == "" {
		return
	}
	snap := c.mgr.hostSnapshot()
	sessions := c.mgr.sessionsStore()
	lc := c.mgr.lifecycleContext()

	c.mgr.mu.Lock()
	f := &c.flow
	client := snap.Client
	if f.busy || client == nil || f.client != client || !f.listing.Available || sessions == nil {
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
	if project.ID == "" || project.CoordinatorSessionID == "" || project.Lifecycle != "ready" || project.Archiving {
		f.err = i18n.T(i18n.KeyCmdRefreshOrRestoreTheOriginalProjectFirs)
		c.mgr.mu.Unlock()
		return
	}
	selectedWork, selectedCard := f.detail, f.requestCard
	if work != nil {
		selectedWork = work
		if sourceValid == nil || !sourceValid(f, client) {
			c.mgr.mu.Unlock()
			return
		}
	}
	var revertState *requestRevertState
	var revertKey requestRevision
	switch kind {
	case "summary":
		if selectedWork == nil || selectedWork.Workstream.SummaryFailure == nil || work == nil && (f.teamID != id || f.team == nil || f.team.Project.CoordinatorSessionID != project.CoordinatorSessionID) {
			c.mgr.mu.Unlock()
			return
		}
	case "handover":
		if len(requestCards) == 1 {
			card := requestCards[0]
			selectedCard = &card
		}
		revertState = c.revertStateLocked()
		if selectedCard == nil || selectedCard.ProjectID != id || selectedCard.HandoverError == "" || selectedCard.AuthorityError != "" || revertState == nil {
			c.mgr.mu.Unlock()
			return
		}
		revertKey = requestRevision{selectedCard.ID, selectedCard.Revision}
		cached, known := c.revertCardLocked(revertState, revertKey, client)
		if !known || cached.HandoverError != selectedCard.HandoverError || cached.AuthorityError != "" {
			c.mgr.mu.Unlock()
			return
		}
		selectedCard = &cached
	case "revert":
		if len(requestCards) > 1 {
			c.mgr.mu.Unlock()
			return
		}
		if len(requestCards) == 1 {
			card := requestCards[0]
			selectedCard = &card
		}
		revertState = c.revertStateLocked()
		if selectedCard == nil || selectedCard.ProjectID != id || revertState == nil {
			c.mgr.mu.Unlock()
			return
		}
		revertKey = requestRevision{selectedCard.ID, selectedCard.Revision}
		cached, known := c.revertCardLocked(revertState, revertKey, client)
		if !known || !c.revertAllowedLocked(cached) || !strings.Contains(revertState.errors[revertKey], "pluginrpc: revert_conflict:") {
			c.mgr.mu.Unlock()
			return
		}
		selectedCard = &cached
	default:
		c.mgr.mu.Unlock()
		return
	}
	f.busy = true
	f.err = ""
	f.generation++
	generation := f.generation
	navigation := f.navigation
	detailGeneration := f.detailGeneration
	c.mgr.mu.Unlock()

	selectedSession := sessions.SelectedSessionID()
	go func() {
		ctx, cancel := context.WithTimeout(lc, 30*time.Second)
		defer cancel()
		rpc := projectRPC(client)
		var draft string
		var err error
		if kind == "summary" {
			draft, err = rpc.PrepareSummaryRetry(ctx, project, *selectedWork)
		} else if kind == "revert" {
			draft, err = rpc.PrepareRevertRequest(ctx, *selectedCard)
		} else {
			draft, err = rpc.PrepareHandoverRetry(ctx, project, *selectedCard)
		}
		c.mgr.updateWindow(win, func() {
			nowSnap := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			f := &c.flow
			if nowSnap.Client != client || f.client != client || c.mgr.sessionsStore() != sessions || f.generation != generation {
				c.mgr.mu.Unlock()
				return
			}
			f.busy = false
			if kind == "summary" && (work != nil && (sourceValid == nil || !sourceValid(f, nowSnap.Client)) || work == nil && (f.teamID != id || f.team == nil || f.team.Project.CoordinatorSessionID != project.CoordinatorSessionID || f.detail != selectedWork || f.detailGeneration != detailGeneration)) {
				c.mgr.mu.Unlock()
				return
			}
			if kind == "revert" || kind == "handover" {
				if _, known := c.revertCardLocked(revertState, revertKey, nowSnap.Client); !known {
					c.mgr.mu.Unlock()
					return
				}
			}
			if err != nil {
				f.err = err.Error()
				c.mgr.mu.Unlock()
				return
			}
			if !f.visible() || f.navigation != navigation || sessions.SelectedSessionID() != selectedSession {
				f.err = i18n.T(i18n.KeyCmdNavigationChangedTheRecoveryDraftWasNo)
				c.mgr.mu.Unlock()
				return
			}
			c.mgr.mu.Unlock()

			var submitErr error
			if c.mgr.deps.SubmitLeadDraft != nil {
				submitErr = c.mgr.deps.SubmitLeadDraft(win, sessions, project.CoordinatorSessionID, draft)
			} else {
				submitErr = errors.New(i18n.T(i18n.KeyCmdProjectRecoveryUnavailable))
			}

			afterSnap := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			if afterSnap.Client == client && c.flow.client == client && c.flow.generation == generation {
				if submitErr != nil {
					c.flow.err = submitErr.Error()
				} else {
					notice := i18n.T(i18n.KeyCmdSendingTheRecoveryRequestToTheOriginal)
					if kind == "summary" {
						c.flow.teamNote = notice
					} else {
						c.flow.assetNote = notice
					}
				}
			}
			c.mgr.mu.Unlock()
		})
	}()
}

// SubmitRecoveryDraft stages the verified request text in target without overwriting existing drafts.
func SubmitRecoveryDraft(target RecoveryDraftTarget, text string, submit func()) error {
	if target == nil || submit == nil || strings.TrimSpace(text) == "" {
		return fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectRecoveryUnavailable))
	}
	snap := target.Snapshot()
	if snap.Text != "" || snap.HasAttachments || snap.Sending || snap.Preparing > 0 || snap.EditSeq > 0 || snap.EditSessionID != "" {
		return fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectLeadDraftPending))
	}
	target.StageRecoveryText(text)
	submit()
	return nil
}
