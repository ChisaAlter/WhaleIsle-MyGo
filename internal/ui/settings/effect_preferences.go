package settings

import (
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"math"
	"reflect"
	"sort"
	"strings"
	"unicode/utf16"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/theme"
)

type effectChoice struct {
	path, zh, en string
	choices      []string
}
type effectNumber struct {
	path, zh, en string
	lo, hi, step float64
}
type effectDraft struct {
	CustomTyping        [3]string
	Style               map[string]any
	Presets             map[string]map[string]any
	Name, Selected, Err string
	Pending             map[string]string
}

func typingCustomSeed(style map[string]any) [3]string {
	colors := [3]string{"#e2e8f0", "#60a5fa", "#64748b"}
	switch effectValue(style, "colors.kind") {
	case "preset":
		id, _ := effectValue(style, "colors.id").(string)
		colors = composer.TypingColorScheme(id)
	case "custom":
		for i, key := range []string{"echo", "caret", "text"} {
			colors[i], _ = effectValue(style, "colors."+key).(string)
		}
	}
	return colors
}

var beamChoices = []effectChoice{
	{"mode", "视觉模式", "Visual mode", []string{"legacy", "lounge", "aurora", "reactive", "custom"}},
	{"direction", "旋转方向", "Direction", []string{"clockwise", "counterclockwise", "pingPong"}},
	{"easing", "缓动", "Easing", []string{"linear", "ease", "ease-in-out", "cubic-bezier"}},
	{"palette.kind", "配色来源", "Color source", []string{"preset", "custom"}},
	{"palette.id", "配色方案", "Palette", []string{"legacy", "warm", "aurora", "rose", "glacier", "cyber", "sunset", "cyan"}},
}
var beamNumbers = []effectNumber{
	{"period", "周期（秒）", "Period (seconds)", .8, 60, .01}, {"intensity", "强度", "Intensity", 40, 140, 5}, {"bloom", "辉光", "Bloom", 0, 160, 5}, {"hue", "色相", "Hue", -180, 180, 5}, {"trackWidth", "边框宽度", "Track width", .5, 4, .1}, {"glowBlur", "辉光模糊", "Glow blur", 0, 12, .5},
	{"breathing.amplitude", "呼吸幅度", "Breathing amplitude", 0, .8, .05}, {"breathing.period", "呼吸周期", "Breathing period", 1, 20, .5}, {"hueCycle.range", "色相循环范围", "Hue cycle range", 0, 360, 5}, {"hueCycle.period", "色相循环周期", "Hue cycle period", 1, 60, .5}, {"night.dim", "夜间亮度降低", "Night dimming", 0, 100, 5},
}
var typingChoices = []effectChoice{
	{"effect", "回声动画", "Echo animation", []string{"drop", "rise", "flash"}}, {"cursor", "光标样式", "Caret", []string{"native", "block", "underline"}}, {"colors.kind", "配色来源", "Color source", []string{"theme", "preset", "custom"}}, {"colors.id", "配色方案", "Color scheme", []string{"ocean", "sunset", "candy", "forest", "violet", "ember"}},
}

