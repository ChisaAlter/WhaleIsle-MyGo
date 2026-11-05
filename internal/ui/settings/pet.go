package settings

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strconv"
	"strings"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/pet"
	"whaleisle/internal/desktop/whale"
	settingsstore "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

type petChoice struct {
	value, zh, en string
}

// petChoiceSelect shows localized option labels while storing the Host's raw
// enum value, so the page never displays "natural" or "balanced" to a user.
func petChoiceSelect(c *ui.Context, current *string, label string, options []petChoice, disabled bool) bool {
	labels := make([]string, 0, len(options)+1)
	selected := ""
	for _, option := range options {
		text := providerCopy(option.zh, option.en)
		labels = append(labels, text)
		if option.value == *current {
			selected = text
		}
	}
	if selected == "" {
		selected = *current
		labels = append(labels, selected)
	}
	if !ui.Select(c, &selected, labels).Label(label).FillWidth().Disabled(disabled).Changed() {
		return false
	}
	for _, option := range options {
		if providerCopy(option.zh, option.en) == selected {
			*current = option.value
			return true
		}
	}
	*current = selected
	return true
}

type PetSettingsAccess struct {
	AssistantEnabled    bool
	AssistantAvailable  bool
	AssistantPreference *ProjectPreferenceAccess
	// DesktopUnavailable is true in portable/QA runs: the pet preference is
	// stored, but the companion window is intentionally not created.
	DesktopUnavailable bool
	// DesktopError is the Companion's last window/asset error, shown instead of
	// silently doing nothing when the toggle is on.
	DesktopError    string
	Busy            bool
	Err             string
	OnSetAssistant  func(bool)
	RenderAssistant func(*ui.Context)
	Store           *settingsstore.Store
	OnSave          func()

	// Whale is the dsh-whale catalog snapshot the Whale page edits. It and
	// every callback stay nil until the Host answers, so the page keeps its
	// current behaviour while the plugin is unavailable.
	Whale           *whale.Catalog
	OnLoadWhale     func()
	OnUpdateWhale   func(whale.Patch)
	OnReplaceMemory func(string)
	OnClearMemory   func()
	OnToggleSkill   func(string, bool)
}

