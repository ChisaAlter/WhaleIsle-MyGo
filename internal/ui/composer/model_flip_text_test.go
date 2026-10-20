package composer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

func TestModelFlipTimelineUsesIndependentVisualAndCleanupClocks(t *testing.T) {
	start := time.Unix(1000, 0)
	var state modelFlipTimeline
	state.update("A", start, false)
	if !state.Initialized || state.HasOutgoing || state.Shown != "A" {
		t.Fatal("first value must be static")
	}
	state.update("B", start.Add(10*time.Millisecond), false)
	visualStart, firstExpiry := state.Started, state.Expires
	if state.Outgoing != "A" || !state.HasOutgoing || !visualStart.Equal(start.Add(10*time.Millisecond)) || !firstExpiry.Equal(start.Add(410*time.Millisecond)) {
		t.Fatalf("first transition: %+v", state)
	}
	state.update("B", start.Add(50*time.Millisecond), false)
	if state.Started != visualStart || state.Expires != firstExpiry || state.Outgoing != "A" {
		t.Fatal("same text restarted a clock")
	}
	state.update("C", start.Add(110*time.Millisecond), false)
	if state.Started != visualStart || state.Outgoing != "B" || state.Shown != "C" || !state.Expires.Equal(start.Add(510*time.Millisecond)) {
		t.Fatalf("rapid update must replace the old value without restarting CSS: %+v", state)
	}
	state.update("C", start.Add(410*time.Millisecond), false)
	if !state.HasOutgoing || state.Started != visualStart {
		t.Fatal("completed visual recipe must retain its transparent old value until cleanup")
	}
	state.update("C", start.Add(509*time.Millisecond), false)
	if !state.HasOutgoing {
		t.Fatal("cleanup fired early")
	}
	state.update("C", start.Add(510*time.Millisecond), false)
	if state.HasOutgoing || state.Shown != "C" {
		t.Fatal("cleanup failed to retire the old value")
	}
	state.update("D", start.Add(520*time.Millisecond), false)
	if !state.Started.Equal(start.Add(520*time.Millisecond)) || state.Outgoing != "C" {
		t.Fatal("a later transition did not start a fresh visual recipe")
	}
}

func TestModelFlipTimelineReducedMotionAndPreferenceChanges(t *testing.T) {
	start := time.Unix(2000, 0)
	var state modelFlipTimeline
	state.update("A", start, true)
	state.update("B", start.Add(time.Millisecond), true)
	if state.HasOutgoing || state.Shown != "B" {
		t.Fatal("reduced motion must replace a changed value immediately")
	}
	state.update("C", start.Add(10*time.Millisecond), false)
	expires := state.Expires
	state.update("C", start.Add(50*time.Millisecond), true)
	if !state.HasOutgoing || state.Expires != expires {
		t.Fatal("CSS animation:none alone must not replace the existing cleanup timer")
	}
	state.update("C", start.Add(100*time.Millisecond), false)
	if !state.Started.Equal(start.Add(100*time.Millisecond)) || state.Expires != expires {
		t.Fatal("removing animation:none must restart CSS without extending cleanup")
	}
	state.update("D", start.Add(150*time.Millisecond), true)
	if state.HasOutgoing || state.Shown != "D" {
		t.Fatal("a changed value under reduced motion retained old ink")
	}
	state.update("E", start.Add(200*time.Millisecond), false)
	state.update("E", start.Add(600*time.Millisecond), true)
	state.update("E", start.Add(601*time.Millisecond), false)
	if state.HasOutgoing {
		t.Fatal("an expired old value restarted when motion was restored")
	}
}

