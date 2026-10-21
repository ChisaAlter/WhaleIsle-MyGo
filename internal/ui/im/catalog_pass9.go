package im

import "whaleisle/internal/ui/i18n"

const (
	keyIMEnablePlugin   = "im.enable_plugin"
	keyIMPluginNotReady = "im.plugin_not_ready"
	keyIMSendPending    = "im.send_pending"
	keyIMSendFailed     = "im.send_failed"
	keyIMSendLocal      = "im.send_local"
	keyIMSendUnknown    = "im.send_unknown"
)

func init() {
	i18n.Register(keyIMEnablePlugin, "启用 dsh-im 插件", "Enable the dsh-im plugin")
	i18n.Register(keyIMPluginNotReady, "IM 插件未就绪；打开插件页完成接入。", "The IM plugin is not ready; open the plugin page to finish setup.")
	i18n.Register(keyIMSendPending, "发送中…", "Sending…")
	i18n.Register(keyIMSendFailed, "发送失败", "Not sent")
	i18n.Register(keyIMSendLocal, "仅保存在本机", "Local only")
	i18n.Register(keyIMSendUnknown, "发送状态未知", "Delivery unknown")
}
