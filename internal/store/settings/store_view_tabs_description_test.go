package settings

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"whaleisle/internal/dsh"
)

type viewTabsDescriptionTransport func(*http.Request) (*http.Response, error)

func (f viewTabsDescriptionTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestViewTabsDescriptionTracksAcceptedCurrentClient(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		var fail atomic.Bool
		var gate chan struct{}
		client := func() *dsh.Client {
			return &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://description-owner.invalid"}, HTTP: &http.Client{Transport: viewTabsDescriptionTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if gate != nil {
					<-gate
				}
				if fail.Load() {
					return nil, errors.New("fixture describe refused")
				}
				var request map[string]any
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					return nil, err
				}
				recorder := httptest.NewRecorder()
				value := map[string]any{"writable": true, "namespaces": []any{}}
				_ = json.NewEncoder(recorder).Encode(map[string]any{"type": "server-response", "rpcId": request["rpcId"], "result": map[string]any{"ok": true, "value": value}})
				return recorder.Result(), nil
			})}}
		}
		s := New()
		a := client()
		s.Attach(a, nil)
		if s.Snapshot().Described {
			t.Fatal("Attach manufactured an accepted description")
		}
		s.EnsureLoaded(context.Background())
		synctest.Wait()
		if snap := s.Snapshot(); !snap.Described || len(snap.Namespaces) != 0 {
			t.Fatalf("successful empty describe not loaded: %+v", snap)
		}
		s.EnsureLoaded(context.Background())
		synctest.Wait()
		if calls.Load() != 1 {
			t.Fatal("successful empty description reloaded")
		}
		s.Attach(a, nil)
		s.EnsureLoaded(context.Background())
		synctest.Wait()
		if !s.Snapshot().Described || calls.Load() != 1 {
			t.Fatal("same client Attach erased accepted description")
		}
		b := client()
		s.Attach(b, nil)
		if s.Snapshot().Described {
			t.Fatal("replacement inherited old client's description")
		}
		gate = make(chan struct{})
		s.EnsureLoaded(context.Background())
		synctest.Wait()
		if snap := s.Snapshot(); snap.Described || !snap.Loading {
			t.Fatal("pending describe granted authority")
		}
		close(gate)
		synctest.Wait()
		gate = nil
		if !s.Snapshot().Described {
			t.Fatal("replacement description not accepted")
		}
		fail.Store(true)
		s.RefreshDescribe(context.Background())
		synctest.Wait()
		if snap := s.Snapshot(); snap.Described || snap.Err == "" {
			t.Fatal("failed refresh kept description-ready")
		}
		before := calls.Load()
		s.EnsureLoaded(context.Background())
		synctest.Wait()
		if calls.Load() != before {
			t.Fatal("EnsureLoaded retried a refused description")
		}
		s.Detach()
		if snap := s.Snapshot(); snap.Described || snap.ClientReady {
			t.Fatal("Detach retained description authority")
		}
		fail.Store(false)
		s.Attach(b, nil)
		s.EnsureLoaded(context.Background())
		synctest.Wait()
		if !s.Snapshot().Described || calls.Load() != before+1 {
			t.Fatal("reconnected client did not describe again")
		}
	})
}

func TestViewTabsDescriptionRejectsStaleArrival(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		gate := make(chan struct{})
		a := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://old-description.invalid"}, HTTP: &http.Client{Transport: viewTabsDescriptionTransport(func(r *http.Request) (*http.Response, error) {
			<-gate
			var request map[string]any
			_ = json.NewDecoder(r.Body).Decode(&request)
			recorder := httptest.NewRecorder()
			rpcOK(recorder, request["rpcId"], map[string]any{"writable": true, "namespaces": []any{}})
			return recorder.Result(), nil
		})}}
		s := New()
		s.Attach(a, nil)
		s.EnsureLoaded(context.Background())
		synctest.Wait()
		s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://new-description.invalid"}}, nil)
		close(gate)
		synctest.Wait()
		if snap := s.Snapshot(); snap.Described || len(snap.Namespaces) != 0 {
			t.Fatal("old response described replacement")
		}
	})
}
