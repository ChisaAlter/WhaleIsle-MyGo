package composer

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

type permissionPickerState struct {
	owner              NativeCommandOwner
	open               bool
	nav                presetNavigation
	pointerX, pointerY float32
	pointerPresent     bool
	confirmation       *presetChoice
	acknowledged       bool
}

func permissionDescription(value, description string) string {
	if value == session.PermissionAutoReview {
		return composerText("无沙箱运行；每次原生工具调用和 PTC 内层调用前由同一模型进行实验性审查。", "Run without a sandbox after an experimental same-model review of every native tool call and PTC inner call.")
	}
	return description
}

func permissionRequiresRiskConfirmation(id string) bool {
	return id == session.PermissionFullAccess || id == session.PermissionAutoReview
}

// The current-session permission seat has its own Menu, not the generic preset
// list. Its identity and value come from the Session projection; choosing a row
// neither creates a Session nor borrows or clears the resident composer draft.
func paintPermissionPicker(c *ui.Context, m Model, colors theme.Colors, ink ui.Color) {
	snap := m.Snapshot
	if !snap.NativeContributions.Known || !snap.NativeContributions.PermissionCommand || snap.SelectedID == "" || snap.SelectedPermission == "" {
		return
	}
	options := permissionChoices(m.PermissionOptions)
	value := snap.SelectedPermissionValue
	display, resolved := resolvePresetCaption(snap.SelectedPermission, options)
	if value == "" {
		value = resolved
	}
	display = permissionLabel(value, display)
	description := ""
	for _, option := range options {
		if option.ID == value {
			display, description = option.Label, option.Description
			break
		}
	}
	owner := ui.Box(c.Key("permission-picker-owner")).Absolute().Size(0, 0).PassThrough()
	picker := ui.Local(owner, "permission-menu", func() permissionPickerState { return permissionPickerState{} })
	binding := nativeCommandOwner(snap, "permission")
	if picker.owner != binding {
		*picker = permissionPickerState{owner: binding}
	}
	locked := m.Inert || snap.PermissionSelectionBusy || m.OnPermissionChange == nil || m.State != nil && (m.State.Sending || m.State.Preparing > 0)
	if locked {
		picker.open = false
	}
	label := composerText("访问模式，当前：", "Access mode, current: ") + display
	trigger := chip(c, colors, picker.open, ink).Label(label).Tooltip(description).
		Disabled(locked).Bind(&picker.nav.Anchor)
	trigger.Children(func() {
		if icon := permissionMenuIcon(value); icon != nil {
			ui.Icon(c, icon).Size(14, 14).Shrink(0)
		}
		ui.Text(c, display).Grow(1).MinWidth(0).SingleLine().FontSize(13).FixedLineHeight(20).FontWeight(500).TextColor(ink)
		if value == session.PermissionAutoReview {
			ui.Text(c, "EXP").FontSize(8).FixedLineHeight(10).FontWeight(500).TextColor(chrome.Color(colors.Muted)).Shrink(0)
		}
		if snap.PermissionSelectionBusy {
			ui.Text(c, "…").Size(12, 12).TextColor(ink)
		} else {
			chevron(c, picker.open)
		}
	})
	if m.PermissionSwitchFailed || strings.EqualFold(strings.TrimSpace(snap.PermissionsErr), "switch_failed") {
		errText := i18n.T(i18n.KeyParityPermSwitchFailed)
		errAlt := i18n.T(i18n.KeyParityPermSwitchFailedAlt)
		ui.Text(c, errText).FontSize(11).TextColor(chrome.Color(colors.Danger)).SingleLine()
		if errAlt != "" && errAlt != errText {
			ui.Box(c).Size(0, 0).PassThrough().Label(errAlt)
		}
	}
	if picker.confirmation != nil {
		paintPickerPermissionConfirmation(c, m, picker, chrome.Color(colors.BG), chrome.Color(colors.FG))
	}
	if trigger.Clicked() {
		picker.open = !picker.open
		picker.nav.Focused = false
		picker.pointerPresent = false
		picker.nav.Active = 0
		for i, option := range options {
			if option.ID == value {
				picker.nav.Active = i
				break
			}
		}
	}
	if !picker.open {
		picker.nav.Focused = false
		return
	}
	if len(picker.nav.Rows) != len(options) {
		picker.nav.Rows = make([]*ui.Handle, len(options))
		for i := range picker.nav.Rows {
			picker.nav.Rows[i] = &ui.Handle{}
		}
	}
	if picker.nav.Active < 0 || picker.nav.Active >= len(options) {
		picker.nav.Active = 0
	}
	menuIconColor := chrome.Color(theme.SkinSemanticToken("", isDark(colors), "--dsw-alias-menu-icon", colors.FG))
	ui.PopoverBase(c, trigger, &picker.open, func(panel ui.Element) {
		panel = menuSurface(c, panel.Padding(4).Gap(0).MinWidth(144).MaxWidth(360).
			AttachTo(trigger, ui.AnchorTopLeft, ui.AnchorBottomLeft).Margin(0, 0, 4, 0), colors)
		pointerX, pointerY, pointerPresent := panel.PointerPosition()
		pointerMoved := pointerPresent && (!picker.pointerPresent || picker.pointerX != pointerX || picker.pointerY != pointerY)
		picker.pointerX, picker.pointerY, picker.pointerPresent = pointerX, pointerY, pointerPresent
		reveal := !picker.nav.Focused
		if len(options) > 0 {
			if panel.Shortcut(0, ui.KeyDown) {
				picker.nav.Active = (picker.nav.Active + 1) % len(options)
				picker.nav.Rows[picker.nav.Active].RequestFocus(c)
				reveal = true
			}
			if panel.Shortcut(0, ui.KeyUp) {
				picker.nav.Active = (picker.nav.Active + len(options) - 1) % len(options)
				picker.nav.Rows[picker.nav.Active].RequestFocus(c)
				reveal = true
			}
		}
		panel.Children(func() {
			if snap.PermissionsErr != "" {
				errMsg := snap.PermissionsErr
				if strings.EqualFold(strings.TrimSpace(errMsg), "switch_failed") {
					errMsg = i18n.T(i18n.KeyParityPermSwitchFailed)
				}
				ui.Text(c, errMsg).FontSize(12).TextColor(chrome.Color(colors.FG)).Padding(6).MaxLines(3)
				if m.OnRetryPermissions != nil && ui.Button(c, composerText("重试", "Retry")).Clicked() {
					m.OnRetryPermissions()
				}
			}
			ui.Scroll(c).FillWidth().MaxHeight(320).Gap(0).Children(func() {
				for i, option := range options {
					i, option := i, option
					row := menuRow(c, colors).Label(option.Label).Tooltip(option.Description).Bind(picker.nav.Rows[i])
					if pointerMoved && row.Hovered() {
						picker.nav.Active = i
						reveal = false
					}
					if i == picker.nav.Active {
						row = row.Background(chrome.Color(colors.Hover))
						if reveal {
							row.ScrollIntoView()
						}
					}
					row.Children(func() {
						if icon := permissionMenuIcon(option.ID); icon != nil {
							ui.Icon(c, icon).Size(14, 14).Shrink(0).TextColor(menuIconColor)
						}
						ui.Row(c).Grow(1).MinWidth(0).Gap(4).AlignItems(ui.Center).Children(func() {
							ui.Text(c, option.Label).FontSize(13).FixedLineHeight(20).TextColor(chrome.Color(colors.FG)).MinWidth(0).SingleLine()
							if option.ID == session.PermissionAutoReview {
								ui.Text(c, "EXP").FontSize(8).FixedLineHeight(10).FontWeight(500).TextColor(chrome.Color(colors.Muted)).Shrink(0)
							}
						})
						if option.ID == value {
							ui.Icon(c, checkIcon).Size(14, 14).Shrink(0).TextColor(chrome.Color(colors.FG))
						}
					})
					if row.Clicked() {
						picker.open, picker.nav.Focused = false, false
						picker.nav.Anchor.RequestFocus(c)
						if m.ConfirmPermissionInPicker && option.ID != value && permissionRequiresRiskConfirmation(option.ID) {
							picked := option
							picker.confirmation = &picked
							picker.acknowledged = false
						} else {
							m.OnPermissionChange(option.ID)
						}
					}
				}
			})
		})
		if !picker.nav.Focused && len(picker.nav.Rows) > 0 {
			picker.nav.Focused = true
			picker.nav.Rows[picker.nav.Active].RequestFocus(c)
		}
	})
}

