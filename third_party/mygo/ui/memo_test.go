package ui

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

// memoApp is a view of a header above two memo regions, each of a few
// buttons, with counts of the regions' builds and clicks.
type memoApp struct {
	header      float32
	labels      [2][]string
	version     [2]uint64
	builds      [2]int
	clicks      map[string]int
	animate     bool
	plain       bool // no Memo: the same view without regions
	handle      Element
	handleSaved bool
}

func newMemoApp() *memoApp {
	a := &memoApp{header: 20, clicks: map[string]int{}}
	for i := range a.labels {
		for j := range 3 {
			a.labels[i] = append(a.labels[i], fmt.Sprintf("Region %d button %d", i, j))
		}
	}
	return a
}

func (a *memoApp) view(c *context) {
	coreColumn(c).Padding(8).Gap(4).Children(func() {
		coreBox(c).Height(a.header)
		if coreButton(c, "Outside").Clicked() {
			a.clicks["Outside"]++
		}
		for i := range a.labels {
			region := coreColumn(c).Gap(2)
			if !a.plain {
				region.Memo(a.version[i])
			}
			region.Children(func() {
				a.builds[i]++
				coreText(c, fmt.Sprint("Heading ", i))
				for _, l := range a.labels[i] {
					b := coreButton(c, l)
					if i == 0 && !a.handleSaved {
						a.handle, a.handleSaved = wrapElement(b), true
					}
					if b.Clicked() {
						a.clicks[l]++
					}
				}
				if i == 1 && a.animate {
					c.AnimationFrame()
				}
			})
		}
	})
}

func memoTester(t *testing.T, a *memoApp) *Tester {
	t.Helper()
	tt := coreNewTester(a.view, 400, 400)
	tt.Frame()
	a.builds = [2]int{}
	return tt
}

func TestMemoKeepsCleanRegions(t *testing.T) {
	a := newMemoApp()
	tt := memoTester(t, a)
	texts := tt.Texts()
	where, _ := tt.Find("Region 1 button 2")
	for range 5 {
		tt.Frame()
	}
	if a.builds != [2]int{} {
		t.Fatalf("clean regions built %v times", a.builds)
	}
	if got := tt.Texts(); !slices.Equal(got, texts) {
		t.Fatalf("kept regions show %q, built %q", got, texts)
	}
	if r, ok := tt.Find("Region 1 button 2"); !ok || r != where {
		t.Fatalf("kept button at %v, built at %v", r, where)
	}
	if tt.rt.memoKept == 0 {
		t.Fatal("nothing kept")
	}
	// A new version builds its region alone.
	a.version[0]++
	a.labels[0][1] = "Renamed"
	tt.Frame()
	if a.builds != [2]int{1, 0} || !tt.HasText("Renamed") {
		t.Fatalf("after a new version: built %v, %q", a.builds, tt.Texts())
	}
}

func TestMemoOff(t *testing.T) {
	memoOff = true
	defer func() { memoOff = false }()
	a := newMemoApp()
	tt := memoTester(t, a)
	tt.Frame()
	tt.Frame()
	if a.builds != [2]int{2, 2} {
		t.Fatalf("with memo off, built %v times", a.builds)
	}
}

// What a region is built against: the window's size, the theme, the
// scale, builds it again.
func TestMemoRebuildsOnConstraints(t *testing.T) {
	a := newMemoApp()
	tt := memoTester(t, a)
	tt.SetSize(500, 400)
	if a.builds[0] == 0 || a.builds[1] == 0 {
		t.Fatalf("resized, built %v", a.builds)
	}
	a.builds = [2]int{}
	tt.SetDark(true)
	if a.builds[0] == 0 {
		t.Fatalf("dark, built %v", a.builds)
	}
	a.builds = [2]int{}
	tt.SetScale(2)
	if a.builds[0] == 0 {
		t.Fatalf("scaled, built %v", a.builds)
	}
}

// Input reaching a region builds it again, so its controls see it; the
// other region stays kept.
func TestMemoInputInside(t *testing.T) {
	a := newMemoApp()
	tt := memoTester(t, a)
	if err := tt.Click("Region 1 button 1"); err != nil {
		t.Fatal(err)
	}
	if a.clicks["Region 1 button 1"] != 1 {
		t.Fatalf("clicks %v", a.clicks)
	}
	if a.builds[1] == 0 {
		t.Fatal("the region clicked did not build")
	}
	if a.builds[0] != 0 {
		t.Fatalf("the region not clicked built %d times", a.builds[0])
	}
	// The focus is in region 1 now: it builds while it keeps it.
	a.builds = [2]int{}
	tt.Frame()
	tt.Frame()
	if a.builds[1] != 2 {
		t.Fatalf("the region with the focus built %d times in 2 frames", a.builds[1])
	}
	// Clicking outside takes the focus out: the region is kept again.
	tt.Click("Outside")
	tt.Frame()
	a.builds = [2]int{}
	tt.Frame()
	if a.builds[1] != 0 {
		t.Fatalf("the region without the focus built %d times", a.builds[1])
	}
	// The pointer coming over a region builds it, for its hover.
	r, _ := tt.Find("Region 0 button 0")
	tt.Move(r.X+2, r.Y+2)
	if a.builds[0] == 0 {
		t.Fatal("hovering did not build the region")
	}
}

