package session

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

func TestOldEventsReadyCannotCancelNewSpeechFollow(t *testing.T) {
	s := New()
	client := &dsh.Client{}
	s.Attach(client, nil)
	s.mux = &dsh.Mux{}
	old := s.BeginEvents(context.Background(), client)
	current := s.BeginEvents(context.Background(), client)
	var canceled atomic.Int32
	s.speechFollowCancel = func() { canceled.Add(1) }
	s.speechReadiness = speechReadinessState{client: client, mux: s.mux, generation: 80, view: SpeechSettingsView{Connected: true}}
	s.WatchSpeechCatalogForEvents(old)
	if canceled.Load() != 0 || s.speechReadiness.generation != 80 || !s.speechReadiness.view.Connected {
		t.Fatal("old Ready retired current readiness")
	}
	// Same client/generation is insufficient after the mux has been replaced.
	s.mux = &dsh.Mux{}
	s.WatchSpeechCatalogForEvents(current)
	if canceled.Load() != 0 || s.speechReadiness.generation != 80 {
		t.Fatal("old mux Ready retired current readiness")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.WatchSpeechCatalog(ctx)
	if canceled.Load() != 0 {
		t.Fatal("canceled startup watcher canceled active readiness")
	}
}

func TestSpeechPreparationReceiptKeepsNewOwnerOperationAndSource(t *testing.T) {
	type request struct {
		Path    string
		Args    json.RawMessage
		Release chan struct{}
		Done    chan struct{}
	}
	requests := make(chan request, 4)
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://offline.invalid"}, HTTP: &http.Client{Transport: openInAppTestTransport(func(r *http.Request) (*http.Response, error) {
		var wire struct {
			RPCID   string `json:"rpcId"`
			Payload struct {
				Args json.RawMessage `json:"args"`
			} `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&wire); err != nil {
			t.Error(err)
		}
		q := request{r.URL.Path, wire.Payload.Args, make(chan struct{}), make(chan struct{})}
		t.Cleanup(func() {
			select {
			case <-q.Release:
			default:
				close(q.Release)
			}
		})
		requests <- q
		<-q.Release
		body, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": wire.RPCID, "result": map[string]any{"ok": true, "value": map[string]any{}}})
		close(q.Done)
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(string(body))), Request: r}, nil
	})}}
	s := New()
	s.Attach(client, nil)
	install := func(generation uint64) {
		s.mu.Lock()
		s.speechReadiness = speechReadinessState{client: client, generation: generation, ctx: context.Background(), view: SpeechSettingsView{Connected: true}}
		s.mu.Unlock()
	}
	next := func() request {
		t.Helper()
		select {
		case q := <-requests:
			return q
		case <-time.After(2 * time.Second):
			t.Fatal("offline speech operation not sent")
			return request{}
		}
	}
	install(1)
	old, _ := s.BindSpeechSettings()
	old.Prepare("local", "mirror")
	first := next()
	var firstArgs struct {
		ProviderID string `json:"providerId"`
		Options    struct {
			DownloadSource string `json:"downloadSource"`
		} `json:"options"`
	}
	if err := json.Unmarshal(first.Args, &firstArgs); err != nil || first.Path != "/api/speech/prepare" || firstArgs.ProviderID != "local" || firstArgs.Options.DownloadSource != "mirror" {
		t.Fatalf("preparation source lost: %s %s", first.Path, first.Args)
	}
	install(2)
	fresh, _ := s.BindSpeechSettings()
	fresh.Prepare("local", "official")
	if old.Key() != fresh.Key() {
		t.Fatal("same-Host reconnect retired page-local download source choice")
	}
	second := next()
	close(first.Release)
	<-first.Done
	// Return the old carrier while the replacement remains in flight. A
	// subsequent stale page gesture must not admit another RPC either.
	old.Configure("other", "zh")
	_, v := s.BindSpeechSettings()
	if !v.Submitting["local"] || v.Error != "" {
		t.Fatal("old receipt cleared newer preparation")
	}
	close(second.Release)
	deadline := time.Now().Add(2 * time.Second)
	for {
		_, v = s.BindSpeechSettings()
		if !v.Submitting["local"] {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("current preparation stayed submitting")
		}
		time.Sleep(time.Millisecond)
	}
	fresh.Configure("local", "zh")
	configure := next()
	if configure.Path != "/api/speech/configure" || !strings.Contains(string(configure.Args), `"language":"zh"`) {
		t.Fatalf("selection patch: %s %s", configure.Path, configure.Args)
	}
	close(configure.Release)
	fresh.CancelPreparation("local")
	cancelRequest := next()
	if cancelRequest.Path != "/api/speech/cancelPreparation" || string(cancelRequest.Args) != `{"providerId":"local"}` {
		t.Fatalf("cancel target: %s %s", cancelRequest.Path, cancelRequest.Args)
	}
	close(cancelRequest.Release)
	s.Attach(&dsh.Client{}, nil)
	replacement, _ := s.BindSpeechSettings()
	if replacement.Key() == fresh.Key() {
		t.Fatal("Host replacement retained page-local choice owner")
	}
	fresh.Configure("other", "en")
	select {
	case q := <-requests:
		close(q.Release)
		t.Fatalf("stale owner sent unexpected %s", q.Path)
	default:
	}
}

func TestSpeechRecordingPreparationCapturesSelectionWithoutInstalling(t *testing.T) {
	for _, phase := range []string{"unprepared", "preparing", "failed", "ready", "standby", "waking"} {
		t.Run(phase, func(t *testing.T) {
			var calls atomic.Int32
			client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://offline.invalid"}, HTTP: &http.Client{Transport: openInAppTestTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.URL.Path != "/api/speech/catalog" {
					t.Errorf("recording implicitly called %s", r.URL.Path)
				}
				var wire struct {
					RPCID string `json:"rpcId"`
				}
				_ = json.NewDecoder(r.Body).Decode(&wire)
				body, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": wire.RPCID, "result": map[string]any{"ok": true, "value": map[string]any{"maxAudioBytes": 32044, "maxDurationSeconds": 30, "selection": map[string]string{"providerId": "local", "language": "zh"}, "providers": []map[string]any{{"id": "local", "preparation": map[string]string{"phase": phase}}}}}})
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(string(body))), Request: r}, nil
			})}}
			s := New()
			s.Attach(client, nil)
			s.selectedID = "A"
			plan, err := s.PrepareSpeechRecording(context.Background())
			ready := phase == "ready" || phase == "standby" || phase == "waking"
			if (err == nil) != ready {
				t.Fatalf("%s admission: %v", phase, err)
			}
			if ready {
				if plan.ProviderID != "local" || plan.Language != "zh" || plan.SessionID != "A" || plan.Limits.MaxBytes != 32044 || plan.Limits.MaxDuration != time.Second {
					t.Fatalf("catalog selection/PCM budget lost: %+v", plan)
				}
				s.mu.Lock()
				s.selectSessionLocked("B")
				s.mu.Unlock()
				if plan.Current() {
					t.Fatal("recording survived Session replacement")
				}
				if _, err := plan.Transcribe(context.Background(), nil); err == nil {
					t.Fatal("retired recording accepted audio")
				}
			}
			if calls.Load() != 1 {
				t.Fatalf("preflight installed/transcribed implicitly: %d calls", calls.Load())
			}
		})
	}
}