func paintPickerPermissionConfirmation(c *ui.Context, m Model, picker *permissionPickerState, panel, fg ui.Color) {
	if picker.confirmation == nil {
		return
	}
	title := composerText("确认启用完全权限？", "Enable Full access?")
	description := i18n.T(i18n.KeyParityPermConfirmDescription)
	action := composerText("启用完全权限", "Enable Full access")
	ackLabel := composerText("我已了解风险，并愿意继续", "I understand these risks and want to continue")
	if picker.confirmation.ID == session.PermissionAutoReview {
		title = composerText("确认启用自动审查（实验）？", "Enable Auto review (experimental)?")
		description = composerText("自动审查不使用沙箱。每次原生工具调用和 PTC 内层调用前，都会由与当前 agent 相同的模型进行审查；审查拒绝的调用由你批准或拒绝。此功能仍属实验性，可能误放行或误拒绝，并会消耗额外 token。", "Auto review runs without a sandbox. Before every native tool call and PTC inner call, the same model as the current agent reviews whether to allow it; you approve or reject each call it denies. This feature is experimental, can falsely allow or deny actions, and uses additional tokens.")
		action = composerText("启用自动审查", "Enable Auto review")
		ackLabel = composerText("我已了解这些风险，并愿意继续", "I understand these risks and want to continue")
	}
	open := true
	ui.DialogBase(c.Key("picker-permission-risk"), &open, func(backdrop, dialog ui.Element) {
		mask := chrome.Reveal(c, backdrop, chrome.OverlayDuration)
		chrome.RevealMask(backdrop, ui.RGBA(0, 0, 0, .32), mask)
		dialog.Width(520).MaxWidthPercent(92).Padding(24).Radius(28).Clip().Opacity(mask).
			Background(panel).
			Draw(chrome.Glass(panel, chrome.MaterialPanel, 28)).Children(func() {
			ui.Column(c).Gap(14).Children(func() {
				ui.Text(c, title).FontSize(16).Bold().TextColor(fg)
				ui.Text(c, description).FontSize(12).TextColor(fg).LineHeight(1.5)
				if picker.confirmation.ID != session.PermissionAutoReview {
					ui.Box(c).Size(0, 0).PassThrough().Label(i18n.T(i18n.KeyParityPermConfirmNewSessionDesc))
					ui.Box(c).Size(0, 0).PassThrough().Label(i18n.T(i18n.KeyParityPermConfirmDescriptionAlt))
				}
				cb := ui.Checkbox(c, &picker.acknowledged, ackLabel)
				cb.Children(func() {
					ui.Box(c).Fill().PassThrough().Label(i18n.T(i18n.KeyParityPermAutoConfirmAcknowledge))
					ui.Box(c).Fill().PassThrough().Label(i18n.T(i18n.KeyParityPermAutoConfirmAcknowledgeAlt))
					ui.Box(c).Fill().PassThrough().Label(composerText("我已了解风险，并愿意继续", "I understand these risks and want to continue"))
				})
				ui.Row(c).Gap(8).Children(func() {
					if ui.Button(c, composerText("取消", "Cancel")).Clicked() {
						picker.confirmation = nil
						picker.acknowledged = false
					}
					if ui.PrimaryButton(c, action).Disabled(!picker.acknowledged).Clicked() {
						targetID := picker.confirmation.ID
						picker.confirmation = nil
						picker.acknowledged = false
						if m.OnPermissionChange != nil {
							m.OnPermissionChange(targetID)
						}
					}
				})
			})
		})
		chrome.MenuElevation(dialog)
	})
	if !open {
		picker.confirmation = nil
		picker.acknowledged = false
	}
}

// Host-configured values outside the shipped design set have no made-up glyph.
func permissionMenuIcon(value string) *ui.SVG {
	switch value {
	case permissionReadOnly, permissionWorkspaceWrite, session.PermissionFullAccess, session.PermissionAutoReview:
		return permissionIconFor(value)
	}
	return nil
}
