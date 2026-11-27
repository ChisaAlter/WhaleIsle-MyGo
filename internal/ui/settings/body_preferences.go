package settings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"

	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"

	"github.com/egoist/mygo/ui"
)

// BodySettingsAccess projects described, non-secret feature preferences into
// their original settings rows. The application retains the Host write owner.
type BodySettingsAccess struct {
	Owner    uint64
	Snapshot store.ViewState
	OnChange func(namespace, key, value string)
	OnPatch  func(namespace string, values map[string]string)
}

// settingsPreferenceButton is a preference row's menu trigger: coss's outline
// small button. Native popup/keyboard behavior is the caller's.
func settingsPreferenceButton(c *ui.Context, label string) ui.Element {
	return kButton(c, label)
}

// settingsPreferenceSelect is a preference row's coss Select: native select
// admission, focus, arrow-key navigation and Changed semantics. A uniform
// width keeps triggers across rows aligned on one left edge.
func settingsPreferenceSelect(c *ui.Context, value *string, options []string) ui.Element {
	return kSelect(c, value, options, "", false).Width(design.Sp(52)).MaxWidthPercent(100)
}

// settingsPreferenceSelectRadius is settingsPreferenceSelect; coss gives every
// select trigger the same rounded-lg corner, so radius is ignored.
func settingsPreferenceSelectRadius(c *ui.Context, value *string, options []string, _ float32) ui.Element {
	return kSelect(c, value, options, "", false).Width(design.Sp(52)).MaxWidthPercent(100)
}

// settingsOptionsScroll is the options pane: coss's ScrollArea inset of 24 DIPs
// (p-6) around the page, with the native overlay scrollbar.
func settingsOptionsScroll(c *ui.Context, key string, gap float32, background ui.Color) ui.Element {
	return ui.Scroll(c.Key(key)).Grow(1).MinWidth(0).MinHeight(0).Background(background).Gap(gap).
		Padding(design.Sp(2), design.Sp(6), design.Sp(6), design.Sp(6)).Label(providerCopy("设置内容", "Settings content"))
}

func bodyHeading(c *ui.Context, title, hint string, fg, muted ui.Color) {
	ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(1)).Children(func() {
		kHeading(c, title)
		if hint != "" {
			kText(c, hint, design.TextSM, design.WeightNormal, tk(c).MutedForeground).FillWidth()
		}
	})
}

// A Setting-Cell is a flat, rule-separated title/description and compact
// control: a coss item list row. Measured content width controls stacking,
// including inside a modal; narrow panes stack the group to keep the controls
// keyboard-reachable.
func bodyPreferenceRow(c *ui.Context, key, title, hint string, fg, muted ui.Color, controls func()) ui.Element {
	gear := takeRowGear()
	help := takeRowHelp()
	row := ui.Column(c.Key(key)).FillWidth().MinWidth(0).Gap(0).Label(title)
	settingsSearchTarget(c, key, title, row)
	settingsRowRule(c, row)
	width := settingsContentWidth(c, row, 0)
	horizontal := width >= settingsRowWide && !appearanceChoiceRow(key)
	row.Draw(func(_ *ui.Painter, bounds ui.Rect) {
		if (bounds.W >= settingsRowWide && !appearanceChoiceRow(key)) != horizontal {
			c.Invalidate()
		}
	})
	row.Children(func() {
		content := func() {
			copy := ui.Column(c).MinWidth(0).Gap(design.Sp(1))
			if horizontal {
				copy.Grow(1).Padding(0, design.Sp(12), 0, 0)
			} else {
				copy.FillWidth()
			}
			copy.Children(func() {
				preferenceRowTitle(c, title, fg, help)
				if hint != "" {
					kHint(c, hint)
				}
			})
			if gear != nil || controls != nil {
				group := ui.Row(c).MinWidth(0).Shrink(0).Gap(design.Sp(2)).AlignItems(ui.Center).Justify(ui.End)
				if !horizontal {
					group.FillWidth()
				}
				group.Children(func() {
					if gear != nil {
						rowGearButton(c, *gear)
					}
					if controls != nil {
						control := ui.Column(c).MinWidth(0).Gap(design.Sp(1.5))
						if horizontal {
							control.AlignItems(ui.End)
						} else {
							control.Grow(1).AlignItems(ui.Stretch)
						}
						control.Children(controls)
					}
				})
			}
		}
		if horizontal {
			ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).Padding(design.Sp(4), 0).AlignItems(ui.Center).Children(content)
		} else {
			ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(2.5)).Padding(design.Sp(4), 0).Children(content)
		}
	})
	return row
}

