package settings

import (
	"github.com/egoist/mygo/ui"
	"strings"
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

	if s.Err != "" {
		ui.Text(c, s.Err).TextColor(fg)
	}
	if s.LastAction != "" {
		ui.Text(c, settingsActionText(s.LastAction)).TextColor(muted)
	}
	if s.Busy {
		ui.Text(c, providerCopy("正在处理，请稍候…", "Working, please wait…")).TextColor(muted)
	}
	if !s.Ready {
		ui.Text(c, providerCopy("档案或凭证状态尚未验证，请刷新；不可据此判断模型可用。", "Profile or credential state is unverified. Refresh before relying on availability.")).TextColor(muted)
	}
	for _, r := range s.Rows {
		r := r
		if !r.Removable {
			// Built-in and base providers already appear as the provider rows above.
			continue
		}
		// 0.3.13 lists each configured provider as a bordered 16 DIP card whose
		// head is the provider name followed by its id.
		modelRowCard(c, "provider-profile/"+r.ID, func() {
			ui.Row(c).FillWidth().MinWidth(0).Gap(10).AlignItems(ui.Center).Wrap().Children(func() {
				ui.Text(c, r.DisplayName).FontSize(fonttheme.InterfaceSize(14)).FixedLineHeight(fonttheme.InterfaceSize(22)).FontWeight(600).TextColor(fg)
				ui.Text(c, r.Provider).FontSize(fonttheme.InterfaceSize(13)).FixedLineHeight(fonttheme.InterfaceSize(20)).TextColor(muted)
			})
			ui.Text(c, r.Namespace+" / "+strings.Join(r.Path, " / ")).FillWidth().FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			if r.Removable {
				if a.OnDelete != nil && ui.Button(c, providerCopy("删除档案", "Remove profile")).AlignSelf(ui.Start).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(s.Busy || !s.Ready).Clicked() {
					a.OnDelete(r.ID)
				}
			} else {
				ui.Text(c, providerCopy("内置或基础配置供应商", "Built-in or base provider")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			}
		})
	}
	if s.Pending != nil {
		r := s.Pending
		ui.Text(c, providerCopy("确认删除档案：", "Confirm profile removal: ")+r.DisplayName+" ("+r.Provider+")").FontWeight(600).TextColor(fg)
		ui.Text(c, r.Namespace+" / "+strings.Join(r.Path, " / ")).TextColor(muted)
		if r.CredentialRef != "" {
			ui.Text(c, providerCopy("此操作还会移除关联凭证：", "This will also remove the associated credential: ")+r.CredentialRef+providerCopy("。依赖它的模型可能无法使用。", ". Models depending on it may stop working.")).TextColor(fg)
		} else {
			ui.Text(c, providerCopy("没有可删除的页面管理凭证；仅移除此档案。", "No removable page-managed credential; only this profile will be removed.")).TextColor(muted)
		}
		ui.Row(c).Gap(8).Wrap().Children(func() {
			if a.OnConfirm != nil && ui.Button(c, providerCopy("确认删除", "Confirm removal")).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(s.Busy).Clicked() {
				a.OnConfirm()
			}
			if a.OnCancel != nil && ui.Button(c, providerCopy("取消", "Cancel")).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(s.Busy).Clicked() {
				a.OnCancel()
			}
		})
	}
}
