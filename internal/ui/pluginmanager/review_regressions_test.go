package pluginmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

type reviewFixture struct {
	mu        sync.Mutex
	bundles   []manager.Bundle
	indexes   []manager.Index
	metadata  manager.MarketMetadata
	entryIcon string
}

func newReviewFixture() *reviewFixture {
	return &reviewFixture{
		bundles: []manager.Bundle{
			{
				Name: "fixture-plugin", Version: "1.0.0", Availability: "profile",
				Installed: true, Removable: true, Enabled: true,
				Description: "Fixture desc",
				Meta:        &manager.Meta{Title: manager.Text(`"Fixture plugin"`)},
				Rows: []manager.BundleRow{
					{RowID: "patch-id", ModuleName: "fixture-module", EntryID: "loader-fixture", Meta: &manager.Meta{Title: manager.Text(`"Fixture component"`)}},
				},
			},
		},
		indexes: []manager.Index{
			{URL: "https://one.invalid/index.json", Title: manager.Text(`"Market One"`)},
			{URL: "https://two.invalid/index.json", Title: manager.Text(`"Market Two"`)},
		},
		metadata: manager.MarketMetadata{
			Categories: map[string]manager.NamedMetadata{
				"tools":  {Title: manager.Text(`"Tools"`)},
				"themes": {Title: manager.Text(`"Themes"`)},
			},
			Labels: map[string]manager.NamedMetadata{
				"verified": {Title: manager.Text(`"Verified"`)},
			},
		},
	}
}

func (f *reviewFixture) Call(_ context.Context, method string, args, out any) error {
	f.mu.Lock()
	bundles := append([]manager.Bundle(nil), f.bundles...)
	indexes := append([]manager.Index(nil), f.indexes...)
	meta := f.metadata
	icon := f.entryIcon
	f.mu.Unlock()

	var value any
	switch method {
	case "pluginInventory/list":
		value = map[string]any{"managementAvailable": true, "entries": []any{}}
	case "pluginManager/listBundles":
		value = bundles
	case "pluginManager/listPlugins":
		value = []manager.Plugin{{EntryID: "loader-fixture", Enabled: true, Phase: "active"}}
	case "pluginMarket/listIndexes":
		value = indexes
	case "pluginMarket/search":
		rawArgs, _ := json.Marshal(args)
		var req struct {
			URL string `json:"url"`
		}
		_ = json.Unmarshal(rawArgs, &req)
		entry := manager.MarketEntry{
			Name: "fixture-discover", Title: manager.Text(`"Discover fixture"`),
			Description: manager.Text(`"Desc"`), Author: "Author", DefaultVersion: "2.0.0",
			Icon: icon,
		}
		entry.Metadata.Category = "tools"
		entry.Metadata.Labels = []string{"verified"}
		entry.Installation.Status = "not-installed"
		ver := manager.MarketVersion{Version: "2.0.0", Compatibility: manager.Compatibility{Compatible: true}}
		ver.Install.Spec = "fixture-discover@2.0.0"
		entry.Versions = []manager.MarketVersion{ver}
		value = manager.SearchResult{
			URL: req.URL, Title: manager.Text(`"Market One"`), Metadata: meta,
			FetchedAt: 10, Entries: []manager.MarketEntry{entry}, Total: 1,
		}
	case "pluginManager/registries":
		official := manager.OfficialNPMRegistry
		value = manager.RegistryInfo{Registry: &official, Resolved: &official, FallbackRegistries: []string{manager.NPMMirrorRegistry}}
	default:
		return fmt.Errorf("review fixture refuses %s", method)
	}
	if out == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

func reviewModel(t *testing.T, f *reviewFixture) Model {
	t.Helper()
	c := manager.New()
	c.Attach(f, nil)
	if err := c.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	return Model{Controller: c}
}

func settleReviewFrames(tt *ui.Tester) {
	for range 12 {
		tt.Frame()
		time.Sleep(16 * time.Millisecond)
	}
}

func TestDiscoverCategoryPillSelectsCategory(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	m := reviewModel(t, newReviewFixture())
	owner := m.Controller.Snapshot().Value.Owner
	if err := m.Controller.LoadIndexes(context.Background(), owner); err != nil {
		t.Fatal(err)
	}
	m.Controller.Actions(owner).SelectTab("discover")

	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, m) }) }, 1200, 900)
	tt.Frame()
	if err := tt.Click("Tools"); err != nil {
		t.Fatalf("click Tools: %v", err)
	}
	tt.Frame()
	if got := m.Controller.Snapshot().Value.Discover.Query.Category; got != "tools" {
		t.Fatalf("clicking Tools category pill did not change Discover.Query.Category: got %q, want %q", got, "tools")
	}
}

func TestInstallRegistryRadioSelectsRegistry(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	m := reviewModel(t, newReviewFixture())
	owner := m.Controller.Snapshot().Value.Owner
	a := m.Controller.Actions(owner)
	a.OpenInstall()
	for range 100 {
		if m.Controller.Snapshot().Value.Install.RegistryReadDone {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	a.ToggleRegistryOptions()

	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, m) }) }, 1440, 920)
	settleReviewFrames(tt)

	mirrorLabel := "Mainland China mirror · " + manager.NPMMirrorRegistry
	if err := tt.Click(mirrorLabel); err != nil {
		t.Fatalf("click mirror radio: %v", err)
	}
	tt.Frame()
	if sel := m.Controller.Snapshot().Value.Install.RegistrySelection; sel.Registry == nil || *sel.Registry != manager.NPMMirrorRegistry {
		t.Errorf("clicking mirror radio did not select mirror: got kind=%q registry=%v", sel.Kind, sel.Registry)
	}

	if err := tt.Click("Custom address"); err != nil {
		t.Fatalf("click Custom address radio: %v", err)
	}
	tt.Frame()
	if sel := m.Controller.Snapshot().Value.Install.RegistrySelection; sel.Kind != "custom" {
		t.Errorf("clicking Custom address radio did not select custom: got kind=%q", sel.Kind)
	}
}

