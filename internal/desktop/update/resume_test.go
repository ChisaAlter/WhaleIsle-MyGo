package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func payloadDigest(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

func TestNativeFullPayloadPauseResumeAndRangeFallback(t *testing.T) {
	for _, honorRange := range []bool{true, false} {
		t.Run(fmt.Sprint(honorRange), func(t *testing.T) {
			payload := []byte("first-8!remaining-installer-bytes")
			var calls atomic.Int32
			var resumedRange string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					w.WriteHeader(200)
					_, _ = w.Write(payload[:8])
					w.(http.Flusher).Flush()
					<-r.Context().Done()
					return
				}
				resumedRange = r.Header.Get("Range")
				if honorRange {
					w.Header().Set("Content-Range", fmt.Sprintf("bytes 8-%d/%d", len(payload)-1, len(payload)))
					w.WriteHeader(206)
					_, _ = w.Write(payload[8:])
				} else {
					_, _ = w.Write(payload)
				}
			}))
			defer srv.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			opt := StageOptions{Resume: true, ReleaseURL: srv.URL + "/fixture.exe", DestDir: t.TempDir(), HTTPClient: srv.Client(), ExpectSHA256: payloadDigest(payload), ExpectSize: int64(len(payload)), OnProgress: func(n, _ int64) {
				if n >= 8 {
					cancel()
				}
			}}
			if _, err := DownloadAndStageOpts(ctx, opt); !errors.Is(err, context.Canceled) {
				t.Fatalf("pause=%v", err)
			}
			part := filepath.Join(opt.DestDir, "fixture.exe.part")
			if st, err := os.Stat(part); err != nil || st.Size() != 8 {
				t.Fatalf("partial=%v %v", st, err)
			}
			opt.OnProgress = nil
			got, err := DownloadAndStageOpts(context.Background(), opt)
			if err != nil {
				t.Fatal(err)
			}
			if resumedRange != "bytes=8-" {
				t.Fatalf("resume did not reuse bytes: %q", resumedRange)
			}
			if b, err := os.ReadFile(got.Path); err != nil || string(b) != string(payload) {
				t.Fatalf("recovered bytes %q %v", b, err)
			}
			if _, err := os.Stat(part); !os.IsNotExist(err) {
				t.Fatal("verified partial was not promoted")
			}
		})
	}
}

func TestNativePartialIdentityAndIntegrityRemainBound(t *testing.T) {
	payload := []byte("exact verified package")
	var gotRange string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRange = r.Header.Get("Range")
		_, _ = w.Write(payload)
	}))
	defer srv.Close()
	dir := t.TempDir()
	part := filepath.Join(dir, "fixture.exe.part")
	if err := os.WriteFile(part, []byte("stale"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(part+".json", []byte(`{"URL":"https://different.example.test/fixture.exe","SHA256":"different","Size":5}`), 0600); err != nil {
		t.Fatal(err)
	}
	opt := StageOptions{Resume: true, ReleaseURL: srv.URL + "/fixture.exe", DestDir: dir, HTTPClient: srv.Client(), ExpectSHA256: payloadDigest(payload), ExpectSize: int64(len(payload))}
	if _, err := DownloadAndStageOpts(context.Background(), opt); err != nil {
		t.Fatal(err)
	}
	if gotRange != "" {
		t.Fatal("different installer partial reused")
	}
	opt.ExpectSHA256 = payloadDigest([]byte("wrong"))
	if _, err := DownloadAndStageOpts(context.Background(), opt); err == nil {
		t.Fatal("bad digest staged")
	}
	if _, err := os.Stat(part); !os.IsNotExist(err) {
		t.Fatal("failed integrity left reusable bytes")
	}
}

func TestNativeDeltaResumeReusesVerifiedPartialBlocks(t *testing.T) {
	payload := []byte("AAAAAAAA" + "BBBBBBBB" + "CCCCCCCC")
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start, end, ok := parseBytesRange(r.Header.Get("Range"), int64(len(payload)))
		if !ok {
			t.Error("expected bounded Range")
			w.WriteHeader(400)
			return
		}
		calls.Add(1)
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(payload)))
		w.WriteHeader(206)
		_, _ = w.Write(payload[start : end+1])
	}))
	defer srv.Close()
	old := filepath.Join(t.TempDir(), "base.exe")
	if err := os.WriteFile(old, []byte("ZZZZZZZZZZZZZZZZZZZZZZZZ"), 0600); err != nil {
		t.Fatal(err)
	}
	info := BlockMapInfo{FileSize: int64(len(payload))}
	for off := 0; off < len(payload); off += 8 {
		info.Blocks = append(info.Blocks, BlockEntry{Offset: int64(off), Size: 8, SHA256: payloadDigest(payload[off : off+8])})
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	opt := PatchOptions{Resume: true, OldPath: old, Feed: Feed{URL: srv.URL + "/fixture.exe", Size: int64(len(payload)), SHA256: payloadDigest(payload)}, BlockMap: info, DestDir: t.TempDir(), HTTPClient: srv.Client(), OnProgress: func(n, _ int64) {
		if n == 8 {
			cancel()
		}
	}}
	if _, err := ApplyBlockPatchOpts(ctx, opt); !errors.Is(err, context.Canceled) {
		t.Fatalf("pause=%v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("first attempt fetched %d blocks", calls.Load())
	}
	opt.OnProgress = nil
	got, err := ApplyBlockPatchOpts(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 || got.CopiedBlocks != 1 || got.FetchedBlocks != 2 {
		t.Fatalf("partial block not reused: hits=%d result=%+v", calls.Load(), got)
	}
	if b, _ := os.ReadFile(got.Path); string(b) != string(payload) {
		t.Fatal("bad reconstructed installer")
	}
}
