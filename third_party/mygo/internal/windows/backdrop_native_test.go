//go:build windows && (amd64 || arm64)

package windows

import (
	"os"
	"runtime"
	"testing"
	"unsafe"

	"github.com/egoist/mygo/internal/platform"
)

type backdropFixtureWindow struct{ previewNativeWindow }

func (*backdropFixtureWindow) SurfaceEvent(platform.SurfaceEvent) bool { return false }

func TestBackdropNativeEnableDisable(t *testing.T) {
	if os.Getenv("MYGO_BACKDROP_NATIVE_TEST") != "1" {
		t.Skip("set MYGO_BACKDROP_NATIVE_TEST=1 in a Windows desktop session")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	oldThread, oldBackend := mainThreadID, theBackend
	mainThreadID = currentThreadID()
	defer func() { mainThreadID = oldThread; theBackend = oldBackend }()
	if !systemBackdrops() {
		t.Skip("system backdrop requires Windows 11 22H2")
	}
	b := New()
	if err := b.Init(previewNativeApp{}, platform.AppOptions{Name: "MyGo isolated backdrop fixture"}); err != nil {
		t.Fatal(err)
	}
	defer procDestroyWindow.Call(b.appHwnd)
	created, err := b.NewWindow(&platform.WindowOptions{Surface: true, Transparent: true, WindowsFrameCornerRadius: 20, Title: "MyGo isolated backdrop fixture", Width: 320, Height: 180, Closable: true, Movable: true, HasShadow: true}, &backdropFixtureWindow{})
	if err != nil {
		t.Fatal(err)
	}
	w := created.(*window)
	defer w.destroy()
	if !w.noRedirect {
		t.Fatal("fixture has no DirectComposition surface")
	}
	before := w.hwnd
	w.SetVibrancy("acrylic")
	if !w.surface.ShowsMaterial() {
		t.Fatal("Acrylic API did not apply")
	}
	var value int32
	hr, _, _ := procDwmGetWindowAttribute.Call(w.hwnd, dwmwaSystemBackdropType, uintptr(unsafe.Pointer(&value)), 4)
	if failed(hr) || value != backdropAcrylic {
		t.Fatalf("native backdrop=%d hr=%#x", value, hr)
	}
	w.SetVibrancy("")
	if w.surface.ShowsMaterial() {
		t.Fatal("disabled Acrylic still reported active")
	}
	w.SetVibrancy("mica")
	if !w.surface.ShowsMaterial() {
		t.Fatal("Mica did not apply")
	}
	w.SetVibrancy("")
	if w.hwnd != before {
		t.Fatal("material change recreated fixture window")
	}
	t.Log("DWM Acrylic/Mica actual attributes and in-place disable passed on isolated composed native window")
}
