package settings

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	fonttheme "whaleisle/internal/ui/theme"
)

func harnessRestartPane(c *ui.Context, d *DesktopAccess, fg, muted ui.Color) {
	f := d.Fields
	if f.HarnessAutoRestart == nil || f.HarnessRestartMaxAttempts == nil || f.HarnessRestartBaseDelayMs == nil {
		return
	}
	ui.Column(c).Key("harness-restart-policy").Gap(6).Children(func() {
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