// bodyDisclosure folds a secondary group behind one row: coss's Collapsible
// trigger, a ghost row with a chevron that turns as it opens.
func bodyDisclosure(c *ui.Context, key, title string, body func()) {
	anchor := ui.Column(c.Key(key)).FillWidth().MinWidth(0).Gap(design.Sp(2))
	open := ui.Local(anchor, "open", func() bool { return false })
	if settingsSearchReveal(c, key) {
		*open = true
	}
	settingsSearchTarget(c, key, title, anchor)
	t := tk(c)
	anchor.Children(func() {
		toggle := coss.Button(c, "", coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM, Label: title, Content: func() {
			glyph := coss.IconChevronRight
			if *open {
				glyph = coss.IconChevronDown
			}
			ui.Icon(c, glyph.SVG()).Size(design.IconSM, design.IconSM).TextColor(t.MutedForeground).PassThrough()
			kText(c, title, design.TextSM, design.WeightMedium, t.MutedForeground).Grow(1).MinWidth(0).SingleLine()
		}})
		toggle.FillWidth().Justify(ui.Start).Expanded(*open)
		if toggle.Clicked() {
			*open = !*open
		}
		if *open {
			body()
		}
	})
}

func bodyHostState(c *ui.Context, a *BodySettingsAccess, muted ui.Color) {
	if a == nil {
		return
	}
	s := a.Snapshot
	// A disconnected or read-only Host is a live condition, so it stays as a
	// status line for as long as it holds rather than fading like an outcome.
	hostState := ""
	if !s.ClientReady {
		hostState = providerCopy("Host 未连接；保留上次设置。", "Host disconnected; showing retained settings.")
	} else if !s.Writable {
		hostState = providerCopy("此 Host 的设置为只读。", "Settings on this Host are read-only.")
	}
	if hostState != "" {
		coss.Alert(c, "", hostState, coss.AlertOptions{Variant: coss.AlertWarning, Key: "settings-host-state"})
	}
	failure := s.Err
	for _, fieldError := range s.PreferenceErrors {
		if failure != "" && failure == fieldError {
			failure = "" // Its mounted field owns this refusal, not the whole page.
			break
		}
	}
	settingsAlert(c, "settings-preference-error", failure != "", providerCopy("没能保存这个设置，请重试。", "This setting was not saved. Try again."), failure)
}

func bodyHostValue(a *BodySettingsAccess, namespace, key string) (store.Field, string, bool) {
	if a == nil {
		return store.Field{}, "", false
	}
	for _, ns := range a.Snapshot.Namespaces {
		if ns.NS != namespace {
			continue
		}
		for _, field := range ns.Fields {
			if field.Key != key || SecretLikeKey(field.Key) {
				continue
			}
			values := namespaceStringValues(ns.Value)
			return field, values[key], true
		}
	}
	return store.Field{}, "", false
}

func bodyHostDisabled(a *BodySettingsAccess) bool {
	return a == nil || a.OnChange == nil || !a.Snapshot.ClientReady || !a.Snapshot.Writable || a.Snapshot.Loading
}

func bodyHostFieldDisabled(a *BodySettingsAccess, namespace, key string) bool {
	return bodyHostDisabled(a) || a.Snapshot.PreferencePending[namespace+"."+key]
}

