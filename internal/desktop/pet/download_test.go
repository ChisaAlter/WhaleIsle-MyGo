package pet

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/config"
)

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func TestEmbeddedDownloadManifestClosure(t *testing.T) {
	specs, total, commit, raw, err := activeDownloadSpecs()
	if err != nil {
		t.Fatalf("activeDownloadSpecs failed: %v", err)
	}
	if len(specs) != 10 {
		t.Fatalf("expected 10 pinned files (9 THA4 + anime4k.js), got %d", len(specs))
	}
	const wantTotal = int64(38605660 + 9398501)
	if total != wantTotal {
		t.Fatalf("totalBytes=%d, want %d", total, wantTotal)
	}
	if commit != "90465b67ec16765631d54c195af19d9ff3c3a770" || len(raw) == 0 {
		t.Fatalf("unexpected manifest commit=%q len=%d", commit, len(raw))
	}
	hasAnime4K := false
	for _, s := range specs {
		if !strings.HasPrefix(s.URL, "https://raw.githubusercontent.com/") {
			t.Fatalf("non-pinned raw github URL: %s", s.URL)
		}
		if s.Path == "anime4k.js" {
			hasAnime4K = true
		}
	}
	if !hasAnime4K {
		t.Fatal("anime4k.js missing from pinned download list")
	}
}

