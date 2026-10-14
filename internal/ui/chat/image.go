package chat

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/imgcache"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

var attachmentFetcher struct {
	mu sync.Mutex
	fn func(ctx context.Context, attachmentID string) (string, error)
}

// SetAttachmentFetcher installs session/attachment resolver for attachment: refs.
func SetAttachmentFetcher(fn func(ctx context.Context, attachmentID string) (string, error)) {
	attachmentFetcher.mu.Lock()
	attachmentFetcher.fn = fn
	attachmentLoads.reset()
	attachmentFetcher.mu.Unlock()
}

var chatImageCache struct {
	mu   sync.Mutex
	path string
	bmp  *ui.Bitmap
}

var remoteImageCache struct {
	mu sync.Mutex
	c  *imgcache.Cache
}

// SetRemoteImageCache installs the shared http(s) image cache (userData/cache/images).
func SetRemoteImageCache(c *imgcache.Cache) {
	remoteImageCache.mu.Lock()
	remoteImageCache.c = c
	remoteImageCache.mu.Unlock()
}

func remoteCache() *imgcache.Cache {
	remoteImageCache.mu.Lock()
	defer remoteImageCache.mu.Unlock()
	return remoteImageCache.c
}

// DisplayImagePath returns a local raster path for ui.Image, or "" when unavailable.
func DisplayImagePath(ref string) string {
	ref = strings.TrimSpace(ref)
	if id, ok := strings.CutPrefix(ref, "attachment:"); ok && id != "" {
		if local := session.CachedAttachmentPath(id); local != "" {
			return local
		}
		return ""
	}
	if local := localImagePath(ref); local != "" {
		return local
	}
	if imgcache.IsHTTPURL(ref) {
		cache := remoteCache()
		if cache == nil {
			return ""
		}
		st, local, _ := cache.State(ref)
		if st == imgcache.StatusReady && local != "" {
			return local
		}
	}
	return ""
}

// ImageInlineView paints a local raster when path is a readable file.
// http(s) URLs use imgcache when configured; otherwise (and for data:) a muted label.
// When onOpen is set and a bitmap is shown, clicking opens the lightbox path.
func ImageInlineView(c *ui.Context, path string, fg, muted ui.Color, onOpen func(displayPath string)) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	if id, ok := strings.CutPrefix(path, "attachment:"); ok && id != "" {
		paintAttachmentImage(c, id, fg, muted, onOpen)
		return
	}
	if local := localImagePath(path); local != "" {
		paintInlineImage(c, local, muted, onOpen)
		return
	}
	if imgcache.IsHTTPURL(path) {
		paintRemoteImage(c, path, fg, muted, onOpen)
		return
	}
	ui.Text(c, "[image] "+path).FontSize(12).TextColor(fg)
}

func paintAttachmentImage(c *ui.Context, id string, fg, muted ui.Color, onOpen func(string)) {
	if local := session.CachedAttachmentPath(id); local != "" && loadChatBitmap(local) != nil {
		paintInlineImage(c, local, muted, onOpen)
		return
	}
	attachmentFetcher.mu.Lock()
	fn := attachmentFetcher.fn
	attachmentFetcher.mu.Unlock()
	if fn == nil {
		ui.Text(c, "[image] attachment:"+id).FontSize(12).TextColor(fg)
		return
	}
	svc := c.Services()
	state := attachmentLoads.ensure(id, func(ctx context.Context, id string) (string, error) {
		path, err := fn(ctx, id)
		if err == nil && loadChatBitmap(path) == nil {
			return "", fmt.Errorf("invalid or unreadable image")
		}
		return path, err
	}, svc.Invalidate)
	if state.err != nil {
		ui.Text(c, "图片加载失败 / Image failed: "+state.err.Error()).FontSize(12).TextColor(fg)
		if ui.Button(c, "重试 / Retry").Clicked() {
			session.ForgetCachedAttachment(id)
			attachmentLoads.retry(id)
			svc.Invalidate()
		}
		return
	}
	if state.path != "" {
		if loadChatBitmap(state.path) == nil {
			ui.Text(c, "图片无法解码 / Invalid image").FontSize(12).TextColor(fg)
			return
		}
		paintInlineImage(c, state.path, muted, onOpen)
		return
	}
	ui.Text(c, "[image…] attachment:"+id).FontSize(12).TextColor(muted)
}

func paintRemoteImage(c *ui.Context, rawURL string, fg, muted ui.Color, onOpen func(string)) {
	cache := remoteCache()
	if cache == nil {
		ui.Text(c, "[image] "+rawURL).FontSize(12).TextColor(fg)
		return
	}
	st, local, _ := cache.State(rawURL)
	switch st {
	case imgcache.StatusReady:
		if local != "" {
			paintInlineImage(c, local, muted, func(p string) {
				if onOpen != nil {
					onOpen(p)
				}
			})
			return
		}
	case imgcache.StatusFetching:
		ui.Text(c, "[image…] "+rawURL).FontSize(12).TextColor(muted)
		return
	case imgcache.StatusError:
		ui.Text(c, "[image] "+rawURL).FontSize(12).TextColor(fg)
		return
	}
	svc := c.Services()
	cache.Ensure(rawURL, func() { svc.Invalidate() })
	ui.Text(c, "[image…] "+rawURL).FontSize(12).TextColor(muted)
}

