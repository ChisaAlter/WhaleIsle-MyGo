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
	"whaleisle/internal/errcopy"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

type inlinePageKey struct {
	store                   *session.Store
	sessionID, attachmentID string
}
type inlinePage struct {
	state     session.InlineVisualReply
	handle    *publishedHTMLSurface
	valid     func() bool
	retire    func()
	retryable bool
}

var inlinePages = struct {
	sync.Mutex
	pages map[inlinePageKey]*inlinePage
}{pages: make(map[inlinePageKey]*inlinePage)}

// retireStaleInlineVisualPages immediately evicts and tears down inline visual
// pages whose owning session or host generation is no longer current.
func retireStaleInlineVisualPages() {
	var retire []func()
	inlinePages.Lock()
	for key, page := range inlinePages.pages {
		if page.valid != nil && !page.valid() {
			delete(inlinePages.pages, key)
			if page.retire != nil {
				retire = append(retire, page.retire)
			}
		}
	}
	inlinePages.Unlock()
	for _, fn := range retire {
		fn()
	}
}

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
			page = &inlinePage{state: session.InlineVisualReply{Loading: true, Height: session.DefaultVisualReplyHeight}}
			inlinePages.pages[key] = page
			go loadInlinePublishedHTML(win, key, page, dark)
		}
		state := page.state
		if page.handle != nil {
			if measured := page.handle.ContentHeight(); measured >= 80 {
				state.Height = measured
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
	retireCheck := make(chan struct{}, 1)
	poke := func() {
		select {
		case retireCheck <- struct{}{}:
		default:
		}
		cancel()
	}
	inlinePages.Lock()
	if inlinePages.pages[key] == page {
		page.retire = poke
	}
	inlinePages.Unlock()
	readCtx, stop := context.WithTimeout(ctx, 30*time.Second)
	publication, valid, err := key.store.ReadVisualReplyPageSnapshot(readCtx, key.sessionID, key.attachmentID)
	stop()
	var surface *publishedHTMLSurface
	if err == nil {
		inlinePages.Lock()
		page.valid = valid
		page.state.Height = publication.Publication.Height
		inlinePages.Unlock()
		exe, e := os.Executable()
		err = e
		appearance := "light"
		if dark {
			appearance = "dark"
		}
		if err == nil {
			var h *previewprocess.Session
			h, err = previewprocess.OpenExperimental(ctx, previewprocess.Options{Enabled: true, HelperPath: previewprocess.HelperPath(exe)}, previewprocess.Document{HTML: publication.HTML, Width: 728, InitialHeight: publication.Publication.Height, Appearance: appearance, Inline: true})
			if err == nil {
				surface = newPublishedHTMLSurface(h, win)
			}
		}
	}
	if err == nil && !valid() {
		err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdPublishedHTMLSessionChanged))
	}
	inlinePages.Lock()
	page.state.Loading = false
	if err != nil {
		page.state.Error = errcopy.Sentence(err)
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
	surface.SetHooks(func() {
		select {
		case retireCheck <- struct{}{}:
		default:
		}
	}, func() {
		win.Update(func() {})
	})
	// Perform only a brief bounded settle pass right after opening so initial
	// layout height reaches the frame; thereafter remain 0 Hz until surface exit
	// or explicit off-screen/session retirement.
	shown := surface.ContentHeight()
	for _, wait := range []time.Duration{60 * time.Millisecond, 150 * time.Millisecond, 300 * time.Millisecond} {
		settle := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			settle.Stop()
			if !valid() {
				err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdPublishedHTMLSessionChanged))
			} else {
				err = ctx.Err()
			}
			goto done
		case <-surface.Done():
			settle.Stop()
			err = surface.Close()
			if err == nil {
				err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdIsolatedPageClosed))
			}
			goto done
		case <-retireCheck:
			settle.Stop()
			if !valid() {
				err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdPublishedHTMLSessionChanged))
				goto done
			}
		case <-settle.C:
			if !valid() {
				err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdPublishedHTMLSessionChanged))
				goto done
			}
			if h := surface.ContentHeight(); h != shown {
				shown = h
				win.Update(func() {})
			}
		}
	}
	for {
		select {
		case <-ctx.Done():
			if !valid() {
				err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdPublishedHTMLSessionChanged))
			} else {
				err = ctx.Err()
			}
		case <-surface.Done():
			err = surface.Close()
			if err == nil {
				err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdIsolatedPageClosed))
			}
		case <-retireCheck:
			if valid() {
				continue
			}
			err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdPublishedHTMLSessionChanged))
		}
		break
	}
done:
	ownerGone := !valid()
	inlinePages.Lock()
	if ownerGone && inlinePages.pages[key] == page {
		delete(inlinePages.pages, key)
	}
	page.state.Surface = nil
	page.handle = nil
	page.state.Error = errcopy.Sentence(err)
	page.retryable = session.VisualReplyRetryable(err)
	inlinePages.Unlock()
	win.Update(func() {})
}
