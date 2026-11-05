package settings

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

type SkillHubAccess struct {
	Open, Searching, Installing bool
	Confirming                  *SkillHubHit
	Error                       string
	OnOpen, OnClose             func()
	OnConfirmInstall            func()
	OnCancelInstall             func()
}

func skillHubDialog(c *ui.Context, a *SkillsAccess, fg, muted ui.Color) {
	hub := a.Hub
	if hub == nil || !hub.Open || a.HubQuery == nil {
		return
	}
	open := true
	busy := hub.Searching || hub.Installing
	title := i18n.T(i18n.KeySettingsSkillsHubTitle)
	if hub.Confirming != nil {
		title = providerCopy("确认安装技能", "Confirm skill installation")
	}
	bodyDialogFooterWidth(c, &open, title, 760, func() {
		if hub.Error != "" {
			ui.Text(c, hub.Error).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		}
		if hit := hub.Confirming; hit != nil {
			ui.Text(c, providerCopy("请核对技能的公开来源和安装范围。", "Review the public source and installation scope before continuing.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			for _, detail := range []struct{ label, value string }{
				{providerCopy("技能", "Skill"), hit.SkillName},
				{providerCopy("仓库", "Repository"), hit.Repo},
				{providerCopy("路径", "Path"), hit.Path},
				{providerCopy("安装范围", "Install scope"), skillHubScope(a)},
			} {
				ui.Text(c, detail.label+": "+detail.value).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
			}
			ui.Text(c, providerCopy("安装不会静默覆盖已有技能；若名称冲突，操作会失败并保留现有文件。", "Installation never silently overwrites an existing skill. A conflict fails and preserves the current files.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			return
		}
		ui.Text(c, i18n.T(i18n.KeySettingsSkillsHubHint)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		ui.Text(c, providerCopy("安装范围", "Install scope")).FontSize(fonttheme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
		ui.Row(c).Gap(8).Children(func() {
			user, project := providerCopy("用户技能", "User skills"), providerCopy("项目技能", "Project skills")
			if a.TargetRoot == "project-dsh" {
				project = "✓ " + project
			} else {
				user = "✓ " + user
			}
			if ui.Button(c, user).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(busy || a.OnSelectRoot == nil).Clicked() {
				a.OnSelectRoot("user-dsh")
			}
			if ui.Button(c, project).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(busy || a.OnSelectRoot == nil || a.ProjectCwd == "").Clicked() {
				a.OnSelectRoot("project-dsh")
			}
		})
		ui.Text(c, skillHubScope(a)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.TextInput(c, a.HubQuery).Grow(1).Height(32).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(13)).Disabled(busy).Placeholder(i18n.T(i18n.KeySettingsSkillsHubPlaceholder))
			label := i18n.T(i18n.KeySettingsSkillsHubSearch)
			if hub.Searching {
				label = providerCopy("正在搜索…", "Searching…")
			}
			if ui.Button(c, label).Height(32).Padding(0, 14).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(13)).Disabled(busy || strings.TrimSpace(*a.HubQuery) == "" || a.OnSearchHub == nil).Clicked() {
				a.OnSearchHub(*a.HubQuery)
			}
		})
		if len(a.HubHits) == 0 && !hub.Searching {
			ui.Text(c, providerCopy("输入关键词后搜索公开技能。", "Enter a keyword to search public skills.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		}
		for _, hit := range a.HubHits {
			hit := hit
			bodyRosterRow(c, hit.SkillName, func() {
				ui.Text(c, fmt.Sprintf("%s · %s · ★ %g", hit.Repo, hit.Path, hit.Stars)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
				if hit.Description != "" {
					ui.Text(c, hit.Description).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
				}
				label := i18n.T(i18n.KeySettingsSkillsHubInstall)
				disabled := busy || a.OnInstallHub == nil || hit.Repo == "" || hit.Path == "" || hit.SkillName == ""
				if hit.Status == "installed" {
					label, disabled = providerCopy("已安装", "Installed"), true
				} else if hit.Status == "conflict" {
					label, disabled = providerCopy("同名冲突", "Name conflict"), true
				}
				if ui.Button(c, label).AlignSelf(ui.Start).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(disabled).Clicked() {
					a.OnInstallHub(hit.Repo, hit.Path, hit.SkillName)
				}
			}, fg, muted)
		}
	}, func() {
		if hub.Confirming == nil {
			if ui.Button(c, providerCopy("关闭", "Close")).Disabled(busy || hub.OnClose == nil).Clicked() && hub.OnClose != nil {
				hub.OnClose()
			}
			return
		}
		if ui.Button(c, providerCopy("取消", "Cancel")).Disabled(busy || hub.OnCancelInstall == nil).Clicked() && hub.OnCancelInstall != nil {
			hub.OnCancelInstall()
		}
		label := providerCopy("确认安装", "Confirm installation")
		if hub.Installing {
			label = providerCopy("正在安装…", "Installing…")
		}
		if ui.PrimaryButton(c, label).Disabled(busy || hub.OnConfirmInstall == nil).Clicked() && hub.OnConfirmInstall != nil {
			hub.OnConfirmInstall()
		}
	}, ui.DialogOptions{DismissDisabled: busy})
	if !open && !busy && hub.OnClose != nil {
		hub.OnClose()
	}
}

func skillHubScope(a *SkillsAccess) string {
	if a.TargetRoot == "project-dsh" {
		return providerCopy("项目技能", "Project skills") + ": " + a.ProjectCwd
	}
	return providerCopy("用户技能（所有项目可用）", "User skills (available in every project)")
}
