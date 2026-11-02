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

	"whaleisle/internal/desktop/imgcache"
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
