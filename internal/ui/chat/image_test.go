package chat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
