package browserpreview

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"testing"
	desktop "whaleisle/internal/desktop/webview"
	"whaleisle/internal/ui/design"
)

func TestMiniPlayerGeometryOriginalBounds(t *testing.T) {
	container := mygo.Rectangle{Width: 800, Height: 600}
	r := miniPlayerGeometry(mygo.Rectangle{X: 468, Y: 12, Width: 320, Height: 200}, container)
	if r.X != 468 || r.Y != 12 || r.Width != 320 || r.Height != 200 {
		t.Fatal(r)
	}
	for _, side := range []string{"n", "e", "s", "w", "ne", "nw", "se", "sw"} {
		got := resizeMiniPlayer(r, 1000, 1000, side, container)
		if got.X < 12 || got.Y < 12 || got.Width < 240 || got.Height < 150 || got.X+got.Width > 788 || got.Y+got.Height > 588 {
			t.Fatalf("%s: %+v", side, got)
		}
	}
	small := miniPlayerGeometry(r, mygo.Rectangle{Width: 200, Height: 120})
	if small.Width != 176 || small.Height != 96 || small.X != 12 || small.Y != 12 {
		t.Fatal(small)
	}
	west := resizeMiniPlayer(r, -100, 0, "w", container)
	if west.X+west.Width != r.X+r.Width || west.Width != 420 {
		t.Fatal(west)
	}
}

func TestMiniPlayerCloseAndRestoreRetainExactTabAndHost(t *testing.T) {
	host := &desktop.Host{}
	first := &Tab{ID: 1, Host: host}
	second := &Tab{ID: 2, Draft: "https://second.example"}
	b := NewController(nil)
	b.Tabs = []*Tab{first, second}
	b.Active = 1
	geometry := mygo.Rectangle{X: 50, Y: 60, Width: 400, Height: 300}
	b.mini = &browserMiniPlayer{tab: first, host: host, geometry: geometry}
	b.CloseMiniPlayer()
	if b.mini != nil || len(b.Tabs) != 2 || first.Host != host || b.Active != 1 || b.miniGeometry != geometry {
		t.Fatal("close replaced guest or selected another tab")
	}
	restored := 0
	b.onMiniRestore = func() { restored++ }
	b.mini = &browserMiniPlayer{tab: first, host: host, geometry: geometry}
	b.restoreMiniPlayer()
	if b.mini != nil || b.current() != first || first.Host != host || len(b.Tabs) != 2 || restored != 1 {
		t.Fatal("restore did not return exact live resource")
	}
	b.mini = &browserMiniPlayer{tab: first, host: host}
	b.Hide()
	if b.mini == nil || first.Host != host {
		t.Fatal("session hide discarded mini-player or guest")
	}
}

func TestMiniPlayerChrome48pxAndSeatInsets(t *testing.T) {
	bounds := mygo.Rectangle{X: 40, Y: 40, Width: 360, Height: 260}
	seat := miniPlayerSeat(bounds)
	if seat != (mygo.Rectangle{X: 46, Y: 88, Width: 348, Height: 206}) {
		t.Fatalf("miniPlayerSeat = %+v, want {46 88 348 206}", seat)
	}
	b := NewController(nil)
	p := &browserMiniPlayer{bounds: bounds, geometry: bounds}
	tt := ui.NewTester(func(c *ui.Context) {
		b.paintMiniPlayerOverlay(c, p, mygo.Rectangle{Width: 600, Height: 400}, bounds)
	}, 600, 400)
	tt.Frame()
	if _, ok := tt.Find(browserPreviewText("miniTitle")); !ok {
		t.Fatalf("mini player drag title missing: %v", tt.Texts())
	}
	if _, ok := tt.Find(browserPreviewText("miniRestore")); !ok {
		t.Fatalf("mini player restore action missing: %v", tt.Texts())
	}
	if _, ok := tt.Find(browserPreviewText("miniClose")); !ok {
		t.Fatalf("mini player close action missing: %v", tt.Texts())
	}
}

func TestBrowserPanelTransparentUnderPureDesignGlass(t *testing.T) {
	defer design.SetGlass(false, nil)
	design.SetGlass(true, func(_ design.Scheme, _ *design.Palette) {})

	var urlDraft string
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Background(ui.RGB(15, 95, 195)).Children(func() {
			View(c, Model{URLDraft: &urlDraft, HostActive: true})
		})
	}, 400, 300)
	tt.Frame()
	if got := tt.Image().RGBAAt(200, 220); got.R != 15 || got.G != 95 || got.B != 195 {
		t.Fatalf("browser panel under pure design.Glass leaked opaque fill: %+v", got)
	}
}
