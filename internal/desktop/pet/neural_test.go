package pet

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNeuralResourceBoundary(t *testing.T) {
	if _, err := loadRenderAssets(t.TempDir()); err == nil {
		t.Fatal("missing resources admitted")
	}
	p := &Companion{neuralActive: true, renderAssets: map[string][]byte{"/render/avatar/model.onnx": []byte("synthetic")}}
	for _, tc := range []struct {
		path   string
		status int
	}{
		{"/render/avatar/model.onnx", 200}, {"/render/avatar/other.onnx", 404},
		{"/render/../config.json", 404}, {"/render/ort/ort.all.min.js?extra=1", 404},
		{"/task.json", 200}, {"/tha4.js", 200}, {"/pet-runtime.js", 200},
	} {
		r := httptest.NewRecorder()
		p.ServeHTTP(r, httptest.NewRequest("GET", "whale-pet://localhost"+tc.path, nil))
		if r.Code != tc.status {
			t.Fatalf("%s: %d", tc.path, r.Code)
		}
		if strings.Contains(r.Header().Get("Content-Security-Policy"), "connect-src *") {
			t.Fatal("broad network access")
		}
	}
	p.close()
	r := httptest.NewRecorder()
	p.ServeHTTP(r, httptest.NewRequest("GET", "whale-pet://localhost/render/avatar/model.onnx", nil))
	if r.Code != 404 {
		t.Fatal("closed renderer still publishes assets")
	}
}
