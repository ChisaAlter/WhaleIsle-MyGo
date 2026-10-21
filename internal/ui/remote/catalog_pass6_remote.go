package remote

import "whaleisle/internal/ui/i18n"

// Pass 6 remote gateway copy lives in the remote package: the settings gateway
// page is owned here, and another agent edits the shared i18n catalogs in the
// same round. Registration is init-only, like every other surface catalog.
const (
	KeyPass6RemoteTitle       = "surfaces.pass6.remote.title"
	KeyPass6RemoteIntro       = "surfaces.pass6.remote.intro"
	KeyPass6RemoteTabGateway  = "surfaces.pass6.remote.tab_gateway"
	KeyPass6RemoteTabChannels = "surfaces.pass6.remote.tab_channels"
	KeyPass6RemoteUnavailable = "surfaces.pass6.remote.unavailable"

	KeyPass6RemoteRelay            = "surfaces.pass6.remote.relay"
	KeyPass6RemoteRelayLabel       = "surfaces.pass6.remote.relay_label"
	KeyPass6RemoteRelayHint        = "surfaces.pass6.remote.relay_hint"
	KeyPass6RemoteRestoreDefault   = "surfaces.pass6.remote.restore_default"
	KeyPass6RemoteSaveRelay        = "surfaces.pass6.remote.save_relay"
	KeyPass6RemoteLegacySection    = "surfaces.pass6.remote.legacy_section"
	KeyPass6RemoteLegacyTitle      = "surfaces.pass6.remote.legacy_title"
	KeyPass6RemoteLegacyHint       = "surfaces.pass6.remote.legacy_hint"
	KeyPass6RemoteLegacyClear      = "surfaces.pass6.remote.legacy_clear"
	KeyPass6RemoteMode             = "surfaces.pass6.remote.mode"
	KeyPass6RemoteModeHint         = "surfaces.pass6.remote.mode_hint"
	KeyPass6RemoteModeLAN          = "surfaces.pass6.remote.mode_lan"
	KeyPass6RemoteModeServer       = "surfaces.pass6.remote.mode_server"
	KeyPass6RemotePort             = "surfaces.pass6.remote.port"
	KeyPass6RemotePortLabel        = "surfaces.pass6.remote.port_label"
	KeyPass6RemotePortHint         = "surfaces.pass6.remote.port_hint"
	KeyPass6RemoteSavePort         = "surfaces.pass6.remote.save_port"
	KeyPass6RemoteBind             = "surfaces.pass6.remote.bind"
	KeyPass6RemoteBindLabel        = "surfaces.pass6.remote.bind_label"
	KeyPass6RemoteBindHintLocal    = "surfaces.pass6.remote.bind_hint_local"
	KeyPass6RemoteBindHintLAN      = "surfaces.pass6.remote.bind_hint_lan"
	KeyPass6RemoteBindHintNIC      = "surfaces.pass6.remote.bind_hint_nic"
	KeyPass6RemoteBindHintStale    = "surfaces.pass6.remote.bind_hint_stale"
	KeyPass6RemoteEncryption       = "surfaces.pass6.remote.encryption"
	KeyPass6RemoteEncryptionHint   = "surfaces.pass6.remote.encryption_hint"
	KeyPass6RemoteRefresh          = "surfaces.pass6.remote.refresh"
	KeyPass6RemoteRefreshHint      = "surfaces.pass6.remote.refresh_hint"
	KeyPass6RemoteRefreshAction    = "surfaces.pass6.remote.refresh_action"
	KeyPass6RemoteLegacyNoCallback = "surfaces.pass6.remote.legacy_no_callback"
)