// A rapid replacement may extend the outgoing ink's cleanup deadline, but the
// recipe itself still ends 400ms after the change that started it: the label
// must return to ordinary text on that clock and the next value must start a
// fresh recipe instead of inheriting the finished one.
func TestModelFlipRecipeEndsOnItsOwnClock(t *testing.T) {
	start := time.Unix(5000, 0)
	var state modelFlipTimeline
	state.update("A", start, false)
	state.update("B", start.Add(10*time.Millisecond), false)
	state.update("C", start.Add(110*time.Millisecond), false)
	if state.recipeOver(start.Add(409 * time.Millisecond)) {
		t.Fatal("recipe ended before its 400ms clock")
	}
	if !state.recipeOver(start.Add(410 * time.Millisecond)) {
		t.Fatal("a rapid replacement's cleanup deadline prolonged the recipe")
	}
	if !state.HasOutgoing {
		t.Fatal("cleanup bookkeeping retired with the recipe")
	}
	state.update("D", start.Add(420*time.Millisecond), false)
	if !state.HasOutgoing || !state.Started.Equal(start.Add(420*time.Millisecond)) || state.Outgoing != "C" {
		t.Fatalf("value after the finished recipe did not start a fresh one: %+v", state)
	}
	if state.recipeOver(start.Add(500 * time.Millisecond)) {
		t.Fatal("fresh recipe ended immediately")
	}
}

func modelFlipImageCopy(src image.Image) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	draw.Draw(dst, dst.Bounds(), src, src.Bounds().Min, draw.Src)
	return dst
}

func modelFlipImagesEqual(a, b *image.RGBA) bool {
	return a.Bounds() == b.Bounds() && bytes.Equal(a.Pix, b.Pix)
}

func modelFlipCrop(src image.Image, rect ui.Rect, scale float32) *image.RGBA {
	bounds := image.Rect(int(math.Floor(float64(rect.X*scale))), int(math.Floor(float64(rect.Y*scale))),
		int(math.Ceil(float64((rect.X+rect.W)*scale))), int(math.Ceil(float64((rect.Y+rect.H)*scale)))).Intersect(src.Bounds())
	dst := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(dst, dst.Bounds(), src, bounds.Min, draw.Src)
	return dst
}

func modelFlipLabelTester(snap *session.ViewState, placement ModelLabelPlacement) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		ui.Row(c).FillWidth().Padding(16).Background(ui.RGB(255, 255, 255)).Children(func() {
			ModelLabel(c, *snap, snap.SelectedModel, placement, true, ui.RGB(0, 0, 0), 13)
		})
	}, 420, 80)
}

func TestModelFlipCurrentWidthClipsOldInkAndKeepsOnlyCurrentText(t *testing.T) {
	start := time.Unix(3000, 0)
	value := "A much longer outgoing model name that must not widen its replacement"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Padding(16).Children(func() { modelFlipText(c, "stable-name", value, ui.RGB(0, 0, 0), 13) })
	}, 600, 80)
	tt.SetTime(start)
	value = "Model now"
	tt.Frame()
	current, ok := tt.Find(value)
	if !ok || tt.HasText("A much longer outgoing") {
		t.Fatal("layout/text collection must expose only the current value, not outgoing decorative ink")
	}
	// This asserts the app's visible layout/text collection. It is deliberately
	// not an assertion against the private platform accessibility tree.
	static := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Padding(16).Children(func() { modelFlipText(c, "reference", value, ui.RGB(0, 0, 0), 13) })
	}, 600, 80)
	want, _ := static.Find(value)
	if current.W != want.W || current.H != want.H {
		t.Fatalf("outgoing text widened the current seat: %v want %v", current, want)
	}
	frame := tt.Image()
	ink := 0
	for y := range frame.Bounds().Dy() {
		for x := range frame.Bounds().Dx() {
			pixel := frame.RGBAAt(x, y)
			if pixel.R < 200 && pixel.G < 200 && pixel.B < 200 && pixel.A > 200 {
				ink++
				if float32(x) < current.X-1 || float32(x) > current.X+current.W+1 || float32(y) < current.Y-1 || float32(y) > current.Y+current.H+1 {
					t.Fatalf("old ellipsized ink escaped the new seat at %d,%d: %v", x, y, current)
				}
			}
		}
	}
	if ink == 0 {
		t.Fatal("first transition frame lost its outgoing ink")
	}
	tt.SetTime(start.Add(200 * time.Millisecond))
	if modelFlipImagesEqual(modelFlipImageCopy(tt.Image()), modelFlipImageCopy(static.Image())) {
		t.Fatal("mid-transition ink jumped directly to the static replacement")
	}
	tt.SetTime(start.Add(400 * time.Millisecond))
	if !modelFlipImagesEqual(modelFlipImageCopy(tt.Image()), modelFlipImageCopy(static.Image())) {
		t.Fatal("completed transition failed to return to ordinary native text")
	}
}

