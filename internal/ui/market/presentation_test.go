package market

import (
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	desktopmarket "whaleisle/internal/desktop/market"
	marketstore "whaleisle/internal/store/market"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestMarketCardsReflowWithTheirContainer(t *testing.T) {
	store := marketstore.NewWithPersonalPath(filepath.Join(t.TempDir(), "market.json"))
	d := marketstore.Discovery{Category: "all", Period: "all", Rows: []desktopmarket.CatalogItem{
		{ID: "owner/first", Repo: "First plugin", Owner: "owner", Description: "First description"},
		{ID: "owner/second", Repo: "Second plugin", Owner: "owner", Description: "Second description"},
	}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			catalogCards(c, Model{Store: store}, d, marketstore.ViewState{}, c.Theme().Text, c.Theme().TextMuted, c.Theme().Surface, c.Theme().Border, c.Theme().Accent)
		})
	}, 800, 1000)
	first, ok1 := tt.Find("First plugin")
	second, ok2 := tt.Find("Second plugin")
	if !ok1 || !ok2 || second.X <= first.X || second.Y != first.Y {
		t.Fatalf("wide container did not place cards side by side: %v %v", first, second)
	}
	saveMarketSnapshot(t, tt, "market-cards-wide.png")
	tt.SetSize(420, 1000)
	first, ok1 = tt.Find("First plugin")
	second, ok2 = tt.Find("Second plugin")
	if !ok1 || !ok2 || second.Y <= first.Y || first.X != second.X || second.X+second.W > 420 {
		t.Fatalf("narrow container did not stack cards within viewport: %v %v", first, second)
	}
	saveMarketSnapshot(t, tt, "market-cards-narrow.png")
}

func TestMarketConfirmationIsModalAndEscapeCancels(t *testing.T) {
	store := marketstore.NewWithPersonalPath(filepath.Join(t.TempDir(), "market.json"))
	run, behind := 0, 0
	store.RequestConfirmation("Install owner/plugin", func() { run++ })
	tt := ui.NewTester(func(c *ui.Context) {
		if ui.Button(c, "Background action").Clicked() {
			behind++
		}
		confirmationView(c, Model{Store: store}, c.Theme().Text, c.Theme().Accent)
	}, 600, 420)
	saveMarketSnapshot(t, tt, "market-confirmation.png")
	if err := tt.Click("Background action"); err != nil {
		t.Fatal(err)
	}
	if run != 0 || behind != 0 {
		t.Fatal("modal permitted an underlying action")
	}
	// A backdrop click may dismiss the dialog. A fresh request checks Escape
	// through the same store owner without executing the operation.
	store.RequestConfirmation("Install owner/plugin", func() { run++ })
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	if store.Confirmation() != "" || run != 0 {
		t.Fatal("Escape did not cancel the pending operation")
	}
	store.RequestConfirmation("Install owner/plugin", func() { run++ })
	tt.Frame()
	if err := tt.Click(marketText(i18n.T(i18n.KeyW4MarketConfirmAndRun), "Confirm and run")); err != nil {
		t.Fatal(err)
	}
	if run != 1 || store.Confirmation() != "" {
		t.Fatal("confirmation did not execute exactly once")
	}
}

func TestMarketDetailsSnapshotFixture(t *testing.T) {
	store := marketstore.NewWithPersonalPath(filepath.Join(t.TempDir(), "market.json"))
	item := desktopmarket.CatalogItem{
		ID: "fixture/workspace-tools", Repo: "Workspace tools", Owner: "fixture",
		Description: "A local fixture showing plugin details and a long repository README.",
		Homepage:    "https://github.com/fixture/workspace-tools", InstallSpec: "fixture-workspace-tools",
	}
	d := marketstore.Discovery{Selected: &item, Details: desktopmarket.Details{
		Version: "1.2.3", Requirements: []string{"node >=22", "Harness >=1.0"},
		Readme: strings.Repeat("Workspace tools keeps related project actions together. This fixture exercises the bounded details scroll area.\n\n", 16),
	}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Background(c.Theme().Background)
		catalogDetailsDialog(c, Model{Store: store}, d, marketstore.ViewState{}, c.Theme().Text, c.Theme().TextMuted, c.Theme().Surface, c.Theme().Accent)
	}, 1000, 800)
	if _, ok := tt.Find("Workspace tools"); !ok {
		t.Fatal("details fixture did not open")
	}
	// The panel fades in from zero opacity; settle the overlay clock so the
	// snapshot shows the shared surface, not its first frame.
	start := time.Unix(1000, 0)
	tt.SetTime(start)
	tt.SetTime(start.Add(time.Second))
	tt.Frame()
	saveMarketSnapshot(t, tt, "market-details-wide.png")
	tt.SetSize(520, 600)
	close, ok := tt.Find(marketText(i18n.T(i18n.KeyW4MarketClose), "Close"))
	if !ok || close.X < 0 || close.Y < 0 || close.X+close.W > 520 || close.Y+close.H > 600 {
		t.Fatalf("narrow details close action escaped viewport: %v", close)
	}
	saveMarketSnapshot(t, tt, "market-details-narrow.png")
}

