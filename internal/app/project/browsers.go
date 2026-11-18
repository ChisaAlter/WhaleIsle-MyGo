package projectctl

import (
	"context"
	"fmt"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/projects"
)

type activityState struct {
	browser *projectflow.ActivityBrowser
	cancel  context.CancelFunc
	owner   *requestRevertState
	request requestRevision
}

type deliveryState struct {
	browser *projectflow.DeliveryBrowser
	cancel  context.CancelFunc
	read    *readOperation
}

type documentState struct {
	browser *projectflow.DocumentBrowser
	cancel  context.CancelFunc
}

type materialCollectionState struct {
	snapshot projectflow.MaterialCollection
	cancel   context.CancelFunc
}

type historicalState struct {
	browser                *projectflow.HistoricalBrowser
	cancel                 context.CancelFunc
	sourceValid            func(*flowState, *dsh.Client) bool
	generation, navigation uint64
}

func (c *Controller) closeActivityLocked() {
	f := &c.flow
	if a := f.activity; a != nil {
		if a.cancel != nil {
			a.cancel()
			a.cancel = nil
			f.busy = false
		}
		a.browser.Close()
		f.activity = nil
	}
}

func (c *Controller) bindActivity(win any, m *projects.Model) {
	snap := c.mgr.hostSnapshot()
	c.mgr.mu.Lock()
	f := &c.flow
	a := f.activity
	if a != nil {
		if _, known := c.revertCardLocked(a.owner, a.request, snap.Client); !known {
			c.closeActivityLocked()
			a = nil
		}
	}
	var view uint64
	if a != nil {
		snapshot := a.browser.Snapshot()
		m.Activity = &snapshot
		view = snapshot.View
	}
	state, card := c.revertStateLocked(), f.requestCard
	c.mgr.mu.Unlock()
	m.OnRequestActivity = func() {
		if card != nil {
			c.openRequestActivity(win, state, requestRevision{card.ID, card.Revision})
		}
	}
	m.OnActivityAction = func(action, arg string) {
		c.runActivity(win, a, view, action, arg, projectflow.DeliveryReportRef{})
	}
	m.OnActivityReport = func(key string, ref projectflow.DeliveryReportRef) {
		c.runActivity(win, a, view, "report", key, ref)
	}
}

func (c *Controller) openRequestActivity(win any, state *requestRevertState, key requestRevision) {
	snap := c.mgr.hostSnapshot()
	client := snap.Client
	c.mgr.mu.Lock()
	f := &c.flow
	card, known := c.revertCardLocked(state, key, client)
	if win == nil || !known || state == nil || client != state.client || f.busy {
		c.mgr.mu.Unlock()
		return
	}
	c.closeActivityLocked()
	c.stopProcessLocked()
	a := &activityState{
		browser: projectflow.NewActivityBrowser(projectflow.ActivityScope{
			Host:       fmt.Sprintf("%p", client),
			ProjectID:  card.ProjectID,
			RequestID:  card.ID,
			Generation: f.generation,
		}),
		owner:   state,
		request: key,
	}
	f.activity = a
	c.mgr.mu.Unlock()
	c.runActivity(win, a, 0, "refresh", "", projectflow.DeliveryReportRef{})
}

func (c *Controller) runActivity(win any, a *activityState, view uint64, action, arg string, ref projectflow.DeliveryReportRef) {
	snap := c.mgr.hostSnapshot()
	client := snap.Client
	lc := c.mgr.lifecycleContext()
	c.mgr.mu.Lock()
	f := &c.flow
	if a == nil || f.activity != a || a.browser.View() != view {
		c.mgr.mu.Unlock()
		return
	}
	if action == "close" {
		c.closeActivityLocked()
		c.mgr.mu.Unlock()
		return
	}
	if action == "back-list" || action == "back-report" {
		if a.cancel != nil {
			a.cancel()
			a.cancel = nil
			f.busy = false
		}
		if action == "back-list" {
			a.browser.BackList()
		} else {
			a.browser.BackReport()
		}
		c.mgr.mu.Unlock()
		return
	}
	snapshot := a.browser.Snapshot()
	_, known := c.revertCardLocked(a.owner, a.request, client)
	if win == nil || !known || snapshot.Scope.Host != fmt.Sprintf("%p", client) || f.busy {
		c.mgr.mu.Unlock()
		return
	}
	switch action {
	case "filter":
		_ = a.browser.Filter(arg)
		c.mgr.mu.Unlock()
		return
	case "next":
		_ = a.browser.MovePage(1)
		c.mgr.mu.Unlock()
		return
	case "previous":
		_ = a.browser.MovePage(-1)
		c.mgr.mu.Unlock()
		return
	case "select":
		_ = a.browser.Select(arg)
		c.mgr.mu.Unlock()
		return
	}
	var req projectflow.ActivityRequest
	var err error
	if action == "refresh" {
		req, err = a.browser.BeginRefresh()
	} else if action == "report" {
		req, err = a.browser.BeginReport(arg, ref)
	} else {
		c.mgr.mu.Unlock()
		return
	}
	if err != nil {
		c.mgr.mu.Unlock()
		return
	}
	ctx, cancel := context.WithTimeout(lc, 30*time.Second)
	a.cancel = cancel
	f.busy = true
	navigation := f.navigation
	c.mgr.mu.Unlock()

	go func() {
		defer cancel()
		rpc := projectRPC(client)
		var card projectflow.RequestActivityCard
		var report projectflow.Report
		var loadErr error
		if action == "refresh" {
			card, loadErr = rpc.RequestActivity(ctx, snapshot.Scope.ProjectID, snapshot.Scope.RequestID)
		} else {
			report, loadErr = rpc.ActivityReport(ctx, req.Event(), req.Reference())
		}
		c.mgr.updateWindow(win, func() {
			nowSnap := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			defer c.mgr.mu.Unlock()
			f := &c.flow
			if f.activity != a || f.client != client || nowSnap.Client != client || !a.browser.Pending(req) {
				return
			}
			if _, known := c.revertCardLocked(a.owner, a.request, nowSnap.Client); !known || f.navigation != navigation {
				c.closeActivityLocked()
				return
			}
			if action == "refresh" {
				a.browser.CompleteRefresh(req, card, loadErr)
			} else {
				a.browser.CompleteReport(req, report, loadErr)
			}
			a.cancel = nil
			f.busy = false
		})
	}()
}