func petPane(c *ui.Context, colors theme.Colors, p *pet.Companion, fg, muted ui.Color, accesses ...*PetSettingsAccess) {
	if p == nil {
		return
	}
	var a *PetSettingsAccess
	if len(accesses) > 0 {
		a = accesses[0]
	}
	anchor := ui.Column(c.Key("pet-settings-body")).FillWidth().MinWidth(0).Gap(8)
	tab := ui.Local(anchor, "tab", func() int {
		if a != nil && a.AssistantAvailable {
			return 0
		}
		return 1
	})
	anchor.Children(func() {
		petHeader(c, fg, muted)
		if picked := petTabs(c, "pet-tabs", []string{providerCopy("聊天与能力", "Chat & capabilities"), providerCopy("桌面形象与行为", "Desktop appearance & behavior")}, *tab, fg); picked >= 0 {
			*tab = picked
		}
		settingsItemVisibility("settings.pet.item", *tab == 0)
		if p.Error != "" && *tab == 1 {
			ui.Text(c, providerCopy("桌面形象加载失败，请重启应用重试。", "The desktop character failed to load. Restart the app to try again.")).
				FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(c.Theme().Warning).Tooltip(p.Error)
		}
		if a != nil && a.Err != "" {
			ui.Text(c, a.Err).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(fg)
		}
		if *tab == 0 {
			if !settingsItemBefore(c, "pet-assistant") {
				return
			}
			if a != nil && a.AssistantAvailable {
				petGroupLabel(c, providerCopy("助理", "Assistant"))
				enabled := a.AssistantEnabled
				busy, onSet := a.Busy, a.OnSetAssistant
				if access := a.AssistantPreference; access != nil {
					enabled, busy, onSet = access.State.Enabled, busy || access.State.Busy, access.OnSet
				}
				featureToggleRow(c, featureRow{
					key:      "pet-assistant",
					title:    providerCopy("鲸鱼娘助理", "Whale assistant"),
					desc:     providerCopy("开启她的常驻对话与侧栏入口；切换后会重启 Harness。", "Mounts her persistent conversation and sidebar entry; toggling restarts Harness."),
					value:    &enabled,
					disabled: busy || onSet == nil,
					changed:  onSet,
					compact:  true,
				}, fg, muted)
				if access := a.AssistantPreference; access != nil {
					settingsNotice(c, "pet:125", "error", access.State.Error != "", func() string { return access.State.Error })
					if access.State.RestartCancelled {
						ui.Text(c, providerCopy("已取消切换，保留原设置。", "Change cancelled; the previous setting is retained.")).FillWidth().TextColor(muted)
					}
				}
				if a.RenderAssistant != nil {
					a.RenderAssistant(c)
				} else {
					petWhaleSection(c, colors, a, fg, muted)
				}
			}
			return
		}
		disabled := p.Store == nil && (a == nil || a.OnSave == nil)
		if a != nil && a.DesktopUnavailable {
			ui.Text(c, providerCopy("当前为便携/测试模式，桌面形象不会显示；安装版可用。", "Portable/QA mode does not show the desktop character; the installed build does.")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
		if a != nil && a.DesktopError != "" {
			ui.Text(c, providerCopy("桌宠窗口未能打开，请检查 WebView2 运行时或重启应用。", "The desktop character window could not open. Check the WebView2 runtime or restart the app.")).
				FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(c.Theme().Warning).Tooltip(a.DesktopError)
		}
		if a != nil && a.Busy {
			disabled = true
			ui.Text(c, providerCopy("正在保存…", "Saving…")).TextColor(muted)
		}
		save := func() {
			p.Behavior = config.NormalizePetBehavior(p.Behavior)
			if a != nil && a.OnSave != nil {
				a.OnSave()
			} else if p.Store != nil {
				p.Save()
			}
		}
		commit := func(field string) { savePetSetting(p, a, field) }
		bodyHeading(c, providerCopy("外观", "Appearance"), "", fg, muted)
		bodyLocalToggle(c, "pet-enabled", providerCopy("在桌面显示鲸鱼娘", "Show Whale on desktop"), providerCopy("关闭后隐藏桌面上的她，随时可以再开。", "Hide the on-screen companion; you can bring her back anytime."), &p.Enabled, disabled, fg, muted, func(bool) { commit("enabled") })
		bodyPreferenceRow(c, "pet-scale-opacity", providerCopy("体型与不透明度", "Size & opacity"), "", fg, muted, func() {
			percentage := func(label, field string, value *float64, choices []string) {
				selected := fmt.Sprintf("%.0f%%", *value*100)
				found := false
				for _, choice := range choices {
					if choice == selected {
						found = true
					}
				}
				if !found {
					choices = append(choices, selected)
				}
				ui.Column(c).FillWidth().MinWidth(0).Gap(2).Children(func() {
					ui.Text(c, label).FontSize(theme.InterfaceSize(11)).TextColor(muted)
					if ui.Select(c, &selected, choices).FillWidth().Label(label).Disabled(disabled).Changed() {
						number, _ := strconv.ParseFloat(strings.TrimSuffix(selected, "%"), 64)
						*value = number / 100
						commit(field)
					}
				})
			}
			percentage(providerCopy("体型", "Size"), "scale", &p.Behavior.Scale, []string{"60%", "80%", "100%", "120%", "140%", "160%"})
			percentage(providerCopy("不透明度", "Opacity"), "opacity", &p.Behavior.Opacity, []string{"30%", "50%", "70%", "85%", "100%"})
		})
		bodyPreferenceRow(c, "pet-personality", providerCopy("性格", "Personality"), providerCopy("影响她的台词与「聊聊」语气；助理人格跟随同一设置。", "Shapes her lines and chat tone; the assistant persona follows the same setting."), fg, muted, func() {
			if petChoiceSelect(c, &p.Behavior.Personality, providerCopy("性格", "Personality"), []petChoice{
				{"natural", "软萌", "Soft"}, {"genki", "元气", "Lively"}, {"tsundere", "傲娇", "Tsundere"}, {"poison", "毒舌", "Sharp-tongued"},
			}, disabled) {
				commit("personality")
			}
		})
		bodyHeading(c, providerCopy("行为", "Behavior"), "", fg, muted)
		bodyPreferenceRow(c, "pet-activity", providerCopy("活跃频率", "Activity"), providerCopy("驱动自言自语与乱逛的间隔。", "Drives self-talk and wander intervals."), fg, muted, func() {
			if petChoiceSelect(c, &p.Behavior.Activity, providerCopy("活跃频率", "Activity"), []petChoice{
				{"quiet", "安静", "Quiet"}, {"balanced", "均衡", "Balanced"}, {"active", "活泼", "Lively"},
			}, disabled) {
				commit("activity")
			}
		})
		bodyPreferenceRow(c, "pet-autonomy", providerCopy("自主行为", "Autonomy"), providerCopy("她自己开口说话、到处乱逛。", "She talks to herself and wanders on her own."), fg, muted, func() {
			ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
				ui.Text(c, providerCopy("自言自语", "Self-talk")).Grow(1).MinWidth(0)
				if ui.Switch(c, &p.Behavior.SelfTalk).Label(providerCopy("自言自语", "Self-talk")).Disabled(disabled).Changed() {
					commit("selfTalk")
				}
			})
			ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
				ui.Text(c, providerCopy("自主乱逛", "Wander")).Grow(1).MinWidth(0)
				if ui.Switch(c, &p.Behavior.Wander).Label(providerCopy("自主乱逛", "Wander")).Disabled(disabled).Changed() {
					commit("wander")
				}
			})
		})
		bodyPreferenceRow(c, "pet-drag-mode", providerCopy("拖拽方式", "Dragging"), providerCopy("锁定后不能拖动她；Shift 模式需按住 Shift 拖。", "Locked pins her in place; Shift mode requires holding Shift."), fg, muted, func() {
			selected := "free"
			if p.Behavior.LockPosition {
				selected = "locked"
			} else if p.Behavior.ShiftToDrag {
				selected = "shift"
			}
			if petChoiceSelect(c, &selected, providerCopy("拖拽方式", "Dragging"), []petChoice{
				{"free", "自由拖拽", "Free drag"}, {"shift", "按住 Shift 拖拽", "Hold Shift to drag"}, {"locked", "锁定位置", "Locked"},
			}, disabled) {
				p.Behavior.LockPosition = selected == "locked"
				p.Behavior.ShiftToDrag = selected == "shift"
				commit("drag")
			}
		})
		bodyLocalToggle(c, "pet-power", providerCopy("省电模式", "Power save"), providerCopy("无交互约 5 分钟后降低帧率。", "Lowers the frame rate after ~5 idle minutes."), &p.Behavior.PowerSave, disabled, fg, muted, func(bool) { commit("powerSave") })
		bodyLocalToggle(c, "pet-sound", providerCopy("点击音效", "Click sound"), "", &p.Behavior.ClickSound, disabled, fg, muted, func(bool) { commit("clickSound") })
		bodyHeading(c, providerCopy("联动", "Integrations"), "", fg, muted)
		bodyLocalToggle(c, "pet-chat", providerCopy("快捷对话", "Quick chat"), providerCopy("右键面板的「聊聊」对话卡。", "The 「chat」card on her panel."), &p.Behavior.ChatEnabled, disabled, fg, muted, func(bool) { commit("chatEnabled") })
		if a != nil {
			petLookPane(c, p, a.Store, fg, muted, disabled)
		}
		reset := ui.Local(anchor, "confirm-reset", func() bool { return false })
		ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
			if widgets.Button(c, colors, providerCopy("恢复桌面形象与行为默认", "Reset desktop appearance & behavior"), widgets.ButtonSpec{Variant: widgets.ButtonDanger, Size: widgets.ButtonSmall, Disabled: p.Store == nil || disabled}).Clicked() {
				*reset = true
			}
		})
		ConfirmDialog(c, petResetQuestion(p, reset, colors))
		bodyDisclosure(c, "pet-local-skins", providerCopy("本地皮肤与动画", "Local skins and animation"), func() {
			if ui.Button(c, providerCopy("保存本地皮肤与导入草稿", "Save local skin and imported draft")).Disabled(disabled).Clicked() {
				save()
			}
			bodyLocalToggle(c, "pet-neural", providerCopy("THA4 动画", "THA4 animation"), providerCopy("使用已验证的本地资源；不可用时使用内置动画。", "Uses verified local resources, with bundled animation when unavailable."), &p.Neural, disabled, fg, muted, nil)
			options := []string{""}
			found := p.SkinID == ""
			for _, skin := range p.Skins {
				options = append(options, skin.ID)
				if skin.ID == p.SkinID {
					found = true
				}
			}
			if !found {
				options = append(options, p.SkinID)
				ui.Text(c, providerCopy("已存皮肤缺失，暂用内置形象。", "Saved skin is missing; showing the bundled character.")).FillWidth().TextColor(muted)
			}
			bodyPreferenceRow(c, "pet-skin", providerCopy("皮肤", "Skin"), providerCopy("留空使用内置鲸鱼娘。", "Leave blank to use the bundled Whale."), fg, muted, func() { ui.Select(c, &p.SkinID, options).FillWidth().Disabled(disabled) })
			if ui.Button(c, providerCopy("刷新本地皮肤", "Refresh local skins")).Disabled(p.Store == nil).Clicked() {
				p.Refresh()
			}
			if ui.Button(c, providerCopy("载入原版设置草稿", "Load original settings draft")).Disabled(p.Store == nil).Clicked() {
				p.LoadOriginalDraft()
			}
			for _, line := range p.OriginalImportReport {
				ui.Text(c, line).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
			}
			if p.PositionDraft != nil {
				ui.Text(c, fmt.Sprintf(providerCopy("待保存位置：(%d, %d)", "Pending position: (%d, %d)"), p.PositionDraft.X, p.PositionDraft.Y)).FillWidth().TextColor(muted)
				if ui.Button(c, providerCopy("取消位置导入", "Discard position import")).Clicked() {
					p.PositionDraft = nil
					p.OriginalImportReport = nil
				}
			}
		})
	})
}