func paintInlineImage(c *ui.Context, local string, muted ui.Color, onOpen func(string)) {
	if bmp := loadChatBitmap(local); bmp != nil {
		ui.Column(c).Width(360).MaxWidthPercent(100).MinWidth(0).Gap(4).Children(func() {
			row := ui.Row(c).FillWidth().MinWidth(0)
			row.Children(func() {
				ui.Image(c, bmp).FillWidth().Height(220).Fit(ui.Contain)
			})
			if onOpen != nil && row.Clicked() {
				onOpen(local)
			}
			ui.Text(c, filepath.Base(local)).MaxWidthPercent(100).FontSize(11).TextColor(muted)
		})
		return
	}
	ui.Text(c, "[image] "+local).FontSize(12).TextColor(muted)
}

// ImageLightboxOverlay paints a full-window image viewer when displayPath is set.
func ImageLightboxOverlay(c *ui.Context, displayPath, themeID string, systemDark bool, onClose func()) {
	displayPath = strings.TrimSpace(displayPath)
	if displayPath == "" || onClose == nil {
		return
	}
	bmp := loadChatBitmap(displayPath)
	if bmp == nil {
		return
	}
	th := theme.Resolve(themeID)
	colors := th.Active(systemDark)
	muted := chrome.Color(colors.Muted)
	ui.Overlay(c, func() {
		ui.Column(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Children(func() {
			dim := ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Background(ui.RGBA(0, 0, 0, 170))
			if dim.Clicked() {
				onClose()
			}
			ui.Column(c).Fill().Padding(20, 24, 20, 24).Gap(10).Children(func() {
				ui.Image(c, bmp).Width(880).Height(620).Fit(ui.Contain)
				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Text(c, filepath.Base(displayPath)).FontSize(12).TextColor(muted).Grow(1)
					if ui.Button(c, i18n.T(i18n.KeyChatImageClose)).Clicked() {
						onClose()
					}
				})
			})
		})
	})
}

func localImagePath(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "data:image/") || strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return ""
	}
	if strings.HasPrefix(lower, "file:") {
		u, err := url.Parse(raw)
		if err != nil {
			return ""
		}
		p := u.Path
		if strings.HasPrefix(p, "/") && len(p) >= 3 && p[2] == ':' {
			// file:///C:/...
			p = p[1:]
		}
		raw = filepath.FromSlash(p)
	}
	st, err := os.Stat(raw)
	if err != nil || st.IsDir() {
		return ""
	}
	return raw
}

func loadChatBitmap(path string) *ui.Bitmap {
	chatImageCache.mu.Lock()
	defer chatImageCache.mu.Unlock()
	if chatImageCache.path == path && chatImageCache.bmp != nil {
		return chatImageCache.bmp
	}
	data, err := os.ReadFile(path)
	if err != nil {
		chatImageCache.path = path
		chatImageCache.bmp = nil
		return nil
	}
	bmp, err := ui.DecodeBitmap(data)
	if err != nil {
		chatImageCache.path = path
		chatImageCache.bmp = nil
		return nil
	}
	chatImageCache.path = path
	chatImageCache.bmp = bmp
	return bmp
}

// attachmentLoadCache suppresses per-frame fetches and retains a failed state
// until the user explicitly retries. A replacement fetcher invalidates old work.
type attachmentLoad struct {
	path string
	err  error
}
type attachmentLoadCache struct {
	mu      sync.Mutex
	entries map[string]*attachmentLoad
}

var attachmentLoads attachmentLoadCache

func (a *attachmentLoadCache) reset() {
	a.mu.Lock()
	a.entries = make(map[string]*attachmentLoad)
	a.mu.Unlock()
}
func (a *attachmentLoadCache) retry(id string) {
	a.mu.Lock()
	delete(a.entries, id)
	a.mu.Unlock()
}
func (a *attachmentLoadCache) ensure(id string, fetch func(context.Context, string) (string, error), invalidate func()) attachmentLoad {
	a.mu.Lock()
	if state := a.entries[id]; state != nil {
		snapshot := *state
		a.mu.Unlock()
		return snapshot
	}
	if a.entries == nil {
		a.entries = make(map[string]*attachmentLoad)
	}
	pending := &attachmentLoad{}
	a.entries[id] = pending
	a.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		path, err := fetch(ctx, id)
		if err == nil && path == "" {
			err = fmt.Errorf("empty attachment response")
		}
		a.mu.Lock()
		current := a.entries[id] == pending
		if current {
			pending.path, pending.err = path, err
		}
		a.mu.Unlock()
		if current && invalidate != nil {
			invalidate()
		}
	}()
	return attachmentLoad{}
}
