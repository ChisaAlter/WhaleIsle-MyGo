package session

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"whaleisle/internal/dsh"
)

type openInAppTestTransport func(*http.Request) (*http.Response, error)

func (f openInAppTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestWorkspaceOpenInAppCapturedGestureDoesNotFollowNextSelection(t *testing.T) {
	type launch struct{ App, Path string }
	var got []launch
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://offline.invalid", Cookie: "fixture=captured"}, HTTP: &http.Client{Transport: openInAppTestTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/open-in-app/open" || r.Header.Get("Cookie") != "fixture=captured" || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("wrong captured carrier: %s %s", r.Method, r.URL.Path)
		}
		var body launch
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		got = append(got, body)
		return &http.Response{StatusCode: 204, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
	})}}
	s := New()
	s.Attach(client, nil)
	s.items = []Item{{SessionID: "A", Cwd: "C:/fixture/A"}, {SessionID: "B", Cwd: "C:/fixture/B"}}
	s.selectedID = "A"
	source := s.OpenInAppSource()
	if _, err := source.CaptureOpen("A", "C:/fixture/B", "vscode"); err == nil {
		t.Fatal("mismatched workspace admitted")
	}
	open, err := source.CaptureOpen("A", "C:/fixture/A", "vscode")
	if err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	s.selectSessionLocked("B")
	s.mu.Unlock()
	if err = open(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != (launch{"vscode", "C:/fixture/A"}) {
		t.Fatalf("gesture redirected to next Session: %+v", got)
	}
	// A stale retained header cannot newly admit the old target.
	if _, err := source.CaptureOpen("A", "C:/fixture/A", "vscode"); err == nil {
		t.Fatal("old rendered Session admitted")
	}
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://next.invalid"}}, nil)
	if source.Current() {
		t.Fatal("captured Host survived replacement")
	}
	if err = open(context.Background()); err == nil || len(got) != 1 {
		t.Fatal("old POST was sent after Host replacement")
	}
}