func (c *Controller) closeDeliveriesLocked() {
	f := &c.flow
	if d := f.deliveries; d != nil {
		if d.read != nil {
			c.finishReadLocked(d.read)
			d.read, d.cancel = nil, nil
		} else if d.cancel != nil {
			d.cancel()
			d.cancel = nil
			f.busy = false
		}
		d.browser.Close()
		f.deliveries = nil
	}
}

func (c *Controller) bindDeliveries(win any, m *projects.Model) {
	c.mgr.mu.Lock()
	f := &c.flow
	d := f.deliveries
	expected, generation, navigation := f.client, f.generation, f.navigation
	if d != nil {
		snap := d.browser.Snapshot()
		m.Deliveries = &snap
	}
	c.mgr.mu.Unlock()
	m.OnDeliveries = func(id string) { c.openDeliveries(win, id, expected, generation, navigation) }
	m.OnDeliveryAction = func(action, arg string) {
		c.runDeliveries(win, d, action, arg, projectflow.DeliveryReportRef{})
	}
	m.OnDeliveryReport = func(id string, ref projectflow.DeliveryReportRef) {
		c.runDeliveries(win, d, "report", id, ref)
	}
	m.OnDeliveryFile = func(file projectflow.Artifact, open func(projectflow.FileTarget)) {
		c.openDeliveryFile(win, d, file, open)
	}
}

func (c *Controller) openDeliveryFile(win any, d *deliveryState, file projectflow.Artifact, open func(projectflow.FileTarget)) {
	snap := c.mgr.hostSnapshot()
	lc := c.mgr.lifecycleContext()
	c.mgr.mu.Lock()
	f := &c.flow
	client := snap.Client
	if win == nil || open == nil || d == nil || f.deliveries != d || client == nil || f.client != client || f.busy || !f.visible() {
		c.mgr.mu.Unlock()
		return
	}
	bSnap := d.browser.Snapshot()
	if bSnap.Selected == nil {
		c.mgr.mu.Unlock()
		return
	}
	found := false
	for _, candidate := range bSnap.Selected.Artifacts {
		if candidate == file {
			found = true
			break
		}
	}
	if !found && bSnap.Report != nil && bSnap.ReportRef != nil {
		for _, candidate := range bSnap.Report.Artifacts {
			if candidate == file {
				found = true
				file.WorkstreamID = bSnap.ReportRef.WorkstreamID
				file.DelegationRef = bSnap.ReportRef.DelegationRef
				break
			}
		}
	}
	if !found {
		c.mgr.mu.Unlock()
		return
	}
	token, generation, navigation := d.browser.NavigationToken(), f.generation, f.navigation
	ctx, cancel := context.WithTimeout(lc, 30*time.Second)
	op := c.beginReadLocked(cancel)
	d.read, d.cancel = op, cancel
	c.mgr.mu.Unlock()

	go func() {
		defer cancel()
		rpc := projectRPC(client)
		resolved, err := rpc.ResolveArtifact(ctx, bSnap.Scope.ProjectID, file)
		c.mgr.updateWindow(win, func() {
			nowSnap := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			f := &c.flow
			completed := c.finishReadLocked(op)
			if d.read == op {
				d.read, d.cancel = nil, nil
			}
			if !completed || f.deliveries != d || nowSnap.Client != client || f.client != client || f.generation != generation || f.navigation != navigation || d.browser.NavigationToken() != token {
				c.mgr.mu.Unlock()
				return
			}
			if err != nil {
				f.err = err.Error()
				c.mgr.mu.Unlock()
				return
			}
			if !f.visible() {
				c.mgr.mu.Unlock()
				return
			}
			c.mgr.mu.Unlock()
			open(c.fileTarget(bSnap.Scope.ProjectID, resolved, client))
		})
	}()
}

func (c *Controller) openDeliveries(win any, id string, expected *dsh.Client, generation, navigation uint64) {
	snap := c.mgr.hostSnapshot()
	client := snap.Client
	c.mgr.mu.Lock()
	f := &c.flow
	if win == nil || client == nil || client != expected || f.client != expected || f.generation != generation || f.navigation != navigation || f.busy || !f.visible() || !f.listing.Available {
		c.mgr.mu.Unlock()
		return
	}
	title := ""
	found := false
	for _, p := range f.listing.Projects {
		if p.ID == id {
			title = p.Title
			found = true
			break
		}
	}
	if !found {
		c.mgr.mu.Unlock()
		return
	}
	c.closeDeliveriesLocked()
	c.stopProcessLocked()
	d := &deliveryState{browser: projectflow.NewDeliveryBrowser(projectflow.DeliveryScope{Host: fmt.Sprintf("%p", client), ProjectID: id, Generation: generation}, title)}
	f.deliveries = d
	c.mgr.mu.Unlock()
	c.runDeliveries(win, d, "refresh", "", projectflow.DeliveryReportRef{})
}

