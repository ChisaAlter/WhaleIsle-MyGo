package market

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/imgcache"
	desktopmarket "whaleisle/internal/desktop/market"
	marketstore "whaleisle/internal/store/market"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

type cossRegressionCatalog struct {
	items []desktopmarket.CatalogItem
}

func (f cossRegressionCatalog) List(context.Context, bool, string) (desktopmarket.Catalog, error) {
	return desktopmarket.Catalog{
		Source:  "fixture",
		Updated: "2026-11-24",
		Categories: []desktopmarket.Category{
			{ID: "all", Label: "All", Count: len(f.items)},
			{ID: "tools", Label: "Tools", Count: len(f.items)},
		},
		Items: f.items,
	}, nil
}

func (f cossRegressionCatalog) Details(context.Context, desktopmarket.CatalogItem, bool) (desktopmarket.Details, error) {
	return desktopmarket.Details{}, nil
}

func newSettledStore(t *testing.T, items []desktopmarket.CatalogItem) *marketstore.Store {
	t.Helper()
	s := marketstore.NewWithPersonalPath(filepath.Join(t.TempDir(), "market.json"))
	s.SetCatalogService(cossRegressionCatalog{items: items})
	s.RefreshCatalog(false, "en")
	deadline := time.Now().Add(2 * time.Second)
	for s.Discovery().Loading && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if s.Discovery().Loading {
		t.Fatalf("catalog did not settle: %+v", s.Discovery())
	}
	return s
}

func solidPNGBytes(t *testing.T, c color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 24, 24))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func paintedColorCount(tt *ui.Tester, want color.RGBA) int {
	const tol = 8
	img := tt.Image()
	n := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			px := img.RGBAAt(x, y)
			if d := int(px.R) - int(want.R); d < -tol || d > tol {
				continue
			}
			if d := int(px.G) - int(want.G); d < -tol || d > tol {
				continue
			}
			if d := int(px.B) - int(want.B); d < -tol || d > tol {
				continue
			}
			n++
		}
	}
	return n
}