func init() {
	i18n.Register(KeyPass6RemoteTitle, "远程", "Remote")
	i18n.Register(KeyPass6RemoteIntro,
		"手机在侧栏扫码配对：Android App 内扫码 = 链接设备，相机 / 浏览器扫码 = 打开 web 端。此处只改连接方式与中继。",
		"Pair a phone by scanning the code in the sidebar: scan in the Android app to link a device, or with the camera or a browser to open the web app. Only the connection and relay are configured here.")
	i18n.Register(KeyPass6RemoteTabGateway, "网关", "Gateway")
	i18n.Register(KeyPass6RemoteTabChannels, "消息渠道", "Channels")
	i18n.Register(KeyPass6RemoteUnavailable, "当前桌面连接未提供网关设置。", "Gateway settings are unavailable on this desktop connection.")

	i18n.Register(KeyPass6RemoteRelay, "中继主机", "Relay host")
	i18n.Register(KeyPass6RemoteRelayLabel, "中继主机输入", "Gateway relay endpoint")
	i18n.Register(KeyPass6RemoteRelayHint, "默认 ayase.cn:443（TLS），一般不用改。", "Defaults to ayase.cn:443 (TLS); usually no need to change.")
	i18n.Register(KeyPass6RemoteRestoreDefault, "恢复默认", "Reset to default")
	i18n.Register(KeyPass6RemoteSaveRelay, "保存中继主机", "Save relay host")
	i18n.Register(KeyPass6RemoteLegacySection, "旧版配置清理", "Legacy configuration cleanup")
	i18n.Register(KeyPass6RemoteLegacyTitle, "旧版中继令牌", "Legacy relay token")
	i18n.Register(KeyPass6RemoteLegacyHint, "旧版令牌不参与当前设备配对，可安全清除。", "Legacy tokens are not used for device pairing and can be cleared safely.")
	i18n.Register(KeyPass6RemoteLegacyClear, "清除旧版令牌", "Clear legacy token")
	i18n.Register(KeyPass6RemoteLegacyNoCallback, "当前桌面连接无法清除旧版令牌。", "Legacy tokens cannot be cleared on this desktop connection.")
	i18n.Register(KeyPass6RemoteMode, "连接方式", "Connection")
	i18n.Register(KeyPass6RemoteModeHint,
		"局域网：扫码打开本机:3180 网页。服务器：扫码打开公网网页；数据走中继。不要用微信转发链接（会丢掉 #offer=）。",
		"LAN: scanning opens the local :3180 page. Server: scanning opens the public page and data goes through the relay. Do not forward links through WeChat: it drops #offer=.")
	i18n.Register(KeyPass6RemoteModeLAN, "局域网", "LAN")
	i18n.Register(KeyPass6RemoteModeServer, "服务器", "Server")
	i18n.Register(KeyPass6RemotePort, "监听端口", "Listen port")
	i18n.Register(KeyPass6RemotePortLabel, "监听端口输入", "Gateway port")
	i18n.Register(KeyPass6RemotePortHint, "本机 daemon 默认 6767；局域网配对页另用 3180。", "The local daemon defaults to 6767; the LAN pairing page uses 3180 separately.")
	i18n.Register(KeyPass6RemoteSavePort, "保存监听端口", "Save listen port")
	i18n.Register(KeyPass6RemoteBind, "监听范围", "Listen scope")
	i18n.Register(KeyPass6RemoteBindLabel, "监听范围选择", "Gateway bind address")
	i18n.Register(KeyPass6RemoteBindHintLocal, "仅本机监听：局域网设备无法直连。", "This machine only: LAN devices cannot connect directly.")
	i18n.Register(KeyPass6RemoteBindHintLAN, "整个局域网监听：同一网络内的设备可直接连接。", "Whole LAN: devices on the same network can connect directly.")
	i18n.Register(KeyPass6RemoteBindHintNIC, "仅监听所选网卡：只有该网段的设备可直接连接。", "Only the selected interface: devices on that subnet can connect directly.")
	i18n.Register(KeyPass6RemoteBindHintStale, "所选网卡已失效：请重新选择监听范围。", "The selected interface is unavailable; choose a listen scope again.")
	i18n.Register(KeyPass6RemoteEncryption, "传输加密", "Transport encryption")
	i18n.Register(KeyPass6RemoteEncryptionHint,
		"手机全用端到端加密，中继 TLS 由中继主机配置决定；局域网配对页使用 HTTP，只提供应用文件。",
		"Phones always use end-to-end encryption; relay TLS follows the relay host configuration. The LAN pairing page uses HTTP and serves only application files.")
	i18n.Register(KeyPass6RemoteRefresh, "刷新配对码", "Refresh pairing code")
	i18n.Register(KeyPass6RemoteRefreshHint, "只换二维码；已配对设备仍有效，除非解除配对。", "Only the QR code changes; paired devices stay paired unless removed.")
	i18n.Register(KeyPass6RemoteRefreshAction, "刷新", "Refresh")
}