func (c *Controller) runDeliveries(win any, d *deliveryState, action, arg string, ref projectflow.DeliveryReportRef) {
	snap := c.mgr.hostSnapshot()
	client := snap.Client
	lc := c.mgr.lifecycleContext()
	c.mgr.mu.Lock()
	f := &c.flow
	if d == nil || f.deliveries != d {
		c.mgr.mu.Unlock()
		return
	}
	if action == "close" {
		c.closeDeliveriesLocked()
		c.mgr.mu.Unlock()
		return
	}
	if action == "back-list" || action == "back-report" {
		if d.read != nil {
			c.finishReadLocked(d.read)
			d.read, d.cancel = nil, nil
		} else if d.cancel != nil {
			d.cancel()
			d.cancel = nil
			f.busy = false
		}
		if action == "back-list" {
			d.browser.BackList()
		} else {
			d.browser.BackReport()
		}
		c.mgr.mu.Unlock()
		return
	}
	if win == nil || client == nil || f.client != client || f.busy || !f.visible() || !f.listing.Available || d.browser.Snapshot().Scope.Host != fmt.Sprintf("%p", client) {
		c.mgr.mu.Unlock()
		return
	}
	if action == "select" {
		_ = d.browser.Select(arg)
		c.mgr.mu.Unlock()
		return
	}
	var req projectflow.DeliveryRequest
	var err error
	if action == "report" {
		req, err = d.browser.BeginReport(arg, ref)
	} else {
		req, err = d.browser.BeginPage(action)
	}
	if err != nil {
		c.mgr.mu.Unlock()
		return
	}
	ctx, cancel := context.WithTimeout(lc, 30*time.Second)
	op := c.beginReadLocked(cancel)
	d.read, d.cancel = op, cancel
	generation, navigation := f.generation, f.navigation
	c.mgr.mu.Unlock()

	go func() {
		defer cancel()
		rpc := projectRPC(client)
		var page projectflow.DeliveryHistoryPage
		var report projectflow.Report
		var loadErr error
		if action == "report" {
			report, loadErr = rpc.DeliveryReport(ctx, req.ProjectID(), req.Delivery(), req.Reference())
		} else {
			page, loadErr = rpc.DeliveryHistory(ctx, req.ProjectID(), req.Cursor())
		}
		c.mgr.updateWindow(win, func() {
			nowSnap := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			defer c.mgr.mu.Unlock()
			f := &c.flow
			completed := c.finishReadLocked(op)
			if d.read == op {
				d.read, d.cancel = nil, nil
			}
			if !completed || f.deliveries != d || f.client != client || nowSnap.Client != client || !d.browser.Pending(req) {
				return
			}
			if f.generation != generation || f.navigation != navigation || !f.visible() {
				c.closeDeliveriesLocked()
				return
			}
			if action == "report" {
				d.browser.CompleteReport(req, report, loadErr)
			} else {
				d.browser.CompletePage(req, page, loadErr)
			}
		})
	}()
}

func (c *Controller) closeDocumentsLocked() {
	f := &c.flow
	if d := f.documents; d != nil {
		if d.cancel != nil {
			d.cancel()
			d.cancel = nil
			f.busy = false
		}
		d.browser.Close()
		f.documents = nil
	}
}

func (c *Controller) bindDocuments(win any, m *projects.Model) {
	c.mgr.mu.Lock()
	d := c.flow.documents
	expectedClient, expectedGeneration, expectedNavigation := c.flow.client, c.flow.generation, c.flow.navigation
	if d != nil {
		snapshot := d.browser.Snapshot()
		m.Documents = &snapshot
	}
	c.mgr.mu.Unlock()
	m.OnDocuments = func(id string) {
		c.openDocuments(win, id, expectedClient, expectedGeneration, expectedNavigation)
	}
	m.OnDocumentAction = func(action, path string) { c.runDocuments(win, d, action, path) }
}

func (c *Controller) openDocuments(win any, id string, expectedClient *dsh.Client, expectedGeneration, expectedNavigation uint64) {
	snap := c.mgr.hostSnapshot()
	client := snap.Client
	c.mgr.mu.Lock()
	f := &c.flow
	if win == nil || client == nil || client != expectedClient || f.client != expectedClient || f.generation != expectedGeneration || f.navigation != expectedNavigation || f.busy || !f.visible() || !f.listing.Available {
		c.mgr.mu.Unlock()
		return
	}
	title := ""
	found := false
	for _, p := range f.listing.Projects {
		if p.ID == id {
			title = p.Title
			found = true
			break
		}
	}
	if !found {
		c.mgr.mu.Unlock()
		return
	}
	c.closeDocumentsLocked()
	c.stopProcessLocked()
	scope := projectflow.DocumentScope{Host: fmt.Sprintf("%p", client), ProjectID: id, Generation: f.generation}
	d := &documentState{browser: projectflow.NewDocumentBrowser(scope, title)}
	f.documents = d
	c.mgr.mu.Unlock()
	c.runDocuments(win, d, "list", "")
}

