package settings

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

// BodySettingsAccess projects described, non-secret feature preferences into
// their original settings rows. The application retains the Host write owner.
type BodySettingsAccess struct {
	Owner    uint64
	Snapshot store.ViewState
	OnChange func(namespace, key, value string)
	OnPatch  func(namespace string, values map[string]string)
}

// settingsPreferenceButton follows SettingsSelect's borderless layer-1 pill
// trigger. It is local to settings; native popup/keyboard behavior is unchanged.
func settingsPreferenceButton(c *ui.Context, label string) ui.Element {
	button := ui.Button(c, label).Height(36).Padding(7, 14).Radius(999).BorderWidth(0).
		Background(c.Theme().Surface).TextColor(c.Theme().Text).FontSize(theme.InterfaceSize(14))
	if button.Hovered() {
		button.Background(c.Theme().SurfaceHover)
	}
	return button
}

// settingsPreferenceSelect retains native select admission, focus, arrow-key
// navigation and Changed semantics while giving its trigger the installed look.
func settingsPreferenceSelect(c *ui.Context, value *string, options []string) ui.Element {
	return settingsPreferenceSelectRadius(c, value, options, 999)
}

// settingsPreferenceSelectRadius is the same trigger with the corner radius of
// the page that hosts it: most rows use the 18 DIP pill, the ui-chat enums the
// installed build draws with a 12 DIP corner.
func settingsPreferenceSelectRadius(c *ui.Context, value *string, options []string, radius float32) ui.Element {
	return widgets.PillSelect(c, modelColors(c), value, options, radius)
}

// settingsOptionsScroll mirrors the source's classic 5 DIP scrollbar gutter.
// Native Scroll overlays its thumb, so only an actually overflowing page needs
// the extra trailing inset. TrackScroll publishes the settled layout extent;
// one invalidation applies/removes a gutter without guessing content height.
func settingsOptionsScroll(c *ui.Context, key string, gap float32, background ui.Color) ui.Element {
	pane := ui.Scroll(c.Key(key)).Grow(1).MinWidth(0).MinHeight(0).Background(background).Gap(gap)
	type optionsState struct {
		scroll ui.ScrollState
		gutter bool
	}
	state := ui.Local(pane, "source-scrollbar-gutter", func() optionsState { return optionsState{} })
	right := float32(24)
	if state.gutter {
		right += 5
	}
	pane.Padding(0, right, 24, 24).TrackScroll(&state.scroll)
	pane.Draw(func(_ *ui.Painter, bounds ui.Rect) {
		if bounds.H <= 0 {
			return
		}
		overflow := state.scroll.MaxY > 0
		if overflow != state.gutter {
			state.gutter = overflow
			c.Invalidate()
		}
	})
	return pane
}

