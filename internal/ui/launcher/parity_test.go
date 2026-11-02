package launcher

import (
	"fmt"
	"image"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/components"
	"whaleisle/internal/product"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/widgets"
)

func chevronInkY(img image.Image, trigger ui.Rect) float64 {
	sum, count := 0, 0
	for y := int(trigger.Y + trigger.H/2 - 6); y < int(trigger.Y+trigger.H/2+6); y++ {
		for x := int(trigger.X + trigger.W - 24); x < int(trigger.X+trigger.W-12); x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if r < 180*257 && g < 180*257 && b < 180*257 {
				sum += y
				count++
			}
		}
	}
	if count == 0 {
		return -1
	}
	return float64(sum) / float64(count)
}

func TestHeaderChevronRotatesWhenOpen(t *testing.T) {
	route := "github"
	tt := ui.NewTester(func(c *ui.Context) { View(c, Model{ActiveTab: TabVersions, DownloadRoute: &route}) }, 1060, 660)
	label := i18n.T(i18n.KeyLauncherRouteCurrentPrefix) + " GitHub"
	trigger, ok := tt.Find(label)
	if !ok {
		t.Fatal("header trigger missing")
	}
	before := chevronInkY(tt.Image(), trigger)
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
	after := chevronInkY(tt.Image(), trigger)
	if before < 0 || after < 0 || after <= before {
		t.Fatalf("open chevron did not rotate upward: closed inkY=%v open inkY=%v", before, after)
	}
}

// Exact equality matters: HasText("鲸") also matches the unrelated desktop title.
func exactTextCount(tt *ui.Tester, want string) int {
	n := 0
	for _, text := range tt.Texts() {
		if text == want {
			n++
		}
	}
	return n
}

func TestRouteKeyboardNavigationCommitsOnlyOnEnter(t *testing.T) {
	for _, keys := range [][]ui.Key{
		{ui.KeyDown},
		{ui.KeyEnd},
		{ui.KeyDown, ui.KeyHome, ui.KeyDown},
		{ui.KeyEnd, ui.KeyUp, ui.KeyDown},
	} {
		t.Run(fmt.Sprint(keys), func(t *testing.T) {
			route, open, saves := "github", false, 0
			tt := ui.NewTester(func(c *ui.Context) {
				View(c, Model{VendorReady: true, DownloadRoute: &route, RoutePopOpen: &open, OnSaveLauncherPrefs: func() { saves++ }})
			}, 1060, 660)
			if err := tt.Click(i18n.T(i18n.KeyLauncherRouteLabel)); err != nil {
				t.Fatal(err)
			}
			for _, key := range keys {
				tt.Key(0, key)
			}
			if route != "github" || saves != 0 {
				t.Fatal("navigation committed a route before Enter")
			}
			tt.Key(0, ui.KeyEnter)
			if route != "cnb" || open || saves != 1 {
				t.Fatalf("route=%q open=%v saves=%d; want cnb, false, 1", route, open, saves)
			}
		})
	}
}

func TestRouteKeyboardSelectionFollowsHighlightAfterTab(t *testing.T) {
	route, open, saves := "github", false, 0
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{VendorReady: true, DownloadRoute: &route, RoutePopOpen: &open, OnSaveLauncherPrefs: func() { saves++ }})
	}, 1060, 660)
	if err := tt.Click(i18n.T(i18n.KeyLauncherRouteLabel)); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeyEnd)
	tt.Key(0, ui.KeyEnter)
	if route != "cnb" || open || saves != 1 {
		t.Fatalf("after Tab/End/Enter: route=%q open=%v saves=%d", route, open, saves)
	}
}

func TestRouteSelectionRestoresTriggerFocus(t *testing.T) {
	route, open := "github", false
	tt := ui.NewTester(func(c *ui.Context) { View(c, Model{VendorReady: true, DownloadRoute: &route, RoutePopOpen: &open}) }, 1060, 660)
	footer := i18n.T(i18n.KeyLauncherRouteLabel)
	if err := tt.Click(footer); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(i18n.T(i18n.KeyLauncherRouteCNB)); err != nil {
		t.Fatal(err)
	}
	if !tt.Focused(footer) {
		t.Fatal("selecting a route did not restore trigger focus")
	}
	tt.Key(0, ui.KeyEnter)
	if !open {
		t.Fatal("focused trigger could not reopen menu")
	}
}

func TestRouteTabSelectionFollowsFocusedOption(t *testing.T) {
	route, open := "github", false
	tt := ui.NewTester(func(c *ui.Context) { View(c, Model{VendorReady: true, DownloadRoute: &route, RoutePopOpen: &open}) }, 1060, 660)
	if err := tt.Click(i18n.T(i18n.KeyLauncherRouteLabel)); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeyTab)
	if !tt.Focused(i18n.T(i18n.KeyLauncherRouteCNB)) {
		t.Fatal("Tab did not focus CNB")
	}
	tt.Key(0, ui.KeyEnter)
	if route != "cnb" || open {
		t.Fatalf("Tab/Tab/Enter: route=%q open=%v", route, open)
	}
}

