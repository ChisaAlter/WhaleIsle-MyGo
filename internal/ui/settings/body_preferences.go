package settings

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/theme"
)

// BodySettingsAccess projects described, non-secret feature preferences into
// their original settings rows. The application retains the Host write owner.
type BodySettingsAccess struct {
	Owner    uint64
	Snapshot store.ViewState
	OnChange func(namespace, key, value string)
	OnPatch  func(namespace string, values map[string]string)
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
// The hairline is drawn above each row instead of below it, and the first row of
// the frame skips it. That way a page can never end on a stray rule above its
// action bar, and a list of generated rows (colours, presets, fonts) needs no
// per-call-site bookkeeping.
func bodyPreferenceRow(c *ui.Context, key, title, hint string, fg, muted ui.Color, controls func()) ui.Element {
	row := ui.Column(c.Key(key)).FillWidth().MinWidth(0).Gap(0)
	width := row.Bounds().W
	leadingRule := nextPreferenceRowIndex(c.Now()) > 0
	row.Draw(func(p *ui.Painter, bounds ui.Rect) {
		if leadingRule {
			p.Fill(ui.Rect{X: bounds.X, Y: bounds.Y, W: bounds.W, H: .5}, ui.RGBA(muted.R, muted.G, muted.B, .25), 0)
		}
		if bounds.W != width {
			c.Invalidate()
		}
	})
	row.Children(func() {
		content := func() {
			copy := ui.Column(c).MinWidth(0).Gap(4)
			if width >= 420 {
				copy.Grow(1)
			} else {
				copy.FillWidth()
			}
			copy.Children(func() {
				ui.Text(c, title).FillWidth().FontSize(theme.InterfaceSize(14)).TextColor(fg)
				if hint != "" {
					ui.Text(c, hint).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
				}
			})
			if gear := takeRowGear(); gear != nil {
				rowGearButton(c, *gear)
			}
			if controls != nil {
				control := ui.Column(c).MinWidth(0).Gap(6)
				if width >= 420 {
					// The reference selector column is about 135 DIP and each
					// control keeps its intrinsic width, right-aligned against
					// the row's trailing edge. The old fixed 180 DIP column
					// stretched every select and stole room from the copy.
					control.Width(140).Shrink(0).AlignItems(ui.End)
				} else {
					control.FillWidth()
				}
				control.Children(controls)
			}
		}
		if width >= 420 {
			ui.Row(c).FillWidth().MinWidth(0).MinHeight(64).Gap(24).Padding(16, 0).AlignItems(ui.Center).Children(content)
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
	if s.Loading {
		ui.Text(c, providerCopy("正在读取设置…", "Loading settings…")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
	}
	if !s.ClientReady {
		ui.Text(c, providerCopy("Host 未连接；保留上次设置。", "Host disconnected; showing retained settings.")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
	} else if !s.Writable {
		ui.Text(c, providerCopy("此 Host 的设置为只读。", "Settings on this Host are read-only.")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
	}
	if s.Saving {
		ui.Text(c, providerCopy("正在保存…", "Saving…")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
	}
	if s.Err != "" {
		ui.Text(c, s.Err).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
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
	return a == nil || a.OnChange == nil || !a.Snapshot.ClientReady || !a.Snapshot.Writable || a.Snapshot.Loading || a.Snapshot.Saving
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
		return providerCopy("应用内侧栏", "In-app sidebar")
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
	if hint == "" {
		hint = field.Description
	}
	bodyPreferenceRow(c, fmt.Sprintf("preference-%d-%s-%s", a.Owner, namespace, key), title, hint, fg, muted, func() {
		switch field.Kind {
		case store.FieldBool:
			checked := value == "true"
			if ui.Switch(c, &checked).Label(title).Disabled(bodyHostDisabled(a)).Changed() {
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
			if ui.Select(c, &display, labels).Label(title).Height(32).MinWidth(84).MaxWidth(140).Disabled(bodyHostDisabled(a) || len(options) == 0).Changed() {
				if index := hostEnumIndex(labels, display); index >= 0 {
					a.OnChange(namespace, key, options[index])
				}
			}
		default:
			anchor := ui.Column(c.Key(fmt.Sprintf("draft-%d-%s-%s", a.Owner, namespace, key))).FillWidth().Gap(4)
			type draftState struct{ Value, Base string }
			draft := ui.Local(anchor, "value", func() draftState { return draftState{value, value} })
			if draft.Value == value || draft.Value == draft.Base && value != draft.Base {
				draft.Value, draft.Base = value, value
			}
			commit := func() {
				if !bodyHostDisabled(a) {
					a.OnChange(namespace, key, draft.Value)
				}
			}
			anchor.Children(func() {
				ui.TextInput(c, &draft.Value).Label(title).FillWidth().Disabled(!a.Snapshot.ClientReady || !a.Snapshot.Writable || a.Snapshot.Loading).OnSubmit(commit)
				if ui.Button(c, providerCopy("保存", "Save")).Disabled(bodyHostDisabled(a) || draft.Value == value).Clicked() {
					commit()
				}
			})
		}
	})
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
	button := ui.Button(c, "").Label(gear.label).Size(24, 24).Padding(4).Radius(6).BorderWidth(0).Background(ui.Transparent).Disabled(gear.disabled)
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
