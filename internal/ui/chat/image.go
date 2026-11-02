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

// chatImageCache keeps the decoded bitmaps of recently drawn local images. It
// used to hold one: two different images in the viewport evicted each other, so
// every frame re-read and re-decoded both files on the UI thread and uploaded a
// new GPU texture for each. Entries are bounded by count and by decoded size.
const (
	chatImageCacheEntries = 32
	chatImageCacheBytes   = 192 << 20
)

type chatImageEntry struct {
	path  string
	bmp   *ui.Bitmap // nil records a file that could not be read or decoded
	bytes int
}

var chatImageCache struct {
	mu      sync.Mutex
	entries []chatImageEntry // most recently used first
	bytes   int
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
	if looksLikeImagePath(path) {
		// A missing or unreadable local file is a broken attachment, not raw
		// placeholder text: paint the Original danger card instead.
		paintImageFailure(c, i18n.T(i18n.KeyChatImageMissing)+": "+filepath.Base(path), nil)
		return
	}
	ui.Text(c, "[image] "+path).FontSize(12).TextColor(fg)
}

// looksLikeImagePath reports whether a ref names an image file by extension.
func looksLikeImagePath(path string) bool {
	switch strings.ToLower(strings.TrimPrefix(filepath.Ext(strings.TrimSpace(path)), ".")) {
	case "png", "jpg", "jpeg", "gif", "webp", "bmp", "avif", "heic", "tif", "tiff":
		return true
	}
	return false
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
		paintImageFailure(c, i18n.T(i18n.KeyChatImageFailed)+": "+state.err.Error(), func() {
			session.ForgetCachedAttachment(id)
			attachmentLoads.retry(id)
			svc.Invalidate()
		})
		return
	}
	if state.path != "" {
		if loadChatBitmap(state.path) == nil {
			paintImageFailure(c, i18n.T(i18n.KeyChatImageInvalid), nil)
			return
		}
		paintInlineImage(c, state.path, muted, onOpen)
		return
	}
	frame := ui.Column(c).Width(240).Height(160).Radius(16).Background(muted.Alpha(0.08)).Center()
	frame.Children(func() {
		ui.Text(c, i18n.T(i18n.KeyChatImageLoading)).FontSize(12).TextColor(muted)
	})
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
		frame := ui.Column(c).Width(240).Height(160).Radius(16).Background(muted.Alpha(0.08)).Center()
		frame.Children(func() {
			ui.Text(c, i18n.T(i18n.KeyChatImageLoading)).FontSize(12).TextColor(muted)
		})
		return
	case imgcache.StatusError:
		paintImageFailure(c, i18n.T(i18n.KeyChatImageFailed)+": "+rawURL, nil)
		return
	}
	svc := c.Services()
	cache.Ensure(rawURL, func() { svc.Invalidate() })
	frame := ui.Column(c).Width(240).Height(160).Radius(16).Background(muted.Alpha(0.08)).Center()
	frame.Children(func() {
		ui.Text(c, i18n.T(i18n.KeyChatImageLoading)).FontSize(12).TextColor(muted)
	})
}

func paintInlineImage(c *ui.Context, local string, muted ui.Color, onOpen func(string)) {
	bmp := loadChatBitmap(local)
	if bmp == nil {
		ui.Text(c, "[image] "+local).FontSize(12).TextColor(muted)
		return
	}
	w, h := inlineImageBounds(bmp)
	ratio := w / h
	card := ui.Column(c).Width(w).MaxWidthPercent(100).AspectRatio(ratio).Radius(16).
		Border(0.5, muted.Alpha(0.28)).Clip()
	card.Children(func() {
		ui.Image(c, bmp).Fill().Fit(ui.Cover)
	})
	if card.Hovered() {
		card.DrawOver(func(p *ui.Painter, r ui.Rect) {
			p.Fill(r, muted.Alpha(0.06), 16)
		})
	}
	if onOpen != nil {
		card.Label("Message image").Focusable().Role(ui.RoleButton)
		if card.Clicked() {
			onOpen(local)
		}
	}
}