func (c *Controller) runDocuments(win any, d *documentState, action, documentPath string) {
	snap := c.mgr.hostSnapshot()
	client := snap.Client
	lc := c.mgr.lifecycleContext()
	c.mgr.mu.Lock()
	f := &c.flow
	if d == nil || f.documents != d {
		c.mgr.mu.Unlock()
		return
	}
	if action == "close" {
		c.closeDocumentsLocked()
		c.mgr.mu.Unlock()
		return
	}
	if action == "back" {
		if d.cancel != nil {
			d.cancel()
			d.cancel = nil
			f.busy = false
		}
		d.browser.Back()
		c.mgr.mu.Unlock()
		return
	}
	if win == nil || client == nil || f.client != client || f.busy || !f.visible() || !f.listing.Available {
		c.mgr.mu.Unlock()
		return
	}
	snapshot := d.browser.Snapshot()
	if snapshot.Scope.Host != fmt.Sprintf("%p", client) {
		c.mgr.mu.Unlock()
		return
	}
	var request projectflow.DocumentRequest
	var err error
	switch action {
	case "list":
		request, err = d.browser.BeginList()
	case "read":
		request, err = d.browser.BeginRead(documentPath)
	default:
		c.mgr.mu.Unlock()
		return
	}
	if err != nil {
		c.mgr.mu.Unlock()
		return
	}
	ctx, cancel := context.WithTimeout(lc, 30*time.Second)
	d.cancel = cancel
	f.busy = true
	generation, navigation := f.generation, f.navigation
	c.mgr.mu.Unlock()

	go func() {
		defer cancel()
		rpc := projectRPC(client)
		var docs []projectflow.DocumentEntry
		var content projectflow.DocumentContent
		var loadErr error
		if action == "list" {
			docs, loadErr = rpc.Documents(ctx, request.ProjectID())
		} else {
			content, loadErr = rpc.ReadDocument(ctx, request.ProjectID(), request.Entry())
		}
		c.mgr.updateWindow(win, func() {
			nowSnap := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			defer c.mgr.mu.Unlock()
			f := &c.flow
			if f.documents != d || f.client != client || nowSnap.Client != client || !d.browser.Pending(request) {
				return
			}
			if f.generation != generation || f.navigation != navigation || !f.visible() {
				c.closeDocumentsLocked()
				return
			}
			if action == "list" {
				d.browser.CompleteList(request, docs, loadErr)
			} else {
				d.browser.CompleteRead(request, content, loadErr)
			}
			d.cancel = nil
			f.busy = false
		})
	}()
}

func (c *Controller) closeMaterialCollectionLocked() {
	f := &c.flow
	if d := f.materialCollection; d != nil {
		if d.cancel != nil {
			d.cancel()
			f.busy = false
		}
		f.materialCollection = nil
	}
}

func (c *Controller) bindMaterialCollection(win any, m *projects.Model) {
	c.mgr.mu.Lock()
	d := c.flow.materialCollection
	client, generation, navigation := c.flow.client, c.flow.generation, c.flow.navigation
	if d != nil {
		snapshot := d.snapshot
		m.MaterialCollection = &snapshot
	}
	c.mgr.mu.Unlock()
	m.OnMaterialCollection = func(id string) {
		snap := c.mgr.hostSnapshot()
		c.mgr.mu.Lock()
		f := &c.flow
		if win == nil || client == nil || f.client != client || snap.Client != client || f.generation != generation || f.navigation != navigation || f.busy || !f.visible() || !f.listing.Available {
			c.mgr.mu.Unlock()
			return
		}
		title, found := "", false
		for _, p := range f.listing.Projects {
			if p.ID == id {
				title, found = p.Title, true
				break
			}
		}
		if !found {
			c.mgr.mu.Unlock()
			return
		}
		c.closeMaterialCollectionLocked()
		c.stopProcessLocked()
		next := &materialCollectionState{snapshot: projectflow.MaterialCollection{ProjectID: id, Title: title, Cursors: []string{""}}}
		f.materialCollection = next
		c.mgr.mu.Unlock()
		c.runMaterialCollection(win, next, "refresh")
	}
	m.OnMaterialCollectionAction = func(action string) { c.runMaterialCollection(win, d, action) }
}

func (c *Controller) runMaterialCollection(win any, d *materialCollectionState, action string) {
	snap := c.mgr.hostSnapshot()
	lc := c.mgr.lifecycleContext()
	c.mgr.mu.Lock()
	f := &c.flow
	if d == nil || f.materialCollection != d {
		c.mgr.mu.Unlock()
		return
	}
	if action == "close" {
		c.closeMaterialCollectionLocked()
		c.mgr.mu.Unlock()
		return
	}
	client := snap.Client
	if win == nil || client == nil || f.client != client || f.busy || !f.visible() || !f.listing.Available {
		c.mgr.mu.Unlock()
		return
	}
	cursors := append([]string(nil), d.snapshot.Cursors...)
	switch action {
	case "refresh":
	case "next":
		if d.snapshot.Requests.NextCursor == "" {
			c.mgr.mu.Unlock()
			return
		}
		cursors = append(cursors, d.snapshot.Requests.NextCursor)
	case "previous":
		if len(cursors) < 2 {
			c.mgr.mu.Unlock()
			return
		}
		cursors = cursors[:len(cursors)-1]
	default:
		c.mgr.mu.Unlock()
		return
	}
	id, cursor := d.snapshot.ProjectID, cursors[len(cursors)-1]
	generation, navigation := f.generation, f.navigation
	ctx, cancel := context.WithTimeout(lc, 30*time.Second)
	d.cancel = cancel
	d.snapshot.Error = ""
	f.busy = true
	c.mgr.mu.Unlock()

	go func() {
		defer cancel()
		rpc := projectRPC(client)
		page, err := rpc.MaterialRequestHistory(ctx, id, cursor)
		c.mgr.updateWindow(win, func() {
			nowSnap := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			defer c.mgr.mu.Unlock()
			f := &c.flow
			if f.materialCollection != d || f.client != client || nowSnap.Client != client {
				return
			}
			if f.generation != generation || f.navigation != navigation || !f.visible() {
				c.closeMaterialCollectionLocked()
				return
			}
			d.cancel = nil
			f.busy = false
			if err != nil {
				d.snapshot.Error = err.Error()
				return
			}
			d.snapshot.Requests, d.snapshot.Cursors, d.snapshot.Loaded = page, cursors, true
		})
	}()
}

