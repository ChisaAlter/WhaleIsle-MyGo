package browserpreview

import (
	"github.com/egoist/mygo"
	"testing"
	desktop "whaleisle/internal/desktop/webview"
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