func bodyHeading(c *ui.Context, title, hint string, fg, muted ui.Color) {
	ui.Column(c).FillWidth().MinWidth(0).Gap(4).Children(func() {
		ui.Text(c, title).FillWidth().FontSize(theme.InterfaceSize(16)).FontWeight(500).TextColor(fg)
		if hint != "" {
			ui.Text(c, hint).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
	})
}

// A Setting-Cell is a flat, hairline-separated title/description and compact
// control. Measured content width controls stacking, including inside a modal.
//
// The source row owns a half-pixel bottom rule, 16 DIP vertical padding,
// 22/18 DIP title/description lines and an intrinsic trailing control group.
// Narrow panes stack the group to retain keyboard-reachable controls.
func bodyPreferenceRow(c *ui.Context, key, title, hint string, fg, muted ui.Color, controls func()) ui.Element {
	gear := takeRowGear()
	help := takeRowHelp()
	if !settingsItemBefore(c, key) {
		return ui.Element{}
	}
	row := ui.Column(c.Key(key)).FillWidth().MinWidth(0).Gap(0)
	settingsSearchTarget(c, key, title, row)
	width := row.Bounds().W
	horizontal := width >= 420 && !appearanceChoiceRow(key)
	row.Draw(func(p *ui.Painter, bounds ui.Rect) {
		p.Fill(ui.Rect{X: bounds.X, Y: bounds.Y + bounds.H - .5, W: bounds.W, H: .5}, c.Theme().Border, 0)
		if bounds.W != width {
			c.Invalidate()
		}
	})
	row.Children(func() {
		content := func() {
			copy := ui.Column(c).MinWidth(0).Gap(4)
			if horizontal {
				copy.Grow(1).Padding(0, 48, 0, 0)
			} else {
				copy.FillWidth()
			}
			copy.Children(func() {
				preferenceRowTitle(c, title, fg, help)
				if hint != "" {
					ui.Text(c, hint).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
				}
			})
			if gear != nil || controls != nil {
				group := ui.Row(c).MinWidth(0).Shrink(0).Gap(8).AlignItems(ui.Center).Justify(ui.End)
				if !horizontal {
					group.FillWidth()
				}
				group.Children(func() {
					if gear != nil {
						rowGearButton(c, *gear)
					}
					if controls != nil {
						control := ui.Column(c).MinWidth(0).Gap(6)
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
			ui.Row(c).FillWidth().MinWidth(0).Gap(8).Padding(16, 0, 16.5, 0).AlignItems(ui.Center).Children(content)
		} else {
			ui.Column(c).FillWidth().MinWidth(0).MinHeight(64).Gap(10).Padding(14, 0).Children(content)
		}
	})
	return row
}

// Preference rows are painted in one pass per frame, so the frame's timestamp
// identifies the page: the same frame's first row starts the list, and the next
// frame starts a fresh count.
var (
	preferenceRowFrame time.Time
	preferenceRowCount int
)

// nextPreferenceRowIndex is this row's ordinal inside the frame.
func nextPreferenceRowIndex(now time.Time) int {
	if !now.Equal(preferenceRowFrame) {
		preferenceRowFrame = now
		preferenceRowCount = 0
	}
	index := preferenceRowCount
	preferenceRowCount++
	return index
}

// bodyDisclosure folds a secondary group behind one row. The chevron and the
// hover fill are what make the row read as openable, so a folded group stays
// discoverable next to the rows that are always visible.
func bodyDisclosure(c *ui.Context, key, title string, body func()) {
	anchor := ui.Column(c.Key(key)).FillWidth().MinWidth(0).Gap(8)
	open := ui.Local(anchor, "open", func() bool { return false })
	if settingsSearchReveal(c, key) {
		*open = true
	}
	settingsSearchTarget(c, key, title, anchor)
	muted := c.Theme().TextMuted
	anchor.Children(func() {
		toggle := ui.Button(c, "").Label(title).FillWidth().Height(34).Padding(0, 8, 0, 4).BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).FocusRing(false)
		toggle.Children(func() {
			if *open {
				ui.Icon(c, bodyIcon("down")).Size(12, 12).TextColor(muted).PassThrough()
			} else {
				ui.Icon(c, bodyIcon("right")).Size(12, 12).TextColor(muted).PassThrough()
			}
			ui.Text(c, " "+title).FontSize(theme.InterfaceSize(13)).TextColor(muted).Grow(1).MinWidth(0).SingleLine()
		})
		if toggle.Hovered() {
			toggle.Background(c.Theme().SurfaceHover)
		}
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
	// A line inserted while the Host connects, or is read-only, would shift every row below it.
	settingsNotice(c, "host-state", "", !s.ClientReady || !s.Writable, func() string {
		if !s.ClientReady {
			return providerCopy("Host 未连接；保留上次设置。", "Host disconnected; showing retained settings.")
		}
		return providerCopy("此 Host 的设置为只读。", "Settings on this Host are read-only.")
	})
	anchor := ui.Box(c.Key("preference-error-notice")).Absolute().Size(0, 0).PassThrough()
	previous := ui.Local(anchor, "error", func() string { return "" })
	failure := s.Err
	for _, fieldError := range s.PreferenceErrors {
		if failure != "" && failure == fieldError {
			failure = "" // Its mounted field owns this refusal, not the whole page.
			c.CloseToast("settings-preference-error")
			break
		}
	}
	if failure == "" {
		*previous = ""
	} else {
		if *previous != failure {
			c.AddToast(ui.Toast{ID: "settings-preference-error", Title: failure})
			*previous = failure
		}
	}
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
			var values map[string]json.RawMessage
			_ = json.Unmarshal(ns.Value, &values)
			value := strings.TrimSpace(string(values[key]))
			var text string
			if json.Unmarshal(values[key], &text) == nil {
				value = text
			}
			return field, value, true
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
	if !settingsItemBefore(c, namespace+"."+key) {
		takeRowGear()
		takeRowHelp()
		return true
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
			radius := float32(999)
			switch key {
			case "transcriptView", "collapseTiming", "linkOpening", "performanceUsage":
				radius = 12
			}
			if settingsPreferenceSelectRadius(c, &display, labels, radius).Label(title).Disabled(bodyHostFieldDisabled(a, namespace, key) || len(options) == 0).Changed() {
				if index := hostEnumIndex(labels, display); index >= 0 {
					clearFailure()
					a.OnChange(namespace, key, options[index])
				}
			}
		default:
			anchor := ui.Column(c.Key(fmt.Sprintf("draft-%d-%s-%s", a.Owner, namespace, key))).Width(140).MaxWidthPercent(100).Gap(4)
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
				ui.TextInput(c, &draft.Value).Label(title).FillWidth().Height(32).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(13)).Disabled(!a.Snapshot.ClientReady || !a.Snapshot.Writable || a.Snapshot.Loading).OnSubmit(commit)
				if ui.Button(c, providerCopy("保存", "Save")).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(12.5)).Disabled(bodyHostFieldDisabled(a, namespace, key) || draft.Value == value).Clicked() {
					commit()
				}
			})
		}
	})
	settingsSearchTarget(c, namespace+"."+key, title, row)
	return true
}

func bodyLocalToggle(c *ui.Context, key, title, hint string, value *bool, disabled bool, fg, muted ui.Color, changed func(bool)) {
	if value == nil {
		return
	}
	bodyPreferenceRow(c, key, title, hint, fg, muted, func() {
		if ui.Switch(c, value).Label(title).Disabled(disabled).Changed() && changed != nil {
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

// rowGearButton paints the 16×16 gear: muted glyph, hover wash, tooltip, and a
// dimmed state when no callback is wired yet.
func rowGearButton(c *ui.Context, gear rowGear) {
	button := ui.Button(c, "").Label(gear.label).Size(28, 28).Padding(6).Radius(6).BorderWidth(0).Background(ui.Transparent).Disabled(gear.disabled)
	if gear.hint != "" {
		button.Tooltip(gear.hint)
	}
	if button.Hovered() && !gear.disabled {
		button.Background(c.Theme().SurfaceHover)
	}
	button.Children(func() { ui.Icon(c, bodyIcon("gear")).Size(16, 16).TextColor(c.Theme().TextMuted) })
	if gear.disabled {
		return
	}
	if button.Clicked() && gear.onOpen != nil {
		gear.onOpen()
	}
}