func (c *Controller) bindMaterials(win any, m *projects.Model) {
	c.mgr.mu.Lock()
	f := &c.flow
	m.Material = f.materials[f.materialKey]
	if m.Material != nil {
		for _, p := range f.listing.Projects {
			if p.ID == m.Material.ProjectID {
				m.MaterialMutable = p.Lifecycle == "ready" && !p.Archiving
			}
		}
	}
	c.mgr.mu.Unlock()
	m.OnMaterial = func(id, path string) { c.runMaterial(win, id, path, "open") }
	m.OnSaveMaterial = func() {
		if m.Material != nil {
			c.runMaterial(win, m.Material.ProjectID, m.Material.Path, "save")
		}
	}
	m.OnReloadMaterial = func() {
		if m.Material != nil {
			c.runMaterial(win, m.Material.ProjectID, m.Material.Path, "reload")
		}
	}
	m.OnDiscardMaterial = func() {
		c.mgr.mu.Lock()
		defer c.mgr.mu.Unlock()
		if d := c.flow.materials[c.flow.materialKey]; d != nil && !c.flow.busy {
			d.Draft = d.Saved
			d.Error = ""
			d.Notice = i18n.T(i18n.KeyCmdProjectDraftDiscarded)
		}
	}
	m.OnCloseMaterial = func() { c.mgr.mu.Lock(); c.flow.materialKey = ""; c.mgr.mu.Unlock() }
}

func (c *Controller) runMaterial(win any, id, path, action string) {
	snap := c.mgr.hostSnapshot()
	lc := c.mgr.lifecycleContext()
	c.mgr.mu.Lock()
	f := &c.flow
	client := snap.Client
	if win == nil || client == nil || f.client != client || f.busy || !f.listing.Available {
		c.mgr.mu.Unlock()
		return
	}
	if path != "charter.md" && path != "notes.md" && path != "preferences.md" {
		c.mgr.mu.Unlock()
		return
	}
	key := id + "\x00" + path
	if f.materials == nil {
		f.materials = map[string]*projectflow.MaterialDraft{}
	}
	d := f.materials[key]
	if d == nil {
		d = &projectflow.MaterialDraft{ProjectID: id, Path: path}
		f.materials[key] = d
	}
	f.materialKey = key
	if action == "open" && d.Loaded {
		c.mgr.mu.Unlock()
		return
	}
	if action == "reload" && d.Dirty() {
		d.Error = i18n.T(i18n.KeyCmdProjectDraftBeforeReload)
		c.mgr.mu.Unlock()
		return
	}
	if action == "save" {
		allowed := false
		for _, p := range f.listing.Projects {
			if p.ID == id {
				allowed = p.Lifecycle == "ready" && !p.Archiving
			}
		}
		if !allowed || !d.Dirty() {
			c.mgr.mu.Unlock()
			return
		}
	}
	draft := d.Draft
	generation := f.generation
	f.busy = true
	d.Error = ""
	d.Notice = ""
	c.mgr.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(lc, 30*time.Second)
		defer cancel()
		rpc := projectRPC(client)
		var material projectflow.Material
		var err error
		if action == "save" {
			material, err = rpc.WriteMaterial(ctx, id, path, draft)
		} else {
			material, err = rpc.ReadMaterial(ctx, id, path)
		}
		c.mgr.updateWindow(win, func() {
			nowSnap := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			defer c.mgr.mu.Unlock()
			f := &c.flow
			if f.client != client || nowSnap.Client != client || f.generation != generation {
				return
			}
			f.busy = false
			d.Apply(material, err)
			if err == nil && action == "save" {
				d.Notice = i18n.T(i18n.KeyCmdProjectMaterialSaved)
			}
		})
	}()
}

