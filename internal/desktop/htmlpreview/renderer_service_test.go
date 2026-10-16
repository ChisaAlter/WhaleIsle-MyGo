package htmlpreview

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRendererServiceUsesItsOwnReadiness(t *testing.T) {
	before := Readiness()
	fail := false
	calls := 0
	service := NewRendererService(func(_ context.Context, html string, width int, appearance string) (CaptureResult, error) {
		calls++
		if fail {
			return CaptureResult{}, errors.New("helper unavailable")
		}
		if calls == 1 && (html != probeHTML || width != 240 || appearance != "dark") {
			t.Fatal("probe must use fixed fixture")
		}
		return CaptureResult{Width: width, ContentHeight: 1, CapturedHeight: 1, PNG: "fixture", ConsoleMessages: []ConsoleMessage{{Text: "preview-probe-passed:opaque/no-native/no-storage/no-rtc"}}}, nil
	})
	request := func(method, path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		service.Handler().ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
		return w
	}
	body := `{"html":"<p>fixture</p>","width":240,"appearance":"dark"}`
	if w := request(http.MethodPost, "/desktop/html-preview", body); w.Code != 503 || calls != 0 {
		t.Fatal("unverified renderer invoked")
	}
	if !service.Probe(context.Background()).Available {
		t.Fatal("helper probe did not verify")
	}
	if w := request(http.MethodGet, "/desktop/html-preview/status", ""); !strings.Contains(w.Body.String(), `"available":true`) {
		t.Fatal(w.Body.String())
	}
	if w := request(http.MethodPost, "/desktop/html-preview", body); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if Readiness() != before {
		t.Fatal("helper must not grant native readiness")
	}
	fail = true
	if w := request(http.MethodPost, "/desktop/html-preview", body); w.Code != 503 {
		t.Fatal("helper failure ignored")
	}
	if service.Readiness().Available {
		t.Fatal("failed helper remains ready")
	}
}

func TestRendererServiceRejectsMissingProbeAttestation(t *testing.T) {
	service := NewRendererService(func(context.Context, string, int, string) (CaptureResult, error) {
		return CaptureResult{Width: 240, ContentHeight: 1, CapturedHeight: 1, PNG: "fixture"}, nil
	})
	if service.Probe(context.Background()).Available {
		t.Fatal("capture alone must not prove isolation")
	}
}
