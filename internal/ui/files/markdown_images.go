package files

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/imgcache"
)

var markdownImageCache struct {
	sync.RWMutex
	cache *imgcache.Cache
}

// SetRemoteImageCache shares the application's public-image cache, never Host
// authentication. Per-resource cancellation remains owned by the file tab.
func SetRemoteImageCache(cache *imgcache.Cache) {
	markdownImageCache.Lock()
	markdownImageCache.cache = cache
	markdownImageCache.Unlock()
}

type markdownImageRead struct {
	owner   imageReadOwner
	queue   *imageReadQueue
	preview *imagePreview
}

func (t *FileTab) stopMarkdownImages() {
	if t == nil {
		return
	}
	for _, r := range t.markdownImages {
		r.queue.stop(true)
		if r.preview != nil && r.preview.svg != nil {
			r.preview.svg.Close()
		}
	}
	t.markdownImages = nil
}

func markdownImagePath(source, ref string) (string, error) {
	u, err := url.Parse(ref)
	if err != nil || u.Scheme != "" || u.Host != "" || strings.HasPrefix(u.Path, "/") || strings.Contains(u.Path, `\`) {
		return "", errors.New("Image is not a relative workspace resource")
	}
	rel := path.Clean(path.Join(path.Dir(source), u.Path))
	if rel == ".." || strings.HasPrefix(rel, "../") || strings.Contains(rel, ":") {
		return "", errors.New("Image is outside this workspace")
	}
	return rel, nil
}
func markdownImageLoader(m Model, tab *FileTab, ref string) func(context.Context) *imagePreview {
	source := tab.Path
	read := m.ReadImageContext
	// Local delivery authority is for that saved file only, not an unrelated Host
	// reader or all sibling resources in its containing directory.
	delivery := tab.deliverySource != nil
	root := m.WorkspaceRoot
	markdownImageCache.RLock()
	cache := markdownImageCache.cache
	markdownImageCache.RUnlock()
	return func(ctx context.Context) *imagePreview {
		if strings.HasPrefix(ref, "data:image/") {
			parts := strings.SplitN(ref, ",", 2)
			if len(parts) != 2 || !strings.HasSuffix(parts[0], ";base64") || len(parts[1]) > int(maxImageBytes)*4/3+8 {
				return &imagePreview{err: errors.New("Unsupported or oversized embedded image")}
			}
			data, err := base64.StdEncoding.DecodeString(parts[1])
			if err != nil {
				return &imagePreview{err: err}
			}
			ext := ".png"
			for mime, suffix := range map[string]string{"jpeg": ".jpg", "gif": ".gif", "webp": ".webp", "svg+xml": ".svg", "avif": ".avif", "bmp": ".bmp", "x-icon": ".ico"} {
				if strings.HasPrefix(parts[0], "data:image/"+mime+";") {
					ext = suffix
				}
			}
			return decodeImagePreviewContext(ctx, "embedded"+ext, data)
		}
		if imgcache.IsHTTPURL(ref) {
			u, err := url.Parse(ref)
			if err != nil || u.User != nil || cache == nil {
				return &imagePreview{err: errors.New("Public image cache is unavailable")}
			}
			file, err := cache.Fetch(ctx, ref)
			if err != nil {
				return &imagePreview{err: err}
			}
			f, err := os.Open(file)
			if err != nil {
				return &imagePreview{err: err}
			}
			defer f.Close()
			data, err := io.ReadAll(io.LimitReader(f, maxImageBytes+1))
			if err != nil {
				return &imagePreview{err: err}
			}
			return decodeImagePreviewContext(ctx, u.Path, data)
		}
		if delivery {
			return &imagePreview{err: errors.New("Relative images require a workspace-owned document")}
		}
		rel, err := markdownImagePath(source, ref)
		if err != nil {
			return &imagePreview{err: err}
		}
		if read != nil {
			data, err := read(ctx, rel, imageByteLimit(rel))
			if err != nil {
				return &imagePreview{err: err}
			}
			return decodeImagePreviewContext(ctx, rel, data)
		}
		return loadImagePreviewContext(ctx, root, rel)
	}
}
func paintMarkdownImage(c *ui.Context, m Model, ref, alt string, muted ui.Color) bool {
	tab := selectedTab(m)
	if tab == nil {
		return false
	}
	owner := imageOwner(m, tab)
	if tab.markdownImages == nil {
		tab.markdownImages = make(map[string]*markdownImageRead)
	}
	r := tab.markdownImages[ref]
	if r != nil && r.owner != owner {
		tab.stopMarkdownImages()
		tab.markdownImages = make(map[string]*markdownImageRead)
		r = nil
	}
	if r == nil {
		r = &markdownImageRead{owner: owner, queue: &imageReadQueue{}, preview: &imagePreview{loading: true}}
		r.queue.setNotify(m.notify)
		tab.markdownImages[ref] = r
		r.queue.request(owner, markdownImageLoader(m, tab, ref))
	}
	if r.preview.loading {
		if result, ok := r.queue.take(owner); ok {
			r.preview = result
		}
	}
	preview := r.preview
	if preview.loading {
		ui.Text(c, label("正在加载图片…", "Loading image…")).TextColor(muted)
		c.After(50 * time.Millisecond)
		return true
	}
	if preview.err != nil {
		ui.Text(c, alt+" · "+preview.err.Error()).TextColor(muted).Tooltip(ref)
		return true
	}
	if preview.svg != nil {
		preview.svg.Prepare(c, m.SVGImages)
		w, h := preview.svg.Size()
		if w > 0 {
			ui.Box(c).FillWidth().Height(min(360, h)).DrawOver(func(p *ui.Painter, rect ui.Rect) { preview.svg.Paint(p, rect, ui.Contain) })
		}
		return true
	}
	if wait := preview.advance(time.Now()); wait > 0 {
		c.After(wait)
	}
	if preview.bitmap != nil {
		ui.Image(c, preview.bitmap).FillWidth().MaxHeight(360).Fit(ui.Contain).Tooltip(alt)
		return true
	}
	return false
}
