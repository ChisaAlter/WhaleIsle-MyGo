package files

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
}
