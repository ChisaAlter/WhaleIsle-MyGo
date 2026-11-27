package chat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/imgcache"
	"whaleisle/internal/ui/i18n"
)

func TestLocalImagePath(t *testing.T) {
	if localImagePath("https://example.com/a.png") != "" {
		t.Fatal("http rejected")
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "shot.png")
	if err := os.WriteFile(p, []byte{0x89, 0x50, 0x4e, 0x47}, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := localImagePath(p); got != p {
		t.Fatalf("got %q", got)
	}
}

func TestLocalImagePathDecodesMarkdownSpacesAndFileURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "My image (1).png")
	if err := os.WriteFile(path, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	encoded := strings.ReplaceAll(path, " ", "%20")
	if got := localImagePath(encoded); got != path {
		t.Fatalf("markdown image path=%q", got)
	}
	uri := (&url.URL{Scheme: "file", Path: "/" + filepath.ToSlash(path)}).String()
	if got := localImagePath(uri); got != path {
		t.Fatalf("file URL=%q, want %q (%s)", got, path, uri)
	}
	if localImagePath("file://server/share/image.png") != "" {
		t.Fatal("remote file URL became a local file")
	}
}

func TestRemoteImageCacheLookup(t *testing.T) {
	t.Cleanup(func() { SetRemoteImageCache(nil) })

	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(png)
	}))
	t.Cleanup(srv.Close)

	cache := imgcache.New(t.TempDir())
	cache.SetHTTPClient(srv.Client())
	SetRemoteImageCache(cache)

	path, err := cache.Fetch(context.Background(), srv.URL+"/a.png")
	if err != nil {
		t.Fatal(err)
	}
	st, local, err := cache.State(srv.URL + "/a.png")
	if err != nil || st != imgcache.StatusReady || local != path {
		t.Fatalf("State = %v %q %v", st, local, err)
	}
	if remoteCache() != cache {
		t.Fatal("SetRemoteImageCache not wired")
	}
}

func TestImageLightbox(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)

	// 1. Corrupt / missing file: must show localized failure copy and allow Esc / close button to invoke onClose.
	badPath := filepath.Join(t.TempDir(), "corrupt.png")
	if err := os.WriteFile(badPath, []byte("not-a-valid-png"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Pre-populate negative cache via loadChatBitmap so cachedChatBitmap returns (nil, false).
	if bmp := loadChatBitmap(badPath); bmp != nil {
		t.Fatal("expected nil bitmap for corrupt image")
	}
	closed := 0
	displayPath := badPath
	tt := ui.NewTester(func(c *ui.Context) {
		ImageLightboxOverlay(c, displayPath, "system", false, func() {
			closed++
			displayPath = ""
		})
	}, 800, 600)
	if !tt.HasText(i18n.T(i18n.KeyChatImageFailed)) {
		t.Fatalf("expected failure copy %q in lightbox, got %v", i18n.T(i18n.KeyChatImageFailed), tt.Texts())
	}
	tt.Key(0, ui.KeyEscape)
	if closed != 1 || displayPath != "" {
		t.Fatalf("Escape did not close failed lightbox: closed=%d displayPath=%q", closed, displayPath)
	}

	// 2. Pending load: must mount DialogBase with spinner & loading copy, and Esc closes cleanly.
	pendingPath := filepath.Join(t.TempDir(), "slow.png")
	block := make(chan struct{})
	origRead := chatImageReadFileFn
	chatImageReadFileFn = func(p string) ([]byte, error) {
		if p == pendingPath {
			<-block
			return nil, os.ErrNotExist
		}
		return origRead(p)
	}
	t.Cleanup(func() {
		select {
		case <-block:
		default:
			close(block)
		}
		chatImageReadFileFn = origRead
	})

	pendingClosed := 0
	pendingDisplay := pendingPath
	ttPending := ui.NewTester(func(c *ui.Context) {
		ImageLightboxOverlay(c, pendingDisplay, "system", false, func() {
			pendingClosed++
			pendingDisplay = ""
		})
	}, 800, 600)
	if !ttPending.HasText(i18n.T(i18n.KeyChatImageLoading)) {
		t.Fatalf("expected loading copy %q while pending, got %v", i18n.T(i18n.KeyChatImageLoading), ttPending.Texts())
	}
	ttPending.Key(0, ui.KeyEscape)
	if pendingClosed != 1 || pendingDisplay != "" {
		t.Fatalf("Escape did not close pending lightbox: closed=%d displayPath=%q", pendingClosed, pendingDisplay)
	}
	close(block)
	time.Sleep(10 * time.Millisecond)
}
