package im

import "whaleisle/internal/ui/i18n"

const keyIMEnablePlugin = "im.enable_plugin"

func init() {
	i18n.Register(keyIMEnablePlugin, "启用 dsh-im 插件", "Enable the dsh-im plugin")
}