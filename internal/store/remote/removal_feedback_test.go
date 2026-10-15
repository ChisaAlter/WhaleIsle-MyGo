package remote

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"whaleisle/internal/dsh"
)

type removalTransport func(*http.Request) (*http.Response, error)

func (f removalTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestUpstreamDeltaRemovalFeedback(t *testing.T) {
	var mu sync.Mutex
	fail, refreshFail, stillPresent := true, false, false
	posts := 0
	c := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://fixture.invalid"}}
	c.HTTP = &http.Client{Transport: removalTransport(func(r *http.Request) (*http.Response, error) {
		mu.Lock()
		defer mu.Unlock()
		data := `{"connected":false}`
		if strings.HasSuffix(r.URL.Path, "machines") {
			if r.Method == "POST" {
				var body map[string]string
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body["action"] != "delete" || body["id"] != "one" {
					t.Errorf("unexpected request: %+v", body)
				}
				posts++
				data = `{"ok":true}`
				if fail {
					data = `{"ok":false,"error":"fixture refused"}`
				}
			} else if refreshFail {
				data = `{"ok":false,"error":"fixture refresh failed"}`
			} else if stillPresent {
				data = `{"machines":[{"id":"one","name":"Dev"}],"currentId":"one"}`
			} else {
				data = `{"machines":[],"currentId":""}`
			}
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(data))}, nil
	})}
	s := New()
	s.Attach(c, nil)
	s.view.Machines = []Machine{{ID: "one", Name: "Dev"}}
	s.view.CurrentID = "one"
	generation := s.Snapshot().Generation
	if !s.RemoveAtGeneration("one", generation) {
		t.Fatal("initial removal rejected")
	}
	v := waitStore(t, s)
	if v.Error == "" || v.RemovedID != "" || len(v.Machines) != 1 || v.MutationApplied {
		t.Fatalf("failed removal lost target: %+v", v)
	}
	mu.Lock()
	fail = false
	refreshFail = true
	mu.Unlock()
	if !s.RemoveAtGeneration("one", generation) {
		t.Fatal("retry rejected")
	}
	v = waitStore(t, s)
	if v.RemovedID != "one" || len(v.Machines) != 0 || v.CurrentID != "" || v.Error == "" || !strings.Contains(v.Notice, "Deleted") {
		t.Fatalf("acknowledged delete/failed refresh: %+v", v)
	}
	s.Attach(c, nil)
	if s.RemoveAtGeneration("one", generation) {
		t.Fatal("old confirmation applied to replacement host")
	}
	mu.Lock()
	n := posts
	refreshFail = false
	stillPresent = true
	mu.Unlock()
	if n != 2 {
		t.Fatalf("unexpected removal count %d", n)
	}
	s.view.Machines = []Machine{{ID: "one", Name: "Dev"}}
	if !s.RemoveAtGeneration("one", s.Snapshot().Generation) {
		t.Fatal("fresh confirmation rejected")
	}
	v = waitStore(t, s)
	if v.RemovedID != "" || v.Error == "" || len(v.Machines) != 1 {
		t.Fatalf("unverified removal claimed success: %+v", v)
	}
	mu.Lock()
	stillPresent = false
	mu.Unlock()
	s.Remove("one")
	v = waitStore(t, s)
	if v.Error != "" || v.RemovedID != "one" || len(v.Machines) != 0 || v.Notice == "" {
		t.Fatalf("success: %+v", v)
	}
}

// A later status error must not discard contradictory registry evidence.
func TestUpstreamDeltaRemovalStatusFailure(t *testing.T) {
	c := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://fixture.invalid"}}
	c.HTTP = &http.Client{Transport: removalTransport(func(r *http.Request) (*http.Response, error) {
		data := `{"ok":true}`
		if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "machines") {
			data = `{"machines":[{"id":"one","name":"Dev"}],"currentId":"one"}`
		}
		if strings.HasSuffix(r.URL.Path, "status") {
			data = `{"ok":false,"error":"fixture status unavailable"}`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(data))}, nil
	})}
	s := New()
	s.Attach(c, nil)
	s.view.Machines = []Machine{{ID: "one", Name: "Dev"}}
	s.Remove("one")
	v := waitStore(t, s)
	if v.RemovedID != "" || v.Notice != "" || len(v.Machines) != 1 || !strings.Contains(v.Error, "still lists") {
		t.Fatalf("status error hid contradictory registry: %+v", v)
	}
}