func TestMarketMemoInvalidatesOnDesignTokensThemeStoreMediaAndLocale(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	itemA := desktopmarket.CatalogItem{
		ID: "fixture/alpha", Repo: "Alpha Plugin", Owner: "fixture-owner",
		Category: "tools", Description: "Alpha description", Stars: 42,
		NativeCompatible: true, CapabilityKind: desktopmarket.CapabilityPureHost,
	}
	itemB := desktopmarket.CatalogItem{
		ID: "fixture/beta", Repo: "Beta Plugin", Owner: "fixture-owner",
		Category: "tools", Description: "Beta description", Stars: 99,
		NativeCompatible: true, CapabilityKind: desktopmarket.CapabilityPureHost,
	}

	cache := imgcache.New(t.TempDir())
	sMedia := useTestMedia(t, cache)
	avatarURL := "https://github.com/fixture-owner.png?size=96"
	magentaAvatar := color.RGBA{R: 216, G: 0, B: 200, A: 255}
	if err := os.WriteFile(cache.FilePath(avatarURL), solidPNGBytes(t, magentaAvatar), 0o600); err != nil {
		t.Fatal(err)
	}

	sA := newSettledStore(t, []desktopmarket.CatalogItem{itemA})
	sB := newSettledStore(t, []desktopmarket.CatalogItem{itemB})
	if sA.Snapshot().Rev != sB.Snapshot().Rev || sA.DiscoveryVersion().Rev != sB.DiscoveryVersion().Rev {
		t.Fatalf("setup revisions differ: snap %d vs %d, disc %d vs %d",
			sA.Snapshot().Rev, sB.Snapshot().Rev, sA.DiscoveryVersion().Rev, sB.DiscoveryVersion().Rev)
	}

	cur := Model{Store: sA, Media: sMedia.NewOwner(nil, nil)}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { View(c, cur) })
	}, 960, 800)
	tt.Frame()
	tt.Frame()

	// 1. design.Revision() invalidates memoized card surface.
	titleBox, ok := tt.Find("Alpha Plugin")
	if !ok {
		t.Fatal("Alpha Plugin card missing")
	}
	sampleX, sampleY := int(titleBox.X), int(titleBox.Y+titleBox.H+2)
	cardMagenta := ui.RGB(255, 0, 255)
	design.SetGlass(true, func(_ design.Scheme, p *design.Palette) {
		p.Card = cardMagenta
	})
	t.Cleanup(func() { design.SetGlass(false, nil) })
	tt.Frame()
	afterGlass := tt.Image().At(sampleX, sampleY)

	ttFresh := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { View(c, cur) })
	}, 960, 800)
	ttFresh.Frame()
	freshGlass := ttFresh.Image().At(sampleX, sampleY)
	design.SetGlass(false, nil)
	if afterGlass != freshGlass {
		t.Fatalf("memoized market card ignored design.SetGlass Card token change: got %v want %v",
			color.RGBAModel.Convert(afterGlass), color.RGBAModel.Convert(freshGlass))
	}

	// 2. Media lease swap + MediaWatch epoch reloads owner avatar inside memoized card.
	cur.Media.Release()
	cur.Media = sMedia.NewOwner(nil, nil)
	for range 3 {
		tt.Frame()
	}
	settleMedia(t, tt, sMedia, func() bool { return paintedColorCount(tt, magentaAvatar) > 0 })

	// 3. Locale switch invalidates memoized card labels.
	i18n.SetLocale(i18n.LocaleZH)
	tt.Frame()
	if !tt.HasText("详情") || !tt.HasText("收藏") {
		t.Fatalf("memoized card retained English labels after locale switch: %v", tt.Texts())
	}
	i18n.SetLocale(i18n.LocaleEN)
	tt.Frame()

	// 4. Store identity swap at identical Rev rebuilds memoized cards.
	cur.Store = sB
	tt.Frame()
	if !tt.HasText("Beta Plugin") || tt.HasText("Alpha Plugin") {
		t.Fatalf("memoized catalog card stayed stale after Store swap at same Rev: %v", tt.Texts())
	}

	// 5. theme.Revision() bumps on SetPreferences and changes memo version.
	prevPrefs := theme.CurrentPreferences()
	t.Cleanup(func() { theme.SetPreferences(prevPrefs) })
	vBefore := testPage(cur, cur.Store.Discovery(), cur.Store.Snapshot().Value).version()
	nextPrefs := prevPrefs
	nextPrefs.Chat.Size = prevPrefs.Chat.Size + 2
	theme.SetPreferences(nextPrefs)
	vAfter := testPage(cur, cur.Store.Discovery(), cur.Store.Snapshot().Value).version()
	if vAfter == vBefore {
		t.Fatal("memo version did not change when theme.Revision() advanced")
	}
}

func TestMarketChangedInputsAndFiltersApplyImmediately(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	items := []desktopmarket.CatalogItem{
		{ID: "org/tool-one", Repo: "Tool One", Owner: "org", Category: "tools", Description: "First tool", NativeCompatible: true, CapabilityKind: desktopmarket.CapabilityPureHost},
		{ID: "org/other-two", Repo: "Other Two", Owner: "org", Category: "themes", Description: "Theme pack", NativeCompatible: true, CapabilityKind: desktopmarket.CapabilityPureHost},
	}
	s := newSettledStore(t, items)
	spec := ""
	var installedSpecs []string
	m := Model{
		Store:        s,
		InstallReady: true,
		InstallSpec:  &spec,
		OnInstallSpec: func(sp string, _ []string) {
			installedSpecs = append(installedSpecs, sp)
		},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { View(c, m) })
	}, 960, 900)
	clock := time.Unix(1000, 0)
	tt.SetTime(clock)

	// Category toggle via Changed().
	if err := tt.Click("Tools 2"); err != nil {
		t.Fatalf("click category toggle failed: %v", err)
	}
	if s.Discovery().Category != "tools" {
		t.Fatalf("category toggle did not update store via Changed(): got %q", s.Discovery().Category)
	}

	// Search input via Changed().
	if err := tt.Click("Search name, package or description"); err != nil {
		t.Fatalf("focus search input failed: %v", err)
	}
	tt.Type("tool-one")
	tt.Frame()
	if s.Discovery().Query != "tool-one" || !tt.HasText("Tool One") || tt.HasText("Other Two") {
		t.Fatalf("search query did not filter via Changed(): query=%q texts=%v", s.Discovery().Query, tt.Texts())
	}

	// Switch to Installed tab, test management filter Changed() and InstallSpec Changed().
	if err := tt.Click("Installed"); err != nil {
		t.Fatalf("click Installed tab failed: %v", err)
	}
	if err := tt.Click("Failed / interrupted"); err != nil {
		t.Fatalf("click Failed / interrupted filter failed: %v", err)
	}
	if s.ManagementFilter() != "failed" {
		t.Fatalf("management filter toggle did not apply via Changed(): got %q", s.ManagementFilter())
	}

	if err := tt.Click(i18n.T(i18n.KeyMarketInstallPlaceholder)); err != nil {
		t.Fatalf("focus install spec input failed: %v", err)
	}
	tt.Type("custom-plugin@1.2.3")
	tt.Frame()
	if err := tt.Click(i18n.T(i18n.KeyMarketInstallButton)); err != nil {
		t.Fatalf("click Install spec button failed: %v", err)
	}
	tt.Frame()
	clock = clock.Add(time.Second)
	tt.SetTime(clock)
	tt.Frame()
	if s.Confirmation() == "" || !strings.Contains(s.Confirmation(), "custom-plugin@1.2.3") {
		t.Fatalf("InstallSpec Changed() did not populate confirmation: %q", s.Confirmation())
	}
	if err := tt.Click("Confirm and run"); err != nil {
		t.Fatalf("click Confirm and run failed: %v", err)
	}
	if len(installedSpecs) != 1 || installedSpecs[0] != "custom-plugin@1.2.3" {
		t.Fatalf("InstallSpec callback got %v, want [custom-plugin@1.2.3]", installedSpecs)
	}
}

