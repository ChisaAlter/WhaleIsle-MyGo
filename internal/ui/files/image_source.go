package files

import (
	"context"
	"time"
)

// A configured Host reader owns the source even if the same path exists on this
// computer. Its failure is displayed and retried by Reload, never read locally.
func loadHostImagePreview(ctx context.Context, path string, read func(context.Context, string, int64) ([]byte, error)) *imagePreview {
	if err := ctx.Err(); err != nil {
		return &imagePreview{err: err}
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	data, err := read(ctx, path, imageByteLimit(path))
	if err != nil {
		return &imagePreview{err: err}
	}
	return decodeImagePreviewContext(ctx, path, data)
}
