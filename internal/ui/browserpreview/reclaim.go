package browserpreview

import (
	"strings"
	"time"

	desktop "whaleisle/internal/desktop/webview"
	"whaleisle/internal/ui/chrome"
)

const defaultBrowserHideTTL = 10 * time.Second

func (b *Controller) effectiveHideTTL() time.Duration {
	if b.hideTTL > 0 {
		return b.hideTTL
	}
	return defaultBrowserHideTTL
}

func (b *Controller) createHost(opts desktop.HostOptions) (*desktop.Host, error) {
	if b.newHostFn != nil {
		return b.newHostFn(opts)
	}
	return desktop.NewHost(opts)
}

func (b *Controller) closeHost(h *desktop.Host) error {
	if h == nil {
		return nil
	}
	if b.closeHostFn != nil {
		return b.closeHostFn(h)
	}
	return h.Close()
}

func (b *Controller) setMiniTab(t *Tab) {
	b.reclaimMu.Lock()
	b.miniTab = t
	b.reclaimMu.Unlock()
}

func (b *Controller) markShown() {
	b.reclaimMu.Lock()
	b.hidden = false
	b.reclaimMu.Unlock()
}

func (b *Controller) captureTabResumeStateLocked(t *Tab) {
	if t == nil || t.Host == nil {
		return
	}
	st := t.Host.State()
	url := strings.TrimSpace(st.URL)
	if url == "" {
		url = strings.TrimSpace(t.Draft)
	}
	if url == "" {
		url = strings.TrimSpace(t.lastURL)
	}
	if url != "" {
		t.resumeURL = url
		t.Draft = url
		t.lastURL = url
	}
	if title := strings.TrimSpace(st.Title); title != "" {
		t.resumeTitle = title
	}
}

func (b *Controller) scheduleTabReclaim(t *Tab) {
	if b == nil || t == nil {
		return
	}
	b.reclaimMu.Lock()
	if b.closed || t.Host == nil || b.miniTab == t {
		b.reclaimMu.Unlock()
		return
	}
	if t.hideTimer != nil {
		t.hideTimer.Stop()
		t.hideTimer = nil
	}
	t.hideGen++
	gen := t.hideGen
	host := t.Host
	ttl := b.effectiveHideTTL()
	t.hideTimer = time.AfterFunc(ttl, func() {
		b.dispatchReclaimHiddenTab(t, host, gen)
	})
	b.reclaimMu.Unlock()
}

func (b *Controller) cancelTabReclaim(t *Tab) {
	if b == nil || t == nil {
		return
	}
	b.reclaimMu.Lock()
	t.hideGen++
	if t.hideTimer != nil {
		t.hideTimer.Stop()
		t.hideTimer = nil
	}
	b.reclaimMu.Unlock()
}

func (b *Controller) stopAllReclaimTimers() {
	b.reclaimMu.Lock()
	b.closed = true
	for _, t := range b.Tabs {
		if t == nil {
			continue
		}
		t.hideGen++
		if t.hideTimer != nil {
			t.hideTimer.Stop()
			t.hideTimer = nil
		}
	}
	b.reclaimMu.Unlock()
}

func (b *Controller) dispatchReclaimHiddenTab(t *Tab, host *desktop.Host, gen uint64) {
	perform := func() {
		b.reclaimHiddenTab(t, host, gen)
	}
	if b.parent != nil && !b.parent.IsDestroyed() {
		b.parent.Update(perform)
		return
	}
	perform()
}

func (b *Controller) reclaimHiddenTab(t *Tab, host *desktop.Host, gen uint64) {
	b.reclaimMu.Lock()
	if b.closed || t == nil || t.hideGen != gen || t.Host != host || host == nil || b.miniTab == t {
		b.reclaimMu.Unlock()
		return
	}
	t.hideTimer = nil
	b.captureTabResumeStateLocked(t)
	t.Host = nil
	t.reclaimed = true
	b.reclaimMu.Unlock()

	_ = b.closeHost(host)
}

func (b *Controller) resumeReclaimedTab(t *Tab) {
	if t == nil {
		return
	}
	b.reclaimMu.Lock()
	shouldResume := t.reclaimed && t.Host == nil && t.resumeURL != "" && t.source == nil && t.Document == nil && t.HTML == nil && t.Media == nil
	targetURL := t.resumeURL
	if shouldResume {
		t.reclaimed = false
	}
	b.reclaimMu.Unlock()
	if !shouldResume {
		return
	}

	h, err := b.createHost(desktop.HostOptions{
		Parent:           b.parent,
		URL:              targetURL,
		Title:            "Browser",
		BrowserProfile:   true,
		BrowserPartition: t.workspace.key,
		TopInset:         chrome.TitleBarHeight + BrowserChromeHeight,
		Untrusted:        true,
		OnChange:         b.update,
		OnFocus:          b.focus,
		OnFailure:        b.pageFailure(t),
	})
	if err != nil {
		t.Error = err.Error()
		b.update()
		return
	}
	b.reclaimMu.Lock()
	t.Host = h
	t.resumeURL = ""
	t.Error = ""
	t.ActionMessage = ""
	b.reclaimMu.Unlock()
	b.showHost(h)
	b.update()
}

func (b *Controller) tabHostSnapshot(t *Tab) (host *desktop.Host, reclaimed bool) {
	b.reclaimMu.Lock()
	if t != nil {
		host = t.Host
		reclaimed = t.reclaimed
	}
	b.reclaimMu.Unlock()
	return host, reclaimed
}
