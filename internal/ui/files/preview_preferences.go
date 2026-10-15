package files

import "errors"

// PreviewPreferences is persisted with the existing Files recovery record.
// A missing record defaults to unwrapped, matching the original Files viewer.
type PreviewPreferences struct{ WordWrap bool }

func sourceAreaWidth(wrap bool, viewport, content float32) float32 {
	viewport = max(1, viewport)
	if wrap {
		return viewport
	}
	if content <= 0 {
		return viewport
	}
	return max(viewport, content+16)
}

type previewSnapshot struct {
	Root, Path, Text, Problem string
	Dirty, Image              bool
}

// Snapshot only values already displayed by Files. This must not turn an opaque
// Host cwd into a local filesystem read or save a pending editor draft.
func makePreviewSnapshot(root, path, preview, draft, problem string, image bool) (previewSnapshot, error) {
	if root == "" || path == "" {
		return previewSnapshot{}, errors.New("select a workspace file before opening a floating preview")
	}
	dirty := preview != draft
	if problem != "" && !dirty {
		return previewSnapshot{}, errors.New(problem)
	}
	return previewSnapshot{Root: root, Path: path, Text: draft, Problem: problem, Dirty: dirty, Image: image && !dirty}, nil
}