func effectDefaults(typing bool) map[string]any {
	raw := `{"direction":"clockwise","period":1.96,"intensity":100,"bloom":100,"hue":0,"mode":"legacy","palette":{"kind":"preset","id":"legacy"},"trackWidth":2,"glowBlur":8,"breathing":{"enabled":false,"amplitude":0.3,"period":6},"hueCycle":{"enabled":true,"range":30,"period":12},"night":{"enabled":false,"fromMinute":1320,"toMinute":360,"dim":40},"easing":"linear"}`
	if typing {
		raw = `{"effect":"drop","cursor":"native","cursorBlink":true,"speed":100,"colors":{"kind":"theme"}}`
	}
	result := map[string]any{}
	_ = json.Unmarshal([]byte(raw), &result)
	return result
}
func effectValue(style map[string]any, path string) any {
	parts := strings.Split(path, ".")
	var value any = style
	for _, key := range parts {
		obj, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		value = obj[key]
	}
	return value
}
func effectSet(style map[string]any, path string, value any) {
	parts := strings.Split(path, ".")
	target := style
	for _, key := range parts[:len(parts)-1] {
		next, ok := target[key].(map[string]any)
		if !ok {
			next = map[string]any{}
			target[key] = next
		}
		target = next
	}
	target[parts[len(parts)-1]] = value
}
func copyEffect(style map[string]any) map[string]any {
	raw, _ := json.Marshal(style)
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out
}
func effectCore(typing bool) string {
	if typing {
		return "dsh-typing-fx"
	}
	return "dsh-composer-beam"
}
func effectPresetName(name string) bool {
	return name != "" && name == strings.TrimSpace(name) && len(utf16.Encode([]rune(name))) <= 32 && name != "__proto__" && name != "constructor" && name != "prototype"
}
func effectHex(value any) bool {
	s, ok := value.(string)
	if !ok || len(s) != 7 || s[0] != '#' {
		return false
	}
	for _, r := range s[1:] {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return false
		}
	}
	return true
}
func normalizeEffect(style map[string]any, typing bool) (map[string]any, error) {
	out := effectDefaults(typing)
	choices, numbers := beamChoices, beamNumbers
	bools := []string{"breathing.enabled", "hueCycle.enabled", "night.enabled"}
	if typing {
		choices = typingChoices
		numbers = []effectNumber{{"speed", "速度", "Speed", 40, 240, 10}}
		bools = []string{"cursorBlink"}
	}
	for _, choice := range choices {
		value := effectValue(style, choice.path)
		if value == nil {
			continue
		}
		text, ok := value.(string)
		valid := false
		for _, candidate := range choice.choices {
			if text == candidate {
				valid = true
			}
		}
		if !ok || !valid {
			return nil, fmt.Errorf("%s: %s", providerCopy(choice.zh, choice.en), providerCopy("选项无效", "invalid option"))
		}
		effectSet(out, choice.path, text)
	}
	for _, number := range numbers {
		value := effectValue(style, number.path)
		if value == nil {
			continue
		}
		n, ok := value.(float64)
		if !ok || math.IsNaN(n) || math.IsInf(n, 0) {
			return nil, fmt.Errorf("%s: %s", providerCopy(number.zh, number.en), providerCopy("数值无效", "invalid number"))
		}
		effectSet(out, number.path, max(number.lo, min(number.hi, n)))
	}
	for _, key := range bools {
		value := effectValue(style, key)
		if value == nil {
			continue
		}
		checked, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("%s: invalid switch", key)
		}
		effectSet(out, key, checked)
	}
	if typing {
		kind, _ := effectValue(out, "colors.kind").(string)
		if kind == "preset" && effectValue(out, "colors.id") == nil {
			effectSet(out, "colors.id", "ocean")
		}
		if kind == "custom" {
			for _, key := range []string{"echo", "caret", "text"} {
				value := effectValue(style, "colors."+key)
				if !effectHex(value) {
					return nil, fmt.Errorf("%s: #RRGGBB", key)
				}
				effectSet(out, "colors."+key, value)
			}
		}
	} else {
		if effectValue(out, "palette.kind") == "custom" {
			colors, ok := effectValue(style, "palette.colors").([]any)
			if !ok || len(colors) < 2 || len(colors) > 6 {
				return nil, fmt.Errorf("%s", providerCopy("自定义配色需要 2–6 个颜色。", "Custom palettes require 2–6 colors."))
			}
			for _, color := range colors {
				if !effectHex(color) {
					return nil, fmt.Errorf("#RRGGBB")
				}
			}
			effectSet(out, "palette", map[string]any{"kind": "custom", "colors": colors})
		}
		for _, key := range []string{"night.fromMinute", "night.toMinute"} {
			if value := effectValue(style, key); value != nil {
				n, ok := value.(float64)
				if !ok {
					return nil, fmt.Errorf("invalid time")
				}
				effectSet(out, key, math.Round(max(0, min(1439, n))))
			}
		}
	}
	return out, nil
}
func effectEnvelope(style map[string]any, presets map[string]map[string]any, typing bool) (string, error) {
	normalized, err := normalizeEffect(style, typing)
	if err != nil {
		return "", err
	}
	if len(presets) > 5 {
		return "", fmt.Errorf("%s", providerCopy("最多保存 5 个预设。", "Save up to 5 presets."))
	}
	clean := map[string]map[string]any{}
	for name, value := range presets {
		if !effectPresetName(name) {
			return "", fmt.Errorf("%s", providerCopy("预设名称需为 1–32 字符。", "Preset names must contain 1–32 characters."))
		}
		normalizedPreset, err := normalizeEffect(value, typing)
		if err != nil {
			return "", err
		}
		clean[name] = normalizedPreset
	}
	raw, err := json.MarshalIndent(map[string]any{"core": effectCore(typing), "version": 1, "style": normalized, "presets": clean}, "", "  ")
	return string(raw), err
}
func parseEffectEnvelope(text string, typing bool) (effectDraft, error) {
	limit := 64 * 1024
	if typing {
		limit = 32 * 1024
	}
	if len(text) > limit {
		return effectDraft{}, fmt.Errorf("%s", providerCopy("配置超过大小限制。", "Configuration exceeds the size limit."))
	}
	var payload struct {
		Core    string                    `json:"core"`
		Version int                       `json:"version"`
		Style   map[string]any            `json:"style"`
		Presets map[string]map[string]any `json:"presets"`
	}
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return effectDraft{}, fmt.Errorf("%s", providerCopy("配置 JSON 无效。", "Invalid configuration JSON."))
	}
	if payload.Core != effectCore(typing) || payload.Version != 1 || payload.Style == nil {
		return effectDraft{}, fmt.Errorf("%s", providerCopy("不支持的配置类型或版本。", "Unsupported configuration type or version."))
	}
	if _, err := effectEnvelope(payload.Style, payload.Presets, typing); err != nil {
		return effectDraft{}, err
	}
	style, _ := normalizeEffect(payload.Style, typing)
	presets := map[string]map[string]any{}
	for name, value := range payload.Presets {
		presets[name], _ = normalizeEffect(value, typing)
	}
	return effectDraft{Style: style, Presets: presets, CustomTyping: typingCustomSeed(style)}, nil
}