// A region whose build asks for frames builds in each.
func TestMemoAnimationInside(t *testing.T) {
	a := newMemoApp()
	a.animate = true
	tt := memoTester(t, a)
	tt.Frame()
	tt.Frame()
	if a.builds[1] < 2 || a.builds[0] != 0 {
		t.Fatalf("built %v while region 1 animates", a.builds)
	}
}

// Tab goes through the controls of kept regions in the order it goes
// through them built.
func TestMemoFocusOrder(t *testing.T) {
	order := func(plain bool) []string {
		a := newMemoApp()
		a.plain = plain
		tt := memoTester(t, a)
		var got []string
		for range 8 {
			tt.Key(0, KeyTab)
			tt.Frame()
			for _, l := range append([]string{"Outside"}, slices.Concat(a.labels[0], a.labels[1])...) {
				if tt.Focused(l) {
					got = append(got, l)
				}
			}
		}
		return got
	}
	kept, built := order(false), order(true)
	if !slices.Equal(kept, built) || len(kept) != 7+1 {
		t.Fatalf("focus order %q with regions, %q without", kept, built)
	}
}

// A region moved by what is above it takes the pointer where it shows.
func TestMemoHitsAfterMove(t *testing.T) {
	a := newMemoApp()
	tt := memoTester(t, a)
	before, _ := tt.Find("Region 1 button 0")
	a.header = 60
	tt.Frame()
	if a.builds != [2]int{} {
		t.Fatalf("moved regions built %v", a.builds)
	}
	after, ok := tt.Find("Region 1 button 0")
	if !ok || after.Y != before.Y+40 || after.X != before.X {
		t.Fatalf("moved button at %v, was at %v", after, before)
	}
	tt.ClickAt(after.X+3, after.Y+3)
	if a.clicks["Region 1 button 0"] != 1 {
		t.Fatalf("clicking the moved button: %v", a.clicks)
	}
}

// Handles from the pass that built a region expire as handles do, though
// its elements are kept.
func TestMemoElementExpiry(t *testing.T) {
	a := newMemoApp()
	tt := memoTester(t, a)
	tt.Frame()
	h := a.handle
	tt.rt.inFrame = true
	defer func() { tt.rt.inFrame = false }()
	if h.Valid() {
		t.Fatal("an element of an earlier pass is valid")
	}
}

// A click outside the regions builds the view again in the same frame;
// the regions are kept in that pass.
func TestMemoKeptAcrossPasses(t *testing.T) {
	a := newMemoApp()
	tt := memoTester(t, a)
	tt.Click("Outside")
	if a.clicks["Outside"] != 1 {
		t.Fatal("no click")
	}
	if tt.rt.pass == 0 {
		t.Skip("the click took one pass")
	}
	if a.builds[0] != 0 {
		t.Fatalf("a clean region built %d times over passes", a.builds[0])
	}
}

// Assistive technology sees the same tree with regions kept as built, on
// views changing at random.
func TestMemoAccessTreeSame(t *testing.T) {
	rng := rand.New(rand.NewPCG(9, 10))
	kept, built := newMemoApp(), newMemoApp()
	built.plain = true
	tk, tb := memoTester(t, kept), memoTester(t, built)
	for _, tt := range []*Tester{tk, tb} {
		tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	}
	for step := range 200 {
		switch rng.IntN(5) {
		case 0:
			i := rng.IntN(2)
			l := fmt.Sprint("Label ", step)
			for _, a := range []*memoApp{kept, built} {
				a.labels[i] = append(a.labels[i], l)
				a.version[i]++
			}
		case 1:
			h := float32(rng.IntN(50))
			kept.header, built.header = h, h
		case 2:
			for _, tt := range []*Tester{tk, tb} {
				tt.Key(0, KeyTab)
			}
		case 3:
			x, y := float32(rng.IntN(300)), float32(rng.IntN(300))
			for _, tt := range []*Tester{tk, tb} {
				tt.Move(x, y)
			}
		}
		tk.Frame()
		tb.Frame()
		a, b := tk.rt.accessTree(), tb.rt.accessTree()
		if !slices.Equal(a.Nodes, b.Nodes) || a.Focus != b.Focus {
			t.Fatalf("step %d: kept\n%+v\nbuilt\n%+v", step, a.Nodes, b.Nodes)
		}
		if !slices.Equal(tk.Texts(), tb.Texts()) {
			t.Fatalf("step %d: kept %q, built %q", step, tk.Texts(), tb.Texts())
		}
	}
	if tk.rt.memoDescribed == 0 || tk.rt.memoCommitted == 0 || tk.rt.memoLaid == 0 {
		t.Fatalf("described %d, committed %d, laid %d kept regions", tk.rt.memoDescribed, tk.rt.memoCommitted, tk.rt.memoLaid)
	}
}

