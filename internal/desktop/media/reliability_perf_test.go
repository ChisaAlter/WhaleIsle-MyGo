package media

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func tinyPNGBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestRemoteFetchPreservesDeriveAndRespectsCancelWhenSlotsFull(t *testing.T) {
	pngData := tinyPNGBytes(t)
	tmpFile := filepath.Join(t.TempDir(), "remote.png")
	if err := os.WriteFile(tmpFile, pngData, 0o600); err != nil {
		t.Fatal(err)
	}

	var derived atomic.Bool
	svc := New(Config{
		Workers: 1,
		Fetch: func(ctx context.Context, rawURL string) (string, error) {
			return tmpFile, nil
		},
	})
	owner := svc.NewOwner(context.Background(), nil)
	defer owner.Release()

	key, src := Remote("https://example.com/a.png")
	src.Derive = func(img image.Image) (image.Image, any) {
		derived.Store(true)
		return img, "derived-tag"
	}
	h := svc.Request(owner, key, src)

	deadline := time.Now().Add(2 * time.Second)
	for h.State() == Pending && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	pic, ok := h.Picture()
	if !ok || !derived.Load() || pic.Extra != "derived-tag" {
		t.Fatalf("remote Source.Derive was lost: ok=%v derived=%v extra=%v", ok, derived.Load(), pic.Extra)
	}

	// Fill all fetchSlots and confirm a canceled waiter exits promptly without leaking.
	for i := 0; i < cap(fetchSlots); i++ {
		fetchSlots <- struct{}{}
	}
	defer func() {
		for i := 0; i < cap(fetchSlots); i++ {
			<-fetchSlots
		}
	}()

	o2 := svc.NewOwner(context.Background(), nil)
	k2, s2 := Remote("https://example.com/blocked.png")
	h2 := svc.Request(o2, k2, s2)
	o2.Release()
	time.Sleep(20 * time.Millisecond)
	if h2.State() != Pending {
		t.Fatalf("dropped handle unexpectedly transitioned to %v", h2.State())
	}
}

func TestDataURLKeyHashedSourceClearedAndFailedBounded(t *testing.T) {
	pngData := tinyPNGBytes(t)
	rawDataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngData)
	key, src := DataURL(rawDataURL)
	if strings.Contains(key.ID, base64.StdEncoding.EncodeToString(pngData)) {
		t.Fatalf("DataURL Key.ID retained raw payload instead of hash: %q", key.ID)
	}
	if !strings.HasPrefix(key.ID, "dataurl:") {
		t.Fatalf("unexpected DataURL key prefix: %q", key.ID)
	}

	svc := New(Config{Workers: -1})
	owner := svc.NewOwner(context.Background(), nil)
	defer owner.Release()
	h := svc.Request(owner, key, src)
	svc.RunPending()
	if h.State() != Ready {
		t.Fatalf("DataURL decode failed: state=%v", h.State())
	}
	svc.mu.Lock()
	if h.src.DataURL != "" || h.src.Bytes != nil {
		svc.mu.Unlock()
		t.Fatal("raw source bytes/DataURL were not cleared after decode")
	}
	svc.mu.Unlock()

	// Inject > maxFailedEntries failing handles and assert bounded map size.
	for i := 0; i < maxFailedEntries+64; i++ {
		k := Key{ID: fmt.Sprintf("bad:%d", i)}
		svc.Request(owner, k, Source{Bytes: []byte("not-an-image")})
	}
	svc.RunPending()
	svc.mu.Lock()
	failedCount := svc.failedLRU.Len()
	entryCount := len(svc.entries)
	svc.mu.Unlock()
	if failedCount > maxFailedEntries {
		t.Fatalf("failedLRU exceeded bound: %d > %d", failedCount, maxFailedEntries)
	}
	if entryCount > maxFailedEntries+2 {
		t.Fatalf("entries map grew unbounded with failed handles: %d", entryCount)
	}
}
