package pluginmanager

import (
	"strings"

	"github.com/egoist/mygo/ui"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/theme"
)

func registryName(registry, resolved *string) string {
	key := manager.RegistryKey(registry, resolved)
	switch key {
	case manager.OfficialNPMRegistry:
		return copyText("npm 官方源", "npm official registry")
	case manager.NPMMirrorRegistry:
		return copyText("中国大陆镜像源", "Mainland China mirror")
	}
	if registry == nil {
		if resolved != nil {
			return *resolved
		}
		return copyText("pnpm 配置的源", "pnpm configured registry")
	}
	return *registry
}

// Keep registry choices in an anchored popover: expanding them does not resize
// the install form, and Escape first dismisses this inner editing scope.
func installRegistryOptions(c *ui.Context, i manager.InstallState, a manager.Actions, disabled bool) {
	var resolved *string
	if i.RegistryInfo != nil {
		resolved = i.RegistryInfo.Resolved
	}
	name := registryName(i.RegistrySelection.Registry, resolved)
	if i.RegistrySelection.Kind == "custom" {
		name = copyText("自定义地址", "Custom address")
	}
	anchor := ui.Button(c, copyText("安装源：", "Registry: ")+name).Disabled(disabled)
	draft := ui.Local(anchor, "custom-registry-draft", func() string { return "" })
	if i.RegistrySelection.Kind == "custom" {
		*draft = i.RegistrySelection.URL
	}
	if anchor.Clicked() {
		a.ToggleRegistryOptions()
	}
	open := i.RegistryOpen && i.Phase == "idle"
	shown := open
	w, _ := c.Size()
	ui.PopoverBase(c.Key("plugin-install-registry-options"), anchor, &open, func(panel ui.Element) {
		panel.Width(max(0, min(float32(420), w-64))).Padding(12).Gap(6).Role(ui.RoleDialog).Label(copyText("安装源", "Installation registry")).Background(c.Theme().Background).Border(.5, c.Theme().Border).Radius(8).Children(func() {
			for _, registry := range manager.OfferedRegistries(i.RegistryInfo) {
				selected := i.RegistrySelection.Kind != "custom" && (registry == nil && i.RegistrySelection.Registry == nil || registry != nil && i.RegistrySelection.Registry != nil && *registry == *i.RegistrySelection.Registry)
				label := registryName(registry, resolved)
				if registry != nil && label != *registry {
					label += " · " + *registry
				}
				if ui.Button(c, label).FillWidth().Justify(ui.Start).Role(ui.RoleRadio).Selected(selected).Clicked() {
					a.ChooseRegistry(manager.RegistryChoice{Kind: "offered", Registry: registry})
				}
			}
			if ui.Button(c, copyText("自定义地址", "Custom address")).FillWidth().Justify(ui.Start).Role(ui.RoleRadio).Selected(i.RegistrySelection.Kind == "custom").Clicked() {
				a.ChooseRegistry(manager.RegistryChoice{Kind: "custom", URL: *draft})
			}
			if ui.TextInput(c, draft).FillWidth().Label(copyText("自定义安装源 URL", "Custom registry URL")).Placeholder("https://npm.example.com/").Changed() {
				a.ChooseRegistry(manager.RegistryChoice{Kind: "custom", URL: *draft})
			}
			if i.RegistryError {
				ui.Text(c, copyText("请输入以 http:// 或 https:// 开头的地址", "Enter an address starting with http:// or https://")).FillWidth().Role(ui.RoleAlert).TextColor(c.Theme().Danger)
			}
			ui.Text(c, copyText("填写内网或私有 npm 源地址。需要登录时，请将凭据保存在本机 ~/.npmrc。", "Enter an internal or private npm registry. Store required login credentials in this machine's ~/.npmrc.")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(c.Theme().TextMuted)
		})
	})
	if shown && !open {
		a.ToggleRegistryOptions()
	}
}

func installGuide(c *ui.Context, i manager.InstallState, a manager.Actions) {
	trigger := ui.Button(c, copyText("安装引导", "Installation guide"))
	open := ui.Local(trigger, "guide-open", func() bool { return i.MirrorRecovery })
	if trigger.Clicked() {
		*open = !*open
	}
	if !*open {
		return
	}
	ui.Text(c, copyText("填入插件 npm 包名", "Enter the plugin's npm package name")).FontSize(theme.InterfaceSize(13)).TextColor(c.Theme().Text)
	ui.Text(c, copyText("插件包名即 npm 包名，如 dsh-xxx 或 @作者/插件名，是社区插件 README 安装命令中 dsh plugin add 或 pnpm add 之后的部分。", "Use the npm package name, such as dsh-xxx or @author/plugin: the part after dsh plugin add or pnpm add in the plugin README.")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(c.Theme().TextMuted)
	if ui.Button(c, "dsh-plugin-whale-pet").Disabled(i.Phase != "idle").Clicked() {
		a.EditInstall("dsh-plugin-whale-pet", i.Registry)
	}
	if strings.TrimSpace(i.Spec) == "" && i.MirrorRecovery {
		ui.Text(c, copyText("请用 npm 包名尝试其他安装来源。", "Try another installation source using the npm package name.")).FillWidth().TextColor(c.Theme().TextMuted)
	}
}
