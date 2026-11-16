package projectctl

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/projects"
)

const (
	visualMinWidth = 240
	visualMaxWidth = 1600
)

type visualOwner struct {
	client                 *dsh.Client
	generation, navigation uint64
	projectID              string
	delivery               *deliveryState
	deliveryView           uint64
	activity               *activityState
	activityView           uint64
	source                 string
	page                   projectflow.DeliveryPageReference
}

type visualPage struct {
	state             projects.VisualPagePreview
	ctx               context.Context
	cancel            context.CancelFunc
	captureCancel     context.CancelFunc
	captureToken      uint64
	interactiveCancel context.CancelFunc
	interactiveToken  uint64
	handle            VisualSurface
	html              string
	width             int
	dark              bool
}

type visualResultsState struct {
	pages map[visualOwner]*visualPage
}

func (c *Controller) closeVisualResultsLocked() {
	c.flow.requestCards = nil
	if v := c.flow.visualResults; v != nil {
		for _, page := range v.pages {
			if page.cancel != nil {
				page.cancel()
			}
		}
		c.flow.visualResults = nil
	}
}

// Project data already carries published attachment identities. Resolve only
// an exact page in the controller's authoritative result snapshots;
// an observer's UI cache or the currently selected chat cannot grant a read.
func (c *Controller) visualOwnerLocked(source string, page projectflow.DeliveryPageReference, currentClient *dsh.Client) (visualOwner, bool) {
	f := &c.flow
	o := visualOwner{client: f.client, navigation: f.navigation, source: source, page: page}
	if f.client == nil || !f.visible() || !f.listing.Available {
		return o, false
	}
	var pages []projectflow.DeliveryPageReference
	switch {
	case strings.HasPrefix(source, "request:"):
		requestID := strings.TrimPrefix(source, "request:")
		card, found := f.requestCards[requestCardKey(f.requestProjectID, requestID)]
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
		if _, known := c.revertCardLocked(f.activity.owner, f.activity.request, currentClient); !known {
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

func (c *Controller) visualOwnerCurrentLocked(owner visualOwner, currentClient *dsh.Client) bool {
	current, ok := c.visualOwnerLocked(owner.source, owner.page, currentClient)
	return ok && current == owner
}

func (c *Controller) visualCurrent(owner visualOwner, state *visualResultsState, page *visualPage) bool {
	snap := c.mgr.hostSnapshot()
	c.mgr.mu.Lock()
	defer c.mgr.mu.Unlock()
	return snap.Client == owner.client && c.flow.visualResults == state && state.pages[owner] == page && c.visualOwnerCurrentLocked(owner, snap.Client)
}

func (c *Controller) bindVisualResults(win any, m *projects.Model) {
	snap := c.mgr.hostSnapshot()
	lc := c.mgr.lifecycleContext()
	c.mgr.mu.Lock()
	f := &c.flow
	client, generation, navigation := f.client, f.generation, f.navigation
	if f.visualResults != nil {
		for owner, page := range f.visualResults.pages {
			if !c.visualOwnerCurrentLocked(owner, snap.Client) {
				if page.cancel != nil {
					page.cancel()
				}
				delete(f.visualResults.pages, owner)
			}
		}
	}
	c.mgr.mu.Unlock()

	dark := m.SystemDark
	m.VisualPage = func(source string, reference projectflow.DeliveryPageReference, width int) projects.VisualPagePreview {
		nowSnap := c.mgr.hostSnapshot()
		c.mgr.mu.Lock()
		f := &c.flow
		if win == nil || nowSnap.Client != client || f.client != client || f.navigation != navigation || (strings.HasPrefix(source, "delivery:") && f.generation != generation) {
			c.mgr.mu.Unlock()
			return projects.VisualPagePreview{}
		}
		owner, allowed := c.visualOwnerLocked(source, reference, nowSnap.Client)
		if !allowed {
			c.mgr.mu.Unlock()
			return projects.VisualPagePreview{}
		}
		if width < 1 {
			c.mgr.mu.Unlock()
			return projects.VisualPagePreview{Available: true, Loading: true}
		}
		width = max(visualMinWidth, min(visualMaxWidth, width))
		if f.visualResults == nil {
			f.visualResults = &visualResultsState{pages: map[visualOwner]*visualPage{}}
		}
		state := f.visualResults
		page := state.pages[owner]
		if page == nil {
			ctx, cancel := context.WithCancel(lc)
			page = &visualPage{state: projects.VisualPagePreview{Available: true, Loading: true}, ctx: ctx, cancel: cancel, width: width, dark: dark}
			state.pages[owner] = page
			go c.loadVisualPage(ctx, win, state, owner, page)
		} else if page.width != width || page.dark != dark {
			widthChanged := page.width != width
			page.width, page.dark = width, dark
			if page.html != "" {
				if page.state.InteractiveError != "" {
					c.startVisualCaptureLocked(win, state, owner, page)
				} else if widthChanged && (runtime.GOOS != "windows" || page.handle == nil) {
					// Windows reflows the same child through the clipped native
					// seat. Pixel-backed helpers need a new viewport snapshot.
					c.startVisualInteractiveLocked(win, state, owner, page)
				}
			}
		}
		snapshot := page.state
		snapshot.Retry = func() {
			retireSnap := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			if c.flow.visualResults == state && state.pages[owner] == page && (page.state.Error != "" || page.state.InteractiveError != "") && c.visualOwnerCurrentLocked(owner, retireSnap.Client) {
				if page.cancel != nil {
					page.cancel()
				}
				delete(state.pages, owner)
			}
			c.mgr.mu.Unlock()
			c.mgr.updateWindow(win, func() {})
		}
		c.mgr.mu.Unlock()
		return snapshot
	}
}

func (c *Controller) loadVisualPage(lifetime context.Context, win any, state *visualResultsState, owner visualOwner, page *visualPage) {
	ctx, cancel := context.WithTimeout(lifetime, 30*time.Second)
	defer cancel()
	valid := func() bool { return ctx.Err() == nil && c.visualCurrent(owner, state, page) }
	var err error
	if owner.page.CallID == "" || strings.TrimSpace(owner.page.Title) == "" || owner.page.Height < 80 || owner.page.Height > 2000 {
		err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectPublishedPageInvalid))
	}
	if err == nil {
		if c.mgr.deps.VisualReaderAvailable != nil {
			available, reason := c.mgr.deps.VisualReaderAvailable()
			if !available {
				err = fmt.Errorf(i18n.T(i18n.KeyCmdProjectHTMLReaderUnavailable), reason)
			}
		} else {
			caps := harness.ReadCurrentHostCapabilities(paths.VendorDir(), harness.CurrentHostBridges{})
			if !caps.VisualReplyReader {
				err = fmt.Errorf(i18n.T(i18n.KeyCmdProjectHTMLReaderUnavailable), caps.Reason)
			}
		}
	}
	var html string
	if err == nil {
		file := owner.page.File
		html, err = session.ReadPublishedVisualReply(ctx, owner.client, owner.page.SessionID, session.VisualReplyFile{AttachmentID: file.AttachmentID, Name: file.Name, Bytes: file.Bytes}, valid)
	}
	c.mgr.updateWindow(win, func() {
		nowSnap := c.mgr.hostSnapshot()
		c.mgr.mu.Lock()
		defer c.mgr.mu.Unlock()
		if nowSnap.Client != owner.client || c.flow.visualResults != state || state.pages[owner] != page || !c.visualOwnerCurrentLocked(owner, nowSnap.Client) {
			return
		}
		if err != nil {
			page.state.Loading = false
			page.state.Error = err.Error()
			return
		}
		page.html = html
		c.startVisualInteractiveLocked(win, state, owner, page)
	})
}

func (c *Controller) startVisualCaptureLocked(win any, state *visualResultsState, owner visualOwner, page *visualPage) {
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
		if !c.visualCurrent(owner, state, page) {
			return
		}
		appearance := "light"
		if dark {
			appearance = "dark"
		}
		var capture VisualCaptureResult
		var err error
		if c.mgr.deps.CaptureVisualHTML == nil {
			err = errors.New("visual HTML capture is unavailable")
		} else {
			// Only the same authorized HTML bytes enter the existing isolated
			// renderer. Width changes do not perform another Host read.
			capture, err = c.mgr.deps.CaptureVisualHTML(ctx, html, width, appearance)
		}
		var png []byte
		if err == nil {
			png, err = base64.StdEncoding.DecodeString(capture.PNG)
		}
		if err == nil {
			err = ctx.Err()
		}
		var decoded projects.VisualPagePreview
		if err == nil && c.mgr.deps.DecodeVisualBitmap != nil {
			err = c.mgr.deps.DecodeVisualBitmap(&decoded, png)
		}
		c.mgr.updateWindow(win, func() {
			nowSnap := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			defer c.mgr.mu.Unlock()
			if nowSnap.Client != owner.client || c.flow.visualResults != state || state.pages[owner] != page || page.captureToken != token || !c.visualOwnerCurrentLocked(owner, nowSnap.Client) {
				return
			}
			page.state.Loading = false
			if err != nil {
				page.state.Error = err.Error()
				return
			}
			page.state.Bitmap = decoded.Bitmap
			page.state.ContentHeight, page.state.CapturedHeight = capture.ContentHeight, capture.CapturedHeight
		})
	}()
}