func TestStageLongComponentListScrollsWithoutMovingRail(t *testing.T) {
	rows := make([]components.Row, 20)
	for i := range rows {
		rows[i] = components.Row{ID: fmt.Sprint(i), Name: fmt.Sprintf("Component-%02d", i), State: "available"}
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, Model{ActiveTab: TabComponents, Components: rows}) }, 1060, 660)
	first, _ := tt.Find("Component-00")
	rail, railFound := tt.Find(product.Name)
	tt.Scroll(800, 450, 0, 600)
	after, _ := tt.Find("Component-00")
	railAfter, railAfterFound := tt.Find(product.Name)
	if !railFound || !railAfterFound {
		t.Fatalf("rail brand %q not found: before=%v after=%v", product.Name, railFound, railAfterFound)
	}
	if after.Y >= first.Y {
		t.Fatalf("stage did not scroll: first=%+v after=%+v", first, after)
	}
	if railAfter != rail {
		t.Fatalf("stage scrolling moved rail: before=%+v after=%+v", rail, railAfter)
	}
	frame := tt.Image()
	for y := 2; y < 56; y++ {
		for x := 250; x < 950; x++ {
			r, g, b, _ := frame.At(x, y).RGBA()
			if r != 65535 || g != 65535 || b != 65535 {
				t.Fatalf("scrolled content paints over fixed stage top padding at (%d,%d)", x, y)
			}
		}
	}
	tt.Scroll(800, 450, 0, 100000)
	last, ok := tt.Find("Component-19")
	if !ok || last.H <= 0 || last.Y < 56 || last.Y+last.H > 636 {
		t.Fatalf("last component cannot be reached inside stage viewport: %+v", last)
	}
}

func TestRailRoutePopupDoesNotMoveTrigger(t *testing.T) {
	route, open := "github", false
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{VendorReady: true, DownloadRoute: &route, RoutePopOpen: &open})
	}, 1060, 660)
	before, ok := tt.Find(i18n.T(i18n.KeyLauncherRouteLabel))
	if !ok {
		t.Fatal("missing footer route trigger")
	}
	if err := tt.Click(i18n.T(i18n.KeyLauncherRouteLabel)); err != nil {
		t.Fatal(err)
	}
	if !open {
		t.Fatal("footer route did not open")
	}
	after, _ := tt.Find(i18n.T(i18n.KeyLauncherRouteLabel))
	if before != after {
		t.Fatalf("opening route popup moved trigger: before=%+v after=%+v", before, after)
	}
	option, ok := tt.Find(i18n.T(i18n.KeyLauncherRouteCNB))
	if !ok || option.Y >= after.Y {
		t.Fatalf("footer popup must overlay above trigger: option=%+v trigger=%+v", option, after)
	}
}

func TestUpwardRoutePopupLeavesSixPixelGap(t *testing.T) {
	route, open := "github", false
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Background(colAccent).Children(func() {
			trigger := ui.Button(c, "Route").Absolute().Left(20).Top(300).Width(140).Height(26)
			routePopup(c, Model{DownloadRoute: &route}, trigger, &open, true)
		})
	}, 400, 400)
	want := tt.Image().RGBAAt(30, 297)
	open = true
	tt.Frame()
	if got := tt.Image().RGBAAt(30, 297); got.B <= got.R*2 || got.B > want.B {
		t.Fatalf("upward popup covers gap above trigger: got=%v want=%v", got, want)
	}
}

func TestRailRoutePopupEscapeAndOutsideDismiss(t *testing.T) {
	for _, dismiss := range []string{"escape", "outside"} {
		t.Run(dismiss, func(t *testing.T) {
			route, open := "github", false
			tt := ui.NewTester(func(c *ui.Context) {
				View(c, Model{VendorReady: true, DownloadRoute: &route, RoutePopOpen: &open})
			}, 1060, 660)
			if err := tt.Click(i18n.T(i18n.KeyLauncherRouteLabel)); err != nil {
				t.Fatal(err)
			}
			if dismiss == "escape" {
				tt.Key(0, ui.KeyEscape)
			} else {
				tt.ClickAt(500, 400)
			}
			if open {
				t.Fatalf("route popup remained open after %s", dismiss)
			}
			if route != "github" {
				t.Fatalf("dismissal changed route to %q", route)
			}
		})
	}
}

