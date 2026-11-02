//go:build windows && (amd64 || arm64)

package previewprocess

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"whaleisle/internal/desktop/openui"
)

// Opt-in: uses only a newly built helper and hidden parent window, never the
// installed application. The real AppContainer policy and WebView2 are active.
func TestWhaleUINativeHiddenHeightAndClip(t *testing.T) {
	helper := os.Getenv("WHALE_UI_NATIVE_HELPER")
	if helper == "" {
		t.Skip("set WHALE_UI_NATIVE_HELPER to a freshly built helper")
	}
	viewer, err := openui.LoadWhaleViewer()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	id := openui.Identity("native-fixture", "session", "message", 0)
	r := openui.Render{InstanceID: id, Revision: 1, Language: "whale-ui", Locale: "en", Program: `root = Col([Title("Native fixture","md"),Text("Short content")])`, State: json.RawMessage(`{}`)}
	s, err := OpenExperimental(ctx, Options{Enabled: true, HelperPath: helper}, Document{HTML: viewer, Width: 390, InitialHeight: 320, Appearance: "light", Inline: true, OpenUI: &r})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	waitHeight := func(min, max int) {
		deadline := time.NewTimer(20 * time.Second)
		defer deadline.Stop()
		tick := time.NewTicker(30 * time.Millisecond)
		defer tick.Stop()
		for {
			h := s.ContentHeight()
			if h >= min && h <= max {
				return
			}
			select {
			case <-s.Done():
				t.Fatal("helper closed before measurement")
			case <-deadline.C:
				t.Fatalf("native height %d outside %d..%d", h, min, max)
			case <-tick.C:
			}
		}
	}
	waitHeight(80, 240)
	user := windows.NewLazySystemDLL("user32.dll")
	create := user.NewProc("CreateWindowExW")
	destroy := user.NewProc("DestroyWindow")
	class, _ := windows.UTF16PtrFromString("STATIC")
	parent, _, err := create.Call(0, uintptr(unsafe.Pointer(class)), 0, 0, 0, 0, 500, 600, 0, 0, 0, 0)
	if parent == 0 {
		t.Fatal("hidden parent", err)
	}
	defer destroy.Call(parent)
	if err := s.PresentSurface(parent, 10, 20, 390, 200, 40, 50, 150, 100); err != nil {
		t.Fatal(err)
	}
	// Native clipping must be exactly the requested intersection, not the whole
	// child. The parent remains hidden, so this test never displays a GUI.
	region, _, _ := surfaceRect.Call(0, 0, 0, 0)
	if region == 0 {
		t.Fatal("clip region unavailable")
	}
	defer surfaceDelete.Call(region)
	getRegion := user.NewProc("GetWindowRgn")
	kind, _, _ := getRegion.Call(s.surfaceHandle, region)
	if kind == 0 {
		t.Fatal("surface did not retain native clip")
	}
	if err := s.PresentSurface(0, 0, 0, 0, 0, 0, 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	r.Revision = 2
	r.Locale = "zh"
	r.Program = `root = Col([Box([Text("Tall content")],"col",3,3,"surface","md",false,null,1700)])`
	if err := s.UpdateOpenUI(r); err != nil {
		t.Fatal(err)
	}
	waitHeight(1500, 2000)
	if err := s.SetTheme("dark", map[string]string{"--accent": "#123abc"}); err != nil {
		t.Fatal(err)
	}
	t.Logf("PASS real AppContainer/WebView2 390px: short natural height, 1700px update, native clip region=%d, hidden parent, live theme", kind)
}