// hostEnumLabel turns the Host's stored enum token into the reference's visible
// option copy. Unknown tokens stay as-is so a new Host option is never hidden.
func hostEnumLabel(value string) string {
	switch value {
	case "workspace-write":
		return providerCopy("工作区内修改", "Workspace write")
	case "danger-full-access":
		return providerCopy("完全权限", "Full access")
	case "read-only":
		return providerCopy("仅可查看", "Read Only")
	case "normal", "standard":
		return providerCopy("标准", "Standard")
	case "detailed":
		return providerCopy("详细", "Detailed")
	case "compact":
		return providerCopy("压缩", "Compact")
	case "verbose":
		return providerCopy("完整", "Verbose")
	case "expanded":
		return providerCopy("展开", "Expanded")
	case "completion":
		return providerCopy("回答结束后", "After the answer")
	case "next-input":
		return providerCopy("输入下一条后", "After the next input")
	case "immediate":
		return providerCopy("立即", "Immediately")
	case "sidebar":
		return providerCopy("应用内侧边栏", "In-app sidebar")
	case "queue":
		return providerCopy("排队发送", "Queue")
	case "steer":
		return providerCopy("插话发送", "Steer")
	case "new-tab":
		return providerCopy("新标签页", "New tab")
	case "system":
		return providerCopy("系统默认", "System default")
	case "zh":
		return "中文"
	case "en":
		return "English"
	default:
		return value
	}
}

func hostEnumIndex(labels []string, selected string) int {
	for i, label := range labels {
		if label == selected {
			return i
		}
	}
	return -1
}

func bodyHostPreference(c *ui.Context, a *BodySettingsAccess, namespace, key, title, hint string, fg, muted ui.Color) bool {
	field, value, found := bodyHostValue(a, namespace, key)
	if !found {
		return false
	}
	clearFailure := bodyPreferenceFieldError(c, a, namespace, key, title)
	if hint == "" {
		hint = field.Description
	}
	row := bodyPreferenceRow(c, fmt.Sprintf("preference-%d-%s-%s", a.Owner, namespace, key), title, hint, fg, muted, func() {
		switch field.Kind {
		case store.FieldBool:
			checked := value == "true"
			style := settingsSwitchLabelled
			if namespace == "ui-git" || namespace == "ui-titlebar" || namespace == "ui-conversation" {
				style = settingsSwitchInput
			}
			if settingsPreferenceSwitch(c, &checked, style).Label(title).Disabled(bodyHostFieldDisabled(a, namespace, key)).Changed() {
				clearFailure()
				a.OnChange(namespace, key, fmt.Sprint(checked))
			}
		case store.FieldEnum:
			options := field.EnumOptions
			if key == "transcriptView" {
				options = nil
				for _, option := range field.EnumOptions {
					if option != "normal" && option != "expanded" {
						options = append(options, option)
					}
				}
				if value == "normal" || value == "expanded" {
					for _, option := range options {
						if option == "detailed" {
							value = option
							break
						}
					}
				}
			}
			if value == "" && len(options) > 0 {
				// An unset Host preference still shows the first option the Host
				// would use, instead of an empty select box.
				value = options[0]
			}
			labels := make([]string, len(options))
			for i := range options {
				labels[i] = hostEnumLabel(options[i])
			}
			display := hostEnumLabel(value)
			if settingsPreferenceSelect(c, &display, labels).Label(title).Disabled(bodyHostFieldDisabled(a, namespace, key) || len(options) == 0).Changed() {
				if index := hostEnumIndex(labels, display); index >= 0 {
					clearFailure()
					a.OnChange(namespace, key, options[index])
				}
			}
		default:
			anchor := ui.Column(c.Key(fmt.Sprintf("draft-%d-%s-%s", a.Owner, namespace, key))).Width(design.Sp(35)).MaxWidthPercent(100).Gap(design.Sp(1))
			type draftState struct{ Value, Base string }
			draft := ui.Local(anchor, "value", func() draftState { return draftState{value, value} })
			if draft.Value == value || draft.Value == draft.Base && value != draft.Base {
				draft.Value, draft.Base = value, value
			}
			commit := func() {
				if !bodyHostFieldDisabled(a, namespace, key) {
					clearFailure()
					a.OnChange(namespace, key, draft.Value)
				}
			}
			anchor.Children(func() {
				kInput(c, &draft.Value, coss.InputOptions{Label: title, Disabled: !a.Snapshot.ClientReady || !a.Snapshot.Writable || a.Snapshot.Loading}).OnSubmit(commit)
				if kButtonDisabled(c, providerCopy("保存", "Save"), bodyHostFieldDisabled(a, namespace, key) || draft.Value == value).Clicked() {
					commit()
				}
			})
		}
	})
	settingsSearchTarget(c, namespace+"."+key, title, row)
	bodyPreferenceFieldAlert(c, a, namespace, key, title)
	return true
}

