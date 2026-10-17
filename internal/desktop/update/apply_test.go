package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloadAndStage(t *testing.T) {
	body := []byte("MZ-fake-installer")
	sum := sha256.Sum256(body)
	digest := hex.EncodeToString(sum[:])
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	dir := t.TempDir()
	res, err := DownloadAndStageOpts(context.Background(), StageOptions{
		ReleaseURL:   srv.URL + "/WhaleIsle-1.0.0.exe",
		DestDir:      dir,
		HTTPClient:   srv.Client(),
		ExpectSHA256: digest,
		ExpectSize:   int64(len(body)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Path == "" || res.Started {
		t.Fatalf("%+v", res)
	}
	data, err := os.ReadFile(res.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(body) {
		t.Fatalf("%q", data)
	}
	if filepath.Base(res.Path) != "WhaleIsle-1.0.0.exe" {
		t.Fatalf("name %s", res.Path)
	}
	if res.SHA256 != digest {
		t.Fatalf("sha %s", res.SHA256)
	}
}

func TestDownloadAndStageRejectsBadHash(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("payload"))
	}))
	defer srv.Close()
	_, err := DownloadAndStageOpts(context.Background(), StageOptions{
		ReleaseURL:   srv.URL + "/x.exe",
		DestDir:      t.TempDir(),
		HTTPClient:   srv.Client(),
		ExpectSHA256: "deadbeef",
	})
	if err == nil {
		t.Fatal("expected hash error")
	}
}

func TestDownloadAndStageRequireSignature(t *testing.T) {
	old := authenticodeProbe
	t.Cleanup(func() { authenticodeProbe = old })

	body := []byte("MZ-signed-fake")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	authenticodeProbe = func(context.Context, string) (SignatureInfo, error) {
		return SignatureInfo{Status: SignatureNone, Detail: "NotSigned"}, nil
	}
	dir := t.TempDir()
	_, err := DownloadAndStageOpts(context.Background(), StageOptions{
		ReleaseURL:       srv.URL + "/WhaleIsle.exe",
		DestDir:          dir,
		HTTPClient:       srv.Client(),
		RequireSignature: true,
	})
	if err == nil {
		t.Fatal("expected authenticode failure")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatalf("failed staging should remove file, got %v", entries)
	}

	authenticodeProbe = func(context.Context, string) (SignatureInfo, error) {
		return SignatureInfo{Status: SignatureValid, Publisher: "CN=WhaleIsle"}, nil
	}
	res, err := DownloadAndStageOpts(context.Background(), StageOptions{
		ReleaseURL:        srv.URL + "/WhaleIsle.exe",
		DestDir:           dir,
		HTTPClient:        srv.Client(),
		RequireSignature:  true,
		ExpectedPublisher: "WhaleIsle",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Signature != SignatureValid || res.Publisher != "CN=WhaleIsle" {
		t.Fatalf("%+v", res)
	}
	if !strings.Contains(res.Message, "authenticode ok") {
		t.Fatalf("message %q", res.Message)
	}
}

func TestApplyIncrementalFullDownloadThenVerify(t *testing.T) {
	payload := []byte("MZ-incremental-stub")
	whole := sha256.Sum256(payload)
	b0 := sha256.Sum256(payload[:8])
	b1 := sha256.Sum256(payload[8:])
	mapDoc := BlockMapInfo{
		FileSize: int64(len(payload)),
		Blocks: []BlockEntry{
			{Offset: 0, Size: 8, SHA256: hex.EncodeToString(b0[:])},
			{Offset: 8, Size: int64(len(payload) - 8), SHA256: hex.EncodeToString(b1[:])},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/WhaleIsle.exe", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	})
	mux.HandleFunc("/map.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(mapDoc)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	dir := t.TempDir()
	res, err := ApplyIncremental(context.Background(), IncrementalOptions{
		CurrentVersion: "1.0.0",
		Feed: Feed{
			Version:     "1.1.0",
			URL:         srv.URL + "/WhaleIsle.exe",
			SHA256:      hex.EncodeToString(whole[:]),
			Size:        int64(len(payload)),
			BlockMapURL: srv.URL + "/map.json",
		},
		DestDir:    dir,
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Path == "" || res.Size != int64(len(payload)) {
		t.Fatalf("%+v", res)
	}
	if !strings.Contains(res.Message, "blockmap verified") {
		t.Fatalf("message %q", res.Message)
	}
	data, err := os.ReadFile(res.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(payload) {
		t.Fatalf("%q", data)
	}
}

func TestApplyIncrementalInlineBlockMap(t *testing.T) {
	payload := []byte("inline-map-payload!")
	sum := sha256.Sum256(payload)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	res, err := ApplyIncremental(context.Background(), IncrementalOptions{
		CurrentVersion: "1.0.0",
		Feed: Feed{
			Version: "1.0.1",
			URL:     srv.URL + "/x.exe",
			BlockMap: &BlockMapInfo{
				FileSize: int64(len(payload)),
				Blocks: []BlockEntry{
					{Offset: 0, Size: int64(len(payload)), SHA256: hex.EncodeToString(sum[:])},
				},
			},
		},
		DestDir:    t.TempDir(),
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.SHA256 != hex.EncodeToString(sum[:]) {
		t.Fatalf("%+v", res)
	}
}

func TestApplyIncrementalUnsupportedFallsBackHint(t *testing.T) {
	_, err := ApplyIncremental(context.Background(), IncrementalOptions{
		CurrentVersion: "1.0.0",
		Feed:           Feed{Version: "1.1.0", URL: "https://example.com/x.exe"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "DownloadAndStage") {
		t.Fatalf("want full-path hint, got %v", err)
	}

	_, err = ApplyIncremental(context.Background(), IncrementalOptions{
		CurrentVersion: "1.0.0",
		Feed: Feed{
			Version:     "2.0.0",
			URL:         "https://example.com/x.exe",
			BlockMapURL: "https://example.com/m.json",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "DownloadAndStage") {
		t.Fatalf("major bump should hint full path: %v", err)
	}
}

func TestApplyIncrementalVerifyFailClosed(t *testing.T) {
	payload := []byte("payload-bytes")
	mux := http.NewServeMux()
	mux.HandleFunc("/WhaleIsle.exe", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	})
	mux.HandleFunc("/map.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(BlockMapInfo{
			FileSize: int64(len(payload)),
			Blocks: []BlockEntry{{
				Offset: 0,
				Size:   int64(len(payload)),
				SHA256: "0000000000000000000000000000000000000000000000000000000000000000",
			}},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	dir := t.TempDir()
	_, err := ApplyIncremental(context.Background(), IncrementalOptions{
		CurrentVersion: "1.0.0",
		Feed: Feed{
			Version:     "1.1.0",
			URL:         srv.URL + "/WhaleIsle.exe",
			BlockMapURL: srv.URL + "/map.json",
		},
		DestDir:    dir,
		HTTPClient: srv.Client(),
	})
	if err == nil {
		t.Fatal("expected verify failure")
	}
	if !strings.Contains(err.Error(), "blockmap verify") {
		t.Fatalf("%v", err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatalf("fail closed should remove staged file, got %v", entries)
	}
}