func (c *Controller) bindHistory(win any, m *projects.Model) {
	c.mgr.mu.Lock()
	m.Reports = c.flow.reports
	c.mgr.mu.Unlock()
	m.OnReports = func(id string) { c.runHistory(win, id, "list", "", nil) }
	m.OnReportAction = func(action, arg string) {
		if m.Reports != nil {
			c.runHistory(win, m.Reports.ProjectID, action, arg, nil)
		}
	}
	m.OnReportFile = func(path string, open func(string)) {
		if m.Reports != nil {
			c.runHistory(win, m.Reports.ProjectID, "file", path, open)
		}
	}
	m.OnReportFileTarget = func(path string, open func(projectflow.FileTarget)) {
		if m.Reports != nil {
			c.runHistory(win, m.Reports.ProjectID, "file", path, nil, open)
		}
	}
	m.OnCloseReports = func() {
		c.mgr.mu.Lock()
		if !c.flow.busy {
			c.flow.reports = nil
		}
		c.mgr.mu.Unlock()
	}
	m.OnReportWork = func() {
		if m.Reports != nil {
			c.runHistory(win, m.Reports.ProjectID, "inspect", "", nil)
		}
	}
}

func (c *Controller) runHistory(win any, id, action, arg string, open func(string), targets ...func(projectflow.FileTarget)) {
	snap := c.mgr.hostSnapshot()
	sessions := c.mgr.sessionsStore()
	lc := c.mgr.lifecycleContext()

	c.mgr.mu.Lock()
	f := &c.flow
	client := snap.Client
	if win == nil || client == nil || f.client != client || f.busy || !f.listing.Available {
		c.mgr.mu.Unlock()
		return
	}
	original := f.reports
	var next projectflow.ReportBrowser
	if action == "list" {
		next = projectflow.ReportBrowser{ProjectID: id, ListCursors: []string{""}}
	} else {
		if original == nil || original.ProjectID != id {
			c.mgr.mu.Unlock()
			return
		}
		next = *original
		next.ListCursors = append([]string(nil), original.ListCursors...)
		next.WorkCursors = append([]string(nil), original.WorkCursors...)
	}
	var file projectflow.Artifact
	switch action {
	case "back-detail":
		next.Selected = nil
		next.Work = nil
		f.reports = &next
		c.mgr.mu.Unlock()
		return
	case "back-work":
		next.WorkID = ""
		next.Selected = nil
		next.Work = nil
		f.reports = &next
		c.mgr.mu.Unlock()
		return
	case "work":
		found := false
		for _, e := range next.Listing.Results {
			if e.WorkstreamID == arg {
				next.WorkID = arg
				next.Title = e.Title
				found = true
				break
			}
		}
		if !found {
			c.mgr.mu.Unlock()
			return
		}
		next.WorkCursors = []string{""}
		next.Selected = nil
		next.Work = nil
	case "list-next":
		if next.Listing.NextCursor == "" {
			c.mgr.mu.Unlock()
			return
		}
		next.ListCursors = append(next.ListCursors, next.Listing.NextCursor)
	case "list-prev":
		if len(next.ListCursors) < 2 {
			c.mgr.mu.Unlock()
			return
		}
		next.ListCursors = next.ListCursors[:len(next.ListCursors)-1]
	case "work-next":
		if next.History.NextCursor == "" {
			c.mgr.mu.Unlock()
			return
		}
		next.WorkCursors = append(next.WorkCursors, next.History.NextCursor)
	case "work-prev":
		if len(next.WorkCursors) < 2 {
			c.mgr.mu.Unlock()
			return
		}
		next.WorkCursors = next.WorkCursors[:len(next.WorkCursors)-1]
	case "detail":
		found := false
		for _, r := range next.History.Items {
			if r.DelegationRef == arg {
				found = true
				break
			}
		}
		if !found {
			c.mgr.mu.Unlock()
			return
		}
	case "file":
		if next.Selected == nil || open == nil && (len(targets) == 0 || targets[0] == nil) {
			c.mgr.mu.Unlock()
			return
		}
		var err error
		file, err = next.Selected.File(next.WorkID, arg)
		if err != nil {
			c.mgr.mu.Unlock()
			return
		}
	}
	f.busy = true
	f.err = ""
	next.Error = ""
	next.Notice = ""
	generation := f.generation
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
		var err error
		var resolved projectflow.ResolvedArtifact
		var overview projectflow.Overview
		switch action {
		case "list", "list-next", "list-prev":
			next.Listing, err = rpc.Reports(ctx, id, next.ListCursors[len(next.ListCursors)-1])
		case "work", "work-next", "work-prev":
			next.History, err = rpc.ReportHistory(ctx, id, next.WorkID, next.WorkCursors[len(next.WorkCursors)-1], "")
		case "detail":
			var page projectflow.ReportHistory
			page, err = rpc.ReportHistory(ctx, id, next.WorkID, "", arg)
			if err == nil {
				r := page.Items[0]
				if len(r.Request) == 0 {
					r.Request = page.Request
				}
				next.Selected = &r
				var work projectflow.WorkDetail
				work, err = rpc.WorkDetail(ctx, id, next.WorkID, "")
				if err == nil {
					next.Work = &work
				}
			}
		case "inspect":
			if next.WorkID == "" || next.Selected == nil {
				err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectSelectReport))
				break
			}
			overview, err = rpc.TeamOverview(ctx, id)
			if err == nil {
				var work projectflow.WorkDetail
				work, err = rpc.WorkDetail(ctx, id, next.WorkID, "")
				next.Work = &work
			}
		case "file":
			resolved, err = rpc.ResolveArtifact(ctx, id, file)
		default:
			err = fmt.Errorf("unsupported report action")
		}
		c.mgr.updateWindow(win, func() {
			nowSnap := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			f := &c.flow
			if f.client != client || nowSnap.Client != client || f.generation != generation {
				c.mgr.mu.Unlock()
				return
			}
			f.busy = false
			if f.reports != original {
				c.mgr.mu.Unlock()
				return
			}
			if err != nil {
				f.err = err.Error()
				c.mgr.mu.Unlock()
				return
			}
			if action == "inspect" {
				f.reports = nil
				f.teamID = id
				f.team = &overview
				f.detail = next.Work
				f.process = nil
				f.detailGeneration++
				c.mgr.mu.Unlock()
				return
			}
			if action == "file" {
				if !f.visible() || f.navigation != navigation || sessions == nil || sessions.SelectedSessionID() != selected {
					f.err = i18n.T(i18n.KeyCmdNavigationChangedTheFileWasNotOpened)
					c.mgr.mu.Unlock()
					return
				}
				next.Notice = i18n.T(i18n.KeyCmdProjectCurrentFileNotice)
				if resolved.ModifiedSinceReport {
					next.Notice = i18n.T(i18n.KeyCmdProjectFileChangedNotice)
				}
				f.reports = &next
				c.mgr.mu.Unlock()
				if len(targets) > 0 && targets[0] != nil {
					targets[0](c.fileTarget(id, resolved, client))
				} else if open != nil {
					open(resolved.CanonicalPath)
				}
				return
			}
			f.reports = &next
			c.mgr.mu.Unlock()
		})
	}()
}

