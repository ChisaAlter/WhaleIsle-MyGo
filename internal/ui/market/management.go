package market

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	marketstore "whaleisle/internal/store/market"
	fonttheme "whaleisle/internal/ui/theme"
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
	ui.Row(c).Gap(8).Children(func() {
		for _, f := range []struct{ id, zh, en string }{{"", "全部", "All"}, {"updates", "可更新 / 更新历史", "Updates"}, {"failed", "失败 / 中断", "Failed / interrupted"}} {
			label := marketText(f.zh, f.en)
			if m.Store.ManagementFilter() == f.id {
				label = "● " + label
			}
			if ui.Button(c, label).Clicked() {
				m.Store.SetManagementFilter(f.id)
			}
		}
	})
}
func confirmationView(c *ui.Context, m Model, fg, accent ui.Color) {
	if description := m.Store.Confirmation(); description != "" {
		ui.Column(c).Gap(8).Children(func() {
			ui.Text(c, marketText("确认操作", "Confirm operation")).Bold().TextColor(accent)
			ui.Text(c, description).TextColor(fg)
			ui.Row(c).Gap(8).Children(func() {
				if ui.Button(c, marketText("取消", "Cancel")).Clicked() {
					m.Store.ResolveConfirmation(false)
				}
				if !m.InstallBusy && ui.Button(c, marketText("确认并执行", "Confirm and run")).Clicked() {
					m.Store.ResolveConfirmation(true)
				}
			})
		})
	}
}
func withMarketConfirmations(m Model) Model {
	if action := m.OnInstallCatalog; action != nil {
		m.OnInstallCatalog = func(t marketstore.InstallTarget) {
			m.Store.RequestConfirmation(marketText("安装 / 更新：", "Install / update: ")+t.ID+"\n"+t.CurrentVersion+" → "+t.UpdateVersion+"\n"+t.Spec, func() { action(t) })
		}
	}
	if action := m.OnInstallSpec; action != nil {
		m.OnInstallSpec = func(spec string, builds []string) {
			approved := append([]string(nil), builds...)
			text := marketText("安装目标：", "Install target: ") + spec
			if len(approved) > 0 {
				text += "\n" + marketText("允许执行这些构建脚本：", "Allow these build scripts to execute: ") + strings.Join(approved, ", ")
			}
			m.Store.RequestConfirmation(text, func() { action(spec, approved) })
		}
	}
	if action := m.OnRemoveBundle; action != nil {
		m.OnRemoveBundle = func(name string) {
			m.Store.RequestConfirmation(marketText("移除插件：", "Remove plugin: ")+name, func() { action(name) })
		}
	}
	if action := m.OnSetBundleEnabled; action != nil {
		m.OnSetBundleEnabled = func(name string, enabled bool) {
			m.Store.RequestConfirmation(fmt.Sprintf("%s · %s: %t", name, marketText("启用", "Enabled"), enabled), func() { action(name, enabled) })
		}
	}
	if action := m.OnAcceptExemption; action != nil {
		m.OnAcceptExemption = func(pkg, runtime string) {
			m.Store.RequestConfirmation(marketText("接受兼容性风险并豁免版本检查：", "Accept compatibility risk and exempt version check: ")+pkg+" · runtime "+runtime, func() { action(pkg, runtime) })
		}
	}
	return m
}
func batchSelectionView(c *ui.Context, m Model, u marketstore.Updates, muted ui.Color) {
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
			if ui.Button(c, marketText("全选可更新项", "Select all updates")).Clicked() {
				m.Store.SelectAvailableUpdates(true)
			}
			if ui.Button(c, marketText("清空选择", "Clear selection")).Clicked() {
				m.Store.SelectAvailableUpdates(false)
			}
			if len(targets) > 0 && m.OnBatchUpdate != nil && ui.Button(c, fmt.Sprintf(marketText("批量更新 (%d)", "Update selected (%d)"), len(targets))).Clicked() {
				confirmed := append([]marketstore.InstallTarget(nil), targets...)
				lines := []string{marketText("按顺序更新以下插件；构建脚本和版本豁免将另外确认，失败后停止。", "Update these plugins sequentially. Build scripts and version exemptions require separate approval. Stop on failure.")}
				for _, t := range confirmed {
					lines = append(lines, t.ID+" · "+t.CurrentVersion+" → "+t.UpdateVersion)
				}
				m.Store.RequestConfirmation(strings.Join(lines, "\n"), func() { m.OnBatchUpdate(confirmed) })
			}
		}
		if m.OnStopBatch != nil && ui.Button(c, marketText("停止剩余批量项目", "Stop queued updates")).Clicked() {
			m.OnStopBatch()
		}
	})
	ui.Text(c, marketText("中断记录仅供核对；不会自动重试或恢复第三方安装。", "Interrupted receipts are for reconciliation; third-party installs are never automatically replayed.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
}
