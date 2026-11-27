package settings

import (
	"math"
	"time"
	"whaleisle/internal/desktop/voicecapture"
	"whaleisle/internal/desktop/voiceinput"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo/ui"
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
			choices := append([]voicecapture.Device{{}}, state.Devices...)
			if state.Selected.ID != "" && state.Missing {
				choices = append(choices, state.Selected)
			}
			opts := make([]coss.SelectOption, 0, len(choices))
			for _, device := range choices {
				label := device.Label
				if device.ID == "" {
					label = text("系统默认麦克风", "System microphone")
				}
				if label == "" {
					label = text("未命名麦克风", "Unnamed microphone")
				}
				if state.Missing && device.ID == state.Selected.ID {
					label += text(" — 不可用", " — unavailable")
				}
				opts = append(opts, coss.SelectOption{
					Value:    device.ID,
					Label:    label,
					Disabled: state.Missing && device.ID == state.Selected.ID,
				})
			}
			changed := false
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				changed = coss.Select(c, &choice.Value, opts, coss.SelectOptions{
					Label:    text("选择麦克风", "Select microphone"),
					Disabled: state.Loading,
				}).FillWidth().Changed()
			})
			if changed {
				for _, device := range choices {
					if device.ID == choice.Value {
						microphone.Select(device)
						break
					}
				}
			}
			if kButtonDisabled(c, text("刷新", "Refresh"), state.Loading).Clicked() {
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
			c.After(33 * time.Millisecond)
		}
		settingsAlert(c, "voice_input:89", state.Error != "", providerCopy("没能保存语音输入设置，请重试。", "The voice input setting was not saved. Try again."), state.Error)
	})
}

type voiceDeviceChoice struct{ Source, Value string }
