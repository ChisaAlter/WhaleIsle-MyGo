package settings

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/providerprofiles"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

type ProviderProfilesAccess struct {
	Snapshot                       providerprofiles.Snapshot
	OnRefresh, OnConfirm, OnCancel func()
	OnDelete                       func(string)
}

func providerCopy(zh, en string) string {
	if i18n.Locale() == i18n.LocaleZH {
		return zh
	}
	return en
}
func providerProfilesPane(c *ui.Context, a *ProviderProfilesAccess, fg, muted ui.Color) {
	if a == nil {
		return
	}
	s := a.Snapshot

	settingsNotice(c, "provider_profiles:29", "error", s.Err != "", func() string { return s.Err })
	if s.Pending != nil {
		r := s.Pending
		open := true
		accountDialog(c, &open, providerCopy("移除供应商", "Remove provider"), 480, s.Busy, func() {
			ui.Text(c, providerCopy("确认删除档案：", "Confirm profile removal: ")+r.DisplayName+" ("+r.Provider+")").FontWeight(600).TextColor(fg)
			if r.CredentialRef != "" {
				ui.Text(c, providerCopy("此操作还会移除关联凭证：", "This will also remove the associated credential: ")+r.CredentialRef+providerCopy("。依赖它的模型可能无法使用。", ". Models depending on it may stop working.")).TextColor(fg)
			} else {
				ui.Text(c, providerCopy("没有可删除的页面管理凭证；仅移除此档案。", "No removable page-managed credential; only this profile will be removed.")).TextColor(muted)
			}
		}, func() {
			ui.Row(c).Gap(8).Wrap().Children(func() {
				if a.OnConfirm != nil && ui.Button(c, providerCopy("确认删除", "Confirm removal")).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(s.Busy).Clicked() {
					a.OnConfirm()
				}
				if a.OnCancel != nil && ui.Button(c, providerCopy("取消", "Cancel")).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(s.Busy).Clicked() {
					a.OnCancel()
				}
			})
		})
		if !open && !s.Busy && a.OnCancel != nil {
			a.OnCancel()
		}
	}
}

func providerRemovalButton(c *ui.Context, a *ProviderProfilesAccess, namespace, provider string, locked bool) {
	if a == nil {
		return
	}
	for _, row := range a.Snapshot.Rows {
		if row.Namespace != namespace || row.Provider != provider || !row.Removable {
			continue
		}
		if modelPill(c, providerCopy("移除", "Remove"), false, true, locked || a.Snapshot.Busy || !a.Snapshot.Ready || a.OnDelete == nil).Label(providerCopy("移除 ", "Remove ")+row.DisplayName).Clicked() && a.OnDelete != nil {
			a.OnDelete(row.ID)
		}
		return
	}
}
