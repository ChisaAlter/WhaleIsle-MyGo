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

type busyEnterTransport func(*http.Request) (*http.Response, error)

func (f busyEnterTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestBusyEnterCapturesModeAndActualPromptBoundary(t *testing.T) {
	var requests []struct {
		SessionID, Mode string
		Content         []map[string]string
	}
	transport := busyEnterTransport(func(r *http.Request) (*http.Response, error) {
		var body struct {
			RPCID   any `json:"rpcId"`
			Payload struct {
				Args struct {
					Request struct {
						SessionID string              `json:"sessionId"`
						Mode      string              `json:"mode"`
						Content   []map[string]string `json:"content"`
					} `json:"request"`
				} `json:"args"`
			} `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(r.URL.Path, "/session/prompt") {
			t.Fatalf("unexpected RPC %s", r.URL.Path)
		}
		v := body.Payload.Args.Request
		requests = append(requests, struct {
			SessionID, Mode string
			Content         []map[string]string
		}{v.SessionID, v.Mode, v.Content})
		raw, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": body.RPCID, "result": map[string]any{"ok": true, "value": map[string]any{"accepted": true}}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw))), Header: make(http.Header)}, nil
	})
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://fixture.invalid"}, HTTP: &http.Client{Transport: transport}}
	s := New()
	s.Attach(client, nil)
	s.selectedID = "A"
	s.composerRosterClient = client
	s.items = []Item{{SessionID: "A", Running: true}, {SessionID: "B"}}
	preferred := "steer"
	read := func(c *dsh.Client) string {
		if c != client {
			t.Fatal("preference read for wrong Host")
		}
		return preferred
	}
	for _, tc := range []struct {
		p                               string
		running, accelerated, available bool
		want                            PromptMode
	}{
		{"queue", true, false, true, PromptModeQueue}, {"queue", true, true, true, PromptModeSteer}, {"steer", true, false, true, PromptModeSteer}, {"steer", true, true, true, PromptModeQueue}, {"steer", false, false, true, PromptModeQueue}, {"steer", true, false, false, PromptModeQueue},
	} {
		if got := ResolveBusyEnterMode(tc.p, tc.running, tc.accelerated, tc.available); got != tc.want {
			t.Fatalf("%+v: %s", tc, got)
		}
	}
	a, err := s.CaptureComposerGestureTarget("A", "A", s.composerHostGeneration, false, read)
	if err != nil {
		t.Fatal(err)
	}
	preferred = "queue"              // saving preferences cannot rewrite an existing admission
	s.SetPromptMode(PromptModeQueue) // explicit same-as-default choice must be remembered
	manual, err := s.CaptureComposerGestureTarget("A", "A", s.composerHostGeneration, false, read)
	if err != nil || manual.mode != PromptModeQueue || !s.Snapshot().PromptModeExplicit {
		t.Fatal("manual choice lost", err)
	}
	preferred = "steer"
	manual, err = s.CaptureComposerGestureTarget("A", "A", s.composerHostGeneration, false, read)
	if err != nil || manual.mode != PromptModeQueue {
		t.Fatal("setting overrode explicit mode", err)
	}
	opposite, err := s.CaptureComposerGestureTarget("A", "A", s.composerHostGeneration, true, read)
	if err != nil || opposite.mode != PromptModeSteer {
		t.Fatal("manual opposite", err)
	}
	s.UseBusyEnterSetting()
	auto, err := s.CaptureComposerGestureTarget("A", "A", s.composerHostGeneration, false, read)
	if err != nil || auto.mode != PromptModeSteer {
		t.Fatal("return to Host preference", err)
	}
	if len(requests) != 0 {
		t.Fatal("capture/settings/mode click sent a prompt")
	}
	// An unrelated selected session's running state must never affect B.
	s.selectedID = "B"
	s.promptBusy = true
	idle, err := s.CaptureComposerGestureTarget("B", "B", s.composerHostGeneration, true, read)
	if err != nil || idle.mode != PromptModeQueue {
		t.Fatal("busy admission mistaken for running", err)
	}
	if _, err = s.CaptureComposerGestureTarget("A", "A", s.composerHostGeneration, false, read); err == nil {
		t.Fatal("stale view admitted")
	}
	if err = a.Submit(context.Background(), "captured text", nil, 0, ""); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 1 || requests[0].SessionID != "A" || requests[0].Mode != "steer" || requests[0].Content[0]["text"] != "captured text" {
		t.Fatalf("prompt retargeted: %+v", requests)
	}
	// Attachment-only explicit accelerated action retains its captured mode.
	attachment := ComposerAttachment{Name: "fixture.png", SessionID: "A", HostScope: opposite.scope(), ImageData: "ZmFrZQ==", MediaType: "image/png"}
	if err = opposite.Submit(context.Background(), "", []ComposerAttachment{attachment}, 0, ""); err != nil {
		t.Fatal(err)
	}
	if err = idle.Submit(context.Background(), "idle", nil, 0, ""); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 3 || requests[1].Mode != "steer" || requests[1].Content[0]["type"] != "image" || requests[2].Mode != "queue" || requests[2].SessionID != "B" {
		t.Fatalf("attachment/idle mode: %+v", requests)
	}
	s.SetPromptMode(PromptModeSteer)
	s.Attach(&dsh.Client{}, nil)
	if s.Snapshot().PromptModeExplicit {
		t.Fatal("manual override crossed Host")
	}
	if err = a.Submit(context.Background(), "never", nil, 0, ""); err == nil || len(requests) != 3 {
		t.Fatal("old Host target dispatched")
	}
}

// New final-boundary fixture; previously passed scenarios are not rerun.
func TestBusyEnterReplacementHostRejectsOldGestureAndRunningRoster(t *testing.T) {
	oldHost, newHost := &dsh.Client{}, &dsh.Client{}
	s := New()
	s.Attach(oldHost, nil)
	s.selectedID = "A"
	s.items = []Item{{SessionID: "A", Running: true}}
	s.composerRosterClient = oldHost
	oldEpoch := s.Snapshot().ComposerHostGeneration
	pref := func(*dsh.Client) string { return "steer" }
	s.Attach(newHost, nil)
	epoch := s.Snapshot().ComposerHostGeneration
	if _, err := s.CaptureComposerGestureTarget("A", "A", oldEpoch, false, pref); err == nil {
		t.Fatal("old rendered gesture retargeted the replacement Host")
	}
	target, err := s.CaptureComposerGestureTarget("A", "A", epoch, false, pref)
	if err != nil || target.mode != PromptModeQueue {
		t.Fatal("old roster marked replacement Host busy", err)
	}
	s.composerRosterClient = newHost
	target, err = s.CaptureComposerGestureTarget("A", "A", epoch, false, pref)
	if err != nil || target.mode != PromptModeSteer {
		t.Fatal("current roster busy evidence lost", err)
	}
	s.composerRosterClient = nil
	s.turnStartMs = 1
	target, err = s.CaptureComposerGestureTarget("A", "A", epoch, false, pref)
	if err != nil || target.mode != PromptModeSteer {
		t.Fatal("current follow busy evidence lost", err)
	}
}