func TestMarketNeverPaintsRawErrors(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	s := marketstore.NewWithPersonalPath(filepath.Join(t.TempDir(), "market.json"))
	m := Model{
		Store:          s,
		InstallMessage: "EACCES: open /secret/install/path: permission denied",
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			View(c, m)
			renderCatalogWarning(c, "Catalog refresh failed: open /secret/catalog/path: EACCES")
			renderPersonalError(c, "open /secret/personal/path: permission denied")
			marketHistoryRow(c, marketstore.InstallHistory{
				At:         time.Now(),
				Plugin:     "org/plugin",
				Result:     "failed",
				Error:      "EACCES: /secret/history/error",
				Diagnostic: "open /secret/diagnostic/path: permission denied",
			})
		})
	}, 960, 900)
	joined := strings.Join(tt.Texts(), "\n")
	for _, secret := range []string{"EACCES", "/secret/", "permission denied"} {
		if strings.Contains(joined, secret) {
			t.Fatalf("market surface leaked raw error substring %q into UI:\n%s", secret, joined)
		}
	}
}

func TestMarketStandaloneVsEmbeddedBehavior(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	s := newSettledStore(t, nil)
	s.SetTab("installed")
	closed, openedManager := 0, 0
	spec := ""

	// Standalone: Back button closes; InstallSpec is shown in Installed tab.
	standalone := Model{
		Store:       s,
		Embedded:    false,
		InstallSpec: &spec,
		OnClose:     func() { closed++ },
	}
	ttStandalone := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { View(c, standalone) })
	}, 960, 800)
	if !ttStandalone.HasText(i18n.T(i18n.KeyMarketInstallSpecLabel)) {
		t.Fatal("standalone Installed tab omitted InstallSpec input")
	}
	if err := ttStandalone.Click(i18n.T(i18n.KeyMarketBack)); err != nil {
		t.Fatalf("standalone Back button missing or unclickable: %v", err)
	}
	if closed != 1 {
		t.Fatalf("standalone Back button calls=%d, want 1", closed)
	}

	// Embedded: Back button omitted; Manage plugin packages opens manager.
	embedded := Model{
		Store:         s,
		Embedded:      true,
		OnClose:       func() { closed++ },
		OnOpenManager: func() { openedManager++ },
	}
	ttEmbedded := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { View(c, embedded) })
	}, 960, 800)
	if ttEmbedded.HasText(i18n.T(i18n.KeyMarketBack)) {
		t.Fatal("embedded mode rendered standalone Back button")
	}
	if ttEmbedded.HasText(i18n.T(i18n.KeyMarketInstallSpecLabel)) {
		t.Fatal("embedded mode rendered standalone InstallSpec input")
	}
	if err := ttEmbedded.Click("Manage plugin packages"); err != nil {
		t.Fatalf("embedded Manage plugin packages button missing: %v", err)
	}
	if openedManager != 1 {
		t.Fatalf("embedded OnOpenManager calls=%d, want 1", openedManager)
	}
}
