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

	"whaleisle/internal/desktop/workspacefs"
)

func imageRoutePNG(t *testing.T, width, height int) []byte {
	t.Helper()
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewNRGBA(image.Rect(0, 0, width, height))); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func imageRouteModel(root string) (Model, *FileTab, chan struct{}) {
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

func awaitImageRoute(t *testing.T, m Model, tab *FileTab, notified <-chan struct{}) *imagePreview {
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
			t.Fatal("image route did not settle")
		}
	}
}

func TestDesktopImageUsesLocalSourceWithCapturedHostReaders(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "same.png"), imageRoutePNG(t, 3, 2), 0600); err != nil {
		t.Fatal(err)
	}
	m, tab, notified := imageRouteModel(root)
	t.Cleanup(func() { m.Tabs.stopImageRead(true) })
	denied := errors.New("Host callbacks are unrelated to the desktop image source")
	m.ReadRemoteContext = func(context.Context, string) (string, error) {
		t.Error("image used Host text reader")
		return "", denied
	}
	m.ReadTextPageContext = func(context.Context, string, int64) (workspacefs.TextPage, error) {
		t.Error("image used Host page reader")
		return workspacefs.TextPage{}, denied
	}
	m.ReadTableContext = func(context.Context, string, workspacefs.XLSXPageRequest) (*workspacefs.XLSXPage, error) {
		t.Error("image used Host workbook reader")
		return nil, denied
	}
	m.StatRemoteContext = func(context.Context, string) string { t.Error("image used Host stat reader"); return "" }
	result := awaitImageRoute(t, m, tab, notified)
	if result.err != nil || result.note != "3 × 2" {
		t.Fatalf("local desktop image unavailable: %+v", result)
	}
}

func TestDesktopImageReadFailureDoesNotChangeSource(t *testing.T) {
	m, tab, notified := imageRouteModel(t.TempDir())
	t.Cleanup(func() { m.Tabs.stopImageRead(true) })
	m.ReadRemoteContext = func(context.Context, string) (string, error) {
		t.Error("local failure changed source to Host")
		return "", nil
	}
	if result := awaitImageRoute(t, m, tab, notified); result.err == nil || result.bitmap != nil {
		t.Fatalf("missing local image silently changed source: %+v", result)
	}
}

func TestDesktopImageEpochDiscardsCachedPixels(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "same.png")
	if err := os.WriteFile(path, imageRoutePNG(t, 1, 1), 0600); err != nil {
		t.Fatal(err)
	}
	m, tab, notified := imageRouteModel(root)
	t.Cleanup(func() { m.Tabs.stopImageRead(true) })
	if result := awaitImageRoute(t, m, tab, notified); result.err != nil || result.note != "1 × 1" {
		t.Fatal(result)
	}
	if err := os.WriteFile(path, imageRoutePNG(t, 2, 3), 0600); err != nil {
		t.Fatal(err)
	}
	m.ImageGeneration++
	if result := awaitImageRoute(t, m, tab, notified); result.err != nil || result.note != "2 × 3" {
		t.Fatalf("old epoch pixels retained: %+v", result)
	}
}

func TestDesktopImageSharedDecoderKeepsBounds(t *testing.T) {
	for _, tc := range []struct {
		path string
		data []byte
	}{
		{"invalid.png", []byte("not a PNG")},
		{"oversized.svg", make([]byte, (1<<20)+1)},
	} {
		t.Run(tc.path, func(t *testing.T) {
			if result := decodeImagePreviewContext(context.Background(), tc.path, tc.data); result.err == nil || result.bitmap != nil {
				t.Fatal("invalid or oversized image displayed")
			}
		})
	}
}