func bodyLocalToggle(c *ui.Context, key, title, hint string, value *bool, disabled bool, fg, muted ui.Color, changed func(bool)) {
	if value == nil {
		return
	}
	bodyPreferenceRow(c, key, title, hint, fg, muted, func() {
		if coss.Switch(c, value, "", coss.CheckOptions{Label: title, Disabled: disabled}).Changed() && changed != nil {
			changed(*value)
		}
	})
}

// rowGear is the optional 16×16 gear a preference row can carry to the left of
// its control. A row renders at most one, and only when the caller explicitly
// asks for it through withRowGear, so unrelated rows never grow one.
type rowGear struct {
	label    string
	hint     string
	disabled bool
	onOpen   func()
}

var pendingRowGear *rowGear

// withRowGear renders the next bodyPreferenceRow with a gear.
func withRowGear(gear rowGear, render func()) {
	pendingRowGear = &gear
	defer func() { pendingRowGear = nil }()
	render()
}

// takeRowGear hands the pending gear to the row that is being built now.
func takeRowGear() *rowGear {
	gear := pendingRowGear
	pendingRowGear = nil
	return gear
}

// rowGearButton is the row's gear: a ghost icon button with its tooltip,
// disabled while no callback is wired yet.
func rowGearButton(c *ui.Context, gear rowGear) {
	button := kIconButton(c, bodyIcon("gear"), gear.label, gear.disabled)
	if gear.hint != "" {
		button.Tooltip(gear.hint)
	}
	if !gear.disabled && button.Clicked() && gear.onOpen != nil {
		gear.onOpen()
	}
}

// namespaceStringValues decodes a namespace value document into its flattened
// string values once per unique document payload so rendering frames never call
// json.Unmarshal repeatedly.
func namespaceStringValues(raw json.RawMessage) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	valuesMu.Lock()
	if v, ok := valuesCache[string(raw)]; ok {
		valuesMu.Unlock()
		return v
	}
	valuesMu.Unlock()

	var rawMap map[string]json.RawMessage
	_ = json.Unmarshal(raw, &rawMap)
	var decoded map[string]string
	if len(rawMap) > 0 {
		decoded = make(map[string]string, len(rawMap))
		for k, v := range rawMap {
			decoded[k] = decodeHostFieldValue(v)
		}
	}

	valuesMu.Lock()
	if len(valuesCache) >= 64 {
		clear(valuesCache)
	}
	valuesCache[string(raw)] = decoded
	valuesMu.Unlock()
	return decoded
}

func decodeHostFieldValue(raw json.RawMessage) string {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return ""
	}
	if trimmed[0] == '"' {
		var text string
		if json.Unmarshal(trimmed, &text) == nil {
			return text
		}
	}
	return string(trimmed)
}

var valuesMu sync.Mutex

var valuesCache = map[string]map[string]string{}