// Project pages use the same authorized bytes, isolated helper and native seat
// as published chat replies. There is no document-to-application capability.
func (c *Controller) startVisualInteractiveLocked(win any, state *visualResultsState, owner visualOwner, page *visualPage) {
	if page.interactiveCancel != nil {
		page.interactiveCancel()
	}
	if page.captureCancel != nil {
		page.captureCancel()
	}
	page.captureToken++
	page.interactiveToken++
	ctx, cancel := context.WithCancel(page.ctx)
	page.interactiveCancel = cancel
	token, width, dark, html := page.interactiveToken, page.width, page.dark, page.html
	page.handle = nil
	page.state.Surface, page.state.Bitmap = nil, nil
	page.state.Loading, page.state.Error, page.state.InteractiveError = true, "", ""
	go func() {
		defer cancel()
		settle := time.NewTimer(120 * time.Millisecond)
		defer settle.Stop()
		select {
		case <-ctx.Done():
			return
		case <-settle.C:
		}
		if !c.visualCurrent(owner, state, page) {
			return
		}
		appearance := "light"
		if dark {
			appearance = "dark"
		}
		var surface VisualSurface
		var err error
		if c.mgr.deps.NewVisualSurface == nil {
			err = errors.New("interactive visual preview surface is unavailable")
		} else {
			surface, err = c.mgr.deps.NewVisualSurface(ctx, win, VisualSurfaceOptions{
				HTML:       html,
				Width:      width,
				Appearance: appearance,
				Inline:     true,
			})
			if err == nil && surface == nil {
				err = errors.New("interactive visual preview returned no surface")
			}
		}
		if err == nil {
			nowSnap := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			installed := ctx.Err() == nil && nowSnap.Client == owner.client && c.flow.visualResults == state && state.pages[owner] == page && page.interactiveToken == token && c.visualOwnerCurrentLocked(owner, nowSnap.Client)
			if installed {
				page.handle, page.state.Surface, page.state.Loading = surface, surface, false
			}
			c.mgr.mu.Unlock()
			if !installed {
				_ = surface.Close()
				return
			}
			// Trigger a redraw once when the interactive surface becomes ready.
			c.mgr.invalidateWindow(win)

			retireCheck := make(chan struct{}, 1)
			var lastHeight atomic.Int64
			lastHeight.Store(int64(surface.ContentHeight()))
			surface.SetHooks(func() {
				select {
				case retireCheck <- struct{}{}:
				default:
				}
			}, func() {
				if h := surface.ContentHeight(); h >= 80 {
					if lastHeight.Swap(int64(h)) != int64(h) {
						c.mgr.invalidateWindow(win)
					}
				}
			})

			// Reuse the seat's paint lease for scrolling/virtualization. The 250ms
			// watch only runs while an active interactive surface is open, and only
			// invalidates when ContentHeight actually changes.
			tick := time.NewTicker(250 * time.Millisecond)
		wait:
			for {
				select {
				case <-ctx.Done():
					err = ctx.Err()
					break wait
				case <-surface.Done():
					err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectPublishedPageClosed))
					break wait
				case <-retireCheck:
					if !c.visualCurrent(owner, state, page) {
						err = fmt.Errorf("published Project page changed")
						break wait
					}
				case <-tick.C:
					if !c.visualCurrent(owner, state, page) {
						err = fmt.Errorf("published Project page changed")
						break wait
					}
					if h := surface.ContentHeight(); h >= 80 {
						prev := lastHeight.Load()
						if int64(h) != prev && lastHeight.CompareAndSwap(prev, int64(h)) {
							c.mgr.invalidateWindow(win)
						}
					}
				}
			}
			tick.Stop()
			if closeErr := surface.Close(); closeErr != nil {
				err = closeErr
			}
		}
		if ctx.Err() != nil {
			return
		}
		c.mgr.updateWindow(win, func() {
			nowSnap := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			defer c.mgr.mu.Unlock()
			if nowSnap.Client != owner.client || c.flow.visualResults != state || state.pages[owner] != page || page.interactiveToken != token || !c.visualOwnerCurrentLocked(owner, nowSnap.Client) {
				return
			}
			page.handle, page.state.Surface = nil, nil
			page.state.InteractiveError = err.Error()
			// This is an explicit fallback: the UI labels the PNG as static,
			// never reports a failed interactive helper as an active surface.
			c.startVisualCaptureLocked(win, state, owner, page)
		})
	}()
}
