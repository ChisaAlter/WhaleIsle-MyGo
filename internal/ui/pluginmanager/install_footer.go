package pluginmanager

import (
	"strings"

	"github.com/egoist/mygo/ui"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
)

// installSafetyCopy is the two lines of trust guidance: who the plugin runs
// as, and that installed plugins do not update themselves yet.
func installSafetyCopy() (safety, upgrade string) {
	return copyText("请确认插件来源可信。插件在本机以你的权限运行，来源不明的插件可能损坏 鲸屿 MyGo，或读取和泄露你的数据。",
			"Install only plugins you trust: they run with your permissions and can damage Whale Isle MyGo or leak your data."),
		copyText("插件安装后，暂不支持自动更新。若需升级，请先卸载再安装新版，后续版本会持续改善升级体验。",
			"Installed plugins do not update automatically yet. To upgrade a plugin, uninstall it and install the new version. Later releases will keep improving the upgrade experience.")
}

// installSafetyNote is the warning Alert both install dialogs end with.
func installSafetyNote(c *ui.Context) {
	safety, upgrade := installSafetyCopy()
	coss.Alert(c, safety, upgrade, coss.AlertOptions{Variant: coss.AlertWarning, Key: "install-safety"})
}

// installFooter is the spec screen's fixed footer, the reference's
// `.installFooter` on the shared dialog surface: the trust note and the
// full-width 40 DIP action stay put while the body above them scrolls. The
// button is `安装`, or while the spec is being checked it loads as
// 正在检查….
func installFooter(c *ui.Context, i manager.InstallState, a manager.Actions, busy bool) {
	checking := i.Phase == "checking"
	label := copyText("安装", "Install")
	if checking {
		label = copyText("正在检查…", "Checking…")
	}
	ui.Column(c).Grow(1).MinWidth(0).Gap(design.Sp(5)).Children(func() {
		installSafetyNote(c)
		if coss.Button(c, label, coss.ButtonOptions{Icon: coss.IconDownload, Size: coss.ButtonXL, Loading: checking,
			Disabled: busy || strings.TrimSpace(i.Spec) == ""}).FillWidth().Label(label).Clicked() {
			a.RetryInstall()
		}
	})
}
