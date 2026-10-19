//go:build windows && svg_media_acceptance

package market

import (
	"context"
	"encoding/base64"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/imagepreview"
	"whaleisle/internal/ui/svgimage"
)

var svgNativeAcceptanceError error

// This opt-in entry uses the already-built real isolated helper and a private,
// hidden native parent. It does no screen capture, input injection, external
// fetching or app-profile reuse. Headless native painting observes geometry;
// it does not establish visible animation quality or real screen clipping.
func TestMain(m *testing.M) {
	helper := os.Getenv("WHALEISLE_SVG_ACCEPTANCE_HELPER")
	if helper != "" {
		svgNativeAcceptanceError = runSVGNativeAcceptance(helper)
	}
	os.Exit(m.Run())
}

func TestNativeSVGPluginIntrinsicAndOwnerAcceptance(t *testing.T) {
	if os.Getenv("WHALEISLE_SVG_ACCEPTANCE_HELPER") == "" {
		t.Fatal("explicit absolute trusted helper path required")
	}
	if svgNativeAcceptanceError != nil {
		t.Fatal(svgNativeAcceptanceError)
	}
}

func runSVGNativeAcceptance(helper string) error {
	if !filepath.IsAbs(helper) {
		return fmt.Errorf("absolute trusted helper path required")
	}
	info, err := os.Stat(helper)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("helper is not a regular file")
	}
	root, err := os.MkdirTemp(os.TempDir(), "whale-svg-acceptance-")
	if err != nil {
		return err
	}
	if err := os.Setenv("LOCALAPPDATA", filepath.Join(root, "browser")); err != nil {
		return err
	}
	mygo.App.SetName("WhaleIsle SVG acceptance " + filepath.Base(root))
	mygo.App.SetPath(mygo.PathUserData, filepath.Join(root, "native"))
	mygo.App.SetPath(mygo.PathCache, filepath.Join(root, "cache"))
	mygo.App.SetPath(mygo.PathLogs, filepath.Join(root, "logs"))
	completed := make(chan error, 1)
	mygo.App.OnReady(func() {
		parent := mygo.NewWindow(mygo.WindowOptions{Title: "Owned SVG metadata acceptance", Width: 800, Height: 1400, Hidden: true, Content: ui.View(func(c *ui.Context) {})})
		go func() {
			defer mygo.App.Quit()
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			cases := []struct {
				name, data string
				fractional bool
			}{
				// Actual contained Chromium reports the default object size here;
				// root CSS alone does not make this image's natural size 80x160.
				{"css-default-object", `<svg xmlns="http://www.w3.org/2000/svg" style="width:80px;height:160px"><rect width="80" height="160" fill="red"><animate attributeName="fill" values="red;blue;red" dur="1s" repeatCount="indefinite"/></rect></svg>`, false},
				// Go's provisional hint retains fractions. The browser's actual
				// naturalWidth/naturalHeight metadata uses integer image dimensions.
				{"fractional-intrinsic", `<svg xmlns="http://www.w3.org/2000/svg" width="80.5" height="160.5"><rect width="80" height="160" fill="red"><animate attributeName="fill" values="red;blue;red" dur="1s" repeatCount="indefinite"/></rect></svg>`, true},
			}
			for _, tc := range cases {
				caseCtx, stopCase := context.WithCancel(ctx)
				owner := &svgimage.Owner{ID: "owned-svg-acceptance/" + tc.name, Window: parent, Context: caseCtx, HelperPath: helper, Valid: func() bool { return caseCtx.Err() == nil && !parent.IsDestroyed() }}
				if err := checkNativePluginSVG(caseCtx, owner, []byte(tc.data), tc.fractional, stopCase); err != nil {
					stopCase()
					completed <- fmt.Errorf("%s: %w", tc.name, err)
					return
				}
				stopCase()
			}
			completed <- nil
		}()
	})
	if err := mygo.App.Run(); err != nil {
		return err
	}
	select {
	case err := <-completed:
		return err
	default:
		return fmt.Errorf("native app exited before SVG acceptance completed")
	}
}

func checkNativePluginSVG(ctx context.Context, owner *svgimage.Owner, data []byte, fractional bool, revoke context.CancelFunc) error {
	hintW, hintH, err := imagepreview.SVGImageSize(data)
	if err != nil || !imagepreview.NeedsBrowserSVG(data) {
		return fmt.Errorf("fixture must enter actual browser SVG path: %v", err)
	}
	source := "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(data)
	ready := false
	var decoded *readmeDecodedImage
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Label("owned SVG artwork").Width(640).Children(func() { ready, decoded = pluginImage(c, source, 640, 0, owner) })
	}, 800, 1400)
	deadline := time.NewTimer(35 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(40 * time.Millisecond)
	defer tick.Stop()
	for {
		tt.Frame()
		r, ok := tt.Find("owned SVG artwork")
		if decoded != nil && decoded.svgSeat != nil {
			seat := decoded.svgSeat
			status := seat.LoadStatus()
			w, h := seat.Size()
			if seat.Error() != nil {
				return fmt.Errorf("helper load: %w (%+v)", seat.Error(), status)
			}
			if status.Loaded && !status.Pending && !status.Retired {
				if fractional {
					if w == hintW && h == hintH || math.Trunc(float64(w)) != float64(w) || math.Trunc(float64(h)) != float64(h) || math.Abs(float64(w-hintW)) > 1 || math.Abs(float64(h-hintH)) > 1 {
						return fmt.Errorf("helper did not refine fractional hint %gx%g with browser natural dimensions: %gx%g", hintW, hintH, w, h)
					}
				} else if w != 300 || h != 150 {
					return fmt.Errorf("CSS-only object size differs from observed Chromium baseline: %gx%g", w, h)
				}
				if ready && ok && r.W == 640 && math.Abs(float64(r.H-640*h/w)) < .1 {
					fmt.Printf("SVG native loaded: hint=%gx%g actual=%gx%g rect=%+v load=%+v\n", hintW, hintH, w, h, r, status)
					break
				}
			}
		}
		select {
		case <-deadline.C:
			return fmt.Errorf("helper did not complete with fitted intrinsic geometry (rect=%+v ready=%v)", r, ready)
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
	}
	revoke()
	tt.Frame()
	if ready || decoded.svgSeat.LoadStatus().Loaded {
		return fmt.Errorf("revoked owner retained live PluginImage authority")
	}
	decoded.svgSeat.Close()
	return nil
}
