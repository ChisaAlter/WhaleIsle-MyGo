package composer

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/widgets"
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
func paintPermissionPicker(c *ui.Context, m Model, ink ui.Color) {
	snap := m.Snapshot
	if !snap.NativeContributions.Known || !snap.NativeContributions.PermissionCommand || snap.SelectedID == "" || snap.SelectedPermission == "" {
		return
	}
	t := design.For(c)
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
	trigger := chip(c, picker.open, ink).Label(label).Tooltip(description).
		Disabled(locked).Bind(&picker.nav.Anchor)
	trigger.Children(func() {
		if icon := permissionMenuIcon(value); icon != nil {
			ui.Icon(c, icon).Size(design.IconXS, design.IconXS).Shrink(0)
		}
		widgets.FlipText(c, display, widgets.FlipTextOptions{
			Key:        "permission-trigger",
			Color:      ink,
			FontSize:   design.TextSM.Size,
			LineHeight: design.TextSM.LineHeight,
			FontWeight: design.WeightMedium,
		}).Grow(1).MinWidth(0)
		if value == session.PermissionAutoReview {
			coss.Badge(c, "EXP", coss.BadgeOptions{Variant: coss.BadgeOutline, Size: coss.BadgeSM}).Shrink(0)
		}
		if snap.PermissionSelectionBusy {
			ui.Text(c, "…").Size(12, 12).TextColor(ink)
		} else {
			chevron(c, picker.open)
		}
	})
	if m.PermissionSwitchFailed || (snap.PermissionsErr != nil && (strings.EqualFold(strings.TrimSpace(snap.PermissionsErr.Code), "switch_failed") || strings.EqualFold(strings.TrimSpace(snap.PermissionsErr.Text), "switch_failed"))) {
		errText := i18n.T(i18n.KeyParityPermSwitchFailed)
		errAlt := i18n.T(i18n.KeyParityPermSwitchFailedAlt)
		ui.Text(c, errText).FontSize(design.TextXS.Size).TextColor(t.DestructiveForeground).SingleLine()
		if errAlt != "" && errAlt != errText {
			ui.Box(c).Size(0, 0).PassThrough().Label(errAlt)
		}
	}
	if picker.confirmation != nil {
		paintPickerPermissionConfirmation(c, m, picker)
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
	menuIconColor := t.PopoverForeground
	ui.PopoverBase(c, trigger, &picker.open, func(panel ui.Element) {
		panel = menuSurface(c, panel.Padding(design.Sp(1)).Gap(0).MinWidth(144).MaxWidth(360).
			AttachTo(trigger, ui.AnchorTopLeft, ui.AnchorBottomLeft).Margin(0, 0, design.Sp(1), 0))
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
			if snap.PermissionsErr != nil {
				errMsg := snap.PermissionsErr.SentenceText()
				if strings.EqualFold(strings.TrimSpace(snap.PermissionsErr.Code), "switch_failed") || strings.EqualFold(strings.TrimSpace(snap.PermissionsErr.Text), "switch_failed") {
					errMsg = i18n.T(i18n.KeyParityPermSwitchFailed)
				}
				ui.Text(c, errMsg).FontSize(design.TextXS.Size).TextColor(t.PopoverForeground).Padding(design.Sp(1.5)).MaxLines(3)
				if m.OnRetryPermissions != nil && coss.Button(c, composerText("重试", "Retry"), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS}).Clicked() {
					m.OnRetryPermissions()
				}
			}
			coss.ScrollArea(c, coss.ScrollAreaOptions{}, func() {
				for i, option := range options {
					i, option := i, option
					row := menuRow(c).Label(option.Label).Tooltip(option.Description).Bind(picker.nav.Rows[i])
					if pointerMoved && row.Hovered() {
						picker.nav.Active = i
						reveal = false
					}
					if i == picker.nav.Active {
						row = row.Background(t.Accent).TextColor(t.AccentForeground)
						if reveal {
							row.ScrollIntoView()
						}
					}
					row.Children(func() {
						if icon := permissionMenuIcon(option.ID); icon != nil {
							ui.Icon(c, icon).Size(design.IconXS, design.IconXS).Shrink(0).TextColor(menuIconColor)
						}
						ui.Row(c).Grow(1).MinWidth(0).Gap(design.Sp(1)).AlignItems(ui.Center).Children(func() {
							ui.Text(c, option.Label).FontSize(design.TextSM.Size).FixedLineHeight(design.TextSM.LineHeight).TextColor(t.PopoverForeground).MinWidth(0).SingleLine()
							if option.ID == session.PermissionAutoReview {
								coss.Badge(c, "EXP", coss.BadgeOptions{Variant: coss.BadgeOutline, Size: coss.BadgeSM}).Shrink(0)
							}
						})
						if option.ID == value {
							cossIcon(c, coss.IconCheck, design.IconXS, t.PopoverForeground)
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
			}).FillWidth().MaxHeight(320)
		})
		if !picker.nav.Focused && len(picker.nav.Rows) > 0 {
			picker.nav.Focused = true
			picker.nav.Rows[picker.nav.Active].RequestFocus(c)
		}
	})
}

func paintPickerPermissionConfirmation(c *ui.Context, m Model, picker *permissionPickerState) {
	if picker.confirmation == nil {
		return
	}
	targetID := picker.confirmation.ID
	paintPermissionRiskDialog(c, "picker-permission-risk", targetID, &picker.acknowledged, false, i18n.T(i18n.KeyParityPermConfirmNewSessionDesc),
		func() {
			picker.confirmation = nil
			picker.acknowledged = false
		},
		func() {
			picker.confirmation = nil
			picker.acknowledged = false
			if m.OnPermissionChange != nil {
				m.OnPermissionChange(targetID)
			}
		},
	)
}

// Host-configured values outside the shipped design set have no made-up glyph.
func permissionMenuIcon(value string) *ui.SVG {
	switch value {
	case permissionReadOnly, permissionWorkspaceWrite, session.PermissionFullAccess, session.PermissionAutoReview:
		return permissionIconFor(value)
	}
	return nil
}