// Activity belongs to an admitted cached request revision, independently of the
// controller's most recently opened detail card. Caller holds mgr.mu.
func (c *Controller) activityHistoricalSourceValidLocked(f *flowState, activity *activityState, view uint64, currentClient *dsh.Client) bool {
	if activity == nil || f.activity != activity || activity.browser.View() != view {
		return false
	}
	_, known := c.revertCardLocked(activity.owner, activity.request, currentClient)
	return known
}

func (c *Controller) closeHistoricalLocked() {
	f := &c.flow
	if h := f.historical; h != nil {
		if h.cancel != nil {
			h.cancel()
			h.cancel = nil
			f.busy = false
		}
		h.browser.Close()
		f.historical = nil
		f.historicalGeneration++
	}
}

func (c *Controller) bindHistorical(win any, m *projects.Model) {
	snap := c.mgr.hostSnapshot()
	c.mgr.mu.Lock()
	f := &c.flow
	h := f.historical
	if h != nil && (h.generation != f.generation || h.navigation != f.navigation || !h.sourceValid(f, snap.Client)) {
		c.closeHistoricalLocked()
		h = nil
	}
	var view uint64
	if h != nil {
		snapshot := h.browser.Snapshot()
		m.Historical = &snapshot
		view = snapshot.View
	}
	expected, generation, navigation, historicalGeneration := f.client, f.generation, f.navigation, f.historicalGeneration
	reports, delivery, activity := f.reports, f.deliveries, f.activity
	var deliveryToken, activityToken uint64
	if delivery != nil {
		deliveryToken = delivery.browser.NavigationToken()
	}
	if activity != nil {
		activityToken = activity.browser.View()
	}
	c.mgr.mu.Unlock()

	openSource := func(source string, terminal *projectflow.DeliveryTerminalRef) {
		nowSnap := c.mgr.hostSnapshot()
		client := nowSnap.Client
		c.mgr.mu.Lock()
		f := &c.flow
		if win == nil || client == nil || expected != client || f.client != client || f.generation != generation || f.navigation != navigation || f.busy || !f.visible() || !f.listing.Available || f.historical != nil || f.historicalGeneration != historicalGeneration {
			c.mgr.mu.Unlock()
			return
		}
		var target projectflow.HistoricalTarget
		var valid func(*flowState, *dsh.Client) bool
		switch source {
		case "reports":
			if reports == nil || f.reports != reports || reports.Selected == nil {
				c.mgr.mu.Unlock()
				return
			}
			r := *reports.Selected
			target = projectflow.HistoricalTarget{ProjectID: reports.ProjectID, WorkstreamID: reports.WorkID, DelegationRef: r.DelegationRef, ReportID: r.CallID, Report: r}
			valid = func(f *flowState, _ *dsh.Client) bool {
				return f.reports == reports && reports.Selected != nil && reports.Selected.DelegationRef == target.DelegationRef && reports.Selected.CallID == target.ReportID
			}
		case "delivery", "delivery-terminal":
			if delivery == nil || f.deliveries != delivery || delivery.browser.NavigationToken() != deliveryToken {
				c.mgr.mu.Unlock()
				return
			}
			d := delivery.browser.Snapshot()
			if d.Selected == nil || source == "delivery" && (d.Report == nil || d.ReportRef == nil) {
				c.mgr.mu.Unlock()
				return
			}
			if source == "delivery-terminal" {
				found := false
				if terminal != nil {
					for _, ref := range d.Selected.TerminalRefs {
						if ref == *terminal {
							found = true
							break
						}
					}
				}
				if !found {
					c.mgr.mu.Unlock()
					return
				}
				target = projectflow.HistoricalTarget{ProjectID: d.Scope.ProjectID, WorkstreamID: terminal.WorkstreamID, DelegationRef: terminal.DelegationRef, RunID: terminal.RunID, Terminal: true, RequestID: d.Selected.RequestID, RequestRevision: d.Selected.RequestRevision}
			} else {
				target = projectflow.HistoricalTarget{ProjectID: d.Scope.ProjectID, WorkstreamID: d.ReportRef.WorkstreamID, DelegationRef: d.ReportRef.DelegationRef, ReportID: d.Report.CallID, Report: *d.Report}
			}
			for _, ref := range d.Selected.TerminalRefs {
				if ref.WorkstreamID == target.WorkstreamID && ref.DelegationRef == target.DelegationRef {
					if target.RunID != "" && target.RunID != ref.RunID {
						f.err = i18n.T(i18n.KeyCmdProjectDeliveryAmbiguousRuns)
						c.mgr.mu.Unlock()
						return
					}
					target.RunID = ref.RunID
				}
			}
			valid = func(f *flowState, _ *dsh.Client) bool {
				return f.deliveries == delivery && delivery.browser.NavigationToken() == deliveryToken
			}
		case "activity":
			if !c.activityHistoricalSourceValidLocked(f, activity, activityToken, client) {
				c.mgr.mu.Unlock()
				return
			}
			a := activity.browser.Snapshot()
			if a.Selected == nil || a.Report == nil || a.ReportRef == nil {
				c.mgr.mu.Unlock()
				return
			}
			target = projectflow.HistoricalTarget{ProjectID: a.Scope.ProjectID, WorkstreamID: a.ReportRef.WorkstreamID, DelegationRef: a.ReportRef.DelegationRef, ReportID: a.Report.CallID, Report: *a.Report}
			if w := a.Selected.Work; w != nil && w.WorkstreamID == target.WorkstreamID && w.DelegationRef == target.DelegationRef {
				target.RunID = w.RunID
				target.WorkerSessionID = w.WorkerSessionID
			}
			if d := a.Selected.Delivery; d != nil {
				for _, ref := range d.TerminalRefs {
					if ref.WorkstreamID == target.WorkstreamID && ref.DelegationRef == target.DelegationRef {
						if target.RunID != "" && target.RunID != ref.RunID {
							f.err = i18n.T(i18n.KeyCmdProjectEventAmbiguousRuns)
							c.mgr.mu.Unlock()
							return
						}
						target.RunID = ref.RunID
					}
				}
			}
			valid = func(f *flowState, currentClient *dsh.Client) bool {
				return c.activityHistoricalSourceValidLocked(f, activity, activityToken, currentClient)
			}
		default:
			c.mgr.mu.Unlock()
			return
		}
		c.stopProcessLocked()
		h := &historicalState{browser: projectflow.NewHistoricalBrowser(fmt.Sprintf("%p", client), generation, target), sourceValid: valid, generation: generation, navigation: navigation}
		f.historical = h
		f.historicalGeneration++
		c.mgr.mu.Unlock()
		c.runHistorical(win, h, 0, "record")
	}
	m.OnHistoricalSource = func(source string) { openSource(source, nil) }
	m.OnDeliveryTerminal = func(ref projectflow.DeliveryTerminalRef) { openSource("delivery-terminal", &ref) }
	m.OnHistoricalAction = func(action string) { c.runHistorical(win, h, view, action) }
}

