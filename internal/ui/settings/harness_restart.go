package settings

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strconv"
	"whaleisle/internal/config"
	fonttheme "whaleisle/internal/ui/theme"
)

func harnessRestartPane(c *ui.Context, d *DesktopAccess, fg, muted ui.Color) {
	f := d.Fields
	if f.HarnessAutoRestart == nil || f.HarnessRestartMaxAttempts == nil || f.HarnessRestartBaseDelayMs == nil {
		return
	}
	block := ui.Column(c).Key("harness-restart-policy").Gap(6)
	settingsSearchTarget(c, "harness-restart-policy", providerCopy("内核崩溃自动重启", "Harness crash auto-restart"), block)
	block.Children(func() {
		ui.Text(c, providerCopy("内核崩溃自动重启", "Harness crash auto-restart")).Bold().FontSize(fonttheme.InterfaceSize(14)).TextColor(fg)
		ui.Text(c, providerCopy("保存仅影响后续新重试决策，不会停启当前内核；已排定的重试计时器保持原决定。", "Saving affects only future retry decisions. It does not stop or start the current Harness; already scheduled retry timers keep their original decision.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		ui.Text(c, providerCopy("插件恢复和手动重启不受此开关控制。每次重试延迟翻倍；内核就绪后重试次数重新计数。", "Plugin recovery and manual restart are separate. Retry delays double each time; the attempt count resets when Harness becomes ready.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if d.RestartPolicy != nil {
			p := *d.RestartPolicy
			enabled := providerCopy("关闭", "off")
			if p.AutoRestart {
				enabled = providerCopy("开启", "on")
			}
			ui.Text(c, fmt.Sprintf(providerCopy("已保存的后续策略：%s；最多 %d 次；基础延迟 %d ms", "Saved policy for future retries: %s; up to %d attempts; base delay %d ms"), enabled, p.MaxAttempts, p.BaseDelayMs)).Label("harness-restart-saved").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			draft, err := config.ParseRestartPreferences(*f.HarnessAutoRestart, *f.HarnessRestartMaxAttempts, *f.HarnessRestartBaseDelayMs)
			if err != nil || draft != p {
				ui.Text(c, providerCopy("有未保存的重试策略草稿。", "Unsaved retry policy draft.")).Label("harness-restart-dirty").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			}
		} else {
			ui.Text(c, providerCopy("本地配置不可用，无法保存重试策略。", "Local configuration is unavailable; retry policy cannot be saved.")).TextColor(muted)
		}
		ui.Checkbox(c, f.HarnessAutoRestart, providerCopy("允许崩溃后自动重启", "Restart automatically after a crash")).Label("harness-restart-enabled").Disabled(d.RestartPolicy == nil)
		desktopRow(c, providerCopy("最多重试次数（1–10）", "Maximum retry attempts (1–10)"), func() {
			ui.TextInput(c, f.HarnessRestartMaxAttempts).Label("harness-restart-attempts").FillWidth().Disabled(d.RestartPolicy == nil)
		}, fg, muted)
		desktopRow(c, providerCopy("基础延迟（500–30000 毫秒）", "Base delay (500–30000 ms)"), func() {
			ui.TextInput(c, f.HarnessRestartBaseDelayMs).Label("harness-restart-delay").FillWidth().Disabled(d.RestartPolicy == nil)
		}, fg, muted)
	})
}

// restartOption is one preset of a recovery picker: the visible label and the
// stored draft value.
type restartOption struct{ label, value string }

// harnessAutoRecoveryRow is the 0.3.13 "Harness 自动恢复" row: copy on the left,
// the enable switch above two labelled 88 DIP pickers on the right. Every
// change writes its own committed field; the verbose diagnostic form keeps its
// separate explicit Save operation.
func harnessAutoRecoveryRow(c *ui.Context, d *DesktopAccess, fg, muted ui.Color) {
	f := d.Fields
	if f.HarnessAutoRestart == nil || f.HarnessRestartMaxAttempts == nil || f.HarnessRestartBaseDelayMs == nil {
		return
	}
	if !settingsItemBefore(c, "harness-auto-recovery") {
		return
	}
	disabled := d.RestartPolicy == nil || d.OnSetRestartPreference == nil
	anchor := ui.Column(c.Key("harness-auto-recovery")).FillWidth().MinWidth(0).Gap(0)
	settingsSearchTarget(c, "harness-auto-recovery", providerCopy("Harness 自动恢复", "Harness auto-recovery"), anchor)
	failure := ui.Local(anchor, "save-error", func() string { return "" })
	save := func(key, value string) {
		accepted, err := d.OnSetRestartPreference(key, value)
		*f.HarnessAutoRestart = accepted.AutoRestart
		*f.HarnessRestartMaxAttempts = strconv.Itoa(accepted.MaxAttempts)
		*f.HarnessRestartBaseDelayMs = strconv.Itoa(accepted.BaseDelayMs)
		if d.RestartPolicy != nil {
			*d.RestartPolicy = accepted
		}
		*failure = ""
		if err != nil {
			*failure = err.Error()
			c.AddToast(ui.Toast{ID: "harness-recovery-save", Title: *failure, Type: "error"})
		}
	}
	seconds := providerCopy(" 秒", " s")
	attempts := make([]restartOption, 0, 3)
	for _, n := range []int{1, 3, 5} {
		attempts = append(attempts, restartOption{strconv.Itoa(n), strconv.Itoa(n)})
	}
	delays := []restartOption{{"1" + seconds, "1000"}, {"2" + seconds, "2000"}, {"5" + seconds, "5000"}}
	title := providerCopy("Harness 自动恢复", "Harness auto-recovery")
	anchor.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).Padding(16, 0).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Padding(0, 48, 0, 0).Children(func() {
				ui.Text(c, title).FillWidth().FontSize(fonttheme.InterfaceSize(14)).FixedLineHeight(fonttheme.InterfaceSize(22)).TextColor(fg)
				ui.Text(c, providerCopy("Harness 进程意外退出后自动重启，最多重试有限次数，重试之间按基础延迟退避等待。", "Restart the Harness automatically after an unexpected exit, with a bounded number of retries, backing off by the base delay between them.")).FillWidth().FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).TextColor(muted)
			})
			ui.Column(c).Shrink(0).Gap(12).AlignItems(ui.End).Children(func() {
				if settingsPreferenceSwitch(c.Key("harness-auto-recovery-switch"), f.HarnessAutoRestart, settingsSwitchInput).Label(providerCopy("启用自动恢复", "Enable auto-recovery")).Disabled(disabled).Changed() {
					save("harnessAutoRestart", strconv.FormatBool(*f.HarnessAutoRestart))
				}
				ui.Row(c).Gap(8).Children(func() {
					restartPicker(c, "harness-restart-attempts-picker", providerCopy("最大重启次数", "Max restarts"), f.HarnessRestartMaxAttempts, attempts, disabled, func() { save("harnessRestartMaxAttempts", *f.HarnessRestartMaxAttempts) }, muted)
					restartPicker(c, "harness-restart-delay-picker", providerCopy("基础重启延迟", "Base delay"), f.HarnessRestartBaseDelayMs, delays, disabled, func() { save("harnessRestartBaseDelayMs", *f.HarnessRestartBaseDelayMs) }, muted)
				})
			})
		})
	})
}

func restartPicker(c *ui.Context, key, label string, value *string, options []restartOption, disabled bool, changed func(), muted ui.Color) {
	labels := make([]string, 0, len(options)+1)
	display := ""
	for _, option := range options {
		labels = append(labels, option.label)
		if option.value == *value {
			display = option.label
		}
	}
	if display == "" {
		display = *value
		labels = append([]string{display}, labels...)
	}
	row := ui.Column(c.Key(key)).Gap(4).AlignItems(ui.End)
	settingsSearchTarget(c, key, label, row)
	row.Children(func() {
		ui.Text(c, label).FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).TextColor(muted)
		if settingsPreferenceSelect(c, &display, labels).Label(label).Width(88).Disabled(disabled).Changed() {
			for _, option := range options {
				if option.label == display {
					*value = option.value
					changed()
					break
				}
			}
		}
	})
}