func TestModelFlipRapidValuesDoNotRestartTheRenderedRecipe(t *testing.T) {
	start := time.Unix(3500, 0)
	actualValue, referenceValue := "First label", "Second label"
	newTester := func(value *string) *ui.Tester {
		return ui.NewTester(func(c *ui.Context) {
			ui.Row(c).Padding(16).Children(func() { modelFlipText(c, "name", *value, ui.RGB(0, 0, 0), 13) })
		}, 300, 80)
	}
	actual, reference := newTester(&actualValue), newTester(&referenceValue)
	actual.SetTime(start)
	reference.SetTime(start)
	actualValue, referenceValue = "Second label", "Third label"
	actual.Frame()
	reference.Frame()
	actual.SetTime(start.Add(100 * time.Millisecond))
	actualValue = "Third label"
	actual.Frame()
	reference.SetTime(start.Add(100 * time.Millisecond))
	// Both now show Second -> Third at the original recipe's 100ms. The
	// actual path got there through a rapid update without remounting spans.
	if !modelFlipImagesEqual(modelFlipImageCopy(actual.Image()), modelFlipImageCopy(reference.Image())) {
		t.Fatal("rapid text replacement restarted the mounted visual recipe")
	}
	actual.SetTime(start.Add(400 * time.Millisecond))
	reference.SetTime(start.Add(400 * time.Millisecond))
	if !modelFlipImagesEqual(modelFlipImageCopy(actual.Image()), modelFlipImageCopy(reference.Image())) {
		t.Fatal("extended cleanup deadline replayed or prolonged the visual recipe")
	}
}

func TestModelFlipReducedMotionShowsOrdinaryNativeReplacement(t *testing.T) {
	value := "Old model"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Padding(16).Children(func() { modelFlipText(c, "name", value, ui.RGB(0, 0, 0), 13) })
	}, 300, 80)
	tt.SetTime(time.Unix(3600, 0))
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	value = "Current model"
	tt.Frame()
	static := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Padding(16).Children(func() { modelFlipText(c, "reference", value, ui.RGB(0, 0, 0), 13) })
	}, 300, 80)
	static.SetPreferences(ui.Preferences{ReduceMotion: true})
	if !modelFlipImagesEqual(modelFlipImageCopy(tt.Image()), modelFlipImageCopy(static.Image())) || tt.HasText("Old model") {
		t.Fatal("reduced motion did not replace both native ink and current layout text immediately")
	}
}

func TestModelFlipRouteChangesRetainNameAnimationAndBadgeIsStatic(t *testing.T) {
	start := time.Unix(4000, 0)
	snap := modelLabelFixture()
	snap.ModelCatalogChoices[0].Name = "Previous model"
	tt := modelFlipLabelTester(&snap, ModelLabelTrigger)
	tt.SetTime(start)
	snap.SelectedModel = "account opaque key"
	tt.Frame()
	static := modelFlipLabelTester(&snap, ModelLabelTrigger)
	if modelFlipImagesEqual(modelFlipImageCopy(tt.Image()), modelFlipImageCopy(static.Image())) {
		t.Fatal("changing a provider/model route remounted the same label renderer instead of flipping its name")
	}
	badge, ok := tt.Find("1017")
	if !ok {
		t.Fatal("new matching route did not show the independent badge immediately")
	}
	badgePixels := modelFlipCrop(static.Image(), badge, 1)
	for _, elapsed := range []time.Duration{0, 100 * time.Millisecond, 200 * time.Millisecond} {
		tt.SetTime(start.Add(elapsed))
		if !modelFlipImagesEqual(modelFlipCrop(tt.Image(), badge, 1), badgePixels) {
			t.Fatalf("badge participated in the name transform at %s", elapsed)
		}
	}
	for _, placement := range []ModelLabelPlacement{ModelLabelMenu, ModelLabelOption} {
		s := modelLabelFixture()
		menu := modelFlipLabelTester(&s, placement)
		menu.SetTime(start)
		s.ModelCatalogChoices[0].Name = "A different static name"
		menu.Frame()
		want := modelFlipLabelTester(&s, placement)
		if !modelFlipImagesEqual(modelFlipImageCopy(menu.Image()), modelFlipImageCopy(want.Image())) {
			t.Fatalf("%s incorrectly acquired a flip animation", placement)
		}
	}
}

