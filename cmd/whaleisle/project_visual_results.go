package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/htmlpreview"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/projects"
)

type projectVisualOwner struct {
	client                 *dsh.Client
	generation, navigation uint64
	projectID              string
	delivery               *projectDeliveryState
	deliveryView           uint64
	activity               *projectActivityState
	activityView           uint64
	source                 string
	page                   projectflow.DeliveryPageReference
}

type projectVisualPage struct {
	state             projects.VisualPagePreview
	ctx               context.Context
	cancel            context.CancelFunc
	captureCancel     context.CancelFunc
	captureToken      uint64
	interactiveCancel context.CancelFunc
	interactiveToken  uint64
	handle            *publishedHTMLSurface
	html              string
	width             int
	dark              bool
}

type projectVisualResultsState struct {
	pages map[projectVisualOwner]*projectVisualPage
}

func (s *projectController) closeProjectVisualResultsLocked() {
	s.projectFlow.requestCards = nil
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
// an exact page in the controller's authoritative result snapshots;
// an observer's UI cache or the currently selected chat cannot grant a read.
func (s *projectController) projectVisualOwnerLocked(source string, page projectflow.DeliveryPageReference) (projectVisualOwner, bool) {
	f := s.projectFlow
	o := projectVisualOwner{client: f.client, navigation: f.navigation, source: source, page: page}
	if f.client == nil || !f.visible() || !f.listing.Available {
		return o, false
	}
	var pages []projectflow.DeliveryPageReference
	switch {
	case strings.HasPrefix(source, "request:"):
		requestID := strings.TrimPrefix(source, "request:")
		card, found := f.requestCards[projectRequestCardKey(f.requestProjectID, requestID)]
		if !found || card.ID != requestID || card.ProjectID != f.requestProjectID || card.Delivery == nil {
			return o, false
		}
		projectFound := false
		for _, project := range f.listing.Projects {
			if project.ID == card.ProjectID {
				projectFound = true
				break
			}
		}
		if !projectFound {
			return o, false
		}
		// Reading card B increments the request/global operation generations,
		// but does not revoke card A's immutable publication. Only an actual
		// owner/navigation or recorded page identity change retires its seat.
		o.projectID = card.ProjectID
		pages = card.Delivery.Pages
	case strings.HasPrefix(source, "delivery:"):
		o.generation = f.generation
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
		if _, known := s.requestRevertCardLocked(f.activity.owner, f.activity.request); !known {
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
	m.VisualPage = func(source string, reference projectflow.DeliveryPageReference, width int) projects.VisualPagePreview {
		s.dshMu.Lock()
		current := s.dshClient
		s.dshMu.Unlock()
		s.mu.Lock()
		f := s.projectFlow
		if win == nil || current != client || f.client != client || f.navigation != navigation || (strings.HasPrefix(source, "delivery:") && f.generation != generation) {
			s.mu.Unlock()
			return projects.VisualPagePreview{}
		}
		owner, allowed := s.projectVisualOwnerLocked(source, reference)
		if !allowed {
			s.mu.Unlock()
			return projects.VisualPagePreview{}
		}
		if width < 1 {
			s.mu.Unlock()
			return projects.VisualPagePreview{Available: true, Loading: true}
		}
		width = max(htmlpreview.MinWidth, min(htmlpreview.MaxWidth, width))
		if f.visualResults == nil {
			f.visualResults = &projectVisualResultsState{pages: map[projectVisualOwner]*projectVisualPage{}}
		}
		state := f.visualResults
		page := state.pages[owner]
		if page == nil {
			ctx, cancel := context.WithCancel(context.Background())
			page = &projectVisualPage{state: projects.VisualPagePreview{Available: true, Loading: true}, ctx: ctx, cancel: cancel, width: width, dark: dark}
			state.pages[owner] = page
			go s.loadProjectVisualPage(ctx, win, state, owner, page)
		} else if page.width != width || page.dark != dark {
			widthChanged := page.width != width
			page.width, page.dark = width, dark
			if page.html != "" {
				if page.state.InteractiveError != "" {
					s.startProjectVisualCaptureLocked(win, state, owner, page)
				} else if widthChanged && (runtime.GOOS != "windows" || page.handle == nil) {
					// Windows reflows the same child through the clipped native
					// seat. Pixel-backed helpers need a new viewport snapshot.
					s.startProjectVisualInteractiveLocked(win, state, owner, page)
				}
			}
		}
		snapshot := page.state
		snapshot.Retry = func() {
			s.mu.Lock()
			if s.projectFlow.visualResults == state && state.pages[owner] == page && (page.state.Error != "" || page.state.InteractiveError != "") && s.projectVisualCurrentLocked(owner) {
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

func (s *projectController) loadProjectVisualPage(lifetime context.Context, win *mygo.Window, state *projectVisualResultsState, owner projectVisualOwner, page *projectVisualPage) {
	ctx, cancel := context.WithTimeout(lifetime, 30*time.Second)
	defer cancel()
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
	win.Update(func() {
		s.dshMu.Lock()
		current := s.dshClient
		s.dshMu.Unlock()
		s.mu.Lock()
		defer s.mu.Unlock()
		if current != owner.client || s.projectFlow.visualResults != state || state.pages[owner] != page || !s.projectVisualCurrentLocked(owner) {
			return
		}
		if err != nil {
			page.state.Loading = false
			page.state.Error = err.Error()
			return
		}
		page.html = html
		s.startProjectVisualInteractiveLocked(win, state, owner, page)
	})
}

func (s *projectController) startProjectVisualCaptureLocked(win *mygo.Window, state *projectVisualResultsState, owner projectVisualOwner, page *projectVisualPage) {
	if page.captureCancel != nil {
		page.captureCancel()
	}
	ctx, cancel := context.WithTimeout(page.ctx, 60*time.Second)
	page.captureCancel = cancel
	page.captureToken++
	token, width, dark, html := page.captureToken, page.width, page.dark, page.html
	page.state.Loading, page.state.Error, page.state.Bitmap = true, "", nil
	go func() {
		defer cancel()
		// Layout can change through many DIPs during one resize gesture. Keep
		// only the last requested width before starting the isolated renderer.
		settle := time.NewTimer(120 * time.Millisecond)
		defer settle.Stop()
		select {
		case <-ctx.Done():
			return
		case <-settle.C:
		}
		if !s.projectVisualCurrent(owner, state, page) {
			return
		}
		appearance := "light"
		if dark {
			appearance = "dark"
		}
		// Only the same authorized HTML bytes enter the existing isolated
		// renderer. Width changes do not perform another Host read.
		capture, err := captureHTMLInHelper(ctx, html, width, appearance)
		var png []byte
		if err == nil {
			png, err = base64.StdEncoding.DecodeString(capture.PNG)
		}
		if err == nil {
			err = ctx.Err()
		}
		win.Update(func() {
			s.dshMu.Lock()
			current := s.dshClient
			s.dshMu.Unlock()
			s.mu.Lock()
			defer s.mu.Unlock()
			if current != owner.client || s.projectFlow.visualResults != state || state.pages[owner] != page || page.captureToken != token || !s.projectVisualCurrentLocked(owner) {
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
			page.state.ContentHeight, page.state.CapturedHeight = capture.ContentHeight, capture.CapturedHeight
		})
	}()
}
