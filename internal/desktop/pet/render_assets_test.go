package pet

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"
)

// The executable reads THA4 resources only from pet-render-assets/ beside
// itself (neural.go:55), and scripts/pet-render-assets.mjs is the only writer
// of that directory: it verifies every manifest entry before writing and emits
// the provenance sidecar. A staged copy is therefore the packaging contract's
// product, and this fixture exercises the runtime side of that contract.
//
// Point WHALEISLE_PET_RENDER_ASSETS at a staged directory to run it. Without
// that input the fixture skips, so a checkout without the (never committed)
// binary assets cannot report a vacuous pass.
func TestStagedRenderAssetsServeThroughPublishedRoutes(t *testing.T) {
	dir := os.Getenv("WHALEISLE_PET_RENDER_ASSETS")
	if dir == "" {
		t.Skip("WHALEISLE_PET_RENDER_ASSETS not set")
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("staged directory: %v", err)
	}
	loaded, err := loadRenderAssets(dir)
	if err != nil {
		t.Fatalf("staged THA4 resources rejected: %v", err)
	}
	raw, err := assets.ReadFile("assets/render-manifest.json")
	if err != nil {
		t.Fatalf("embedded manifest: %v", err)
	}
	var manifest struct {
		Files []renderAsset `json:"files"`
	}
	if err = json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("embedded manifest: %v", err)
	}
	// Licenses are validated for packaging but never published: the resource
	// route set stays derived from executable content only.
	published := 0
	for _, file := range manifest.Files {
		route := "/render/" + file.Path
		body, ok := loaded[route]
		if file.Path == "LICENSE" || file.Path == "ort/LICENSE" {
			if ok {
				t.Fatalf("%s must not be published", file.Path)
			}
			continue
		}
		if !ok {
			t.Fatalf("%s missing from the staged resources", file.Path)
		}
		if int64(len(body)) != file.Size {
			t.Fatalf("%s staged with %d bytes, manifest pins %d", file.Path, len(body), file.Size)
		}
		published++
	}
	if published != 7 || len(loaded) != published {
		t.Fatalf("published %d routes for %d manifest entries, want exactly the 7 runtime/model files", len(loaded), published)
	}
	// Serving goes through the same bare dispatcher the scheme forwards to, so
	// a route name or content-type regression fails here rather than in the
	// WebView2 page.
	p := &Companion{neuralActive: true, renderAssets: loaded}
	defer p.close()
	for _, file := range manifest.Files {
		if file.Path == "LICENSE" || file.Path == "ort/LICENSE" {
			continue
		}
		r := httptest.NewRecorder()
		p.ServeHTTP(r, httptest.NewRequest("GET", "whale-pet://localhost/render/"+file.Path, nil))
		if r.Code != 200 || int64(r.Body.Len()) != file.Size {
			t.Fatalf("%s: code=%d body=%d want 200/%d", file.Path, r.Code, r.Body.Len(), file.Size)
		}
		if contentType := r.Header().Get("Content-Type"); contentType == "" {
			t.Fatalf("%s served without a content type", file.Path)
		}
	}
}
