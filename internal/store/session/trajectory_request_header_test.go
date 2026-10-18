package session

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTrajectoryRequestHeaderTypedRetentionAndIsolation(t *testing.T) {
	s := New()
	s.captureTrajectoryBoundaryLocked([]byte(`{"type":"request/header","seq":4,"data":{"reason":"initial","header":{"config":{"model":"m","provider":"p","authorization":"SECRET"},"tools":[{"name":"shell","description":"Run shell","parameters":{"type":"object"},"token":"SECRET"}],"authentication":"SECRET"}}}`))
	headers := s.trajectoryRequestHeadersLocked()
	if len(headers) != 1 || headers[0].Header.Config.Model != "m" || len(headers[0].Header.Tools) != 1 {
		t.Fatalf("headers %+v", headers)
	}
	raw, _ := json.Marshal(headers)
	if strings.Contains(string(raw), "SECRET") {
		t.Fatal("unknown authentication retained")
	}
	headers[0].Header.Tools[0].Parameters["type"] = "changed"
	if s.trajectoryRequestHeadersLocked()[0].Header.Tools[0].Parameters["type"] != "object" {
		t.Fatal("snapshot aliases stored boundary")
	}
	s.captureTrajectoryBoundaryLocked([]byte(`{"type":"request/header","seq":8,"data":{"header":{"tools":[]}}}`))
	headers = s.trajectoryRequestHeadersLocked()
	if len(headers) != 2 || len(headers[1].Header.Tools) != 0 {
		t.Fatal("empty catalog did not clear historical schemas")
	}
	s.invalidateFollowLocked()
	if len(s.trajectoryRequestHeadersLocked()) != 0 {
		t.Fatal("header survived follow invalidation")
	}
}