func TestCodedErrorWithoutDiagnosticStillRendersSentence(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	m := pageModel(t)
	s := m.Controller.Snapshot().Value

	s.Packages[0].Error = &manager.ManagementError{Code: "plugin/load-failed"}
	s.View = manager.Subject{Kind: "package", Name: s.Packages[0].Name}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { detailView(c, testPage(m, s)) })
	}, 1200, 900)
	if !tt.HasText("The plugin has a problem") {
		t.Errorf("detailView rendered no error alert when pkg.Error.Diagnostic is empty; texts=%v", tt.Texts())
	}

	s2 := m.Controller.Snapshot().Value
	s2.Install = manager.InstallState{
		Open: true, Phase: "failed", Spec: "fixture-plugin",
		Result: &manager.Change{Application: "failed", Error: &manager.ManagementError{Code: "native-install/failed"}},
	}
	tt2 := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { installDialog(c, testPage(m, s2)) })
	}, 1440, 920)
	settleReviewFrames(tt2)
	if !tt2.HasText("Installation failed") {
		t.Errorf("installDialog rendered no error alert when result.Error.Diagnostic is empty; texts=%v", tt2.Texts())
	}
}

func TestNoticeClassifiesAppliedWithErrorOutcome(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	rawNotice := "applied · EACCES · open /secret/path: permission denied"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { notice(c, rawNotice) })
	}, 800, 200)
	for _, tx := range tt.Texts() {
		if strings.Contains(tx, "EACCES") || strings.Contains(tx, "/secret/path") || strings.Contains(tx, "permission denied") {
			t.Fatalf("notice() painted raw error text into UI: %q", tx)
		}
	}
}

func TestPageMemoInvalidatesOnDesignTokensAndController(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	m1 := reviewModel(t, newReviewFixture())
	owner1 := m1.Controller.Snapshot().Value.Owner
	if err := m1.Controller.LoadIndexes(context.Background(), owner1); err != nil {
		t.Fatal(err)
	}
	m1.Controller.Actions(owner1).SelectTab("discover")

	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, m1) }) }, 1200, 900)
	tt.Frame()
	tt.Frame()
	box, ok := tt.Find("Discover fixture")
	if !ok {
		t.Fatalf("Discover fixture not found")
	}
	sampleX, sampleY := int(box.X), int(box.Y+box.H+2)
	before := tt.Image().At(sampleX, sampleY)

	magenta := ui.RGB(255, 0, 255)
	design.SetGlass(true, func(_ design.Scheme, p *design.Palette) {
		p.Card = magenta
	})
	t.Cleanup(func() { design.SetGlass(false, nil) })
	tt.Frame()
	after := tt.Image().At(sampleX, sampleY)

	ttFresh := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, m1) }) }, 1200, 900)
	ttFresh.Frame()
	fresh := ttFresh.Image().At(sampleX, sampleY)
	design.SetGlass(false, nil)

	if after != fresh {
		t.Errorf("memoized discover-listing ignored design.SetGlass Card token change: before=%v after=%v fresh=%v",
			color.RGBAModel.Convert(before), color.RGBAModel.Convert(after), color.RGBAModel.Convert(fresh))
	}

	f2 := newReviewFixture()
	f2.bundles = []manager.Bundle{{Name: "pkg-beta", Version: "2.0.0", Availability: "profile", Installed: true, Enabled: true, Meta: &manager.Meta{Title: manager.Text(`"Beta Plugin"`)}}}
	mA := reviewModel(t, newReviewFixture())
	mB := reviewModel(t, f2)
	mA.Controller.Actions(mA.Controller.Snapshot().Value.Owner).OpenPackage("fixture-plugin")
	mB.Controller.Actions(mB.Controller.Snapshot().Value.Owner).OpenPackage("pkg-beta")
	if mA.Controller.Snapshot().Rev != mB.Controller.Snapshot().Rev {
		t.Fatalf("setup: revs differ %d vs %d", mA.Controller.Snapshot().Rev, mB.Controller.Snapshot().Rev)
	}
	cur := mA
	tt2 := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, cur) }) }, 1200, 900)
	tt2.Frame()
	tt2.Frame()
	cur = mB
	tt2.Frame()
	if !tt2.HasText("Beta Plugin") || tt2.HasText("Fixture plugin") {
		t.Errorf("memoized plugin-detail-head stayed stale after Controller swap at same Rev %d: texts=%v", mB.Controller.Snapshot().Rev, tt2.Texts())
	}
}

func TestInstallDialogCloseHidesWhileApplying(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	m := reviewModel(t, newReviewFixture())
	owner := m.Controller.Snapshot().Value.Owner
	m.Controller.Actions(owner).OpenInstall()
	s := m.Controller.Snapshot().Value
	s.Install.Phase = "applying"
	s.Install.RequestID = "req-1"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			installDialog(c, testPage(m, s))
		})
	}, 1440, 920)
	settleReviewFrames(tt)
	if err := tt.Click("Close"); err != nil {
		t.Fatalf("expected Close button to be absent or disabled during applying, got click err: %v", err)
	}
	tt.Frame()
	tt.Frame()
	if m.Controller.Snapshot().Value.Install.Open {
		t.Fatalf("during applying phase, installDialog renders an enabled Close button that silently does nothing when clicked (Install.Open stayed true)")
	}
}
