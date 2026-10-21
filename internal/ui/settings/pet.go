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

type PetSettingsAccess struct {
	AssistantEnabled   bool
	AssistantAvailable bool
	// DesktopUnavailable is true in portable/QA runs: the pet preference is
	// stored, but the companion window is intentionally not created.
	DesktopUnavailable bool
	// DesktopError is the Companion's last window/asset error, shown instead of
	// silently doing nothing when the toggle is on.
	DesktopError string
	Busy         bool
	Err                string
	OnSetAssistant     func(bool)
	RenderAssistant    func(*ui.Context)
	Store              *settingsstore.Store
	OnSave             func()

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
		bodyHeading(c, providerCopy("鲸鱼娘", "Whale"), providerCopy("你的常驻助理与桌面伙伴。", "Your resident assistant and desktop companion."), fg, muted)
		if _, picked := settingsSegmented(c, "pet-tabs", []string{providerCopy("聊天与能力", "Chat and abilities"), providerCopy("桌面形象与行为", "Appearance and behavior")}, *tab, false); picked >= 0 {
			*tab = picked
		}
		if p.Error != "" {
			ui.Text(c, p.Error).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(fg)
		}
		if a != nil && a.Err != "" {
			ui.Text(c, a.Err).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(fg)
		}
		if *tab == 0 {
			if a != nil && a.AssistantAvailable {
				bodyHeading(c, providerCopy("助理", "Assistant"), "", fg, muted)
				enabled := a.AssistantEnabled
				bodyLocalToggle(c, "pet-assistant", providerCopy("鲸鱼娘助理", "Whale assistant"), providerCopy("常驻对话与侧栏入口；切换后重启 Harness。", "Resident conversation and sidebar entry. Changing this restarts Harness."), &enabled, a.Busy || a.OnSetAssistant == nil, fg, muted, a.OnSetAssistant)
				// The dsh-whale catalog drives the rest of the page; the legacy
				// assistant form is no longer rendered here.
				petWhaleSection(c, colors, a, fg, muted)
			}
			return
		}
		disabled := p.Store == nil && (a == nil || a.OnSave == nil)
		if a != nil && a.DesktopUnavailable {
			ui.Text(c, providerCopy("当前为便携/测试模式，桌面形象不会显示；安装版可用。", "Portable/QA mode does not show the desktop character; the installed build does.")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
		if a != nil && a.DesktopError != "" {
			ui.Text(c, a.DesktopError).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(ui.RGB(220, 38, 38))
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
		bodyLocalToggle(c, "pet-enabled", providerCopy("在桌面显示鲸鱼娘", "Show Whale on the desktop"), providerCopy("关闭后隐藏桌面形象，随时可以再开。", "Hide the desktop character; you can show her again at any time."), &p.Enabled, disabled, fg, muted, func(bool) { commit("enabled") })
		bodyPreferenceRow(c, "pet-scale-opacity", providerCopy("体型与不透明度", "Scale and opacity"), "", fg, muted, func() {
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
				if ui.Select(c, &selected, choices).FillWidth().Label(label).Disabled(disabled).Changed() {
					number, _ := strconv.ParseFloat(strings.TrimSuffix(selected, "%"), 64)
					*value = number / 100
					commit(field)
				}
			}
			percentage(providerCopy("体型", "Scale"), "scale", &p.Behavior.Scale, []string{"60%", "80%", "100%", "120%", "140%", "160%"})
			percentage(providerCopy("不透明度", "Opacity"), "opacity", &p.Behavior.Opacity, []string{"30%", "50%", "70%", "85%", "100%"})
		})
		bodyPreferenceRow(c, "pet-personality", providerCopy("性格", "Personality"), providerCopy("影响台词与「聊聊」语气，助理人格跟随同一设置。", "Affects dialogue and quick-chat tone; the assistant uses the same personality."), fg, muted, func() {
			if ui.Select(c, &p.Behavior.Personality, []string{"natural", "genki", "tsundere", "poison"}).FillWidth().Label(providerCopy("性格", "Personality")).Disabled(disabled).Changed() {
				commit("personality")
			}
		})
		bodyHeading(c, providerCopy("行为", "Behavior"), "", fg, muted)
		bodyPreferenceRow(c, "pet-activity", providerCopy("活跃频率", "Activity"), providerCopy("驱动自言自语与乱逛的间隔。", "Controls the interval between self-talk and wandering."), fg, muted, func() {
			if ui.Select(c, &p.Behavior.Activity, []string{"quiet", "balanced", "active"}).FillWidth().Label(providerCopy("活跃频率", "Activity")).Disabled(disabled).Changed() {
				commit("activity")
			}
		})
		bodyPreferenceRow(c, "pet-autonomy", providerCopy("自主行为", "Autonomous behavior"), providerCopy("她自己开口说话、到处乱逛。", "Let her talk to herself and wander around."), fg, muted, func() {
			ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
				ui.Text(c, providerCopy("自言自语", "Self-talk")).Grow(1).MinWidth(0)
				if ui.Switch(c, &p.Behavior.SelfTalk).Label(providerCopy("自言自语", "Self-talk")).Disabled(disabled).Changed() {
					commit("selfTalk")
				}
			})
			ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
				ui.Text(c, providerCopy("自主乱逛", "Wandering")).Grow(1).MinWidth(0)
				if ui.Switch(c, &p.Behavior.Wander).Label(providerCopy("自主乱逛", "Wandering")).Disabled(disabled).Changed() {
					commit("wander")
				}
			})
		})
		bodyPreferenceRow(c, "pet-drag-mode", providerCopy("拖拽方式", "Dragging"), providerCopy("自由拖动、按住 Shift，或锁定位置。", "Drag freely, hold Shift to drag, or lock the position."), fg, muted, func() {
			selected := "free"
			if p.Behavior.LockPosition {
				selected = "locked"
			} else if p.Behavior.ShiftToDrag {
				selected = "shift"
			}
			if ui.Select(c, &selected, []string{"free", "shift", "locked"}).FillWidth().Label(providerCopy("拖拽方式", "Dragging")).Disabled(disabled).Changed() {
				p.Behavior.LockPosition = selected == "locked"
				p.Behavior.ShiftToDrag = selected == "shift"
				commit("drag")
			}
		})
		bodyLocalToggle(c, "pet-power", providerCopy("省电模式", "Power saving"), providerCopy("无交互约五分钟后降低帧率。", "Reduce the frame rate after about five idle minutes."), &p.Behavior.PowerSave, disabled, fg, muted, func(bool) { commit("powerSave") })
		bodyLocalToggle(c, "pet-sound", providerCopy("点击音效", "Click sound"), "", &p.Behavior.ClickSound, disabled, fg, muted, func(bool) { commit("clickSound") })
		bodyHeading(c, providerCopy("联动", "Integration"), "", fg, muted)
		bodyLocalToggle(c, "pet-chat", providerCopy("快捷对话", "Quick chat"), providerCopy("右键面板中的「聊聊」对话卡。", "The quick-chat card in the context menu."), &p.Behavior.ChatEnabled, disabled, fg, muted, func(bool) { commit("chatEnabled") })
		if a != nil {
			petLookPane(c, p, a.Store, fg, muted, disabled)
		}
		reset := ui.Local(anchor, "confirm-reset", func() bool { return false })
		ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
			if widgets.Button(c, colors, providerCopy("恢复外观与行为默认", "Reset appearance and behavior"), widgets.ButtonSpec{Variant: widgets.ButtonDanger, Size: widgets.ButtonSmall, Disabled: p.Store == nil || disabled}).Clicked() {
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
