package market

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/imgcache"
)

type readmeImageTransport func(*http.Request) (*http.Response, error)

func (f readmeImageTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func readmeImageCache(t *testing.T) *imgcache.Cache {
	t.Helper()
	readmeRemoteImages.RLock()
	previous := readmeRemoteImages.cache
	readmeRemoteImages.RUnlock()
	cache := imgcache.New(t.TempDir())
	SetRemoteImageCache(cache)
	t.Cleanup(func() { SetRemoteImageCache(previous) })
	return cache
}

func readmeImagePNG(t *testing.T) []byte {
	t.Helper()
	canvas := image.NewRGBA(image.Rect(0, 0, 800, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 800; x++ {
			canvas.SetRGBA(x, y, color.RGBA{R: uint8(50 + x/8), G: uint8(90 + y/4), B: 160, A: 255})
		}
	}
	var body bytes.Buffer
	if err := png.Encode(&body, canvas); err != nil {
		t.Fatal(err)
	}
	return body.Bytes()
}

func awaitReadmeImage(t *testing.T, cache *imgcache.Cache, url string, want imgcache.Status) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		status, _, _ := cache.State(url)
		if status == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	status, _, err := cache.State(url)
	t.Fatalf("image state=%v want=%v error=%v", status, want, err)
}

func TestMarketReadmeRemoteImageLoadsInPlaceWithoutNavigation(t *testing.T) {
	cache := readmeImageCache(t)
	imageURL := "https://example.invalid/preview.png"
	started, release := make(chan struct{}), make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	var calls atomic.Int32
	picture := readmeImagePNG(t)
	cache.SetHTTPClient(&http.Client{Transport: readmeImageTransport(func(r *http.Request) (*http.Response, error) {
		if calls.Add(1) == 1 {
			close(started)
		}
		<-release
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"image/png"}}, Body: io.NopCloser(bytes.NewReader(picture))}, nil
	})})
	opened := 0
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			marketReadme(c, Model{OnOpenURL: func(string) { opened++ }}, "Before image\n\n![Preview]("+imageURL+")\n\nAfter image", c.Theme().Text, c.Theme().TextMuted, c.Theme().Surface, c.Theme().Accent)
		})
	}, 400, 600)
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("image did not request through injected cache transport")
	}
	if !tt.HasText("Preview") || opened != 0 {
		t.Fatal("loading image lost alt text or activated navigation")
	}
	tt.Frame()
	close(release)
	awaitReadmeImage(t, cache, imageURL, imgcache.StatusReady)
	tt.Frame()
	pictureRect, ok := tt.Find("README image " + imageURL)
	var before, after ui.Rect
	var beforeOK, afterOK bool
	// A native RichText paragraph retains its terminating newline in its
	// accessible text; locate that exact element rather than its trimmed copy.
	for _, text := range tt.Texts() {
		switch strings.TrimSpace(text) {
		case "Before image":
			before, beforeOK = tt.Find(text)
		case "After image":
			after, afterOK = tt.Find(text)
		}
	}
	if !ok || !beforeOK || !afterOK || pictureRect.W != 400 || pictureRect.H != 200 || before.Y >= pictureRect.Y || after.Y < pictureRect.Y+pictureRect.H || calls.Load() != 1 || opened != 0 {
		t.Fatalf("image order/ratio: before=%+v image=%+v after=%+v calls=%d opened=%d", before, pictureRect, after, calls.Load(), opened)
	}
	saveMarketSnapshot(t, tt, "market-readme-image-loaded.png")
}

func TestMarketReadmeCachedImagesKeepNaturalSizeAndAspectRatio(t *testing.T) {
	cache := readmeImageCache(t)
	cache.SetHTTPClient(&http.Client{Transport: readmeImageTransport(func(*http.Request) (*http.Response, error) {
		t.Error("disk-cached image attempted an HTTP request")
		return nil, fmt.Errorf("network is unavailable in this fixture")
	})})
	for _, tc := range []struct {
		name string
		body []byte
		w, h float32
	}{
		{"wide.png", readmeImagePNG(t), 800, 400},
		{"small.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 40 20"><rect width="40" height="20" fill="#4176e6"/></svg>`), 40, 20},
	} {
		url := "http://example.invalid/" + tc.name
		if err := os.WriteFile(cache.FilePath(url), tc.body, 0600); err != nil {
			t.Fatal(err)
		}
		for _, width := range []int{320, 800} {
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() {
					renderReadmeImage(c, url, "Cached preview")
				})
			}, width, 600)
			r, ok := tt.Find("README image " + url)
			wantW := min(float32(width), tc.w)
			if !ok || r.W != wantW || r.H != wantW*tc.h/tc.w {
				t.Fatalf("%s width%d image=%+v", tc.name, width, r)
			}
		}
	}
}

func TestMarketReadmeImageFailureAndUnsupportedDestinationsRetainAlt(t *testing.T) {
	cache := readmeImageCache(t)
	url := "https://example.invalid/missing.png"
	var calls atomic.Int32
	cache.SetHTTPClient(&http.Client{Transport: readmeImageTransport(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return &http.Response{StatusCode: 404, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("missing"))}, nil
	})})
	source := "![Missing preview](" + url + ")\n\n![Relative preview](images/local.png)\n\n![Local preview](file:///C:/private.png)\n\n![Embedded preview](data:image/png;base64,AAAA)"
	tt := ui.NewTester(func(c *ui.Context) {
		marketReadme(c, Model{}, source, c.Theme().Text, c.Theme().TextMuted, c.Theme().Surface, c.Theme().Accent)
	}, 420, 600)
	awaitReadmeImage(t, cache, url, imgcache.StatusError)
	for range 3 {
		tt.Frame()
	}
	for _, text := range []string{"Missing preview", "Relative preview", "Local preview", "Embedded preview"} {
		if !tt.HasText(text) {
			t.Errorf("failed or non-web image lost authored alt %q", text)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("failed image retried or unsupported destination fetched: %d calls", calls.Load())
	}
	saveMarketSnapshot(t, tt, "market-readme-image-failed.png")
}
