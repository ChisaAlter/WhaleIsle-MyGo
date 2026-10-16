package workspacefs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"whaleisle/internal/dsh"
)

func TestB26TextPagesKeepUTF8AndEOF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.md")
	body := strings.Repeat("a", TextPreviewLimit-1) + "鲸" + strings.Repeat("b", TextPreviewLimit) + "\nend"
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	var offset int64
	var combined, version string
	for n := 0; n < 4; n++ {
		page, err := ReadFileTextPage(path, offset)
		if err != nil {
			t.Fatal(err)
		}
		if n == 0 {
			version = page.Version
		} else if page.Version != version {
			t.Fatal("version changed")
		}
		combined += page.Text
		if page.EOF {
			if combined != body {
				t.Fatal("page lost or repeated bytes")
			}
			return
		}
		if page.Next <= offset {
			t.Fatal("no progress")
		}
		offset = page.Next
	}
	t.Fatal("no EOF")
}

func TestB26RemotePageUsesLineCursor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			RPCID string `json:"rpcId"`
			Args  struct {
				Path  string                          `json:"path"`
				Range struct{ Offset, Limit float64 } `json:"range"`
			} `json:"args"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		// Request envelopes are covered by dsh.Client; assert the returned cursor contract here.
		json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": map[string]any{"ok": true, "value": map[string]any{"offset": 2001, "lines": 2, "text": "next\nlast", "eof": true, "version": "v1"}}})
	}))
	defer server.Close()
	page, err := ReadRemoteTextPage(context.Background(), &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}, "large.txt", 2001)
	if err != nil {
		t.Fatal(err)
	}
	if page.Next != 2003 || page.Separator != "\n" || !page.EOF || page.Version != "v1" {
		t.Fatalf("page %#v", page)
	}
}