func TestDownloadRenderAssetsVerifyResumeCancelAndFallback(t *testing.T) {
	prevRewrite := renderAssetDownloadURLRewrite
	prevSpecs := testDownloadSpecs
	prevVerify := testVerifyRoot
	prevRoot := configuredRenderRoot.Load()
	prevExplicit := explicitDevRenderRoot.Load()
	t.Cleanup(func() {
		renderAssetDownloadURLRewrite = prevRewrite
		testDownloadSpecs = prevSpecs
		testVerifyRoot = prevVerify
		configuredRenderRoot.Store(prevRoot)
		explicitDevRenderRoot.Store(prevExplicit)
	})

	fileA := []byte("model-bytes-verified-a")
	fileB := []byte("ort-bytes-verified-b-12345")
	var hitsA, hitsB atomic.Int32
	var corruptB atomic.Bool
	blockB := make(chan struct{})
	bStarted := make(chan struct{})
	var waitForB atomic.Bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/a":
			hitsA.Add(1)
			_, _ = w.Write(fileA)
		case "/b":
			hitsB.Add(1)
			if waitForB.Load() {
				select {
				case <-bStarted:
				default:
					close(bStarted)
				}
				select {
				case <-blockB:
				case <-r.Context().Done():
					return
				}
			}
			if corruptB.Load() {
				_, _ = w.Write([]byte("corrupt-bytes-wrong-len!!"))
				return
			}
			_, _ = w.Write(fileB)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	testDownloadSpecs = []downloadFileSpec{
		{Path: "avatar/model.onnx", Size: int64(len(fileA)), SHA256: sha256Hex(fileA), URL: srv.URL + "/a"},
		{Path: "ort/ort.all.min.js", Size: int64(len(fileB)), SHA256: sha256Hex(fileB), URL: srv.URL + "/b"},
	}
	renderAssetDownloadURLRewrite = func(u string) string { return u }
	testVerifyRoot = func(dir string) error {
		for _, spec := range testDownloadSpecs {
			if !verifyFileSHA256(filepath.Join(dir, filepath.FromSlash(spec.Path)), spec.Size, spec.SHA256) {
				return errors.New("verify failed")
			}
		}
		return nil
	}

	userDir := t.TempDir()
	store, err := config.Open(userDir)
	if err != nil {
		t.Fatal(err)
	}
	// Point defaultRenderRoot to a missing sibling dir beside executable via test hook override?
	// Keep defaultRenderRoot active via empty explicitRoot in packaged mode.
	ConfigureRenderAssets(false, "")
	p := &Companion{Store: store, Enabled: true, Neural: true, Behavior: config.DefaultPetBehavior()}

	if snap := p.RenderAssetsStatus(); snap.State != RenderAssetsMissing {
		t.Fatalf("initial status=%v, want RenderAssetsMissing", snap.State)
	}

	// 1. Corrupt download must fail closed and remove .partial.
	corruptB.Store(true)
	doneCh := make(chan struct{})
	p.StartDownloadRenderAssets(func() {
		if p.RenderAssetsStatus().State == RenderAssetsError {
			select {
			case <-doneCh:
			default:
				close(doneCh)
			}
		}
	})
	select {
	case <-doneCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for error state")
	}
	destRoot := p.userRenderRoot()
	if _, err := os.Stat(filepath.Join(destRoot, "ort", "ort.all.min.js.partial")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal(".partial file leaked after failed checksum")
	}
	if snap := p.RenderAssetsStatus(); snap.State != RenderAssetsError {
		t.Fatalf("expected error state, got %+v", snap)
	}
	if hitsA.Load() != 1 || hitsB.Load() != 1 {
		t.Fatalf("unexpected initial fetch counts a=%d b=%d", hitsA.Load(), hitsB.Load())
	}

	// 2. Retry succeeds and skips fileA since it already matches SHA-256 on disk.
	corruptB.Store(false)
	readyCh := make(chan struct{})
	p.StartDownloadRenderAssets(func() {
		if p.RenderAssetsStatus().State == RenderAssetsReady {
			select {
			case <-readyCh:
			default:
				close(readyCh)
			}
		}
	})
	select {
	case <-readyCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for ready state")
	}
	if hitsA.Load() != 1 {
		t.Fatalf("already-verified fileA should have been skipped on resume; fetches=%d", hitsA.Load())
	}
	if hitsB.Load() != 2 {
		t.Fatalf("expected fileB retry count=2, got %d", hitsB.Load())
	}
	if snap := p.RenderAssetsStatus(); snap.State != RenderAssetsReady || snap.DownloadedBytes != snap.TotalBytes {
		t.Fatalf("expected ready snapshot, got %+v", snap)
	}
	// UserDataDir fallback resolves only if adjacent exe pet-render-assets doesn't exist.
	exe, _ := os.Executable()
	adj := filepath.Join(filepath.Dir(exe), "pet-render-assets")
	if _, err := os.Lstat(adj); errors.Is(err, os.ErrNotExist) {
		if got := p.resolveActiveRenderRoot(); got != destRoot {
			t.Fatalf("expected UserDataDir fallback %q, got %q", destRoot, got)
		}
	}
	p.Error = "prior missing error"
	if !p.PollDownloadedRenderAssets() || p.Error != "" {
		t.Fatalf("PollDownloadedRenderAssets did not clear stale error")
	}

	// 3. Cancellation cleans .partial and leaves RenderAssetsMissing when target not ready.
	_ = os.Remove(filepath.Join(destRoot, "ort", "ort.all.min.js"))
	waitForB.Store(true)
	cancelledDone := make(chan struct{})
	p.StartDownloadRenderAssets(func() {
		if !p.renderDownloadIsRunning() {
			select {
			case <-cancelledDone:
			default:
				close(cancelledDone)
			}
		}
	})
	select {
	case <-bStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("fileB request never started")
	}
	p.CancelDownloadRenderAssets()
	select {
	case <-cancelledDone:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for cancel completion")
	}
	close(blockB)
	if _, err := os.Stat(filepath.Join(destRoot, "ort", "ort.all.min.js.partial")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal(".partial remained after cancel")
	}
	if snap := p.RenderAssetsStatus(); snap.State != RenderAssetsMissing {
		t.Fatalf("expected missing state after clean cancel, got %+v", snap)
	}
	_ = io.Discard
}

func (p *Companion) renderDownloadIsRunning() bool {
	p.renderDownload.mu.Lock()
	defer p.renderDownload.mu.Unlock()
	return p.renderDownload.downloading
}