func effectJSONEqual(left, right string) bool {
	var a, b any
	return json.Unmarshal([]byte(left), &a) == nil && json.Unmarshal([]byte(right), &b) == nil && reflect.DeepEqual(a, b)
}

func acknowledgeEffect(a *BodySettingsAccess, draft *effectDraft) bool {
	if len(draft.Pending) == 0 || a.Snapshot.Saving {
		return false
	}
	if a.Snapshot.Err != "" {
		draft.Err = a.Snapshot.Err
		draft.Pending = nil
		return false
	}
	for key, submitted := range draft.Pending {
		_, value, found := bodyHostValue(a, "ui-conversation", key)
		if !found || !effectJSONEqual(value, submitted) {
			return false
		}
	}
	draft.Pending = nil
	return true
}

func bodyEffectPreference(c *ui.Context, a *BodySettingsAccess, typing bool, fg, muted ui.Color) {
	enabledKey, styleKey, presetsKey := "composerBeam", "composerBeamStyle", "composerBeamPresets"
	title := providerCopy("发送消息时的思考炫光", "Thinking glow when sending")
	if typing {
		enabledKey, styleKey, presetsKey = "typingFx", "typingFxStyle", "typingFxPresets"
		title = providerCopy("输入特效", "Typing effects")
	}
	_, enabled, found := bodyHostValue(a, "ui-conversation", enabledKey)
	if !found {
		return
	}
	_, styleValue, styleFound := bodyHostValue(a, "ui-conversation", styleKey)
	_, presetsValue, presetsFound := bodyHostValue(a, "ui-conversation", presetsKey)
	anchor := ui.Column(c.Key(fmt.Sprintf("effect-settings-%d-%s", a.Owner, enabledKey))).FillWidth().MinWidth(0)
	open := ui.Local(anchor, "open", func() bool { return false })
	draft := ui.Local(anchor, "draft", func() effectDraft { return effectDraft{} })
	openSettings := func() {
		style := effectDefaults(typing)
		_ = json.Unmarshal([]byte(styleValue), &style)
		if style == nil {
			style = effectDefaults(typing)
		}
		normalized, normalizeErr := normalizeEffect(style, typing)
		if normalizeErr != nil {
			normalized = effectDefaults(typing)
		}
		presets := map[string]map[string]any{}
		_ = json.Unmarshal([]byte(presetsValue), &presets)
		if presets == nil {
			presets = map[string]map[string]any{}
		}
		*draft = effectDraft{Style: normalized, Presets: presets, CustomTyping: typingCustomSeed(normalized)}
		if normalizeErr != nil {
			draft.Err = normalizeErr.Error()
		}
		*open = true
	}
	anchor.Children(func() {
		gearLabel := providerCopy("思考炫光设置", "Thinking glow settings")
		if typing {
			gearLabel = providerCopy("打字特效设置", "Typing effect settings")
		}
		// This row carries the 16×16 gear that opens the effect dialog; no other
		// preference row asks for one.
		withRowGear(rowGear{
			label:    gearLabel,
			hint:     providerCopy("打开输入框光效与指针特效设置", "Open the composer beam and pointer effect settings"),
			disabled: bodyHostDisabled(a) || !styleFound,
			onOpen:   openSettings,
		}, func() {
			hint := providerCopy("智能体发送或思考时，在输入条周围播放彩色边光。", "Play a colored edge glow around the composer while the agent sends or thinks.")
			if typing {
				hint = providerCopy("点击齿轮调整光效样式；开关控制是否启用。", "Use the cog to tune the effect; the switch only enables it.")
			}
			bodyPreferenceRow(c, "effect-row-"+enabledKey, title, hint, fg, muted, func() {
				ui.Row(c).Wrap().Gap(8).Justify(ui.End).Children(func() {
					checked := enabled == "true"
					if settingsPreferenceSwitch(c, &checked, settingsSwitchInput).Label(title).Disabled(bodyHostDisabled(a)).Changed() {
						a.OnChange("ui-conversation", enabledKey, fmt.Sprint(checked))
					}
				})
			})
		})
	})
	if !*open {
		return
	}
	if acknowledgeEffect(a, draft) {
		*open = false
		return
	}
	ui.DialogBase(c, open, func(backdrop, panel ui.Element) {
		vw, vh := c.Size()
		// The effect editor is a feature dialog on the shared modal surface, like
		// the reference settings dialogs: the same fade, glass, 28 DIP radius and
		// 0.5 DIP outer hairline as commit, permission and settings editors.
		chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), min(float32(620), max(float32(0), vw-48)), 0).
			Height(min(float32(760), max(float32(0), vh*.85))).Children(func() {
			ui.Column(c).Fill().MinWidth(0).MinHeight(0).Children(func() {
				ui.Row(c).FillWidth().Padding(16, 20).Wrap().Gap(8).Children(func() {
					ui.Text(c, title).Grow(1).MinWidth(0).FontSize(theme.InterfaceSize(16)).Bold().TextColor(fg)
					if ui.Button(c, providerCopy("关闭", "Close")).Clicked() {
						*open = false
					}
				})
				ui.Scroll(c).Grow(1).FillWidth().MinHeight(0).MinWidth(0).Padding(0, 24, 24, 24).Gap(12).Children(func() {
					previewStyle, previewErr := normalizeEffect(draft.Style, typing)
					if previewErr != nil {
						previewStyle = effectDefaults(typing)
					}
					preview, _ := json.Marshal(previewStyle)
					composer.SettingsEffectPreview(c, preview, typing)
					renderEffectControls(c, draft, typing, bodyHostDisabled(a) || len(draft.Pending) > 0, fg, muted)
					bodyHeading(c, providerCopy("个人预设", "Personal presets"), providerCopy("最多 5 个预设，名称不超过 32 字符。", "Up to 5 presets, with names of at most 32 characters."), fg, muted)
					names := []string{""}
					for name := range draft.Presets {
						names = append(names, name)
					}
					sort.Strings(names)
					if ui.Select(c, &draft.Selected, names).FillWidth().Label("effect-preset-selection").Disabled(bodyHostDisabled(a) || len(draft.Pending) > 0).Changed() {
						if value, ok := draft.Presets[draft.Selected]; ok {
							if normalized, err := normalizeEffect(value, typing); err != nil {
								draft.Err = err.Error()
							} else {
								draft.Style = normalized
								draft.Err = ""
							}
						}
					}
					ui.TextInput(c, &draft.Name).FillWidth().Label("effect-preset-name").Disabled(bodyHostDisabled(a) || len(draft.Pending) > 0).Placeholder(providerCopy("预设名称", "Preset name"))
					ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
						if ui.Button(c, providerCopy("保存预设", "Save preset")).Label("effect-save-preset").Disabled(bodyHostDisabled(a) || len(draft.Pending) > 0 || !presetsFound || a.OnPatch == nil).Clicked() {
							name := strings.TrimSpace(draft.Name)
							units := utf16.Encode([]rune(name))
							if len(units) > 32 {
								name = string(utf16.Decode(units[:32]))
							}
							if !effectPresetName(name) {
								draft.Err = providerCopy("预设名称需为 1–32 字符。", "Preset names must contain 1–32 characters.")
							} else if _, exists := draft.Presets[name]; !exists && len(draft.Presets) >= 5 {
								draft.Err = providerCopy("最多保存 5 个预设。", "Save up to 5 presets.")
							} else if normalized, err := normalizeEffect(draft.Style, typing); err != nil {
								draft.Err = err.Error()
							} else {
								draft.Presets[name] = normalized
								draft.Selected = name
								draft.Name = ""
								draft.Err = ""
							}
						}
						if ui.Button(c, providerCopy("删除预设", "Delete preset")).Disabled(bodyHostDisabled(a) || len(draft.Pending) > 0 || draft.Selected == "" || !presetsFound || a.OnPatch == nil).Clicked() {
							delete(draft.Presets, draft.Selected)
							draft.Selected = ""
						}
					})
					ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
						if ui.Button(c, providerCopy("复制配置 JSON", "Copy configuration JSON")).Clicked() {
							if text, err := effectEnvelope(draft.Style, draft.Presets, typing); err != nil {
								draft.Err = err.Error()
							} else {
								c.WriteClipboard(text)
								draft.Err = ""
							}
						}
						if ui.Button(c, providerCopy("从剪贴板导入", "Import from clipboard")).Disabled(bodyHostDisabled(a) || len(draft.Pending) > 0).Clicked() {
							if imported, err := parseEffectEnvelope(c.ReadClipboard(), typing); err != nil {
								draft.Err = err.Error()
							} else {
								*draft = imported
							}
						}
						if ui.Button(c, providerCopy("恢复默认", "Reset")).Disabled(bodyHostDisabled(a) || len(draft.Pending) > 0).Clicked() {
							draft.Style = effectDefaults(typing)
							draft.Selected = ""
							draft.Err = ""
						}
					})
				})
				if draft.Err != "" || len(draft.Pending) > 0 {
					ui.Column(c).FillWidth().Padding(8, 24).Children(func() {
						if draft.Err != "" {
							ui.Text(c, draft.Err).FillWidth().TextColor(fg)
						}
						if len(draft.Pending) > 0 {
							ui.Text(c, providerCopy("正在保存，确认后应用。", "Saving; waiting for confirmation.")).FillWidth().TextColor(muted)
						}
					})
				}
				ui.Row(c).FillWidth().Padding(12, 24).Wrap().Gap(8).Children(func() {
					if ui.Button(c, providerCopy("取消", "Cancel")).Disabled(len(draft.Pending) > 0).Clicked() {
						*open = false
					}
					if ui.Button(c, providerCopy("保存并应用", "Save and apply")).Label("save-" + enabledKey + "-configuration").Disabled(bodyHostDisabled(a) || len(draft.Pending) > 0).Clicked() {
						normalized, err := normalizeEffect(draft.Style, typing)
						if err != nil {
							draft.Err = err.Error()
							return
						}
						style, _ := json.Marshal(normalized)
						draft.Err = ""
						draft.Pending = map[string]string{styleKey: string(style)}
						if presetsFound && a.OnPatch != nil {
							if _, err := effectEnvelope(normalized, draft.Presets, typing); err != nil {
								draft.Err = err.Error()
								draft.Pending = nil
								return
							}
							presets, _ := json.Marshal(draft.Presets)
							draft.Pending[presetsKey] = string(presets)
							a.OnPatch("ui-conversation", draft.Pending)
						} else {
							a.OnChange("ui-conversation", styleKey, string(style))
						}
					}
				})
			})
		})
		chrome.DialogDecorTheme(c, backdrop, panel, c.Theme(), 28)
	})
}

