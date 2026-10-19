package files

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func imageSourcePNG(t *testing.T, width, height int) []byte {
	t.Helper()
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewNRGBA(image.Rect(0, 0, width, height))); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func awaitImageSource(t *testing.T, m Model, tab *FileTab, notified <-chan struct{}) *imagePreview {
	t.Helper()
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	for {
		result := readImagePreview(m, tab)
		if !result.loading {
			return result
		}
		select {
		case <-notified:
		case <-deadline.C:
			t.Fatal("image source did not settle")
		}
	}
}

func imageSourceModel(root string) (Model, *FileTab, chan struct{}) {
	notified := make(chan struct{}, 4)
	tab := &FileTab{Path: "same.png"}
	m := Model{Tabs: &Tabs{}, WorkspaceRoot: root, BindingID: "host-a", SessionID: "session-a", ImageGeneration: 1,
		notify: func() {
			select {
			case notified <- struct{}{}:
			default:
			}
		}}
	tabsFor(m).Open = []*FileTab{tab}
	return m, tab, notified
}

func TestHostImageNeverReadsSameNamedLocalFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "same.png"), imageSourcePNG(t, 1, 1), 0600); err != nil {
		t.Fatal(err)
	}
	m, tab, notified := imageSourceModel(root)
	t.Cleanup(func() { m.Tabs.stopImageRead(true) })
	data := imageSourcePNG(t, 3, 2)
	m.ReadImageContext = func(_ context.Context, path string, limit int64) ([]byte, error) {
		if path != "same.png" || limit != maxImageBytes {
			t.Errorf("wrong image request: %q %d", path, limit)
		}
		return data, nil
	}
	result := awaitImageSource(t, m, tab, notified)
	if result.err != nil || result.note != "3 × 2" {
		t.Fatalf("used local bytes instead of Host snapshot: %+v", result)
	}
	denied := errors.New("Host denied this image")
	m.ReadImageContext = func(context.Context, string, int64) ([]byte, error) { return nil, denied }
	tab.image = nil // Explicit Reload also clears the previous decoded pixels.
	result = awaitImageSource(t, m, tab, notified)
	if !errors.Is(result.err, denied) || result.bitmap != nil {
		t.Fatalf("Host failure fell back to local pixels: %+v", result)
	}
}

func TestHostImageRejectsInvalidOversizedAndCancelledSnapshots(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		data       []byte
		limit      int64
	}{
		{"invalid-png", "a.png", []byte("not a PNG"), maxImageBytes},
		{"oversized-svg", "a.svg", make([]byte, (1<<20)+1), 1 << 20},
		{"oversized-png", "a.png", make([]byte, maxImageBytes+1), maxImageBytes},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := loadHostImagePreview(context.Background(), tc.path, func(_ context.Context, _ string, limit int64) ([]byte, error) {
				if limit != tc.limit {
					t.Errorf("limit = %d", limit)
				}
				return tc.data, nil
			})
			if result.err == nil || result.bitmap != nil {
				t.Fatal("invalid Host snapshot displayed")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := loadHostImagePreview(ctx, "a.png", func(context.Context, string, int64) ([]byte, error) {
		t.Error("cancelled source invoked Host")
		return nil, nil
	})
	if !errors.Is(result.err, context.Canceled) {
		t.Fatal(result.err)
	}
}

func TestHostImageReplacesCachedAndInFlightOwnership(t *testing.T) {
	m, tab, notified := imageSourceModel("unused-host-root")
	t.Cleanup(func() { m.Tabs.stopImageRead(true) })
	started, release := make(chan struct{}), make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	old := imageSourcePNG(t, 1, 1)
	m.ReadImageContext = func(context.Context, string, int64) ([]byte, error) { close(started); <-release; return old, nil }
	readImagePreview(m, tab)
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("old reader not started")
	}
	data := imageSourcePNG(t, 2, 3)
	m.BindingID, m.SessionID, m.ImageGeneration = "host-b", "session-b", 2
	m.ReadImageContext = func(context.Context, string, int64) ([]byte, error) { return data, nil }
	readImagePreview(m, tab)
	close(release)
	if result := awaitImageSource(t, m, tab, notified); result.err != nil || result.note != "2 × 3" {
		t.Fatalf("stale Host image applied: %+v", result)
	}
	// The same client and session can receive a new follow generation.
	m.ImageGeneration++
	failure := errors.New("new generation must reread")
	m.ReadImageContext = func(context.Context, string, int64) ([]byte, error) { return nil, failure }
	if result := awaitImageSource(t, m, tab, notified); !errors.Is(result.err, failure) || result.bitmap != nil {
		t.Fatalf("cached previous generation image survived: %+v", result)
	}
}
