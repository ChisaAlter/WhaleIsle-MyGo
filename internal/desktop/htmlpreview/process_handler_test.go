package htmlpreview

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProcessRendererRoute(t *testing.T) {
	before := Readiness()
	calls := 0
	handler := HandlerWithRenderer(func(ctx context.Context, html string, width int, appearance string) (CaptureResult, error) {
		calls++
		if html != "<p>fixture</p>" || width != 240 || appearance != "dark" {
			t.Fatal("changed prepared input")
		}
		return CaptureResult{PNG: "fixture", Width: width, ContentHeight: 1, CapturedHeight: 1}, nil
	})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/desktop/html-preview", strings.NewReader(`{"html":"<p>fixture</p>","width":240,"appearance":"dark"}`)))
	if w.Code != 200 || calls != 1 || !strings.Contains(w.Body.String(), `"png":"fixture"`) {
		t.Fatalf("capture response %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/desktop/html-preview", strings.NewReader(`{"html":"x","width":1,"appearance":"dark"}`)))
	if w.Code != 400 || calls != 1 {
		t.Fatal("validation bypassed")
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/desktop/html-preview/status", nil))
	if w.Code != 200 || calls != 1 || Readiness() != before {
		t.Fatal("capture changed global readiness")
	}
}
