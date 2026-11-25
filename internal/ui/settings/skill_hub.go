package settings

import (
	"fmt"
	"strings"

	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo/ui"
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
			settingsAuthoredFailure(c, hub.Error)
		}
		if hit := hub.Confirming; hit != nil {
			kHint(c, providerCopy("请核对技能的公开来源和安装范围。", "Review the public source and installation scope before continuing."))
			for _, detail := range []struct{ label, value string }{
				{providerCopy("技能", "Skill"), hit.SkillName},
				{providerCopy("仓库", "Repository"), hit.Repo},
				{providerCopy("路径", "Path"), hit.Path},
				{providerCopy("安装范围", "Install scope"), skillHubScope(a)},
			} {
				kText(c, detail.label+": "+detail.value, design.TextXS, design.WeightNormal, fg).FillWidth()
			}
			kHint(c, providerCopy("安装不会静默覆盖已有技能；若名称冲突，操作会失败并保留现有文件。", "Installation never silently overwrites an existing skill. A conflict fails and preserves the current files."))
			return
		}
		kHint(c, i18n.T(i18n.KeySettingsSkillsHubHint))
		kHeading(c, providerCopy("安装范围", "Install scope"))
		ui.Row(c).Gap(design.Sp(2)).Children(func() {
			user, project := providerCopy("用户技能", "User skills"), providerCopy("项目技能", "Project skills")
			isProject := a.TargetRoot == "project-dsh"
			if isProject {
				project = "✓ " + project
			} else {
				user = "✓ " + user
			}
			if kChoiceButton(c, user, !isProject, busy || a.OnSelectRoot == nil).Clicked() {
				a.OnSelectRoot("user-dsh")
			}
			if kChoiceButton(c, project, isProject, busy || a.OnSelectRoot == nil || a.ProjectCwd == "").Clicked() {
				a.OnSelectRoot("project-dsh")
			}
		})
		kHint(c, skillHubScope(a))
		canSearch := !busy && strings.TrimSpace(*a.HubQuery) != "" && a.OnSearchHub != nil
		ui.Row(c).FillWidth().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
			searchInput := kInputRow(c, a.HubQuery, coss.InputOptions{
				Disabled:    busy,
				Placeholder: i18n.T(i18n.KeySettingsSkillsHubPlaceholder),
				Label:       i18n.T(i18n.KeySettingsSkillsHubSearch),
			})
			if searchInput.Submitted() && canSearch {
				a.OnSearchHub(*a.HubQuery)
			}
			label := i18n.T(i18n.KeySettingsSkillsHubSearch)
			if hub.Searching {
				label = providerCopy("正在搜索…", "Searching…")
			}
			if kPrimary(c, label, !canSearch).Clicked() {
				a.OnSearchHub(*a.HubQuery)
			}
		})
		if len(a.HubHits) == 0 && !hub.Searching {
			kHint(c, providerCopy("输入关键词后搜索公开技能。", "Enter a keyword to search public skills."))
		}
		for _, hit := range a.HubHits {
			hit := hit
			bodyRosterRow(c, hit.SkillName, func() {
				kHint(c, fmt.Sprintf("%s · %s · ★ %g", hit.Repo, hit.Path, hit.Stars))
				if hit.Description != "" {
					kHint(c, hit.Description)
				}
				label := i18n.T(i18n.KeySettingsSkillsHubInstall)
				disabled := busy || a.OnInstallHub == nil || hit.Repo == "" || hit.Path == "" || hit.SkillName == ""
				if hit.Status == "installed" {
					label, disabled = providerCopy("已安装", "Installed"), true
				} else if hit.Status == "conflict" {
					label, disabled = providerCopy("同名冲突", "Name conflict"), true
				}
				if kButtonDisabled(c, label, disabled).AlignSelf(ui.Start).Clicked() {
					a.OnInstallHub(hit.Repo, hit.Path, hit.SkillName)
				}
			}, fg, muted)
		}
	}, func() {
		if hub.Confirming == nil {
			if kButtonDisabled(c, providerCopy("关闭", "Close"), busy || hub.OnClose == nil).Clicked() && hub.OnClose != nil {
				hub.OnClose()
			}
			return
		}
		if kButtonDisabled(c, providerCopy("取消", "Cancel"), busy || hub.OnCancelInstall == nil).Clicked() && hub.OnCancelInstall != nil {
			hub.OnCancelInstall()
		}
		label := providerCopy("确认安装", "Confirm installation")
		if hub.Installing {
			label = providerCopy("正在安装…", "Installing…")
		}
		if kPrimary(c, label, busy || hub.OnConfirmInstall == nil).Clicked() && hub.OnConfirmInstall != nil {
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
