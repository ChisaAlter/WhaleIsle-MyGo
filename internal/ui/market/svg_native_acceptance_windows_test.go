//go:build windows && svg_media_acceptance

package market

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
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
			owner := &svgimage.Owner{ID: "owned-svg-acceptance", Window: parent, Context: ctx, HelperPath: helper, Valid: func() bool { return ctx.Err() == nil && !parent.IsDestroyed() }}
			// Root CSS geometry is intentionally absent from width/height attrs.
			// Go must begin with 300x150 and then accept 80x160 from this helper.
			data := []byte(`<svg xmlns="http://www.w3.org/2000/svg" style="width:80px;height:160px"><rect width="80" height="160" fill="red"><animate attributeName="fill" values="red;blue;red" dur="1s" repeatCount="indefinite"/></rect></svg>`)
			source := "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(data)
			ready := false
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Label("owned SVG artwork").Width(640).Children(func() { ready = PluginImage(c, source, 640, 0, owner) })
			}, 800, 1400)
			deadline := time.NewTimer(35 * time.Second)
			defer deadline.Stop()
			tick := time.NewTicker(40 * time.Millisecond)
			defer tick.Stop()
			for {
				tt.Frame()
				r, ok := tt.Find("owned SVG artwork")
				if ready && ok && r.W == 640 && r.H == 1280 {
					break
				}
				select {
				case <-deadline.C:
					completed <- fmt.Errorf("real helper never replaced plugin 300x150 hint with 80x160 intrinsic ratio (last=%+v,ready=%v)", r, ready)
					return
				case <-ctx.Done():
					completed <- ctx.Err()
					return
				case <-tick.C:
				}
			}
			cancel()
			tt.Frame()
			if ready {
				completed <- fmt.Errorf("revoked owner retained live PluginImage authority")
				return
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