func TestModelFlipPresentationOwnersRetireIndependently(t *testing.T) {
	baseGraph := nativeCommandContributionFixture(t, nil)
	reloadedGraph := nativeCommandContributionFixture(t, map[string]string{"ui-deepseek-model-label-1017": "label-r2"})
	if baseGraph.ModelLabelRevision() == "" || baseGraph.ModelLabelRevision() == reloadedGraph.ModelLabelRevision() {
		t.Fatal("fixture did not produce distinct real label registration revisions")
	}
	for _, tc := range []struct {
		name   string
		retire bool
		change func(*session.ViewState)
	}{
		{"unrelated graph revision", false, func(s *session.ViewState) { s.NativeContributions.Revision = "unrelated-change" }},
		{"label reload", true, func(s *session.ViewState) { s.NativeContributions = reloadedGraph }},
		{"label withdrawn and reinstated between frames", true, func(s *session.ViewState) { s.NativeModelLabelGeneration++ }},
		{"registration owner replaced between frames", true, func(s *session.ViewState) { s.NativeContributionsGeneration++ }},
		{"renderer replaced by fallback", true, func(s *session.ViewState) { s.NativeContributions.ModelLabel1017 = false }},
		{"session generation", true, func(s *session.ViewState) { s.ComposerSessionGeneration++ }},
		{"host generation", true, func(s *session.ViewState) { s.ComposerHostGeneration++ }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Unix(5000, 0)
			snap := modelLabelFixture()
			snap.NativeContributions = baseGraph
			snap.ModelCatalogChoices[0].Name = "Previous display value"
			tt := modelFlipLabelTester(&snap, ModelLabelTrigger)
			tt.SetTime(start)
			snap.ModelCatalogChoices[0].Name = "Current display value"
			tt.Frame()
			tc.change(&snap)
			tt.SetTime(start.Add(100 * time.Millisecond))
			static := modelFlipLabelTester(&snap, ModelLabelTrigger)
			isStatic := modelFlipImagesEqual(modelFlipImageCopy(tt.Image()), modelFlipImageCopy(static.Image()))
			if isStatic != tc.retire {
				t.Fatalf("source retirement=%v want %v", isStatic, tc.retire)
			}
		})
	}
}

func TestModelFlipLabelReloadDoesNotRestartEffort(t *testing.T) {
	start := time.Unix(6000, 0)
	snap := modelLabelFixture()
	snap.ModelCatalogChoices[0].Name = "Previous name"
	snap.SelectedModelEffort = "Low"
	tt := ui.NewTester(func(c *ui.Context) {
		paintModelPicker(c, Model{Snapshot: snap}, testColors(), 28)
	}, 600, 100)
	tt.SetTime(start)
	snap.ModelCatalogChoices[0].Name = "Current name"
	snap.SelectedModelEffort = "High"
	tt.Frame()
	tt.SetTime(start.Add(100 * time.Millisecond))
	effort, ok := tt.Find("High")
	if !ok {
		t.Fatal("actual grouped trigger lost its current effort")
	}
	before := modelFlipCrop(tt.Image(), effort, 1)
	snap.NativeModelLabelGeneration++
	tt.Frame()
	after, ok := tt.Find("High")
	if !ok || effort != after || !modelFlipImagesEqual(before, modelFlipCrop(tt.Image(), after, 1)) {
		t.Fatal("label renderer reload restarted or moved the independently owned effort animation")
	}
	static := ui.NewTester(func(c *ui.Context) {
		paintModelPicker(c, Model{Snapshot: snap}, testColors(), 28)
	}, 600, 100)
	if modelFlipImagesEqual(modelFlipCrop(tt.Image(), after, 1), modelFlipCrop(static.Image(), after, 1)) {
		t.Fatal("effort animation was already static during the retained transition")
	}
}

