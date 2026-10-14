package htmlpreview

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUnavailableGate(t *testing.T) {
	if Available() {
		t.Fatal("unverified renderer must not advertise availability")
	}
	cases := []struct {
		name, body string
		want       int
	}{
		{"valid prepared HTML", `{"html":"<html><body>fixture</body></html>","width":728,"appearance":"dark"}`, 503},
		{"minimum width", `{"html":"x","width":240,"appearance":"light"}`, 503},
		{"maximum width", `{"html":"x","width":1600,"appearance":"dark"}`, 503},
		{"empty", `{"html":"  ","width":728,"appearance":"dark"}`, 400},
		{"fractional", `{"html":"x","width":240.5,"appearance":"dark"}`, 400},
		{"small", `{"html":"x","width":239,"appearance":"dark"}`, 400},
		{"large", `{"html":"x","width":1601,"appearance":"dark"}`, 400},
		{"appearance", `{"html":"x","width":728,"appearance":"auto"}`, 400},
		{"trailing JSON", `{"html":"x","width":728,"appearance":"dark"} {}`, 400},
		{"null", `null`, 400},
		{"oversized prepared HTML", `{"html":"` + strings.Repeat("x", MaxPreparedHTMLBytes+1) + `","width":728,"appearance":"dark"}`, 400},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/desktop/html-preview", strings.NewReader(tt.body))
			Handler().ServeHTTP(w, r)
			if w.Code != tt.want {
				t.Fatalf("status %d, want %d: %s", w.Code, tt.want, w.Body.String())
			}
			if strings.Contains(w.Body.String(), `"png"`) {
				t.Fatal("must never return a fake capture")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w := httptest.NewRecorder()
	Handler().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/desktop/html-preview", strings.NewReader(`{}`)).WithContext(ctx))
	if w.Body.Len() != 0 {
		t.Fatal("cancelled request wrote a response")
	}
}
