package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/projects"
)

type projectVisualOwner struct {
	client                                    *dsh.Client
	generation, navigation, requestGeneration uint64
	card                                      *projectflow.RequestCard
	delivery                                  *projectDeliveryState
	deliveryView                              uint64
	activity                                  *projectActivityState
	activityView                              uint64
	source                                    string
	page                                      projectflow.DeliveryPageReference
}

type projectVisualPage struct {
	state  projects.VisualPagePreview
	cancel context.CancelFunc
}

type projectVisualResultsState struct {
	pages map[projectVisualOwner]*projectVisualPage
}

func (s *projectController) closeProjectVisualResultsLocked() {
	if v := s.projectFlow.visualResults; v != nil {
		for _, page := range v.pages {
			if page.cancel != nil {
				page.cancel()
			}
		}
		s.projectFlow.visualResults = nil
	}
}

// Project data already carries published attachment identities. Resolve only
// an exact page in the controller's current authoritative result snapshot;
// an observer's UI cache or the currently selected chat cannot grant a read.
func (s *projectController) projectVisualOwnerLocked(source string, page projectflow.DeliveryPageReference) (projectVisualOwner, bool) {
	f := s.projectFlow
	o := projectVisualOwner{client: f.client, generation: f.generation, navigation: f.navigation, requestGeneration: f.requestGeneration, card: f.requestCard, source: source, page: page}
	if f.client == nil || !f.visible() || !f.listing.Available {
		return o, false
	}
	var pages []projectflow.DeliveryPageReference
	switch {
	case strings.HasPrefix(source, "request:"):
		if f.requestCard == nil || source != "request:"+f.requestCard.ID || f.requestProjectID != f.requestCard.ProjectID || f.requestCard.Delivery == nil {
			return o, false
		}
		pages = f.requestCard.Delivery.Pages
	case strings.HasPrefix(source, "delivery:"):
		if f.deliveries == nil {
			return o, false
		}
		d := f.deliveries.browser.Snapshot()
		if d.Selected == nil || d.ReportRef != nil || source != "delivery:"+d.Selected.ID {
			return o, false
		}
		o.delivery, o.deliveryView = f.deliveries, f.deliveries.browser.NavigationToken()
		pages = d.Selected.Pages
	case strings.HasPrefix(source, "activity:"):
		if f.activity == nil {
			return o, false
		}
		a := f.activity.browser.Snapshot()
		if a.Selected == nil || a.ReportRef != nil || a.Selected.Delivery == nil || source != "activity:"+a.Selected.Key {
			return o, false
		}
		o.activity, o.activityView = f.activity, a.View
		pages = a.Selected.Delivery.Pages
	default:
		return o, false
	}
	for _, recorded := range pages {
		if recorded == page {
			return o, true
		}
	}
	return o, false
}

func (s *projectController) projectVisualCurrentLocked(owner projectVisualOwner) bool {
	current, ok := s.projectVisualOwnerLocked(owner.source, owner.page)
	return ok && current == owner
}

func (s *projectController) projectVisualCurrent(owner projectVisualOwner, state *projectVisualResultsState, page *projectVisualPage) bool {
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	return client == owner.client && s.projectFlow.visualResults == state && state.pages[owner] == page && s.projectVisualCurrentLocked(owner)
}

