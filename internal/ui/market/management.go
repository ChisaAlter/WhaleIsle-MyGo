package market

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	marketstore "whaleisle/internal/store/market"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

func failedOutcome(result string) bool {
	return result == "failed" || result == "interrupted" || result == "unknown" || result == "not-started"
}
func failedPlugin(m Model, id, name string) bool {
	for _, row := range m.Store.Snapshot().Rows {
		if row.Label == name && row.Error != "" {
			return true
		}
	}
	for _, entry := range m.Store.Personal().History {
		if entry.Plugin == id {
			return failedOutcome(entry.Result)
		}
	}
	return false
}
func showInstalledRow(m Model, row marketstore.Entry) bool {
	filter := m.Store.ManagementFilter()
	if filter == "" {
		return true
	}
	if filter == "failed" {
		for _, item := range m.Store.Discovery().Catalog.Items {
			if marketstore.InstalledName(item, []marketstore.Entry{row}) != "" && failedPlugin(m, item.ID, row.Label) {
				return true
			}
		}
	}
	for _, update := range m.Store.Updates().Rows {
		if update.Name == row.Label {
			if filter == "updates" {
				return update.Available
			}
			if filter == "failed" && failedPlugin(m, update.ID, row.Label) {
				return true
			}
		}
	}
	if filter == "failed" {
		return row.Error != ""
	}
	return false
}
func managementFilters(c *ui.Context, m Model) {
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	ui.Row(c).Gap(8).Children(func() {
		for _, f := range []struct{ id, zh, en string }{{"", i18n.T(i18n.KeyW4MarketAll), "All"}, {"updates", i18n.T(i18n.KeyW4MarketUpdatableUpdateHistory), "Updates"}, {"failed", i18n.T(i18n.KeyW4MarketFailedInterrupted), "Failed / interrupted"}} {
			selected := m.Store.ManagementFilter() == f.id
			label := marketText(f.zh, f.en)
			if selected {
				label = "● " + label
			}
			b := ui.Button(c, label).Height(28).Padding(0, 10).Radius(999).Border(.5, chrome.Color(colors.Border)).FontSize(fonttheme.InterfaceSize(12.5)).Selected(selected)
			if selected {
				b = b.Background(chrome.Color(colors.Field)).TextColor(chrome.Color(colors.FG)).FontWeight(500)
			}
			if b.Clicked() {
				m.Store.SetManagementFilter(f.id)
			}
		}
	})
}
func confirmationView(c *ui.Context, m Model, fg, accent ui.Color) {
	if description := m.Store.Confirmation(); description != "" {
		open := true
		w, h := c.Size()
		ui.DialogBase(c.Key("market-confirm-dialog"), &open, func(backdrop, panel ui.Element) {
			chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), max(0, min(float32(560), w-32)), 20).
				Role(ui.RoleAlertDialog).Label(marketText(i18n.T(i18n.KeyW4MarketConfirmAction), "Confirm operation")).MaxHeight(max(0, h-48)).Gap(16).Children(func() {
				ui.Text(c, marketText(i18n.T(i18n.KeyW4MarketConfirmAction), "Confirm operation")).Bold().TextColor(fg)
				ui.Scroll(c).FillWidth().MaxHeight(max(0, h-200)).MinHeight(0).Children(func() {
					ui.Text(c, description).FillWidth().MinWidth(0).Selectable().TextColor(fg)
				})
				ui.Row(c).FillWidth().Wrap().Gap(8).Justify(ui.End).Children(func() {
					colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
					if widgets.Button(c, colors, marketText(i18n.T(i18n.KeyW4MarketCancel), "Cancel"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonMedium}).Clicked() {
						m.Store.ResolveConfirmation(false)
					}
					if widgets.Button(c, colors, marketText(i18n.T(i18n.KeyW4MarketConfirmAndRun), "Confirm and run"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonMedium, Disabled: m.InstallBusy}).Clicked() {
						m.Store.ResolveConfirmation(true)
					}
				})
			})
		})
		if !open {
			m.Store.ResolveConfirmation(false)
		}
	}
}
func withMarketConfirmations(m Model) Model {
	if action := m.OnInstallCatalog; action != nil {
		m.OnInstallCatalog = func(t marketstore.InstallTarget) {
			m.Store.RequestConfirmation(marketText(i18n.T(i18n.KeyW4MarketInstallUpdate), "Install / update: ")+t.ID+"\n"+t.CurrentVersion+" → "+t.UpdateVersion+"\n"+t.Spec, func() { action(t) })
		}
	}
	if action := m.OnInstallSpec; action != nil {
		m.OnInstallSpec = func(spec string, builds []string) {
			approved := append([]string(nil), builds...)
			text := marketText(i18n.T(i18n.KeyW4MarketInstallTarget), "Install target: ") + spec
			if len(approved) > 0 {
				text += "\n" + marketText(i18n.T(i18n.KeyW4MarketAllowTheseBuildScripts), "Allow these build scripts to execute: ") + strings.Join(approved, ", ")
			}
			m.Store.RequestConfirmation(text, func() { action(spec, approved) })
		}
	}
	if action := m.OnRemoveBundle; action != nil {
		m.OnRemoveBundle = func(name string) {
			m.Store.RequestConfirmation(marketText(i18n.T(i18n.KeyW4MarketRemovePlugin), "Remove plugin: ")+name, func() { action(name) })
		}
	}
	if action := m.OnSetBundleEnabled; action != nil {
		m.OnSetBundleEnabled = func(name string, enabled bool) {
			m.Store.RequestConfirmation(fmt.Sprintf("%s · %s: %t", name, marketText(i18n.T(i18n.KeyW4MarketEnable), "Enabled"), enabled), func() { action(name, enabled) })
		}
	}
	if action := m.OnAcceptExemption; action != nil {
		m.OnAcceptExemption = func(pkg, runtime string) {
			m.Store.RequestConfirmation(marketText(i18n.T(i18n.KeyW4MarketAcceptTheCompatibilityRiskAndWaiveThe), "Accept compatibility risk and exempt version check: ")+pkg+" · runtime "+runtime, func() { action(pkg, runtime) })
		}
	}
	return m
}
func batchSelectionView(c *ui.Context, m Model, u marketstore.Updates, muted, border ui.Color) {
	var targets []marketstore.InstallTarget
	for _, row := range u.Rows {
		if !row.Available || !m.Store.UpdateSelected(row.ID) {
			continue
		}
		if target, err := m.Store.CaptureUpdateTarget(row.ID); err == nil {
			targets = append(targets, target)
		}
	}
	ui.Row(c).Gap(8).Children(func() {
		if !m.InstallBusy && !u.Loading {
			if ui.Button(c, marketText(i18n.T(i18n.KeyW4MarketSelectAllUpdatable), "Select all updates")).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Clicked() {
				m.Store.SelectAvailableUpdates(true)
			}
			if ui.Button(c, marketText(i18n.T(i18n.KeyW4MarketClearSelection2), "Clear selection")).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Clicked() {
				m.Store.SelectAvailableUpdates(false)
			}
			if len(targets) > 0 && m.OnBatchUpdate != nil && ui.Button(c, fmt.Sprintf(marketText(i18n.T(i18n.KeyW4MarketUpdateAllD), "Update selected (%d)"), len(targets))).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Clicked() {
				confirmed := append([]marketstore.InstallTarget(nil), targets...)
				lines := []string{marketText(i18n.T(i18n.KeyW4MarketTheFollowingPluginsUpdateInOrderBuild), "Update these plugins sequentially. Build scripts and version exemptions require separate approval. Stop on failure.")}
				for _, t := range confirmed {
					lines = append(lines, t.ID+" · "+t.CurrentVersion+" → "+t.UpdateVersion)
				}
				m.Store.RequestConfirmation(strings.Join(lines, "\n"), func() { m.OnBatchUpdate(confirmed) })
			}
		}
		if m.OnStopBatch != nil && ui.Button(c, marketText(i18n.T(i18n.KeyW4MarketStopRemainingBatchItems), "Stop queued updates")).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Clicked() {
			m.OnStopBatch()
		}
	})
	ui.Text(c, marketText(i18n.T(i18n.KeyW4MarketInterruptedRecordsAreForReviewOnlyThir), "Interrupted receipts are for reconciliation; third-party installs are never automatically replayed.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
}