func (c *Controller) runHistorical(win any, h *historicalState, view uint64, action string) {
	snap := c.mgr.hostSnapshot()
	client, mux := snap.Client, snap.Mux
	lc := c.mgr.lifecycleContext()
	c.mgr.mu.Lock()
	f := &c.flow
	if h == nil || f.historical != h || h.browser.View() != view {
		c.mgr.mu.Unlock()
		return
	}
	if action == "close" {
		c.closeHistoricalLocked()
		c.mgr.mu.Unlock()
		return
	}
	if client == nil || win == nil || f.client != client || h.browser.Snapshot().Host != fmt.Sprintf("%p", client) || f.generation != h.generation || f.navigation != h.navigation || !h.sourceValid(f, client) || !f.visible() || f.busy || !f.listing.Available {
		c.mgr.mu.Unlock()
		return
	}
	if action == "next" || action == "previous" {
		delta := 1
		if action == "previous" {
			delta = -1
		}
		h.browser.MovePage(delta)
		c.mgr.mu.Unlock()
		return
	}
	snapshot := h.browser.Snapshot()
	req, err := h.browser.Begin(action)
	if err != nil {
		f.err = err.Error()
		c.mgr.mu.Unlock()
		return
	}
	ctx, cancel := context.WithTimeout(lc, 60*time.Second)
	h.cancel = cancel
	f.busy = true
	c.mgr.mu.Unlock()

	go func() {
		defer cancel()
		rpc := projectRPC(client)
		var binding projectflow.HistoricalBinding
		var window projectflow.HistoricalWindow
		var loadErr error
		if action == "record" {
			binding, loadErr = rpc.HistoricalAssignment(ctx, snapshot.Target)
		} else if action == "process" {
			window, loadErr = rpc.ReadHistoricalProcess(ctx, mux, *snapshot.Binding)
		}
		c.mgr.updateWindow(win, func() {
			nowSnap := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			defer c.mgr.mu.Unlock()
			f := &c.flow
			if f.historical != h || f.client != client || nowSnap.Client != client || !h.browser.Pending(req) {
				return
			}
			if f.generation != h.generation || f.navigation != h.navigation || !h.sourceValid(f, nowSnap.Client) || !f.visible() || action == "process" && nowSnap.Mux != mux {
				c.closeHistoricalLocked()
				return
			}
			if action == "record" {
				h.browser.CompleteRecord(req, binding, loadErr)
			} else {
				h.browser.CompleteProcess(req, window, loadErr)
			}
			h.cancel = nil
			f.busy = false
		})
	}()
}