// inlineImageBounds implements the Original MessageImage sizing: the longer
// side caps at 240 DIP, the natural aspect ratio stays inside 0.25–4, and the
// short side follows from it.
func inlineImageBounds(bmp *ui.Bitmap) (w, h float32) {
	const (
		longEdge = float32(240)
		minRatio = float32(0.25)
		maxRatio = float32(4)
	)
	width, height := bmp.Size()
	if width <= 0 || height <= 0 {
		return longEdge, longEdge * .75
	}
	ratio := float32(width) / float32(height)
	if ratio < minRatio {
		ratio = minRatio
	}
	if ratio > maxRatio {
		ratio = maxRatio
	}
	if ratio >= 1 {
		return longEdge, longEdge / ratio
	}
	return longEdge * ratio, longEdge
}

// paintImageTile is the 64 DIP multi-image thumbnail the Original uses when a
// message carries more than one picture.
func paintImageTile(c *ui.Context, path string, muted ui.Color, onOpen func(string)) {
	const size = float32(64)
	local := DisplayImagePath(path)
	bmp := (*ui.Bitmap)(nil)
	if local != "" {
		bmp = loadChatBitmap(local)
	}
	tile := ui.Column(c).Width(size).Height(size).Radius(12).Border(0.5, muted.Alpha(0.28)).Clip()
	tile.Children(func() {
		switch {
		case bmp != nil:
			ui.Image(c, bmp).Fill().Fit(ui.Cover)
		case path != "":
			ui.Box(c).Fill().Background(muted.Alpha(0.10)).Center().Children(func() {
				ui.Icon(c, attachmentIconImage).Width(18).Height(18).TextColor(muted)
			})
		}
	})
	if onOpen != nil {
		tile.Label("Message image").Focusable().Role(ui.RoleButton)
		if tile.Clicked() {
			onOpen(local)
		}
	}
}

// paintImageFailure is the danger card the Original shows instead of a broken
// picture: a tinted panel, the reason and an explicit retry.
func paintImageFailure(c *ui.Context, message string, retry func()) {
	ui.Column(c).Width(240).MaxWidthPercent(100).MinWidth(0).Radius(16).Padding(10, 12, 10, 12).Gap(6).
		Background(ui.RGBA(200, 60, 60, .10)).Border(0.5, ui.RGBA(200, 60, 60, .45)).Children(func() {
		ui.Text(c, message).FontSize(12).TextColor(ui.RGB(190, 60, 60))
		if retry != nil {
			if ui.Button(c, i18n.T(i18n.KeyChatImageRetry)).FontSize(11).Clicked() {
				retry()
			}
		}
	})
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
	width, height := c.Size()
	border := chrome.Color(colors.Border)
	if darkSurface(chrome.Color(colors.FG)) {
		border = chrome.Color(colors.BorderL1)
	}
	caption := c.TitleBar().Height
	pw, ph := imageLightboxSize(bmp, width, height-caption)
	open := true
	ui.DialogBase(c.Key("image-lightbox"), &open, func(backdrop, panel ui.Element) {
		progress := chrome.Reveal(c, backdrop, chrome.OverlayDuration)
		mask := ui.RGBA(0, 0, 0, chrome.MaskAlpha(darkSurface(chrome.Color(colors.FG))))
		backdrop.Draw(func(p *ui.Painter, r ui.Rect) {
			p.Fill(ui.Rect{X: r.X, Y: r.Y + caption, W: r.W, H: max(0, r.H-caption)}, mask.Alpha(progress), 0)
		})
		backdrop.Padding(40+caption, 40, 40, 40)
		panel.Label(i18n.T(i18n.KeyToolImagePreview)).Size(pw, ph).Padding(0).Radius(16).
			Background(chrome.Color(colors.InputMajor)).Opacity(progress).
			Shadow(0, 12, 32, 0, ui.RGBA(0, 0, 0, .2)).Children(func() {
			ui.Image(c, bmp).Fill().Fit(ui.Contain).Radius(16).Label(filepath.Base(displayPath))
		})
		// A sibling of the centered image stays anchored to the window, and the
		// modal scope keeps Tab on this sole control and restores the opener.
		backdrop.Children(func() {
			close := ui.ButtonBase(c.Key("close")).Absolute().Top(20+c.TitleBar().Height).Right(20).Size(36, 36).Padding(0).
				Radius(18).Border(.5, border).Background(chrome.Color(colors.InputMajor)).TextColor(chrome.Color(colors.FG)).
				Label(i18n.T(i18n.KeyToolImageClosePreview)).Cursor(ui.CursorPointer).Center().FocusRing(false).Opacity(progress)
			if close.Clicked() {
				open = false
			}
			focused := close.FocusVisible()
			close.DrawOver(func(p *ui.Painter, r ui.Rect) {
				if focused {
					p.Stroke(ui.Rect{X: r.X - 5, Y: r.Y - 5, W: r.W + 10, H: r.H + 10}, chrome.Color(colors.Accent), 23, 2)
				}
			})
			close.Children(func() { ui.Icon(c, messageIconCloseOutline).Size(16, 16).PassThrough() })
		})
	})
	if !open {
		onClose()
	}
}