// An action of assistive technology on an element of a kept region builds
// the region, which sees it.
func TestMemoAccessAction(t *testing.T) {
	a := newMemoApp()
	tt := memoTester(t, a)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Frame()
	n := accessNode(t, tt.rt.accessTree(), platform.RoleButton, "Region 0 button 2")
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: n.ID, Action: platform.AccessPress})
	tt.Frame()
	if a.clicks["Region 0 button 2"] != 1 {
		t.Fatalf("pressed through assistive technology: %v", a.clicks)
	}
}

// A region leaving with an exit transition shows what it kept.
func TestMemoExit(t *testing.T) {
	show := true
	builds := 0
	tt, now := clockTester(func(c *context) {
		coreColumn(c).Children(func() {
			if show {
				coreColumn(c).Key("r").Transition(ElementTransition{Exit: &Motion{Opacity: 0}}).Memo(1).Children(func() {
					builds++
					coreText(c, "Kept text")
				})
			}
			coreText(c, "Below")
		})
	}, 300, 300)
	tt.Frame()
	tt.Frame()
	if builds != 1 {
		t.Fatalf("built %d times", builds)
	}
	show = false
	*now = now.Add(10 * time.Millisecond)
	tt.Frame()
	var ghost *node
	for _, r := range tt.rt.trans {
		if r.ghost != nil {
			ghost = r.ghost
		}
	}
	if ghost == nil || ghost.first == nil || ghost.first.text != "Kept text" {
		t.Fatalf("leaving copy %+v", ghost)
	}
}

// A region inside a kept region is kept with it, and kept within it when
// only the outer one builds again.
func TestMemoNested(t *testing.T) {
	outerV, innerV := uint64(1), uint64(1)
	outerText, innerText := "Outer 1", "Inner 1"
	outerBuilds, innerBuilds := 0, 0
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Memo(outerV).Children(func() {
			outerBuilds++
			coreText(c, outerText)
			coreColumn(c).Memo(innerV).Children(func() {
				innerBuilds++
				coreText(c, innerText)
				coreButton(c, "Inner button")
			})
		})
	}, 300, 300)
	for range 3 {
		tt.Frame()
	}
	if outerBuilds != 1 || innerBuilds != 1 {
		t.Fatalf("built %d, %d", outerBuilds, innerBuilds)
	}
	outerV, outerText = 2, "Outer 2"
	tt.Frame()
	tt.Frame()
	if outerBuilds != 2 || innerBuilds != 1 || !tt.HasText("Outer 2") || !tt.HasText("Inner 1") {
		t.Fatalf("built %d, %d: %q", outerBuilds, innerBuilds, tt.Texts())
	}
	outerV, innerV, innerText = 3, 2, "Inner 2"
	tt.Frame()
	if innerBuilds != 2 || !tt.HasText("Inner 2") {
		t.Fatalf("built %d, %d: %q", outerBuilds, innerBuilds, tt.Texts())
	}
	if err := tt.Click("Inner button"); err != nil {
		t.Fatal(err)
	}
	if innerBuilds < 3 || outerBuilds < 4 {
		t.Fatalf("a click in the inner region built %d, %d", outerBuilds, innerBuilds)
	}
}

// A region whose element is not built for a frame lets go of what it kept.
func TestMemoRegionGone(t *testing.T) {
	show := true
	builds := 0
	tt := coreNewTester(func(c *context) {
		if show {
			coreColumn(c).Memo(1).Children(func() {
				builds++
				coreText(c, "Shown")
			})
		}
	}, 300, 300)
	tt.Frame()
	show = false
	tt.Frame()
	if len(tt.rt.memos) != 0 {
		t.Fatalf("%d regions kept after their element went", len(tt.rt.memos))
	}
	show = true
	tt.Frame()
	if builds != 2 || !tt.HasText("Shown") {
		t.Fatalf("built %d times, %q", builds, tt.Texts())
	}
}

// Typing into a text input in a region builds it, which shows the text.
func TestMemoTextInput(t *testing.T) {
	value := ""
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Memo(0).Children(func() {
			coreTextInput(c, &value)
		})
	}, 300, 300)
	tt.Key(0, KeyTab)
	tt.Type("hello")
	tt.Frame()
	if value != "hello" {
		t.Fatalf("typed %q", value)
	}
}

// A scroll container in a kept region scrolled by the app takes the
// pointer where its content shows.
func TestMemoScrolledByApp(t *testing.T) {
	var track ScrollState
	builds := 0
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Memo(1).Children(func() {
			builds++
			coreScroll(c).Height(100).TrackScroll(&track).Children(func() {
				for i := range 20 {
					coreButton(c, fmt.Sprint("Row ", i))
				}
			})
		})
	}, 300, 300)
	tt.Frame()
	before, _ := tt.Find("Row 5")
	track.Y = 60
	tt.Frame()
	tt.Frame()
	after, ok := tt.Find("Row 5")
	if !ok || after.Y >= before.Y {
		t.Fatalf("scrolled, row 5 at %v, was at %v (built %d times)", after, before, builds)
	}
	if builds != 1 || tt.rt.memoKept == 0 {
		t.Fatalf("built %d times, kept %d", builds, tt.rt.memoKept)
	}
}
