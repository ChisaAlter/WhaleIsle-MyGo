package workspacefs

import (
	"context"
	"encoding/json"
	"fmt"
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
	for _, offset := range []int64{0, 2001} {
		t.Run(fmt.Sprint(offset), func(t *testing.T) {
			start := offset
			if start == 0 {
				start = 1
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request struct {
					RPCID   string `json:"rpcId"`
					Payload struct {
						Args struct {
							Scope string                          `json:"workspaceFileScopeId"`
							Path  string                          `json:"path"`
							Range struct{ Offset, Limit float64 } `json:"range"`
						} `json:"args"`
					} `json:"payload"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
				}
				args := request.Payload.Args
				if args.Scope != "session-B26" || args.Path != "large.txt" || args.Range.Offset != float64(start) || args.Range.Limit != 2000 {
					t.Errorf("missing session scope or wrong page args: %+v", args)
				}
				json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": map[string]any{"ok": true, "value": map[string]any{"offset": start, "lines": 2, "text": "next\nlast", "eof": true, "version": "v1"}}})
			}))
			defer server.Close()
			page, err := ReadRemoteTextPage(context.Background(), &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}, "session-B26", "large.txt", offset)
			if err != nil {
				t.Fatal(err)
			}
			separator := ""
			if start > 1 {
				separator = "\n"
			}
			if page.Next != start+2 || page.Separator != separator || !page.EOF || page.Version != "v1" {
				t.Fatalf("page %#v", page)
			}
		})
	}
}

func TestB26RemotePageRejectsMissingScope(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(http.StatusInternalServerError) }))
	defer server.Close()
	for _, scope := range []string{"", "  "} {
		if _, err := ReadRemoteTextPage(context.Background(), &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}, scope, "large.txt", 0); err == nil {
			t.Fatal("missing scope accepted")
		}
	}
	if calls != 0 {
		t.Fatalf("sent %d unscoped reads", calls)
	}
}
