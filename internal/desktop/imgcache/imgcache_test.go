package imgcache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestDir(t *testing.T) {
	got := Dir(`C:\data`)
	want := filepath.Join(`C:\data`, "cache", "images")
	if got != want {
		t.Fatalf("Dir = %q, want %q", got, want)
	}
}

func TestFetchCachesByURLHash(t *testing.T) {
	png := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(png)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	c := New(dir)
	c.SetHTTPClient(srv.Client())

	url := srv.URL + "/shot.png"
	path, err := c.Fetch(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(url))
	want := filepath.Join(dir, hex.EncodeToString(sum[:]))
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(png) {
		t.Fatalf("cached bytes mismatch")
	}
	if p, ok := c.Lookup(url); !ok || p != path {
		t.Fatalf("Lookup = %q %v", p, ok)
	}
	st, p, err := c.State(url)
	if st != StatusReady || p != path || err != nil {
		t.Fatalf("State = %v %q %v", st, p, err)
	}
}

func TestFetchRejectsNonImageContentType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("nope"))
	}))
	t.Cleanup(srv.Close)

	c := New(t.TempDir())
	c.SetHTTPClient(srv.Client())
	if _, err := c.Fetch(context.Background(), srv.URL); err == nil {
		t.Fatal("expected content-type error")
	}
}

func TestFetchRejectsOversize(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(make([]byte, 32))
	}))
	t.Cleanup(srv.Close)

	c := New(t.TempDir())
	c.SetHTTPClient(srv.Client())
	c.SetMaxBytes(8)
	if _, err := c.Fetch(context.Background(), srv.URL); err == nil {
		t.Fatal("expected size error")
	}
}

func TestEnsureFetchesOnce(t *testing.T) {
	var hits int
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits++
		mu.Unlock()
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte{0xff, 0xd8, 0xff, 0xd9})
	}))
	t.Cleanup(srv.Close)

	c := New(t.TempDir())
	c.SetHTTPClient(srv.Client())

	done := make(chan struct{}, 1)
	c.Ensure(srv.URL, func() { done <- struct{}{} })
	c.Ensure(srv.URL, nil) // second call while fetching / after

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Ensure timed out")
	}
	// Allow in-flight second Ensure to settle.
	deadline := time.Now().Add(time.Second)
	for {
		st, path, _ := c.State(srv.URL)
		if st == StatusReady && path != "" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("state = %v path=%q", st, path)
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	n := hits
	mu.Unlock()
	if n != 1 {
		t.Fatalf("hits = %d, want 1", n)
	}
}

func TestIsHTTPURL(t *testing.T) {
	if !IsHTTPURL("https://example.com/a.png") {
		t.Fatal("https")
	}
	if IsHTTPURL("data:image/png;base64,xx") {
		t.Fatal("data rejected")
	}
	if IsHTTPURL(`C:\tmp\a.png`) {
		t.Fatal("local rejected")
	}
}