func saveMarketSnapshot(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("SECONDARY_SNAPSHOT_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, tt.Image()); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestMarketTabsRetainStoreAndEmbeddedRefresh(t *testing.T) {
	store := marketstore.NewWithPersonalPath(filepath.Join(t.TempDir(), "market.json"))
	model := Model{Store: store, Embedded: true}
	tt := ui.NewTester(func(c *ui.Context) { View(c, model) }, 650, 800)
	if err := tt.Click(marketText(i18n.T(i18n.KeyW4MarketInstallHistory), "History")); err != nil {
		t.Fatal(err)
	}
	if store.Discovery().Tab != "history" {
		t.Fatal("tab did not update the existing marketplace store")
	}
	if _, ok := tt.Find(i18n.T(i18n.KeyMarketBack)); ok {
		t.Fatal("embedded section retained a global back action")
	}
	if _, ok := tt.Find(i18n.T(i18n.KeyMarketRefresh)); !ok {
		t.Fatal("embedded section lost catalog refresh")
	}
}

// TestMarketDetailsDrawsItsHairlineInTheDark pins the elevation stroke: with no
// wallpaper the glass fills the panel opaquely, so the hairline has to paint
// after it or it disappears.
func TestMarketDetailsDrawsItsHairlineInTheDark(t *testing.T) {
	store := marketstore.NewWithPersonalPath(filepath.Join(t.TempDir(), "market.json"))
	item := desktopmarket.CatalogItem{
		ID: "fixture/hairline", Repo: "Hairline fixture", Owner: "fixture",
		Description: "A local fixture for the details dialog's elevation hairline.",
	}
	d := marketstore.Discovery{Selected: &item}
	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(ui.DarkTheme())
		ui.Column(c).Fill().Background(c.Theme().Background)
		catalogDetailsDialog(c, Model{Store: store, ThemeID: theme.IDDeepseekDark, SystemDark: true}, d, marketstore.ViewState{}, c.Theme().Text, c.Theme().TextMuted, c.Theme().Surface, c.Theme().Accent)
	}, 900, 700)
	// The panel fades in; a settled clock shows its material, not frame one.
	start := time.Unix(1000, 0)
	tt.SetTime(start)
	tt.SetTime(start.Add(time.Second))
	// The panel is min(680, w-32) × min(720, h-48), centered by DialogBase with
	// its 16 DIP padding: 680×652 at (110, 24).
	image := tt.Image()
	const panelTop, panelLeft = 24, 110
	inside := ttColor(image.RGBAAt(panelLeft+200, panelTop+6))
	edge := ttColor(image.RGBAAt(panelLeft+200, panelTop))
	contrast := chrome.ContrastRatio(edge, inside)
	t.Logf("details hairline %v on fill %v = %.2f:1", edge, inside, contrast)
	if contrast < 1.15 {
		t.Fatalf("details hairline invisible in the dark: %.2f:1 (edge %v on fill %v)", contrast, edge, inside)
	}
	if edge.R <= inside.R {
		t.Fatalf("details hairline is darker than its fill: %v on %v", edge, inside)
	}
}

// ttColor reads a rendered pixel as a theme color.
func ttColor(c color.RGBA) ui.Color { return ui.RGBA(c.R, c.G, c.B, float32(c.A)/255) }
