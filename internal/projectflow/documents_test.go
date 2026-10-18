package projectflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type documentFixtureTransport func(*http.Request) (*http.Response, error)

func (f documentFixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// One bounded in-memory fixture: no real Host, project directory, model or tool.
func TestNativeProjectDocumentFixtures(t *testing.T) {
	calls := 0
	bodyPath, body := "docs/work-a/evidence.md", "# Full evidence\nSecond paragraph\n[untrusted](https://example.invalid)\n"
	fail, wrongPath := false, false
	c := Client{Origin: "http://127.0.0.1", Cookie: "fixture-only", HTTP: &http.Client{Transport: documentFixtureTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var q struct {
			Method, RPCID string
			Payload       map[string]any
		}
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
			return nil, err
		}
		if r.Method != "POST" || r.URL.Host != "127.0.0.1" || r.URL.Path != "/dsh-project/"+q.Method || q.Payload["projectId"] != "project-a" || r.Header.Get("Cookie") != "fixture-only" {
			t.Fatalf("source identity changed: %s %#v", r.URL, q)
		}
		var value any
		switch q.Method {
		case "store/list":
			value = map[string]any{"docs": []DocumentEntry{{Path: bodyPath, Size: 80}, {Path: "docs/work-b/empty.txt", Size: 0}}}
		case "store/read":
			if q.Payload["path"] != bodyPath {
				t.Fatalf("wrong selected source: %#v", q.Payload)
			}
			path := bodyPath
			if wrongPath {
				path = "docs/other.txt"
			}
			value = map[string]any{"path": path, "text": body}
		default:
			t.Fatalf("unexpected mutation/open RPC: %s", q.Method)
		}
		result := map[string]any{"ok": true, "value": value}
		if fail {
			result = map[string]any{"ok": false, "error": map[string]any{"message": "synthetic disconnected"}}
		}
		data, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": q.RPCID, "result": result})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(data))), Header: make(http.Header)}, nil
	})}}
	ctx := context.Background()
	b := NewDocumentBrowser(DocumentScope{Host: "host-a", ProjectID: "project-a", Generation: 3}, "Project A")
	list, err := b.BeginList()
	if err != nil {
		t.Fatal(err)
	}
	docs, err := c.Documents(ctx, list.ProjectID())
	if !b.CompleteList(list, docs, err) || !b.Snapshot().Loaded || len(b.Snapshot().Docs) != 2 {
		t.Fatal("full inventory not available")
	}
	read, err := b.BeginRead(bodyPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = b.BeginRead(bodyPath); err == nil {
		t.Fatal("duplicate click admitted")
	}
	content, err := c.ReadDocument(ctx, read.ProjectID(), read.Entry())
	if !b.CompleteRead(read, content, err) || b.Snapshot().Text != body || !b.Snapshot().TextLoaded {
		t.Fatal("body was truncated or replaced by summary")
	}
	snapshot := b.Snapshot()
	snapshot.Docs[0].Path = "docs/spoof.txt"
	if b.Snapshot().Docs[0].Path != bodyPath {
		t.Fatal("UI mutated source inventory")
	}
	// A failed reload preserves this document's previously fetched full body.
	read, _ = b.BeginRead(bodyPath)
	fail = true
	content, err = c.ReadDocument(ctx, read.ProjectID(), read.Entry())
	b.CompleteRead(read, content, err)
	fail = false
	if b.Snapshot().Text != body || b.Snapshot().Error == "" || b.Snapshot().Busy {
		t.Fatal("read failure lost body or left spinner")
	}
	read, _ = b.BeginRead(bodyPath)
	wrongPath = true
	content, err = c.ReadDocument(ctx, read.ProjectID(), read.Entry())
	b.CompleteRead(read, content, err)
	wrongPath = false
	if err == nil || b.Snapshot().Text != body {
		t.Fatal("wrong response path admitted")
	}
	// Back/close and new source scope fence even already-returned completions.
	read, _ = b.BeginRead(bodyPath)
	b.Back()
	if b.CompleteRead(read, DocumentContent{Path: bodyPath, Text: "late"}, nil) || b.Snapshot().TextLoaded {
		t.Fatal("late read crossed Back")
	}
	read, _ = b.BeginRead(bodyPath)
	other := NewDocumentBrowser(DocumentScope{Host: "host-b", ProjectID: "project-a", Generation: 3}, "B")
	if other.CompleteRead(read, DocumentContent{Path: bodyPath, Text: "foreign"}, nil) {
		t.Fatal("cross-Host read accepted")
	}
	other = NewDocumentBrowser(DocumentScope{Host: "host-a", ProjectID: "project-b", Generation: 3}, "B")
	if other.CompleteRead(read, DocumentContent{Path: bodyPath, Text: "foreign"}, nil) {
		t.Fatal("cross-project read accepted")
	}
	b.Close()
	if b.CompleteRead(read, DocumentContent{Path: bodyPath, Text: "late"}, nil) {
		t.Fatal("closed reader accepted completion")
	}
	before := calls
	for _, p := range []string{"notes.md", "../docs/a", "docs/../a", "docs/internal/a", "docs/.git/config", "docs/a\\b", "docs//a", "docs/./a", "docs/a%2fb", "docs/a:stream", "https://else.invalid/a", "C:/docs/a", "/docs/a", "docs/a\x00b"} {
		if _, err := c.ReadDocument(ctx, "project-a", DocumentEntry{Path: p, Size: 1}); err == nil {
			t.Fatalf("unsafe source accepted: %q", p)
		}
	}
	if _, err := c.ReadDocument(ctx, "project-a", DocumentEntry{Path: bodyPath, Size: DocumentReadLimit + 1}); err == nil {
		t.Fatal("oversized source admitted")
	}
	if calls != before {
		t.Fatal("invalid document contacted transport")
	}
	body = strings.Repeat("x", DocumentReadLimit+1)
	if _, err := c.ReadDocument(ctx, "project-a", DocumentEntry{Path: bodyPath, Size: 1}); err == nil {
		t.Fatal("growth past Host text limit admitted")
	}
	body = "a\x00b"
	if _, err := c.ReadDocument(ctx, "project-a", DocumentEntry{Path: bodyPath, Size: 3}); err == nil {
		t.Fatal("binary source displayed as text")
	}
	b = NewDocumentBrowser(DocumentScope{Host: "host-a", ProjectID: "project-a", Generation: 4}, "A")
	list, _ = b.BeginList()
	b.CompleteList(list, []DocumentEntry{{Path: bodyPath, Size: 1}, {Path: bodyPath, Size: 2}}, nil)
	if b.Snapshot().Loaded || b.Snapshot().Error == "" {
		t.Fatal("ambiguous document identity accepted")
	}
	list, _ = b.BeginList()
	b.CompleteList(list, []DocumentEntry{}, nil)
	if !b.Snapshot().Loaded || len(b.Snapshot().Docs) != 0 {
		t.Fatal("valid empty library confused with missing result")
	}
	if _, err = b.BeginRead(bodyPath); err == nil {
		t.Fatal("unlisted file read admitted")
	}
	list, _ = b.BeginList()
	b.CompleteList(list, nil, fmt.Errorf("fixture failure"))
	if !b.Snapshot().Loaded || b.Snapshot().Error == "" {
		t.Fatal("failed refresh erased readable inventory")
	}
}

// Missing text must not be confused with a valid, explicitly empty document.
func TestNativeProjectDocumentMissingBodyFixture(t *testing.T) {
	for _, wire := range []string{`{"path":"docs/empty.txt"}`, `{"path":"docs/empty.txt","text":null}`, `{"path":"docs/empty.txt","text":""}`} {
		c := Client{Origin: "http://127.0.0.1", HTTP: &http.Client{Transport: documentFixtureTransport(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"result":{"ok":true,"value":` + wire + `}}`)), Header: make(http.Header)}, nil
		})}}
		value, err := c.ReadDocument(context.Background(), "project-a", DocumentEntry{Path: "docs/empty.txt"})
		if strings.Contains(wire, `"text":""`) {
			if err != nil || value.Text != "" {
				t.Fatal("explicit empty document rejected", err)
			}
		} else if err == nil {
			t.Fatal("missing document body reported as empty")
		}
	}
}
