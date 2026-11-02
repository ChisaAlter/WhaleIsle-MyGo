//go:build windows && (amd64 || arm64)

package windows

import (
	"os"
	"runtime"
	"testing"
	"unsafe"

	"github.com/egoist/mygo/internal/platform"
)

type roundedFrameFixtureWindow struct{ previewNativeWindow }

func (*roundedFrameFixtureWindow) SurfaceEvent(platform.SurfaceEvent) bool { return false }

// TestRoundedFrameLeavesTheSilhouetteToTheContent pins the one-contour rule: a
// transparent window with a frame radius draws its own silhouette (the 20 DIP
// clip plus the app's hairline ring), so DWM must neither round the frame nor
// paint a non-client frame under it. DWMWCP_ROUND gave the corners a second,
// 8 DIP system outline with a light wedge between the two curves.
func TestRoundedFrameLeavesTheSilhouetteToTheContent(t *testing.T) {
	if os.Getenv("MYGO_ROUNDED_FRAME_NATIVE_TEST") != "1" {
		t.Skip("set MYGO_ROUNDED_FRAME_NATIVE_TEST=1 in a Windows desktop session")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	oldThread, oldBackend := mainThreadID, theBackend
	mainThreadID = currentThreadID()
	defer func() { mainThreadID = oldThread; theBackend = oldBackend }()
	if !windows11() {
		t.Skip("the system corner preference exists on Windows 11 only")
	}
	b := New()
	if err := b.Init(previewNativeApp{}, platform.AppOptions{Name: "MyGo isolated rounded frame fixture"}); err != nil {
		t.Fatal(err)
	}
	defer procDestroyWindow.Call(b.appHwnd)
	created, err := b.NewWindow(&platform.WindowOptions{Surface: true, Transparent: true, WindowsFrameCornerRadius: 20, Title: "MyGo isolated rounded frame fixture", Width: 320, Height: 180, Closable: true, Movable: true, Resizable: true}, &roundedFrameFixtureWindow{})
	if err != nil {
		t.Fatal(err)
	}
	w := created.(*window)
	defer w.destroy()
	if w.frameRadius() != 20 {
		t.Fatalf("fixture frame radius = %d, want 20", w.frameRadius())
	}

	var corner, ncEnabled int32
	if hr, _, _ := procDwmGetWindowAttribute.Call(w.hwnd, dwmwaCornerPreference, uintptr(unsafe.Pointer(&corner)), 4); failed(hr) {
		t.Fatalf("reading the corner preference: HRESULT %#x", hr)
	}
	if corner != dwmDoNotRound {
		t.Fatalf("DWM corner preference = %d, want DWMWCP_DONOTROUND (%d): the system would draw its own rounded frame under the content's silhouette", corner, dwmDoNotRound)
	}
	// The policy itself is write-only; DWMWA_NCRENDERING_ENABLED reports its effect.
	const dwmwaNCRenderingEnabled = 1
	if hr, _, _ := procDwmGetWindowAttribute.Call(w.hwnd, dwmwaNCRenderingEnabled, uintptr(unsafe.Pointer(&ncEnabled)), 4); failed(hr) {
		t.Fatalf("reading whether DWM renders the non-client frame: HRESULT %#x", hr)
	}
	if ncEnabled != 0 {
		t.Fatal("DWM still renders a non-client frame under the content's silhouette")
	}
}