func TestRailBrandShowsFullProductName(t *testing.T) {
	tt := ui.NewTester(railHead, 188, 100)
	if n := exactTextCount(tt, product.Name); n != 1 {
		t.Fatalf("product name count = %d, want 1; texts=%q", n, tt.Texts())
	}
	for _, old := range []string{"鲸", "屿", "WHALE ISLE"} {
		if exactTextCount(tt, old) != 0 {
			t.Errorf("old wordmark %q remains in brand: %q", old, tt.Texts())
		}
	}
}

func TestBrandWordmarkFitsNarrowRail(t *testing.T) {
	tt := ui.NewTester(railHead, 164, 100)
	name, ok := tt.Find(product.Name)
	if !ok || name.X < 0 || name.X+name.W > 164 || name.H > 36 {
		t.Fatalf("product name overflows narrow rail: %+v found=%v", name, ok)
	}
	role, ok := tt.Find(i18n.T(i18n.KeyLauncherBrandRole))
	if !ok || role.Y < name.Y+name.H || role.Y+role.H > 100 {
		t.Errorf("role must follow product name: name=%+v role=%+v", name, role)
	}
}

func TestRouteOptionsShowReferenceCopyAndSelectedCheck(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	route, open := "github", true
	tt := ui.NewTester(func(c *ui.Context) {
		trigger := ui.Button(c, "Route").Absolute().Left(30).Top(300).Width(140).Height(26)
		routePopup(c, Model{DownloadRoute: &route}, trigger, &open, true)
	}, 500, 400)
	// The menu enters on the popover token, so read it once that has run.
	start := time.Unix(1000, 0)
	settled := start.Add(chrome.PopoverDuration + time.Millisecond)
	tt.SetTime(start)
	tt.SetTime(settled)
	for _, label := range []string{"GitHub", "CNB", "国外线路 · GitHub Releases 直连", "国内线路 · CNB 镜像", "✓"} {
		if exactTextCount(tt, label) != 1 {
			t.Errorf("missing exact route content %q; texts=%q", label, tt.Texts())
		}
	}
	check, ok := tt.Find("✓")
	if !ok {
		return
	}
	github, _ := tt.Find("GitHub")
	desc, _ := tt.Find("国外线路 · GitHub Releases 直连")
	if github.X-check.X < 23 || github.X-check.X > 25 {
		t.Errorf("expected14px check column +10px gap: check=%+v title=%+v", check, github)
	}
	if desc.X != github.X || desc.Y <= github.Y {
		t.Errorf("description must sit below title in same column: title=%+v description=%+v", github, desc)
	}
	if github.H > 18 || desc.Y-github.Y-github.H < 1 || desc.Y-github.Y-github.H > 3 {
		t.Errorf("route title must use normal13px font metrics and2px description gap: title=%+v description=%+v", github, desc)
	}
	// Current selection is shown by the check, with no permanent blue wash.
	plain := tt.Image().RGBAAt(259, int(github.Y+3))
	if plain.R != 255 || plain.G != 255 || plain.B != 255 {
		t.Errorf("selected option has a non-reference fill: %v", plain)
	}
	tt.Move(github.X+github.W/2, github.Y+github.H/2)
	// The hover wash eases in over the fast token.
	tt.SetTime(settled.Add(widgets.FastDuration + 10*time.Millisecond))
	hover := tt.Image().RGBAAt(259, int(github.Y+3))
	if hover == plain || hover.R < 238 || hover.B < 238 {
		t.Errorf("route hover must use subtle reference wash: plain=%v hover=%v", plain, hover)
	}
	tt.Move(0, 0)
	tt.Key(0, ui.KeyEnd)
	stillChecked, _ := tt.Find("✓")
	if stillChecked.Y != check.Y {
		t.Fatal("navigation moved selected check before commit")
	}
	if err := tt.Click("CNB"); err != nil {
		t.Fatal(err)
	}
	open = true
	tt.Frame()
	check, _ = tt.Find("✓")
	cnb, _ := tt.Find("CNB")
	if exactTextCount(tt, "✓") != 1 || check.Y < cnb.Y-2 || check.Y > cnb.Y+4 {
		t.Errorf("CNB selection missing check: check=%+v CNB=%+v texts=%q", check, cnb, tt.Texts())
	}
}

func TestDarkRouteMenuUsesReferenceLayerTwo(t *testing.T) {
	route, open := "github", true
	tt := ui.NewTester(func(c *ui.Context) {
		trigger := ui.Button(c, "Route").Absolute().Left(30).Top(300).Width(140).Height(26)
		routePopup(c, Model{DownloadRoute: &route, SystemDark: true}, trigger, &open, true)
	}, 500, 400)
	// The menu enters on the popover token, so read it once that has run: pin the
	// clock and settle the frame, as the shell's other motion tests do.
	start := time.Unix(1000, 0)
	tt.SetTime(start)
	tt.SetTime(start.Add(chrome.PopoverDuration + time.Millisecond))
	got := tt.Image().RGBAAt(45, 290)
	if got.R != 44 || got.G != 44 || got.B != 46 {
		t.Fatalf("dark menu layer2=%v, want rgb(44,44,46)", got)
	}
}