func (s *projectController) bindProjectVisualResults(win *mygo.Window, m *projects.Model) {
	s.mu.Lock()
	f := s.projectFlow
	client, generation, navigation := f.client, f.generation, f.navigation
	if f.visualResults != nil {
		for owner, page := range f.visualResults.pages {
			if !s.projectVisualCurrentLocked(owner) {
				if page.cancel != nil {
					page.cancel()
				}
				delete(f.visualResults.pages, owner)
			}
		}
	}
	s.mu.Unlock()
	dark := mygo.Theme.IsDark()
	m.VisualPage = func(source string, reference projectflow.DeliveryPageReference) projects.VisualPagePreview {
		s.dshMu.Lock()
		current := s.dshClient
		s.dshMu.Unlock()
		s.mu.Lock()
		f := s.projectFlow
		if win == nil || current != client || f.client != client || f.generation != generation || f.navigation != navigation {
			s.mu.Unlock()
			return projects.VisualPagePreview{}
		}
		owner, allowed := s.projectVisualOwnerLocked(source, reference)
		if !allowed {
			s.mu.Unlock()
			return projects.VisualPagePreview{}
		}
		if f.visualResults == nil {
			f.visualResults = &projectVisualResultsState{pages: map[projectVisualOwner]*projectVisualPage{}}
		}
		state := f.visualResults
		page := state.pages[owner]
		if page == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			page = &projectVisualPage{state: projects.VisualPagePreview{Available: true, Loading: true}, cancel: cancel}
			state.pages[owner] = page
			go s.loadProjectVisualPage(ctx, win, state, owner, page, dark)
		}
		snapshot := page.state
		snapshot.Retry = func() {
			s.mu.Lock()
			if s.projectFlow.visualResults == state && state.pages[owner] == page && page.state.Error != "" && s.projectVisualCurrentLocked(owner) {
				if page.cancel != nil {
					page.cancel()
				}
				delete(state.pages, owner)
			}
			s.mu.Unlock()
			win.Update(func() {})
		}
		s.mu.Unlock()
		return snapshot
	}
}

func (s *projectController) loadProjectVisualPage(ctx context.Context, win *mygo.Window, state *projectVisualResultsState, owner projectVisualOwner, page *projectVisualPage, dark bool) {
	defer page.cancel()
	valid := func() bool { return ctx.Err() == nil && s.projectVisualCurrent(owner, state, page) }
	var err error
	if owner.page.CallID == "" || strings.TrimSpace(owner.page.Title) == "" || owner.page.Height < 80 || owner.page.Height > 2000 {
		err = fmt.Errorf("published Project page has an invalid identity or height")
	}
	if err == nil {
		caps := harness.ReadCurrentHostCapabilities(paths.VendorDir(), harness.CurrentHostBridges{})
		if !caps.VisualReplyReader {
			err = fmt.Errorf("published HTML reader is unavailable: %s", caps.Reason)
		}
	}
	var html string
	if err == nil {
		file := owner.page.File
		html, err = session.ReadPublishedVisualReply(ctx, owner.client, owner.page.SessionID, session.VisualReplyFile{AttachmentID: file.AttachmentID, Name: file.Name, Bytes: file.Bytes}, valid)
	}
	var png []byte
	contentHeight, capturedHeight := 0, 0
	if err == nil && valid() {
		appearance := "light"
		if dark {
			appearance = "dark"
		}
		// Reuse the existing isolated saved-page renderer. Only authorized HTML
		// bytes enter the helper; it receives no Host credentials or file paths.
		capture, captureErr := captureHTMLInHelper(ctx, html, 728, appearance)
		err = captureErr
		if err == nil {
			png, err = base64.StdEncoding.DecodeString(capture.PNG)
			contentHeight, capturedHeight = capture.ContentHeight, capture.CapturedHeight
		}
	}
	if err == nil && !valid() {
		err = fmt.Errorf("Project page owner changed while rendering")
	}
	win.Update(func() {
		s.dshMu.Lock()
		current := s.dshClient
		s.dshMu.Unlock()
		s.mu.Lock()
		defer s.mu.Unlock()
		if current != owner.client || s.projectFlow.visualResults != state || state.pages[owner] != page || !s.projectVisualCurrentLocked(owner) {
			return
		}
		page.state.Loading = false
		if err == nil {
			page.state.Bitmap, err = ui.DecodeBitmap(png)
		}
		if err != nil {
			page.state.Error = err.Error()
			return
		}
		page.state.ContentHeight, page.state.CapturedHeight = contentHeight, capturedHeight
	})
}