func imageLightboxSize(bitmap *ui.Bitmap, width, height float32) (float32, float32) {
	bw, bh := bitmap.Size()
	scale := min(float32(1), max(0, min(width-80, 1600))/float32(bw), max(0, height-80)/float32(bh))
	return float32(bw) * scale, float32(bh) * scale
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
		if err != nil || u.Host != "" && !strings.EqualFold(u.Host, "localhost") {
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
	if err != nil && strings.Contains(raw, "%") {
		// Markdown percent escapes are decoded only after trying the literal
		// local filename, so an actual '%20' in a file name keeps its identity.
		if decoded, decodeErr := url.PathUnescape(raw); decodeErr == nil && decoded != raw {
			raw = decoded
			st, err = os.Stat(raw)
		}
	}
	if err != nil || st.IsDir() {
		return ""
	}
	return raw
}

func loadChatBitmap(path string) *ui.Bitmap {
	chatImageCache.mu.Lock()
	defer chatImageCache.mu.Unlock()
	for i, entry := range chatImageCache.entries {
		if entry.path != path {
			continue
		}
		// A failed read keeps its slot too, so a missing file is not statted
		// and opened again every frame; it is retried once it ages out.
		copy(chatImageCache.entries[1:i+1], chatImageCache.entries[:i])
		chatImageCache.entries[0] = entry
		return entry.bmp
	}
	var bmp *ui.Bitmap
	size := 0
	if data, err := os.ReadFile(path); err == nil {
		if decoded, err := ui.DecodeBitmap(data); err == nil {
			bmp = decoded
			w, h := decoded.Size()
			size = 4 * w * h
		}
	}
	storeChatBitmapLocked(chatImageEntry{path: path, bmp: bmp, bytes: size})
	return bmp
}

// storeChatBitmapLocked inserts entry at the front and evicts from the back
// until the cache is within its bounds. The newest entry always stays, even when
// it alone exceeds the byte bound.
func storeChatBitmapLocked(entry chatImageEntry) {
	c := &chatImageCache
	c.entries = append(c.entries, chatImageEntry{})
	copy(c.entries[1:], c.entries)
	c.entries[0] = entry
	c.bytes += entry.bytes
	for len(c.entries) > 1 && (len(c.entries) > chatImageCacheEntries || c.bytes > chatImageCacheBytes) {
		last := len(c.entries) - 1
		c.bytes -= c.entries[last].bytes
		c.entries[last] = chatImageEntry{}
		c.entries = c.entries[:last]
	}
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
