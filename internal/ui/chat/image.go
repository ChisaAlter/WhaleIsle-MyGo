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
	"whaleisle/internal/errcopy"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
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

type bitmapLoadFlight struct {
	done        chan struct{}
	invalidates []func()
}

var chatImageCache struct {
	mu       sync.Mutex
	entries  []chatImageEntry // most recently used first
	bytes    int
	inFlight map[string]*bitmapLoadFlight
}

const localImagePathCacheEntries = 256

type localPathEntry struct {
	raw      string
	resolved string // empty records missing, directory, or non-local path
}

type pathLoadFlight struct {
	done        chan struct{}
	invalidates []func()
}

var localImagePathCache struct {
	mu       sync.Mutex
	entries  []localPathEntry
	inFlight map[string]*pathLoadFlight
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
		if local := attachmentImageRead().Cached(id); local != "" {
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

// cachedDisplayImagePath resolves ref for UI rendering without blocking on os.Stat.
func cachedDisplayImagePath(ref string, invalidate func()) (string, bool) {
	ref = strings.TrimSpace(ref)
	if id, ok := strings.CutPrefix(ref, "attachment:"); ok && id != "" {
		return attachmentImageRead().Cached(id), false
	}
	if imgcache.IsHTTPURL(ref) {
		cache := remoteCache()
		if cache == nil {
			return "", false
		}
		st, local, _ := cache.State(ref)
		if st == imgcache.StatusReady && local != "" {
			return local, false
		}
		return "", st == imgcache.StatusFetching
	}
	return cachedLocalImagePath(ref, invalidate)
}

func paintImageLoading(c *ui.Context, muted ui.Color) {
	t := design.For(c)
	frame := ui.Column(c).Width(240).Height(160).Radius(design.RadiusXL2).Background(t.Muted).Center()
	frame.Children(func() {
		ui.Text(c, i18n.T(i18n.KeyChatImageLoading)).FontSize(12).TextColor(t.MutedForeground)
	})
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
	if imgcache.IsHTTPURL(path) {
		paintRemoteImage(c, path, fg, muted, onOpen)
		return
	}
	if _, _, ok := parseLocalImageCandidate(path); ok {
		local, pending := cachedLocalImagePath(path, c.Services().Invalidate)
		if pending {
			paintImageLoading(c, muted)
			return
		}
		if local != "" {
			paintInlineImage(c, local, muted, onOpen)
			return
		}
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
	read := attachmentImageRead()
	svc := c.Services()
	if local := attachmentImageRead().Cached(id); local != "" {
		bmp, pending := cachedChatBitmap(local, svc.Invalidate)
		if bmp != nil {
			paintInlineImageBitmap(c, local, bmp, muted, onOpen)
			return
		}
		if pending {
			paintImageLoading(c, muted)
			return
		}
	}
	fn := read.Fetch
	if fn == nil {
		ui.Text(c, "[image] attachment:"+id).FontSize(12).TextColor(fg)
		return
	}
	key := read.Identity(id)
	state := attachmentLoads.ensure(key, func(ctx context.Context, _ string) (string, error) {
		path, err := fn(ctx, id)
		if err == nil && loadFetchedChatBitmap(path) == nil {
			return "", fmt.Errorf("invalid or unreadable image")
		}
		return path, err
	}, svc.Invalidate)
	if state.err != nil {
		paintImageFailure(c, errcopy.Under(i18n.T(i18n.KeyChatImageFailed), state.err), func() {
			read.Forget(id)
			attachmentLoads.retry(key)
			svc.Invalidate()
		})
		return
	}
	if state.path != "" {
		bmp, pending := cachedChatBitmap(state.path, svc.Invalidate)
		if pending {
			paintImageLoading(c, muted)
			return
		}
		if bmp == nil {
			paintImageFailure(c, i18n.T(i18n.KeyChatImageInvalid), nil)
			return
		}
		paintInlineImageBitmap(c, state.path, bmp, muted, onOpen)
		return
	}
	paintImageLoading(c, muted)
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
		paintImageLoading(c, muted)
		return
	case imgcache.StatusError:
		paintImageFailure(c, i18n.T(i18n.KeyChatImageFailed)+": "+rawURL, nil)
		return
	}
	svc := c.Services()
	cache.Ensure(rawURL, func() { svc.Invalidate() })
	paintImageLoading(c, muted)
}

func paintInlineImage(c *ui.Context, local string, muted ui.Color, onOpen func(string)) {
	bmp, pending := cachedChatBitmap(local, c.Services().Invalidate)
	if pending {
		paintImageLoading(c, muted)
		return
	}
	if bmp == nil {
		ui.Text(c, "[image] "+local).FontSize(12).TextColor(muted)
		return
	}
	paintInlineImageBitmap(c, local, bmp, muted, onOpen)
}

func paintInlineImageBitmap(c *ui.Context, local string, bmp *ui.Bitmap, muted ui.Color, onOpen func(string)) {
	t := design.For(c)
	w, h := inlineImageBounds(bmp)
	ratio := w / h
	card := ui.Column(c).Width(w).MaxWidthPercent(100).AspectRatio(ratio).Radius(design.RadiusXL2).
		Border(0.5, t.Border).Clip()
	card.Children(func() {
		ui.Image(c, bmp).Fill().Fit(ui.Cover)
	})
	if card.Hovered() {
		card.DrawOver(func(p *ui.Painter, r ui.Rect) {
			p.Fill(r, t.MutedForeground.Alpha(0.06), design.RadiusXL2)
		})
	}
	if onOpen != nil {
		card.Label("Message image").Focusable().Role(ui.RoleButton).Cursor(ui.CursorPointer)
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
	svc := c.Services()
	local, pathPending := cachedDisplayImagePath(path, svc.Invalidate)
	read := attachmentImageRead()
	id, isAttachment := strings.CutPrefix(path, "attachment:")
	state := attachmentLoad{}
	if local == "" && !pathPending && isAttachment && read.Fetch != nil {
		state = attachmentLoads.ensure(read.Identity(id), func(ctx context.Context, _ string) (string, error) {
			local, err := read.Fetch(ctx, id)
			if err == nil && loadFetchedChatBitmap(local) == nil {
				return "", fmt.Errorf("invalid or unreadable image")
			}
			return local, err
		}, svc.Invalidate)
		local = state.path
	}
	if local == "" && imgcache.IsHTTPURL(path) {
		if cache := remoteCache(); cache != nil {
			cache.Ensure(path, svc.Invalidate)
		}
	}
	bmp := (*ui.Bitmap)(nil)
	if local != "" {
		bmp, _ = cachedChatBitmap(local, svc.Invalidate)
	}
	t := design.For(c)
	tile := ui.Column(c).Width(size).Height(size).Radius(design.RadiusLG).Border(0.5, t.Border).Clip()
	tile.Children(func() {
		switch {
		case state.err != nil:
			ui.Box(c).Fill().Background(t.Destructive.Alpha(.10)).Center().Children(func() { ui.Text(c, i18n.T(i18n.KeyToolImageFailed)).FontSize(11).TextColor(t.DestructiveForeground) })
		case bmp != nil:
			ui.Image(c, bmp).Fill().Fit(ui.Cover)
		case path != "":
			ui.Box(c).Fill().Background(t.Muted).Center().Children(func() {
				ui.Icon(c, attachmentIconImage).Width(18).Height(18).TextColor(t.MutedForeground)
			})
		}
	})
	if state.err != nil {
		tile.Label(i18n.T(i18n.KeyChatImageRetry)).Tooltip(errcopy.Sentence(state.err)).Focusable().Role(ui.RoleButton).Cursor(ui.CursorPointer)
		if tile.Clicked() {
			read.Forget(id)
			attachmentLoads.retry(read.Identity(id))
			c.Invalidate()
		}
	} else if onOpen != nil {
		tile.Label("Message image").Focusable().Role(ui.RoleButton).Cursor(ui.CursorPointer)
		if tile.Clicked() && local != "" && bmp != nil {
			onOpen(local)
		}
	}
}

// paintImageFailure is the danger card the Original shows instead of a broken
// picture: a tinted panel, the reason and an explicit retry.
func paintImageFailure(c *ui.Context, message string, retry func()) {
	t := design.For(c)
	ui.Column(c).Width(240).MaxWidthPercent(100).MinWidth(0).Radius(design.RadiusXL2).Padding(design.Sp(2.5), design.Sp(3), design.Sp(2.5), design.Sp(3)).Gap(design.Sp(1.5)).
		Background(t.Destructive.Alpha(.10)).Border(0.5, t.Destructive.Alpha(.45)).Children(func() {
		ui.Text(c, message).FontSize(12).TextColor(t.DestructiveForeground)
		if retry != nil {
			if coss.Button(c, i18n.T(i18n.KeyChatImageRetry), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS}).Clicked() {
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
	bmp, _ := cachedChatBitmap(displayPath, c.Services().Invalidate)
	if bmp == nil {
		return
	}
	t := design.For(c)
	width, height := c.Size()
	border := t.Border
	titleBar := c.TitleBar()
	caption := titleBar.Height
	closeRight := max(float32(20), titleBar.Right+12)
	pw, ph := imageLightboxSize(bmp, width, height-caption)
	open := true
	ui.DialogBase(c.Key("image-lightbox"), &open, func(backdrop, panel ui.Element) {
		progress := float32(1)
		if !c.Preferences().ReduceMotion {
			progress = backdrop.AnimateWith("coss.dialog.mask", 1, design.DurationPanel, design.EaseInOut)
		}
		mask := design.TW.Black.Alpha(0.45)
		if t.IsDark() {
			mask = design.TW.Black.Alpha(0.65)
		}
		backdrop.Draw(func(p *ui.Painter, r ui.Rect) {
			p.Fill(ui.Rect{X: r.X, Y: r.Y + caption, W: r.W, H: max(0, r.H-caption)}, mask.Alpha(progress), 0)
		})
		backdrop.Padding(design.Sp(10)+caption, design.Sp(10), design.Sp(10), design.Sp(10))
		chatPopupSurface(c, panel.Label(i18n.T(i18n.KeyToolImagePreview)).Size(pw, ph).Padding(0).
			Background(t.Card).Opacity(progress), design.RadiusXL2).Children(func() {
			ui.Image(c, bmp).Fill().Fit(ui.Contain).Radius(design.RadiusXL2).Label(filepath.Base(displayPath))
		})
		// A sibling of the centered image stays anchored to the window, and the
		// modal scope keeps Tab on this sole control and restores the opener.
		backdrop.Children(func() {
			close := ui.ButtonBase(c.Key("close")).Absolute().Top(20+caption).Right(closeRight).Size(36, 36).Padding(0).
				Radius(design.RadiusFull).Border(.5, border).Background(t.Card).TextColor(t.Foreground).
				Label(i18n.T(i18n.KeyToolImageClosePreview)).Cursor(ui.CursorPointer).Center().FocusRing(false).Opacity(progress)
			if close.Clicked() {
				open = false
			}
			focused := close.FocusVisible()
			close.DrawOver(func(p *ui.Painter, r ui.Rect) {
				if focused {
					p.Stroke(ui.Rect{X: r.X - 5, Y: r.Y - 5, W: r.W + 10, H: r.H + 10}, t.Ring, 23, 2)
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

func parseLocalImageCandidate(raw string) (candidate string, hasPercent bool, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false, false
	}
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "data:image/") || strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return "", false, false
	}
	if strings.HasPrefix(lower, "file:") {
		u, err := url.Parse(raw)
		if err != nil || u.Host != "" && !strings.EqualFold(u.Host, "localhost") {
			return "", false, false
		}
		p := u.Path
		if strings.HasPrefix(p, "/") && len(p) >= 3 && p[2] == ':' {
			// file:///C:/...
			p = p[1:]
		}
		raw = filepath.FromSlash(p)
	}
	return raw, strings.Contains(raw, "%"), true
}

var (
	localImageStatFn    = os.Stat
	chatImageReadFileFn = os.ReadFile
)

func statLocalImagePath(raw string) string {
	candidate, hasPercent, ok := parseLocalImageCandidate(raw)
	if !ok {
		return ""
	}
	st, err := localImageStatFn(candidate)
	if err != nil && hasPercent {
		// Markdown percent escapes are decoded only after trying the literal
		// local filename, so an actual '%20' in a file name keeps its identity.
		if decoded, decodeErr := url.PathUnescape(candidate); decodeErr == nil && decoded != candidate {
			candidate = decoded
			st, err = localImageStatFn(candidate)
		}
	}
	if err != nil || st.IsDir() {
		return ""
	}
	return candidate
}

func storeLocalPathLocked(raw, resolved string) {
	c := &localImagePathCache
	for i, e := range c.entries {
		if e.raw == raw {
			c.entries[i].resolved = resolved
			if i > 0 {
				entry := c.entries[i]
				copy(c.entries[1:i+1], c.entries[:i])
				c.entries[0] = entry
			}
			return
		}
	}
	c.entries = append(c.entries, localPathEntry{})
	copy(c.entries[1:], c.entries)
	c.entries[0] = localPathEntry{raw: raw, resolved: resolved}
	for len(c.entries) > localImagePathCacheEntries {
		last := len(c.entries) - 1
		c.entries[last] = localPathEntry{}
		c.entries = c.entries[:last]
	}
}

func runLocalPathFlight(raw string, flight *pathLoadFlight) string {
	resolved := statLocalImagePath(raw)
	localImagePathCache.mu.Lock()
	storeLocalPathLocked(raw, resolved)
	if localImagePathCache.inFlight[raw] == flight {
		delete(localImagePathCache.inFlight, raw)
	}
	close(flight.done)
	invalidates := flight.invalidates
	flight.invalidates = nil
	localImagePathCache.mu.Unlock()

	for _, fn := range invalidates {
		if fn != nil {
			fn()
		}
	}
	return resolved
}

// cachedLocalImagePath checks the bounded LRU + negative cache and starts a
// background singleflight os.Stat on misses, calling invalidate upon completion.
func cachedLocalImagePath(raw string, invalidate func()) (string, bool) {
	raw = strings.TrimSpace(raw)
	if _, _, ok := parseLocalImageCandidate(raw); !ok {
		return "", false
	}
	localImagePathCache.mu.Lock()
	for i, entry := range localImagePathCache.entries {
		if entry.raw != raw {
			continue
		}
		copy(localImagePathCache.entries[1:i+1], localImagePathCache.entries[:i])
		localImagePathCache.entries[0] = entry
		localImagePathCache.mu.Unlock()
		return entry.resolved, false
	}
	if flight := localImagePathCache.inFlight[raw]; flight != nil {
		if invalidate != nil && len(flight.invalidates) < 16 {
			flight.invalidates = append(flight.invalidates, invalidate)
		}
		localImagePathCache.mu.Unlock()
		return "", true
	}
	if localImagePathCache.inFlight == nil {
		localImagePathCache.inFlight = make(map[string]*pathLoadFlight)
	}
	flight := &pathLoadFlight{done: make(chan struct{})}
	if invalidate != nil {
		flight.invalidates = []func(){invalidate}
	}
	localImagePathCache.inFlight[raw] = flight
	localImagePathCache.mu.Unlock()

	go runLocalPathFlight(raw, flight)
	return "", true
}

func localImagePath(raw string) string {
	raw = strings.TrimSpace(raw)
	if _, _, ok := parseLocalImageCandidate(raw); !ok {
		return ""
	}
	for {
		localImagePathCache.mu.Lock()
		for i, entry := range localImagePathCache.entries {
			if entry.raw != raw {
				continue
			}
			copy(localImagePathCache.entries[1:i+1], localImagePathCache.entries[:i])
			localImagePathCache.entries[0] = entry
			localImagePathCache.mu.Unlock()
			return entry.resolved
		}
		if flight := localImagePathCache.inFlight[raw]; flight != nil {
			done := flight.done
			localImagePathCache.mu.Unlock()
			<-done
			continue
		}
		if localImagePathCache.inFlight == nil {
			localImagePathCache.inFlight = make(map[string]*pathLoadFlight)
		}
		flight := &pathLoadFlight{done: make(chan struct{})}
		localImagePathCache.inFlight[raw] = flight
		localImagePathCache.mu.Unlock()

		return runLocalPathFlight(raw, flight)
	}
}

// A successful canonical read may replace a corrupt file at the same cache
// path. Retire its failed decode before validating the new bytes, otherwise an
// explicit retry can never recover until this bitmap entry happens to age out.
func loadFetchedChatBitmap(path string) *ui.Bitmap {
	chatImageCache.mu.Lock()
	for i, entry := range chatImageCache.entries {
		if entry.path == path && entry.bmp == nil {
			chatImageCache.entries = append(chatImageCache.entries[:i], chatImageCache.entries[i+1:]...)
			break
		}
	}
	chatImageCache.mu.Unlock()
	return loadChatBitmap(path)
}

func decodeChatBitmapFile(path string) (*ui.Bitmap, int) {
	data, err := chatImageReadFileFn(path)
	if err != nil {
		return nil, 0
	}
	decoded, err := ui.DecodeBitmap(data)
	if err != nil {
		return nil, 0
	}
	w, h := decoded.Size()
	return decoded, 4 * w * h
}

func runBitmapLoadFlight(path string, flight *bitmapLoadFlight) *ui.Bitmap {
	bmp, size := decodeChatBitmapFile(path)
	chatImageCache.mu.Lock()
	storeChatBitmapLocked(chatImageEntry{path: path, bmp: bmp, bytes: size})
	if chatImageCache.inFlight[path] == flight {
		delete(chatImageCache.inFlight, path)
	}
	close(flight.done)
	invalidates := flight.invalidates
	flight.invalidates = nil
	chatImageCache.mu.Unlock()

	for _, fn := range invalidates {
		if fn != nil {
			fn()
		}
	}
	return bmp
}

// cachedChatBitmap returns the cached bitmap (or nil on negative cache) without
// performing disk I/O on the UI thread. On cache miss it spawns a singleflight
// background decode and invokes invalidate when ready.
func cachedChatBitmap(path string, invalidate func()) (*ui.Bitmap, bool) {
	if path == "" {
		return nil, false
	}
	chatImageCache.mu.Lock()
	for i, entry := range chatImageCache.entries {
		if entry.path != path {
			continue
		}
		// A failed read keeps its slot too, so a missing file is not statted
		// and opened again every frame; it is retried once it ages out.
		copy(chatImageCache.entries[1:i+1], chatImageCache.entries[:i])
		chatImageCache.entries[0] = entry
		chatImageCache.mu.Unlock()
		return entry.bmp, false
	}
	if flight := chatImageCache.inFlight[path]; flight != nil {
		if invalidate != nil && len(flight.invalidates) < 16 {
			flight.invalidates = append(flight.invalidates, invalidate)
		}
		chatImageCache.mu.Unlock()
		return nil, true
	}
	if chatImageCache.inFlight == nil {
		chatImageCache.inFlight = make(map[string]*bitmapLoadFlight)
	}
	flight := &bitmapLoadFlight{done: make(chan struct{})}
	if invalidate != nil {
		flight.invalidates = []func(){invalidate}
	}
	chatImageCache.inFlight[path] = flight
	chatImageCache.mu.Unlock()

	go runBitmapLoadFlight(path, flight)
	return nil, true
}

func loadChatBitmap(path string) *ui.Bitmap {
	if path == "" {
		return nil
	}
	for {
		chatImageCache.mu.Lock()
		for i, entry := range chatImageCache.entries {
			if entry.path != path {
				continue
			}
			// A failed read keeps its slot too, so a missing file is not statted
			// and opened again every frame; it is retried once it ages out.
			copy(chatImageCache.entries[1:i+1], chatImageCache.entries[:i])
			chatImageCache.entries[0] = entry
			chatImageCache.mu.Unlock()
			return entry.bmp
		}
		if flight := chatImageCache.inFlight[path]; flight != nil {
			done := flight.done
			chatImageCache.mu.Unlock()
			<-done
			continue
		}
		if chatImageCache.inFlight == nil {
			chatImageCache.inFlight = make(map[string]*bitmapLoadFlight)
		}
		flight := &bitmapLoadFlight{done: make(chan struct{})}
		chatImageCache.inFlight[path] = flight
		chatImageCache.mu.Unlock()

		return runBitmapLoadFlight(path, flight)
	}
}

// storeChatBitmapLocked inserts entry at the front and evicts from the back
// until the cache is within its bounds. The newest entry always stays, even when
// it alone exceeds the byte bound.
func storeChatBitmapLocked(entry chatImageEntry) {
	c := &chatImageCache
	for i, existing := range c.entries {
		if existing.path != entry.path {
			continue
		}
		c.bytes -= existing.bytes
		copy(c.entries[1:i+1], c.entries[:i])
		c.entries[0] = entry
		c.bytes += entry.bytes
		for len(c.entries) > 1 && (len(c.entries) > chatImageCacheEntries || c.bytes > chatImageCacheBytes) {
			last := len(c.entries) - 1
			c.bytes -= c.entries[last].bytes
			c.entries[last] = chatImageEntry{}
			c.entries = c.entries[:last]
		}
		return
	}
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
