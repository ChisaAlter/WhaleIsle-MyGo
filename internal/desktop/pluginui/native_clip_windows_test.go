//go:build windows && functionalacceptance

package pluginui

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
	"unsafe"
	"whaleisle/internal/processutil"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/pluginslots"
	"whaleisle/internal/desktop/webview"
)

func TestNativePluginSurfacePreservesViewportClipAndMenuHoles(t *testing.T) {
	if os.Getenv("WHALEISLE_PLUGIN_CLIP_CHILD") == "1" {
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestNativePluginSurfacePreservesViewportClipAndMenuHoles$", "-test.v")
	cmd.Env = append(os.Environ(), "WHALEISLE_PLUGIN_CLIP_CHILD=1")
	processutil.Background(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("native clip child: %v\n%s", err, out)
	}
}

func TestMain(m *testing.M) {
	if os.Getenv("WHALEISLE_PLUGIN_CLIP_CHILD") != "1" {
		os.Exit(m.Run())
	}
	if err := nativePluginClipFixture(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func nativePluginClipFixture() error {
	httpHost := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<!doctype html><html><body>Plugin clip fixture</body></html>"))
	}))
	defer httpHost.Close()
	capability, err := pluginslots.NewGateway(httpHost.URL, "")
	if err != nil {
		return err
	}
	defer capability.Close()
	entry, _ := capability.EntryURL(httpHost.URL + "/")
	var parent *mygo.Window
	var surface *webview.Host
	var failure error
	mygo.App.SetName("Plugin clipping fixture")
	mygo.App.WhenReady(func() {
		parent = mygo.NewWindow(mygo.WindowOptions{Title: "Owned plugin clip fixture", Width: 800, Height: 600})
		parent.ShowInactive()
		surface, failure = webview.NewPluginSurface(webview.HostOptions{Parent: parent, URL: entry, Title: "Owned plugin surface"}, capability)
		go func() {
			time.Sleep(350 * time.Millisecond)
			mygo.RunOnMain(func() {
				defer mygo.App.Quit()
				defer parent.Destroy()
				if failure != nil {
					return
				}
				defer surface.Close()
				bounds := mygo.Rectangle{X: -40, Y: -70, Width: 600, Height: 560}
				clip := mygo.Rectangle{X: 20, Y: 30, Width: 400, Height: 300}
				shown, e := surface.PresentPlugin(bounds, clip, []mygo.Rectangle{{X: 100, Y: 100, Width: 80, Height: 90}}, capability)
				if e != nil || !shown {
					failure = e
					if failure == nil {
						failure = fmt.Errorf("surface not shown parentVisible=%v minimized=%v content=%+v", parent.IsVisible(), parent.IsMinimized(), parent.ContentBounds())
					}
					return
				}
				// Select the actual owned child. A loaded page may change its title,
				// so a global title search cannot identify a capability-owned HWND.
				user32 := syscall.NewLazyDLL("user32.dll")
				gdi32 := syscall.NewLazyDLL("gdi32.dll")
				var hwnd uintptr
				for _, window := range mygo.Windows() {
					if window.Parent() == parent {
						hwnd = window.NativeHandle()
						break
					}
				}
				if hwnd == 0 {
					failure = fmt.Errorf("surface native handle unavailable")
					return
				}
				region, _, _ := gdi32.NewProc("CreateRectRgn").Call(0, 0, 0, 0)
				defer gdi32.NewProc("DeleteObject").Call(region)
				if code, _, _ := user32.NewProc("GetWindowRgn").Call(hwnd, region); code == 0 {
					failure = fmt.Errorf("surface clip region missing")
					return
				}
				dpi, _, _ := user32.NewProc("GetDpiForWindow").Call(hwnd)
				if dpi == 0 {
					dpi = 96
				}
				inside := func(x, y int) bool {
					v, _, _ := gdi32.NewProc("PtInRegion").Call(region, uintptr(x*int(dpi)/96), uintptr(y*int(dpi)/96))
					return v != 0
				}
				if inside(10, 10) || inside(145, 175) || !inside(80, 120) || inside(500, 120) {
					failure = fmt.Errorf("surface region points %v %v %v %v", inside(10, 10), inside(145, 175), inside(80, 120), inside(500, 120))
					return
				}
				var r struct{ Left, Top, Right, Bottom int32 }
				user32.NewProc("GetClientRect").Call(hwnd, uintptr(unsafe.Pointer(&r)))
				if int(r.Right-r.Left) != 600*int(dpi)/96 || int(r.Bottom-r.Top) != 560*int(dpi)/96 {
					failure = fmt.Errorf("surface viewport resized %v dpi=%d", r, dpi)
				}
			})
		}()
	})
	if err := mygo.App.Run(); err != nil {
		return err
	}
	return failure
}
