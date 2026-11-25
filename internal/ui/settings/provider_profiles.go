package settings

import (
	"whaleisle/internal/desktop/providerprofiles"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo/ui"
)

type ProviderProfilesAccess struct {
	Snapshot                       providerprofiles.Snapshot
	OnRefresh, OnConfirm, OnCancel func()
	OnDelete                       func(string)
}

func providerCopy(zh, en string) string {
	return i18n.Pick(zh, en)
}
func providerProfilesPane(c *ui.Context, a *ProviderProfilesAccess, fg, muted ui.Color) {
	if a == nil {
		return
	}
	s := a.Snapshot

	settingsAlert(c, "provider_profiles:29", s.Err != "", providerCopy("没能读取提供商配置，请重试。", "Provider settings could not be read. Try again."), s.Err)
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
				if a.OnConfirm != nil && kDanger(c, providerCopy("确认删除", "Confirm removal"), s.Busy).Clicked() {
					a.OnConfirm()
				}
				if a.OnCancel != nil && kButtonDisabled(c, providerCopy("取消", "Cancel"), s.Busy).Clicked() {
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
		if kDanger(c, providerCopy("移除", "Remove"), locked || a.Snapshot.Busy || !a.Snapshot.Ready || a.OnDelete == nil).Label(providerCopy("移除 ", "Remove ")+row.DisplayName).Shrink(0).Clicked() && a.OnDelete != nil {
			a.OnDelete(row.ID)
		}
		return
	}
}
