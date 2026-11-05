package settings

import (
	"github.com/egoist/mygo/ui"
	"math"
	"whaleisle/internal/desktop/voicecapture"
	"whaleisle/internal/desktop/voiceinput"
	"whaleisle/internal/ui/i18n"
)

// VoiceInputDeviceView is the microphone row in the existing voice plugin
// settings. The application owns SetActive while that page/window is visible.
func VoiceInputDeviceView(c *ui.Context, microphone *voiceinput.Microphone) {
	if microphone == nil {
		return
	}
	state := microphone.Snapshot()
	text := func(zh, en string) string {
		if i18n.Locale() == "en" {
			return en
		}
		return zh
	}
	root := ui.Column(c.Key("voice-input-device")).FillWidth().Gap(6)
	choice := ui.Local(root, "selection", func() voiceDeviceChoice {
		return voiceDeviceChoice{Source: state.Selected.ID, Value: state.Selected.ID}
	})
	if choice.Source != state.Selected.ID {
		choice.Source, choice.Value = state.Selected.ID, state.Selected.ID
	}
	root.Children(func() {
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, text("麦克风", "Input device")).FontSize(13)
			selected := state.Selected.ID
			choices := append([]voicecapture.Device{{}}, state.Devices...)
			if state.Selected.ID != "" && state.Missing {
				choices = append(choices, state.Selected)
			}
			parts := ui.SelectBase(c, &choice.Value)
			caption := state.Selected.Label
			if selected == "" {
				caption = text("系统默认麦克风", "System microphone")
			}
			if state.Missing {
				caption += text(" — 不可用", " — unavailable")
			}
			parts.Trigger.Label(text("选择麦克风", "Select microphone")).Grow(1).MinWidth(0).Disabled(state.Loading).Children(func() {
				ui.Text(c, caption).Grow(1).MinWidth(0).SingleLine()
				ui.Text(c, "▾")
			})
			parts.Popup(func(panel ui.Element) {
				for _, device := range choices {
					label := device.Label
					if device.ID == "" {
						label = text("系统默认麦克风", "System microphone")
					}
					if label == "" {
						label = text("未命名麦克风", "Unnamed microphone")
					}
					parts.Item(device.ID).Label(label).Disabled(state.Missing && device.ID == state.Selected.ID).Children(func() { ui.Text(c, label) })
				}
			})
			if parts.Trigger.Changed() {
				for _, device := range choices {
					if device.ID == choice.Value {
						microphone.Select(device)
						break
					}
				}
			}
			if ui.Button(c, text("刷新", "Refresh")).Disabled(state.Loading).Clicked() {
				microphone.Refresh()
			}
		})
		if state.Active && !state.Missing && !state.Loading && state.Error == "" {
			ui.Row(c).Gap(3).Label(text("麦克风输入电平", "Microphone input level")).Children(func() {
				bars := int(math.Ceil(state.Level * 8))
				for i := 0; i < 8; i++ {
					color := c.Theme().Border
					if i < bars {
						color = c.Theme().Accent
					}
					ui.Box(c).Width(8).Height(12).Radius(2).Background(color)
				}
			})
			c.Services().Invalidate()
		}
		settingsNotice(c, "voice_input:89", "error", state.Error != "", func() string { return state.Error })
	})
}

type voiceDeviceChoice struct{ Source, Value string }
