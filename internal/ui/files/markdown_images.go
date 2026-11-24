package files

import (
	"context"
	"errors"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/imgcache"
	"whaleisle/internal/desktop/media"
	"whaleisle/internal/errcopy"
)

type markdownImageRead struct {
	owner   imageReadOwner
	preview *imagePreview
}

func (t *FileTab) stopMarkdownImages() {
	if t == nil {
		return
	}
	for _, r := range t.markdownImages {
		r.preview.release()
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

// markdownImagePreview prepares one rendered-Markdown image request: embedded
// data and public URLs are keyed by content, relative resources by the tab's
// authority. Every read and decode runs on a media worker.
func markdownImagePreview(m Model, tab *FileTab, owner imageReadOwner, ref string) *imagePreview {
	if strings.HasPrefix(ref, "data:image/") {
		head, payload, ok := strings.Cut(ref, ",")
		if !ok || !strings.HasSuffix(head, ";base64") || len(payload) > int(maxImageBytes)*4/3+8 {
			return &imagePreview{err: errors.New("Unsupported or oversized embedded image")}
		}
		p := &imagePreview{owner: owner, loading: true, lease: filesMedia().NewOwner(nil, m.notify)}
		p.key, p.src = media.DataURL(ref)
		return p
	}
	if imgcache.IsHTTPURL(ref) {
		u, err := url.Parse(ref)
		if err != nil || u.User != nil {
			return &imagePreview{err: errors.New("Public image cache is unavailable")}
		}
		p := &imagePreview{owner: owner, loading: true, lease: filesMedia().NewOwner(nil, m.notify)}
		p.key, p.src = media.Remote(ref)
		p.src.Kind = strings.ToLower(path.Ext(u.Path))
		return p
	}
	source := tab.Path
	read := m.ReadImageContext
	// Local delivery authority is for that saved file only, not an unrelated Host
	// reader or all sibling resources in its containing directory.
	delivery := tab.deliverySource != nil
	root := m.WorkspaceRoot
	rel, relErr := markdownImagePath(source, ref)
	load := func(ctx context.Context) ([]byte, error) {
		if delivery {
			return nil, errors.New("Relative images require a workspace-owned document")
		}
		if relErr != nil {
			return nil, relErr
		}
		if read != nil {
			return read(ctx, rel, imageByteLimit(rel))
		}
		return readWorkspaceImage(ctx, root, rel)
	}
	return newImagePreview(owner, "files-markdown\x00"+mediaID(owner, nil)+"\x00"+ref, rel, m.notify, load)
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
		r = &markdownImageRead{owner: owner, preview: markdownImagePreview(m, tab, owner, ref)}
		tab.markdownImages[ref] = r
	}
	preview := r.preview
	wait := preview.sync(time.Now())
	if preview.loading {
		ui.Text(c, label("正在加载图片…", "Loading image…")).TextColor(muted)
		return true
	}
	if preview.err != nil {
		ui.Text(c, alt+" · "+errcopy.Sentence(preview.err)).TextColor(muted).Tooltip(ref)
		return true
	}
	if wait > 0 {
		c.After(wait)
	}
	if preview.bitmap != nil {
		ui.Image(c, preview.bitmap).FillWidth().MaxHeight(360).Fit(ui.Contain).Tooltip(alt)
		return true
	}
	if preview.vector != nil {
		ui.Image(c, preview.vector).FillWidth().MaxHeight(360).Fit(ui.Contain).Tooltip(alt)
		return true
	}
	return false
}
