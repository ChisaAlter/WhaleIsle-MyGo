package pet

import (
	"encoding/json"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo"
)

func petHitFixtureURL(value any) string {
	body, _ := json.Marshal(value)
	return "whale-pet://localhost/interactive-regions?data=" + url.QueryEscape(string(body))
}

func TestPetHitReportFixedOriginClippingAndBounds(t *testing.T) {
	fixture := petHitReport{Width: 200, Height: 300, Epoch: 7, Regions: []petHitRect{
		{X: -10, Y: 20, Width: 40, Height: 50},
		{X: 180, Y: 290, Width: 100, Height: 100},
		{X: 400, Y: 20, Width: 10, Height: 10},
		{X: 40, Y: 50, Width: -1, Height: 10},
	}}
	report, ok := decodePetHitReport(petHitFixtureURL(fixture))
	want := []petHitRect{{X: 0, Y: 20, Width: 30, Height: 50}, {X: 180, Y: 290, Width: 20, Height: 10}}
	if !ok || report.Epoch != 7 || !reflect.DeepEqual(report.Regions, want) {
		t.Fatalf("viewport clipping: %+v %t", report, ok)
	}
	valid := petHitFixtureURL(petHitReport{Width: 200, Height: 300, Epoch: 1})
	for _, raw := range []string{
		strings.Replace(valid, "whale-pet://localhost", "https://localhost", 1),
		strings.Replace(valid, "localhost", "other-window", 1),
		strings.Replace(valid, "localhost", "name@localhost", 1),
		valid + "#other", valid + "&other=1", valid + "&data=duplicate",
		petHitFixtureURL(petHitReport{Width: 0, Height: 300, Epoch: 1}),
		petHitFixtureURL(petHitReport{Width: 200, Height: 300}),
		petHitFixtureURL(petHitReport{Width: 65537, Height: 300, Epoch: 1}),
		petHitFixtureURL(petHitReport{Width: 200, Height: 300, Epoch: 1, Regions: make([]petHitRect, 17)}),
		"whale-pet://localhost/interactive-regions?data=" + url.QueryEscape(`{"Width":"200","Height":300,"Epoch":1}`),
		"whale-pet://localhost/interactive-regions?data=" + strings.Repeat("x", 8192),
	} {
		if _, ok := decodePetHitReport(raw); ok {
			t.Fatalf("accepted an invalid geometry report: %.100s", raw)
		}
	}
}

func TestPetHitRegionsMapCSSViewportToDIPAndPreserveGaps(t *testing.T) {
	// A zoomed CSS viewport over a negative-origin monitor. Bounds and cursor
	// are already DIPs: a second devicePixelRatio multiplication would fail.
	report := petHitReport{Width: 400, Height: 400, Epoch: 1, Regions: []petHitRect{
		{X: 100, Y: 40, Width: 80, Height: 120},
		{X: 300, Y: 200, Width: 50, Height: 50},
	}}
	content := mygo.Rectangle{X: -1500, Y: -100, Width: 200, Height: 200}
	for _, tc := range []struct {
		point       mygo.Point
		interactive bool
	}{
		{mygo.Point{X: -1430, Y: -50}, true},
		{mygo.Point{X: -1458, Y: -80}, true}, // Exactly eight DIPs outside left.
		{mygo.Point{X: -1459, Y: -80}, false},
		{mygo.Point{X: -1430, Y: -12}, true}, // Exactly eight below first island.
		{mygo.Point{X: -1430, Y: -11}, false},
		{mygo.Point{X: -1340, Y: 10}, true},
		{mygo.Point{X: -1380, Y: -40}, false}, // Gap between character and panel.
		{mygo.Point{X: 0, Y: 0}, false},
	} {
		if got := petHitContains(report, content, tc.point, false); got != tc.interactive {
			t.Fatalf("point %+v: %t; want %t", tc.point, got, tc.interactive)
		}
	}
	out := mygo.Point{X: 2000, Y: 2000}
	if !petHitContains(report, content, out, true) || petHitContains(report, content, out, false) {
		t.Fatal("capture did not remain interactive until release, or stayed interactive afterwards")
	}
	if petHitContains(report, mygo.Rectangle{}, out, true) {
		t.Fatal("zero content bounds became an interactive surface")
	}
	if petHitContains(petHitReport{}, content, mygo.Point{X: -1430, Y: -50}, false) {
		t.Fatal("page without a published layout blocked the desktop")
	}
}

func TestPetHitRegionEpochRetiresOldPageLayout(t *testing.T) {
	var p Companion // No native window, application runtime, or OS calls.
	p.interaction.generation = 7
	old := petHitReport{Width: 200, Height: 300, Epoch: 7, Regions: []petHitRect{{Width: 30, Height: 30}}}
	if !p.interaction.publishHitReport(old) {
		t.Fatal("current page report rejected")
	}
	p.stopInteraction() // Hide/close/show reset uses this production lifecycle.
	if p.interaction.view.Epoch != 8 || len(p.interaction.hit.report.Regions) != 0 {
		t.Fatal("old layout survived its page lifecycle")
	}
	if p.interaction.publishHitReport(old) {
		t.Fatal("late report revived the retired page's regions")
	}
	fresh := old
	fresh.Epoch = 8
	if !p.interaction.publishHitReport(fresh) {
		t.Fatal("newly shown page could not publish its layout")
	}
	p.stopInteraction()
	if p.interaction.publishHitReport(fresh) {
		t.Fatal("second close accepted the old layout")
	}
}
