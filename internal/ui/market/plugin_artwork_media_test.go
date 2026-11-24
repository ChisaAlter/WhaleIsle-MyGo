package market

import (
	"encoding/base64"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/media"
	"whaleisle/internal/desktop/svgrender"
)

// The 插件 page's artwork: the build path only reads handles. Every frame
// before the decode runs shows the caller's fallback, nothing decodes on the
// UI thread, and after one decode per icon the page rebuilds without any more.
func TestPluginArtworkBuildPathPerformsNoDecode(t *testing.T) {
	s := useTestMedia(t, nil)
	files, err := filepath.Glob("../../desktop/media/testdata/plugin-icons/*.svg")
	if err != nil || len(files) != 14 {
		t.Fatalf("plugin icon fixtures: %d %v", len(files), err)
	}
	var sources []string
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		sources = append(sources, "data:image/svg+xml;base64,"+base64.StdEncoding.EncodeToString(data))
	}
	unique := map[string]bool{}
	for _, source := range sources {
		unique[source] = true
	}
	// Identical icons (inspector / inspector-profile, badge bundles) share one decode.
	want := int64(len(unique))
	owner := s.NewOwner(nil, nil)
	shown := 0
	tt := ui.NewTester(func(c *ui.Context) {
		shown = 0
		ui.Column(c).Fill().Children(func() {
			for _, source := range sources {
				if PluginImage(c, source, 36, 36, owner) {
					shown++
				}
			}
		})
	}, 400, 900)
	for range 3 {
		tt.Frame()
	}
	if shown != 0 || s.Decodes() != 0 || int64(s.Queued()) != want {
		t.Fatalf("build path decoded: shown=%d decodes=%d queued=%d", shown, s.Decodes(), s.Queued())
	}
	fonts := svgrender.FontLoads()
	s.RunPending()
	tt.Frame()
	if shown != len(sources) || s.Decodes() != want {
		t.Fatalf("after one decode each: shown=%d decodes=%d", shown, s.Decodes())
	}
	if s.Rasters() != 0 || svgrender.FontLoads() != fonts {
		t.Fatalf("plugin icons rasterised (%d) or loaded fonts", s.Rasters())
	}
	// Revisiting the page (a new owner, a fresh tester) is a cache hit.
	revisit := s.NewOwner(nil, nil)
	owner.Release()
	owner = revisit
	for range 3 {
		tt.Frame()
	}
	if s.Decodes() != want || s.Queued() != 0 {
		t.Fatalf("revisit decoded again: %d", s.Decodes())
	}
}

// SMIL artwork that once needed a hidden browser now renders its static
// first frame natively; no owner or helper process is involved.
func TestPluginArtworkAnimatedSVGRendersStaticFrame(t *testing.T) {
	body := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="30" height="20"><rect width="30" height="20"><animate attributeName="opacity" values="0;1" dur="1s"/></rect></svg>`)
	cache := readmeImageCache(t)
	s := marketMedia()
	cache.SetHTTPClient(&http.Client{Transport: readmeImageTransport(func(*http.Request) (*http.Response, error) {
		t.Error("cached fixture attempted HTTP")
		return nil, errors.New("fixture forbids network")
	})})
	remote := "https://example.invalid/animated-meta.svg"
	if err := os.WriteFile(cache.FilePath(remote), body, 0600); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{remote, "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(body)} {
		accepted := false
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Children(func() {
				accepted = PluginImage(c, source, 30, 20, nil)
				if !accepted {
					ui.Text(c, "Default plugin artwork")
				}
			})
		}, 160, 100)
		if accepted || !tt.HasText("Default plugin artwork") {
			t.Fatal("artwork shown before its decode ran")
		}
		settleMedia(t, tt, s, func() bool { return accepted })
	}
}

func TestPluginArtworkUnsupportedKeepsFallback(t *testing.T) {
	s := useTestMedia(t, nil)
	source := "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(`<!DOCTYPE svg><svg/>`))
	accepted := true
	tt := ui.NewTester(func(c *ui.Context) {
		accepted = PluginImage(c, source, 30, 20, nil)
	}, 100, 100)
	s.RunPending()
	tt.Frame()
	h := s.Request(nil, media.Key{ID: source, W: 60, H: 40}, media.Source{})
	if accepted || h.State() != media.Failed || h.Failure() != media.FailUnsupported {
		t.Fatalf("unsupported SVG: accepted=%v state=%v failure=%v", accepted, h.State(), h.Failure())
	}
}
