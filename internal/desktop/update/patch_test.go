package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestApplyBlockPatchUsesRangeForChangedBlocks(t *testing.T) {
	oldPayload := []byte("AAAAAAAA" + "BBBBBBBB")
	newPayload := []byte("AAAAAAAA" + "CCCCCCCC")
	b0 := sha256.Sum256(newPayload[:8])
	b1 := sha256.Sum256(newPayload[8:])
	info := BlockMapInfo{
		FileSize: int64(len(newPayload)),
		Blocks: []BlockEntry{
			{Offset: 0, Size: 8, SHA256: hex.EncodeToString(b0[:])},
			{Offset: 8, Size: 8, SHA256: hex.EncodeToString(b1[:])},
		},
	}
	whole := sha256.Sum256(newPayload)

	var rangeHits atomic.Int64
	var fullHits atomic.Int64
	var rangedBytes atomic.Int64
	mux := http.NewServeMux()
	mux.HandleFunc("/WhaleIsle.exe", func(w http.ResponseWriter, r *http.Request) {
		rng := r.Header.Get("Range")
		if rng == "" {
			fullHits.Add(1)
			_, _ = w.Write(newPayload)
			return
		}
		start, end, ok := parseBytesRange(rng, int64(len(newPayload)))
		if !ok {
			http.Error(w, "bad range", http.StatusRequestedRangeNotSatisfiable)
			return
		}
		rangeHits.Add(1)
		rangedBytes.Add(end - start + 1)
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(newPayload)))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(newPayload[start : end+1])
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old.bin")
	if err := os.WriteFile(oldPath, oldPayload, 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := ApplyBlockPatchOpts(context.Background(), PatchOptions{
		OldPath: oldPath,
		Feed: Feed{
			Version: "1.1.0",
			URL:     srv.URL + "/WhaleIsle.exe",
			SHA256:  hex.EncodeToString(whole[:]),
			Size:    int64(len(newPayload)),
		},
		BlockMap:   info,
		DestDir:    dir,
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.UsedRange {
		t.Fatalf("expected UsedRange: %+v", res)
	}
	if res.CopiedBlocks != 1 || res.FetchedBlocks != 1 {
		t.Fatalf("copied/fetched %+v", res)
	}
	if rangeHits.Load() != 1 || fullHits.Load() != 0 {
		t.Fatalf("rangeHits=%d fullHits=%d", rangeHits.Load(), fullHits.Load())
	}
	if rangedBytes.Load() != 8 {
		t.Fatalf("ranged bytes %d", rangedBytes.Load())
	}
	data, err := os.ReadFile(res.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(newPayload) {
		t.Fatalf("reconstructed %q", data)
	}
}

func TestApplyBlockPatchRangeUnsupportedFallsBack(t *testing.T) {
	oldPayload := []byte("AAAAAAAA" + "BBBBBBBB")
	newPayload := []byte("AAAAAAAA" + "CCCCCCCC")
	b0 := sha256.Sum256(newPayload[:8])
	b1 := sha256.Sum256(newPayload[8:])
	info := BlockMapInfo{
		FileSize: int64(len(newPayload)),
		Blocks: []BlockEntry{
			{Offset: 0, Size: 8, SHA256: hex.EncodeToString(b0[:])},
			{Offset: 8, Size: 8, SHA256: hex.EncodeToString(b1[:])},
		},
	}

	var fullHits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ignore Range: always 200 with the whole payload.
		fullHits.Add(1)
		_, _ = w.Write(newPayload)
	}))
	defer srv.Close()

	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old.bin")
	if err := os.WriteFile(oldPath, oldPayload, 0o644); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	res, err := ApplyBlockPatchOpts(context.Background(), PatchOptions{
		OldPath:    oldPath,
		Feed:       Feed{URL: srv.URL + "/WhaleIsle.exe"},
		BlockMap:   info,
		DestDir:    dest,
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.UsedRange {
		t.Fatalf("fallback should not set UsedRange: %+v", res)
	}
	if !strings.Contains(res.Message, "Range unsupported") {
		t.Fatalf("message %q", res.Message)
	}
	if fullHits.Load() < 1 {
		t.Fatal("expected full GET fallback")
	}
	data, _ := os.ReadFile(res.Path)
	if string(data) != string(newPayload) {
		t.Fatalf("%q", data)
	}
}

func TestApplyIncrementalPatchWhenOldExists(t *testing.T) {
	oldPayload := []byte("HEADHEAD" + "OLD-TAIL")
	newPayload := []byte("HEADHEAD" + "NEW-TAIL")
	b0 := sha256.Sum256(newPayload[:8])
	b1 := sha256.Sum256(newPayload[8:])
	info := BlockMapInfo{
		FileSize: int64(len(newPayload)),
		Blocks: []BlockEntry{
			{Offset: 0, Size: 8, SHA256: hex.EncodeToString(b0[:])},
			{Offset: 8, Size: 8, SHA256: hex.EncodeToString(b1[:])},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/map.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(info)
	})
	mux.HandleFunc("/WhaleIsle.exe", func(w http.ResponseWriter, r *http.Request) {
		start, end, ok := parseBytesRange(r.Header.Get("Range"), int64(len(newPayload)))
		if !ok {
			_, _ = w.Write(newPayload)
			return
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(newPayload)))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(newPayload[start : end+1])
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	dir := t.TempDir()
	oldPath := filepath.Join(dir, "prev.exe")
	if err := os.WriteFile(oldPath, oldPayload, 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := ApplyIncremental(context.Background(), IncrementalOptions{
		CurrentVersion: "1.0.0",
		Feed: Feed{
			Version:     "1.1.0",
			URL:         srv.URL + "/WhaleIsle.exe",
			BlockMapURL: srv.URL + "/map.json",
		},
		DestDir:    dir,
		OldPath:    oldPath,
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.UsedRange || res.CopiedBlocks != 1 {
		t.Fatalf("%+v", res)
	}
}

func TestApplyIncrementalOldMissingFullDownload(t *testing.T) {
	payload := []byte("full-when-old-missing")
	sum := sha256.Sum256(payload)
	info := BlockMapInfo{
		FileSize: int64(len(payload)),
		Blocks:   []BlockEntry{{Offset: 0, Size: int64(len(payload)), SHA256: hex.EncodeToString(sum[:])}},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/map.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(info)
	})
	mux.HandleFunc("/x.exe", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	res, err := ApplyIncremental(context.Background(), IncrementalOptions{
		CurrentVersion: "1.0.0",
		Feed: Feed{
			Version:     "1.0.1",
			URL:         srv.URL + "/x.exe",
			BlockMapURL: srv.URL + "/map.json",
		},
		DestDir:    t.TempDir(),
		OldPath:    filepath.Join(t.TempDir(), "does-not-exist.exe"),
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.UsedRange {
		t.Fatalf("missing old should full-download: %+v", res)
	}
	if !strings.Contains(res.Message, "full download") {
		t.Fatalf("message %q", res.Message)
	}
}

func TestApplyBlockPatchVerifyFailClosed(t *testing.T) {
	oldPayload := []byte("AAAAAAAA" + "BBBBBBBB")
	newPayload := []byte("AAAAAAAA" + "CCCCCCCC")
	b0 := sha256.Sum256(newPayload[:8])
	// Claim the second block is CCCCCCCC but serve garbage via Range.
	info := BlockMapInfo{
		FileSize: int64(len(newPayload)),
		Blocks: []BlockEntry{
			{Offset: 0, Size: 8, SHA256: hex.EncodeToString(b0[:])},
			{Offset: 8, Size: 8, SHA256: hex.EncodeToString(b0[:])}, // wrong hash vs served bytes
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start, end, ok := parseBytesRange(r.Header.Get("Range"), int64(len(newPayload)))
		if !ok {
			http.Error(w, "need range", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(newPayload)))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(newPayload[start : end+1])
	}))
	defer srv.Close()

	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old.bin")
	if err := os.WriteFile(oldPath, oldPayload, 0o644); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	_, err := ApplyBlockPatchOpts(context.Background(), PatchOptions{
		OldPath:    oldPath,
		Feed:       Feed{URL: srv.URL + "/WhaleIsle.exe"},
		BlockMap:   info,
		DestDir:    dest,
		HTTPClient: srv.Client(),
	})
	if err == nil || !strings.Contains(err.Error(), "blockmap verify") {
		t.Fatalf("expected verify failure, got %v", err)
	}
	entries, _ := os.ReadDir(dest)
	if len(entries) != 0 {
		t.Fatalf("fail closed should remove staged file, got %v", entries)
	}
}
