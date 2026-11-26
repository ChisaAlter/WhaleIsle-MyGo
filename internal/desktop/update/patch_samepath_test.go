package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestApplyBlockPatchSamePath(t *testing.T) {
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

	mux := http.NewServeMux()
	mux.HandleFunc("/WhaleIsle.exe", func(w http.ResponseWriter, r *http.Request) {
		rng := r.Header.Get("Range")
		start, end, ok := parseBytesRange(rng, int64(len(newPayload)))
		if !ok {
			http.Error(w, "bad range", http.StatusRequestedRangeNotSatisfiable)
			return
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(newPayload)))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(newPayload[start : end+1])
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	dir := t.TempDir()
	feedURL := srv.URL + "/WhaleIsle.exe"
	dest := stageFileName(dir, feedURL)
	if err := os.WriteFile(dest, oldPayload, 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := ApplyBlockPatchOpts(context.Background(), PatchOptions{
		OldPath: dest,
		Feed: Feed{
			Version: "1.2.0",
			URL:     feedURL,
			SHA256:  hex.EncodeToString(whole[:]),
			Size:    int64(len(newPayload)),
		},
		BlockMap:   info,
		DestDir:    dir,
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatalf("in-place same-path block patch failed: %v", err)
	}
	if res.Path != dest || res.CopiedBlocks != 1 || res.FetchedBlocks != 1 {
		t.Fatalf("unexpected same-path result: %+v", res)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(newPayload) {
		t.Fatalf("in-place patched bytes = %q, want %q", got, newPayload)
	}
}
