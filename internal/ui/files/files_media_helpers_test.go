package files

import (
	"context"
	"time"

	"whaleisle/internal/desktop/media"
)

// awaitPreview waits for a preview's media request to settle without
// starting its animation clock.
func awaitPreview(p *imagePreview) *imagePreview {
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		h := filesMedia().Request(p.lease, p.key, p.src)
		if h.State() != media.Pending {
			p.handle = h
			if h.State() == media.Failed {
				if e := p.loadErr.Load(); e != nil {
					p.err = *e
				} else {
					p.err = h.Err()
				}
				p.loading = false
				return p
			}
			pic, _ := h.Picture()
			p.loading, p.bitmap, p.vector = false, pic.Bitmap, pic.SVG
			return p
		}
		time.Sleep(2 * time.Millisecond)
	}
	return p
}

var testPreviewSeq int

// loadImagePreview reads and decodes one workspace image through the media
// service, as a tab would, and waits for the result.
func loadImagePreview(root, rel string) *imagePreview {
	testPreviewSeq++
	id := "test\x00" + root + "\x00" + rel + "\x00" + time.Now().String()
	return awaitPreview(newImagePreview(imageReadOwner{root: root, path: rel}, id, rel, nil, func(ctx context.Context) ([]byte, error) {
		return readWorkspaceImage(ctx, root, rel)
	}))
}

// decodeImagePreviewContext decodes supplied bytes as path through the media
// service; a cancelled ctx fails before any decode.
func decodeImagePreviewContext(ctx context.Context, path string, data []byte) *imagePreview {
	testPreviewSeq++
	id := "test-bytes\x00" + path + "\x00" + time.Now().String()
	return awaitPreview(newImagePreview(imageReadOwner{path: path}, id, path, nil, func(context.Context) ([]byte, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return data, nil
	}))
}

// advance shows the frame due at now and returns the wait to the next.
func (p *imagePreview) advance(now time.Time) time.Duration { return p.sync(now) }