func TestModelFlipGroupedUnknownRouteAndCatalogSourceBoundary(t *testing.T) {
	start := time.Unix(6500, 0)
	snap := modelLabelFixture()
	snap.SelectedModel = "custom/previous-route"
	newTester := func() *ui.Tester {
		return ui.NewTester(func(c *ui.Context) {
			paintModelPicker(c, Model{Snapshot: snap}, testColors(), 28)
		}, 600, 100)
	}
	tt := newTester()
	tt.SetTime(start)
	snap.SelectedModel = "custom/current-route"
	tt.Frame()
	tt.SetTime(start.Add(100 * time.Millisecond))
	static := newTester()
	if modelFlipImagesEqual(modelFlipImageCopy(tt.Image()), modelFlipImageCopy(static.Image())) {
		t.Fatal("unknown-current fallback lost its stable animated owner")
	}
	snap.SelectedModel = "official opaque key"
	tt.Frame()
	static = newTester()
	if !modelFlipImagesEqual(modelFlipImageCopy(tt.Image()), modelFlipImageCopy(static.Image())) {
		t.Fatal("switching from unknown fallback to catalog renderer carried outgoing ink across a source remount")
	}
}

func TestModelFlipSoftwareFramesAtFractionalDPI(t *testing.T) {
	for _, scale := range []float32{1.25, 1.5} {
		t.Run(fmt.Sprintf("dpi%d", int(scale*100)), func(t *testing.T) {
			start := time.Unix(7000, 0)
			snap := modelLabelFixture()
			snap.ModelCatalogChoices[0].Name = "Previous long model display name"
			snap.SelectedModelEffort = "Low"
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Padding(24).Background(ui.RGB(255, 255, 255)).Children(func() {
					paintModelPicker(c, Model{Snapshot: snap}, testColors(), 28)
				})
			}, 500, 100)
			tt.SetScale(scale)
			tt.SetTime(start)
			snap.ModelCatalogChoices[0].Name = "DeepSeek-V4.1-Flash"
			snap.SelectedModelEffort = "High"
			tt.Frame()
			for _, ms := range []int{0, 100, 200, 400} {
				tt.SetTime(start.Add(time.Duration(ms) * time.Millisecond))
				if _, ok := tt.Find("DeepSeek-V4.1-Flash"); !ok || !tt.HasText("High") || modelBadgeCount(tt) != 1 {
					t.Fatalf("dpi %.2f frame %d lost the live caption or independent badge", scale, ms)
				}
				exportModelFlipFrame(t, tt, scale, ms)
			}
		})
	}
}

func exportModelFlipFrame(t *testing.T, tt *ui.Tester, scale float32, ms int) {
	t.Helper()
	dir := os.Getenv("SHELL_SNAPSHOT_DIR")
	if dir == "" {
		return
	}
	dir = filepath.Join(dir, "model-flip")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("model-flip-dpi%d-t%03d", int(scale*100), ms)
	f, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	err = png.Encode(f, tt.Image())
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("write software frame: %v; close: %v", err, closeErr)
	}
	geometry := map[string]any{"scale": scale, "elapsedMS": ms, "softwareOnly": true, "texts": tt.Texts()}
	for _, label := range []string{"Model catalog", "DeepSeek-V4.1-Flash", "High", "1017"} {
		if rect, ok := tt.Find(label); ok {
			geometry[label] = rect
		}
	}
	data, err := json.MarshalIndent(geometry, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".json"), data, 0644); err != nil {
		t.Fatal(err)
	}
}
