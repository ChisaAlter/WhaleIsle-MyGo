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

func TestParseBlockMap(t *testing.T) {
	raw := []byte(`{"fileSize":8192,"blocks":[
		{"offset":0,"size":4096,"sha256":"aaa"},
		{"offset":4096,"size":4096,"sha256":"bbb"}
	]}`)
	info, err := ParseBlockMap(raw)
	if err != nil {
		t.Fatal(err)
	}
	if info.FileSize != 8192 || len(info.Blocks) != 2 {
		t.Fatalf("%+v", info)
	}
	if info.Blocks[1].Offset != 4096 || info.Blocks[1].SHA256 != "bbb" {
		t.Fatalf("block1: %+v", info.Blocks[1])
	}
}

func TestParseBlockMapRejectsEmpty(t *testing.T) {
	if _, err := ParseBlockMap([]byte(`{"blocks":[]}`)); err == nil {
		t.Fatal("expected error")
	}
	if _, err := ParseBlockMap([]byte(`{"blocks":[{"offset":0,"size":0,"sha256":"x"}]}`)); err == nil {
		t.Fatal("expected size error")
	}
	if _, err := ParseBlockMap([]byte(`not-json`)); err == nil {
		t.Fatal("expected JSON error")
	}
}

func TestDownloadBlockMap(t *testing.T) {
	doc := BlockMapInfo{
		FileSize: 100,
		Blocks:   []BlockEntry{{Offset: 0, Size: 100, SHA256: "abcd"}},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q", r.Header.Get("Accept"))
		}
		_ = json.NewEncoder(w).Encode(doc)
	}))
	defer srv.Close()

	got, err := DownloadBlockMap(context.Background(), srv.URL+"/map.json", srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	if got.FileSize != 100 || len(got.Blocks) != 1 || got.Blocks[0].SHA256 != "abcd" {
		t.Fatalf("%+v", got)
	}
}

func TestDownloadBlockMapHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()
	if _, err := DownloadBlockMap(context.Background(), srv.URL, srv.Client()); err == nil {
		t.Fatal("expected HTTP error")
	}
}

func TestCanIncremental(t *testing.T) {
	withURL := Feed{Version: "1.2.0", BlockMapURL: "https://example.com/m.blockmap.json"}
	withInline := Feed{
		Version: "1.3.0",
		BlockMap: &BlockMapInfo{
			Blocks: []BlockEntry{{Offset: 0, Size: 1, SHA256: "x"}},
		},
	}
	noMap := Feed{Version: "1.2.0"}
	emptyInline := Feed{Version: "1.2.0", BlockMap: &BlockMapInfo{}}
	majorBump := Feed{Version: "2.0.0", BlockMapURL: "https://example.com/m.json"}

	if !CanIncremental("1.0.0", withURL) {
		t.Fatal("same major + blockMapURL")
	}
	if !CanIncremental("v1.1.0", withInline) {
		t.Fatal("same major + inline blockMap")
	}
	if CanIncremental("1.0.0", noMap) {
		t.Fatal("no blockmap")
	}
	if CanIncremental("1.0.0", emptyInline) {
		t.Fatal("empty blocks")
	}
	if CanIncremental("1.9.0", majorBump) {
		t.Fatal("major channel change")
	}
	if CanIncremental("", withURL) || CanIncremental("1.0.0", Feed{BlockMapURL: "x"}) {
		t.Fatal("missing versions")
	}
}

func TestCheckIgnoresOptionalBlockMapFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(Feed{
			Version:     "1.2.0",
			URL:         "https://example.com/WhaleIsle-1.2.0.exe",
			BlockMapURL: "https://example.com/WhaleIsle-1.2.0.blockmap.json",
			BlockMap: &BlockMapInfo{
				Blocks: []BlockEntry{{Offset: 0, Size: 8, SHA256: "deadbeef"}},
			},
		})
	}))
	defer srv.Close()

	res, err := Check(context.Background(), Options{
		CurrentVersion: "1.0.0",
		FeedURL:        srv.URL,
		HTTPClient:     srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusUpdateAvailable || res.Version != "1.2.0" {
		t.Fatalf("%+v", res)
	}
	if !res.IncrementalPossible {
		t.Fatal("expected IncrementalPossible")
	}
}

func TestVerifyBlockMap(t *testing.T) {
	// two 4-byte blocks
	payload := []byte("abcdEFGH")
	b0 := sha256.Sum256(payload[:4])
	b1 := sha256.Sum256(payload[4:])
	info := BlockMapInfo{
		FileSize: int64(len(payload)),
		Blocks: []BlockEntry{
			{Offset: 0, Size: 4, SHA256: hex.EncodeToString(b0[:])},
			{Offset: 4, Size: 4, SHA256: hex.EncodeToString(b1[:])},
		},
	}
	path := filepath.Join(t.TempDir(), "artifact.bin")
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyBlockMap(path, info); err != nil {
		t.Fatal(err)
	}

	// case-insensitive digest
	info.Blocks[0].SHA256 = strings.ToUpper(info.Blocks[0].SHA256)
	if err := VerifyBlockMap(path, info); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyBlockMapRejects(t *testing.T) {
	payload := []byte("01234567")
	sum := sha256.Sum256(payload[:4])
	good := BlockMapInfo{
		FileSize: int64(len(payload)),
		Blocks:   []BlockEntry{{Offset: 0, Size: 4, SHA256: hex.EncodeToString(sum[:])}},
	}
	path := filepath.Join(t.TempDir(), "a.bin")
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	badSize := good
	badSize.FileSize = 1
	if err := VerifyBlockMap(path, badSize); err == nil {
		t.Fatal("expected fileSize error")
	}

	badHash := good
	badHash.Blocks = []BlockEntry{{Offset: 0, Size: 4, SHA256: "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"}}
	if err := VerifyBlockMap(path, badHash); err == nil {
		t.Fatal("expected sha256 error")
	}

	overflow := good
	overflow.Blocks = []BlockEntry{{Offset: 6, Size: 4, SHA256: hex.EncodeToString(sum[:])}}
	if err := VerifyBlockMap(path, overflow); err == nil {
		t.Fatal("expected range error")
	}
}
