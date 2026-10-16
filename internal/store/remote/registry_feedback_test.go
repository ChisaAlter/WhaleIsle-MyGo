package remote

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"whaleisle/internal/dsh"
)

// Selecting another machine is committed before the separate status request.
// A status outage must not leave the old target displayed for Connect.
func TestRegistrySelectionSurvivesStatusFailure(t *testing.T) {
	c := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://fixture.invalid"}}
	c.HTTP = &http.Client{Transport: removalTransport(func(r *http.Request) (*http.Response, error) {
		data := `{"ok":true}`
		switch r.URL.Path {
		case "/api/dsh-remote/machines":
			data = `{"machines":[{"id":"two","name":"New target","host":"new.invalid","port":22,"username":"dev"}],"currentId":"two"}`
		case "/api/dsh-remote/status":
			data = `{"ok":false,"error":"fixture status unavailable"}`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(data))}, nil
	})}
	s := New()
	s.Attach(c, nil)
	s.view.Machines = []Machine{{ID: "one", Name: "Old target"}}
	s.view.CurrentID = "one"
	s.view.Connected = true
	s.view.Status = ConnectionStatus{Connected: true, Workspace: "/old", HostKeyKnown: true}
	s.view.Tools = ToolsSnapshot{Loaded: true, Forwards: []Forward{{ID: "old-forward", MachineID: "one"}}}
	s.Select("two")
	v := waitStore(t, s)
	if v.CurrentID != "two" || len(v.Machines) != 1 || v.Machines[0].Host != "new.invalid" {
		t.Fatalf("status failure discarded authoritative registry: %+v", v)
	}
	if v.Connected || v.Status != (ConnectionStatus{}) || v.Tools.Loaded || len(v.Tools.Forwards) != 0 {
		t.Fatalf("old target connection metadata survived: %+v", v)
	}
	if !v.MutationApplied || !strings.Contains(v.Error, "fixture status unavailable") {
		t.Fatalf("lost mutation acknowledgement or status error: %+v", v)
	}
}