func applyBeamMode(style map[string]any, mode string) {
	if mode == "legacy" {
		next := effectDefaults(false)
		for key := range style {
			delete(style, key)
		}
		for key, value := range next {
			style[key] = value
		}
		return
	}
	style["mode"] = mode
	switch mode {
	case "lounge":
		effectSet(style, "palette", map[string]any{"kind": "preset", "id": "legacy"})
		style["period"], style["intensity"], style["bloom"], style["easing"] = float64(10), float64(70), float64(70), "ease-in-out"
		effectSet(style, "breathing.enabled", false)
		effectSet(style, "hueCycle.enabled", false)
	case "aurora":
		effectSet(style, "palette", map[string]any{"kind": "preset", "id": "aurora"})
		style["period"], style["easing"] = float64(10), "cubic-bezier"
		effectSet(style, "hueCycle", map[string]any{"enabled": true, "range": float64(90), "period": float64(16)})
		effectSet(style, "breathing", map[string]any{"enabled": true, "amplitude": .18, "period": float64(8)})
	case "reactive":
		effectSet(style, "palette", map[string]any{"kind": "preset", "id": "cyber"})
		style["period"], style["easing"] = float64(22), "ease-in-out"
		effectSet(style, "hueCycle.enabled", false)
		effectSet(style, "breathing", map[string]any{"enabled": true, "amplitude": .3, "period": float64(6)})
	}
}
func renderEffectControls(c *ui.Context, d *effectDraft, typing, blocked bool, fg, muted ui.Color) {
	choices, numbers := beamChoices, beamNumbers
	if typing {
		choices = typingChoices
		numbers = []effectNumber{{"speed", "速度", "Speed", 40, 240, 10}}
	}
	for _, choice := range choices {
		if choice.path == "palette.id" && effectValue(d.Style, "palette.kind") != "preset" {
			continue
		}
		if choice.path == "colors.id" && effectValue(d.Style, "colors.kind") != "preset" {
			continue
		}
		value, _ := effectValue(d.Style, choice.path).(string)
		if value == "" && len(choice.choices) > 0 {
			value = choice.choices[0]
		}
		bodyPreferenceRow(c, "effect-choice-"+choice.path, providerCopy(choice.zh, choice.en), "", fg, muted, func() {
			if ui.Select(c, &value, choice.choices).FillWidth().Label("effect-" + choice.path).Disabled(blocked).Changed() {
				var palette []string
				if choice.path == "palette.kind" && value == "custom" {
					raw, _ := json.Marshal(d.Style)
					palette = composer.BeamPaletteColors(raw)
				}
				effectSet(d.Style, choice.path, value)
				if !typing && choice.path != "mode" {
					d.Style["mode"] = "custom"
				}
				d.Err = ""
				if !typing && choice.path == "mode" {
					applyBeamMode(d.Style, value)
				}
				if choice.path == "palette.kind" && value == "custom" {
					colors := make([]any, 0, min(6, len(palette)))
					for _, color := range palette[:min(6, len(palette))] {
						colors = append(colors, color)
					}
					effectSet(d.Style, "palette.colors", colors)
				}
				if choice.path == "colors.kind" && value == "custom" {
					if d.CustomTyping == ([3]string{}) {
						d.CustomTyping = typingCustomSeed(d.Style)
					}
					for i, key := range []string{"echo", "caret", "text"} {
						effectSet(d.Style, "colors."+key, d.CustomTyping[i])
					}
				}
				if choice.path == "colors.kind" && value == "preset" && effectValue(d.Style, "colors.id") == nil {
					effectSet(d.Style, "colors.id", "ocean")
				}
			}
		})
	}
	for _, toggle := range []struct{ path, zh, en string }{{"cursorBlink", "光标闪烁", "Blink caret"}, {"breathing.enabled", "呼吸效果", "Breathing"}, {"hueCycle.enabled", "色相循环", "Hue cycling"}, {"night.enabled", "夜间调暗", "Night dimming"}} {
		if typing != (toggle.path == "cursorBlink") {
			continue
		}
		checked, _ := effectValue(d.Style, toggle.path).(bool)
		bodyLocalToggle(c, "effect-toggle-"+toggle.path, providerCopy(toggle.zh, toggle.en), "", &checked, blocked || typing && effectValue(d.Style, "cursor") == "native", fg, muted, func(value bool) {
			effectSet(d.Style, toggle.path, value)
			if !typing {
				d.Style["mode"] = "custom"
			}
		})
	}
	if !typing {
		numbers = append(append([]effectNumber(nil), numbers...), effectNumber{"night.fromMinute", "夜间开始", "Night starts", 0, 1439, 1}, effectNumber{"night.toMinute", "夜间结束", "Night ends", 0, 1439, 1})
	}
	for _, number := range numbers {
		value, _ := effectValue(d.Style, number.path).(float64)
		disabled := strings.HasPrefix(number.path, "breathing.") && effectValue(d.Style, "breathing.enabled") != true || strings.HasPrefix(number.path, "hueCycle.") && effectValue(d.Style, "hueCycle.enabled") != true || strings.HasPrefix(number.path, "night.") && effectValue(d.Style, "night.enabled") != true
		bodyPreferenceRow(c, "effect-number-"+number.path, providerCopy(number.zh, number.en), "", fg, muted, func() {
			label := fmt.Sprintf("%.2f", value)
			if number.path == "night.fromMinute" || number.path == "night.toMinute" {
				label = fmt.Sprintf("%02d:%02d", int(value)/60, int(value)%60)
			}
			ui.Text(c, label).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			if ui.StepSlider(c, &value, number.lo, number.hi, number.step).FillWidth().Label("effect-" + number.path).Disabled(blocked || disabled).Changed() {
				effectSet(d.Style, number.path, value)
				if !typing {
					d.Style["mode"] = "custom"
				}
				d.Err = ""
			}
		})
	}
	if typing && effectValue(d.Style, "colors.kind") == "custom" {
		for _, color := range []struct{ key, zh, en string }{{"echo", "回声颜色", "Echo color"}, {"caret", "光标颜色", "Caret color"}, {"text", "文字颜色", "Text color"}} {
			value, _ := effectValue(d.Style, "colors."+color.key).(string)
			bodyPreferenceRow(c, "typing-color-"+color.key, providerCopy(color.zh, color.en), "#RRGGBB", fg, muted, func() {
				if ui.TextInput(c, &value).FillWidth().Label("effect-color-" + color.key).Disabled(blocked).Changed() {
					effectSet(d.Style, "colors."+color.key, value)
					for i, key := range []string{"echo", "caret", "text"} {
						if key == color.key {
							d.CustomTyping[i] = value
						}
					}
					d.Err = ""
				}
			})
		}
	}
	if !typing && effectValue(d.Style, "palette.kind") == "custom" {
		colors, _ := effectValue(d.Style, "palette.colors").([]any)
		for index, color := range colors {
			value, _ := color.(string)
			bodyPreferenceRow(c, fmt.Sprintf("beam-color-%d", index), fmt.Sprintf(providerCopy("颜色 %d", "Color %d"), index+1), "#RRGGBB", fg, muted, func() {
				if ui.TextInput(c, &value).FillWidth().Label(fmt.Sprintf("effect-color-%d", index)).Disabled(blocked).Changed() {
					colors[index] = value
					effectSet(d.Style, "palette.colors", colors)
					d.Style["mode"] = "custom"
					d.Err = ""
				}
				if ui.Button(c, providerCopy("移除颜色", "Remove color")).Disabled(blocked || len(colors) <= 2).Clicked() {
					effectSet(d.Style, "palette.colors", append(colors[:index], colors[index+1:]...))
					d.Style["mode"] = "custom"
				}
			})
		}
		if ui.Button(c, providerCopy("添加颜色", "Add color")).Disabled(blocked || len(colors) >= 6).Clicked() {
			effectSet(d.Style, "palette.colors", append(colors, "#ffffff"))
			d.Style["mode"] = "custom"
		}
	}
}