func TestRailRouteHasNoQuestionMarkPlaceholder(t *testing.T) {
	route := "github"
	open := false
	tt := ui.NewTester(func(c *ui.Context) {
		railRouteChip(c, Model{VendorReady: true, DownloadRoute: &route, RoutePopOpen: &open})
	}, 164, 60)
	if exactTextCount(tt, "?") != 0 {
		t.Fatalf("route chevron is a literal question mark: %q", tt.Texts())
	}
}

func TestHeaderRoutePopupIsIndependentOfFooter(t *testing.T) {
	for _, tab := range []string{TabComponents, TabVersions} {
		t.Run(tab, func(t *testing.T) {
			route, footerOpen := "github", false
			tt := ui.NewTester(func(c *ui.Context) { View(c, Model{ActiveTab: tab, DownloadRoute: &route, RoutePopOpen: &footerOpen}) }, 1060, 660)
			label := i18n.T(i18n.KeyLauncherRouteCurrentPrefix) + " " + railRouteLabel(Model{VendorReady: true, DownloadRoute: &route})
			trigger, ok := tt.Find(label)
			if !ok {
				t.Fatalf("missing header route trigger %q", label)
			}
			if err := tt.Click(label); err != nil {
				t.Fatal(err)
			}
			if footerOpen {
				t.Fatal("header route trigger opened footer state")
			}
			option, ok := tt.Find(i18n.T(i18n.KeyLauncherRouteCNB))
			if !ok || option.Y < trigger.Y || option.Y > trigger.Y+140 {
				t.Fatalf("header popup is not anchored near header: option=%+v trigger=%+v", option, trigger)
			}
		})
	}
}

func TestHeaderRoutePopupClosesOnProgrammaticTabChange(t *testing.T) {
	route, tab := "github", TabComponents
	tt := ui.NewTester(func(c *ui.Context) { View(c, Model{ActiveTab: tab, VendorReady: true, DownloadRoute: &route}) }, 1060, 660)
	if err := tt.Click(i18n.T(i18n.KeyLauncherRouteCurrentPrefix) + " GitHub"); err != nil {
		t.Fatal(err)
	}
	tab = TabVersions
	tt.Frame()
	if exactTextCount(tt, i18n.T(i18n.KeyLauncherRouteCNB)) != 0 {
		t.Fatal("header popup leaked into another page after programmatic tab change")
	}
}

func TestHomeComponentsUseResponsiveGrid(t *testing.T) {
	rows := []components.Row{{ID: "one", Name: "Card One", InstalledVersion: "1.0", State: "available"}, {ID: "two", Name: "Card Two", InstalledVersion: "1.0", State: "available"}, {ID: "three", Name: "Card Three", InstalledVersion: "1.0", State: "available"}}
	tt := ui.NewTester(func(c *ui.Context) { View(c, Model{ActiveTab: TabHome, VendorReady: true, Components: rows}) }, 1060, 660)
	first, _ := tt.Find("Card One")
	second, _ := tt.Find("Card Two")
	third, _ := tt.Find("Card Three")
	if first.Y != second.Y || second.X-first.X < 296 || third.Y <= first.Y {
		t.Errorf("wide stage must have two columns with >=280 DIP cells and16 DIP gap: first=%+v second=%+v third=%+v", first, second, third)
	}
	// Sampling the gap catches full-grid-width children that overlap the next
	// cell even though their title positions appear to form two columns.
	x := 625 // 188px rail +24px padding +1px border +404px cell +8px half-gap
	y := int(first.Y + first.H/2)
	r, g, b, _ := tt.Image().At(x, y).RGBA()
	if r != 65535 || g != 65535 || b != 65535 {
		t.Errorf("grid gutter at (%d,%d) is covered by a card: rgb=%d,%d,%d", x, y, r, g, b)
	}
	tt.SetSize(700, 660)
	first, _ = tt.Find("Card One")
	second, _ = tt.Find("Card Two")
	if second.Y <= first.Y || second.X != first.X {
		t.Errorf("narrow stage must have one column: first=%+v second=%+v", first, second)
	}
	tt.SetSize(812, 660)
	first, _ = tt.Find("Card One")
	second, _ = tt.Find("Card Two")
	if first.Y == second.Y {
		t.Errorf("shell border leaves only574 DIPs; two280 DIP cards plus16 gap must not fit: first=%+v second=%+v", first, second)
	}
	tt.SetSize(1400, 660)
	first, _ = tt.Find("Card One")
	third, _ = tt.Find("Card Three")
	if first.Y != third.Y || third.X <= first.X {
		t.Errorf("large stage must have three columns: first=%+v third=%+v", first, third)
	}
}
