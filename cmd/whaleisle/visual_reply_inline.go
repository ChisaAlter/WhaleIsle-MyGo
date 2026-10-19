package main

import (
	"context"
	"fmt"
	"github.com/egoist/mygo"
	"os"
	"runtime"
	"sync"
	"time"
	"whaleisle/internal/desktop/previewprocess"
	"whaleisle/internal/store/session"
)

type inlinePageKey struct {
	store                   *session.Store
	sessionID, attachmentID string
}
type inlinePage struct {
	state     session.InlineVisualReply
	handle    *publishedHTMLSurface
	valid     func() bool
	retryable bool
}

var inlinePages = struct {
	sync.Mutex
	pages map[inlinePageKey]*inlinePage
}{pages: make(map[inlinePageKey]*inlinePage)}

func inlinePublishedHTMLHandler(win *mygo.Window, store *session.Store, sid string, dark bool) func(string) session.InlineVisualReply {
	if (runtime.GOOS != "windows" && runtime.GOOS != "linux" && runtime.GOOS != "darwin") || (runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64") {
		return nil
	}
	return func(id string) session.InlineVisualReply {
		key := inlinePageKey{store, sid, id}
		inlinePages.Lock()
		page := inlinePages.pages[key]
		if page != nil && page.valid != nil && !page.valid() {
			delete(inlinePages.pages, key)
			page = nil
		}
		if page == nil {
			page = &inlinePage{state: session.InlineVisualReply{Loading: true, Height: 360}}
			inlinePages.pages[key] = page
			go loadInlinePublishedHTML(win, key, page, dark)
		}
		state := page.state
		if page.handle != nil {
			state.Height = page.handle.ContentHeight()
			if state.Height < 80 {
				state.Height = 360
			}
		}
		state.ToggleExpanded = func() {
			inlinePages.Lock()
			page.state.Expanded = !page.state.Expanded
			inlinePages.Unlock()
			win.Update(func() {})
		}
		if page.state.Error != "" && page.retryable {
			state.Retry = func() {
				inlinePages.Lock()
				if page.state.Error != "" && page.retryable && inlinePages.pages[key] == page {
					delete(inlinePages.pages, key)
				}
				inlinePages.Unlock()
				win.Update(func() {})
			}
		}
		inlinePages.Unlock()
		return state
	}
}
func loadInlinePublishedHTML(win *mygo.Window, key inlinePageKey, page *inlinePage, dark bool) {
	// The mounted publication owns this page, including its JavaScript state.
	// Navigation/Host replacement and helper exit retire it below; elapsed time
	// must not discard a page that is still mounted and being used.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	readCtx, stop := context.WithTimeout(ctx, 30*time.Second)
	html, valid, err := key.store.ReadVisualReplySnapshot(readCtx, key.sessionID, key.attachmentID)
	stop()
	var surface *publishedHTMLSurface
	if err == nil {
		inlinePages.Lock()
		page.valid = valid
		inlinePages.Unlock()
		exe, e := os.Executable()
		err = e
		appearance := "light"
		if dark {
			appearance = "dark"
		}
		if err == nil {
			var h *previewprocess.Session
			h, err = previewprocess.OpenExperimental(ctx, previewprocess.Options{Enabled: true, HelperPath: previewprocess.HelperPath(exe)}, previewprocess.Document{HTML: html, Width: 728, Appearance: appearance, Inline: true})
			if err == nil {
				surface = newPublishedHTMLSurface(h, win)
			}
		}
	}
	if err == nil && !valid() {
		err = fmt.Errorf("published HTML session changed")
	}
	inlinePages.Lock()
	page.state.Loading = false
	if err != nil {
		page.state.Error = err.Error()
		page.retryable = session.VisualReplyRetryable(err)
	} else {
		page.handle = surface
		page.state.Surface = surface
	}
	inlinePages.Unlock()
	win.Update(func() {})
	if err != nil {
		if surface != nil {
			_ = surface.Close()
		}
		return
	}
	defer surface.Close()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
		case <-surface.Done():
			err = surface.Close()
			if err == nil {
				err = fmt.Errorf("isolated page closed")
			}
		case <-tick.C:
			if valid() {
				continue
			}
			err = fmt.Errorf("published HTML session changed")
		}
		break
	}
	ownerGone := !valid()
	inlinePages.Lock()
	if ownerGone && inlinePages.pages[key] == page {
		delete(inlinePages.pages, key)
	}
	page.state.Surface = nil
	page.handle = nil
	page.state.Error = err.Error()
	page.retryable = session.VisualReplyRetryable(err)
	inlinePages.Unlock()
	win.Update(func() {})
}
