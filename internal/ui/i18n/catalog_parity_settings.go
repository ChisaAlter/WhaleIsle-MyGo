package i18n

import "strings"

// CatalogParitySettings 注册设置中心、远程连接、外观主题、关于更新、插件管理与插件市场的全量对齐词条。

// SettingsParityFormat 取词并将 {name} 形式占位符替换为成对给出的键值。
func SettingsParityFormat(key string, kv ...string) string {
	text := T(key)
	for i := 0; i+1 < len(kv); i += 2 {
		text = strings.ReplaceAll(text, "{"+kv[i]+"}", kv[i+1])
	}
	return text
}

const (
	// 1. dsh-client-ui-settings-general
	KeySettingsParityNavGroupAgent          = "settings_parity.general.nav.group.agent"
	KeySettingsParitySearchResults          = "settings_parity.general.search.results"
	KeySettingsParityAboutCenterOpen        = "settings_parity.general.about.center.open"
	KeySettingsParityAboutCenterIdle        = "settings_parity.general.about.center.idle"
	KeySettingsParityAboutCenterCurrent     = "settings_parity.general.about.center.current"
	KeySettingsParityAboutCenterAvailable   = "settings_parity.general.about.center.available"
	KeySettingsParityAboutCenterDownloading = "settings_parity.general.about.center.downloading"
	KeySettingsParityAboutCenterPaused      = "settings_parity.general.about.center.paused"
	KeySettingsParityAboutCenterVerifying   = "settings_parity.general.about.center.verifying"
	KeySettingsParityAboutCenterReady       = "settings_parity.general.about.center.ready"
	KeySettingsParityAboutCenterInstalling  = "settings_parity.general.about.center.installing"
	KeySettingsParityAboutCenterError       = "settings_parity.general.about.center.error"
	KeySettingsParityAboutCenterErrorAlt    = "settings_parity.general.about.center.error.alt"

	// 2. dsh-client-ui-settings-remote
	KeySettingsParityRemoteRelayConnectingHint      = "settings_parity.remote.relayConnectingHint"
	KeySettingsParityRemotePairingLifetimeHint      = "settings_parity.remote.pairingLifetimeHint"
	KeySettingsParityRemotePairingCountdownHint     = "settings_parity.remote.pairingCountdownHint"
	KeySettingsParityRemotePersistentLinkHint       = "settings_parity.remote.persistentLinkHint"
	KeySettingsParityRemotePersistentLinkWindowHint = "settings_parity.remote.persistentLinkWindowHint"
	KeySettingsParityRemoteRevokeLink               = "settings_parity.remote.revokeLink"
	KeySettingsParityRemoteRevokeLinkConfirm        = "settings_parity.remote.revokeLinkConfirm"
	KeySettingsParityRemoteRevokeLinkDisconnect     = "settings_parity.remote.revokeLinkDisconnect"
	KeySettingsParityRemoteRevokeLinkAction         = "settings_parity.remote.revokeLinkAction"
	KeySettingsParityRemoteLinkRevoked              = "settings_parity.remote.linkRevoked"
	KeySettingsParityRemoteLinkRevokedShort         = "settings_parity.remote.linkRevokedShort"

	// 3. dsh-client-ui-theme
	KeySettingsParityThemeEffectLowPowerGPU    = "settings_parity.theme.effect.lowPowerGpu"
	KeySettingsParityThemeEffectLowPowerGPUAlt = "settings_parity.theme.effect.lowPowerGpu.alt"
	KeySettingsParityThemePresetDefaultLight   = "settings_parity.theme.preset.defaultLight.label"
	KeySettingsParityThemePresetDefaultDark    = "settings_parity.theme.preset.defaultDark.label"
	KeySettingsParityThemePresetMidnight       = "settings_parity.theme.preset.midnight.label"
	KeySettingsParityThemePresetCeladon        = "settings_parity.theme.preset.celadon.label"
	KeySettingsParityThemePresetViolet         = "settings_parity.theme.preset.violet.label"
	KeySettingsParityThemePresetAmber          = "settings_parity.theme.preset.amber.label"
	KeySettingsParityThemePresetPaper          = "settings_parity.theme.preset.paper.label"
	KeySettingsParityThemePresetContrast       = "settings_parity.theme.preset.contrast.label"
)

func init() {
	// 1. dsh-client-ui-settings-general
	Register(KeySettingsParityNavGroupAgent, "智能体", "Agent")
	Register(KeySettingsParitySearchResults, "搜索结果", "Search results")
	Register(KeySettingsParityAboutCenterOpen, "打开更新", "Open updates")
	Register(KeySettingsParityAboutCenterIdle, "当前版本 {current}", "Current version {current}")
	Register(KeySettingsParityAboutCenterCurrent, "已是最新版本（{current}）", "Up to date ({current})")
	Register(KeySettingsParityAboutCenterAvailable, "发现新版本 {version}", "Version {version} is available")
	Register(KeySettingsParityAboutCenterDownloading, "正在下载 {version} · {percent}%", "Downloading {version} · {percent}%")
	Register(KeySettingsParityAboutCenterPaused, "{version} 下载已暂停 · {percent}%", "{version} download paused · {percent}%")
	Register(KeySettingsParityAboutCenterVerifying, "正在校验 {version}", "Verifying {version}")
	Register(KeySettingsParityAboutCenterReady, "{version} 已准备好，可以安装", "{version} is ready to install")
	Register(KeySettingsParityAboutCenterInstalling, "正在安装 {version}", "Installing {version}")
	Register(KeySettingsParityAboutCenterError, "更新未完成：{message}", "Update not finished: {message}")
	Register(KeySettingsParityAboutCenterErrorAlt, "更新未完成：{message}", "Update did not finish: {message}")

	// 2. dsh-client-ui-settings-remote
	Register(KeySettingsParityRemoteRelayConnectingHint, "正在连接中继…", "Connecting to relay…")
	Register(KeySettingsParityRemotePairingLifetimeHint, "二维码过期或使用后自动刷新；复制的链接长期有效。", "The QR refreshes after expiry or use. Copied links stay valid until revoked.")
	Register(KeySettingsParityRemotePairingCountdownHint, "配对码与二维码将在 {seconds} 秒后刷新；配对成功后无需重复扫码。", "Pairing code and QR refresh in {seconds}s; once paired, you won't need to scan again.")
	Register(KeySettingsParityRemotePersistentLinkHint, "长期链接可重复配对，在这里撤销。已配对设备不受影响。", "Reusable links can pair new devices until revoked here. Paired devices stay connected.")
	Register(KeySettingsParityRemotePersistentLinkWindowHint, "该链接长期有效，拿到链接的人可在配对有效期内连接你的桌面端。", "This link stays valid; anyone with it can connect during the pairing window.")
	Register(KeySettingsParityRemoteRevokeLink, "撤销长期链接", "Revoke persistent link")
	Register(KeySettingsParityRemoteRevokeLinkConfirm, "撤销后，旧链接不能再配对；已配对设备保持连接。", "The old link will stop pairing new devices. Paired devices stay connected.")
	Register(KeySettingsParityRemoteRevokeLinkDisconnect, "撤销后旧链接将立即失效，已连接设备也会断开。确定继续吗？", "Revoking invalidates the current link immediately and disconnects paired devices. Continue?")
	Register(KeySettingsParityRemoteRevokeLinkAction, "撤销并生成新链接", "Revoke and replace link")
	Register(KeySettingsParityRemoteLinkRevoked, "旧链接已撤销，可重新复制新链接。", "Old link revoked. You can copy the replacement link.")
	Register(KeySettingsParityRemoteLinkRevokedShort, "持久链接已重置", "Persistent link reset")

	// 3. dsh-client-ui-theme
	Register(KeySettingsParityThemeEffectLowPowerGPU, "检测到集成显卡，背景特效默认关闭；开启后可能影响流畅度。", "An integrated GPU was detected, so the background effect starts off; turning it on may make the app less smooth.")
	Register(KeySettingsParityThemeEffectLowPowerGPUAlt, "检测到集成显卡，背景特效默认关闭；开启后可能影响流畅度。", "Integrated GPU detected. Background effects are off by default and may affect smoothness when enabled.")
	Register(KeySettingsParityThemePresetDefaultLight, "DeepSeek 浅色", "DeepSeek Light")
	Register(KeySettingsParityThemePresetDefaultDark, "DeepSeek 深色", "DeepSeek Dark")
	Register(KeySettingsParityThemePresetMidnight, "午夜", "Midnight")
	Register(KeySettingsParityThemePresetCeladon, "青瓷", "Celadon")
	Register(KeySettingsParityThemePresetViolet, "暮紫", "Violet")
	Register(KeySettingsParityThemePresetAmber, "琥珀", "Amber")
	Register(KeySettingsParityThemePresetPaper, "宣纸", "Paper")
	Register(KeySettingsParityThemePresetContrast, "对比", "Contrast")

	// 4. dsh-client-ui-plugin-manager
	Register("settings_parity.plugin_manager.loading", "正在读取插件列表…", "Reading plugin list…")
	Register("settings_parity.plugin_manager.error", "暂时无法读取插件列表。", "Plugin list is temporarily unavailable.")
	Register("settings_parity.plugin_manager.unavailable", "本部署没有可管理的 profile，无法安装或启停插件。", "This deployment has no writable profile, so plugins cannot be installed or toggled.")
	Register("settings_parity.plugin_manager.updating", "正在保存更改…", "Saving changes…")
	Register("settings_parity.plugin_manager.mutationError", "未能保存插件更改：{message}", "Could not save plugin changes: {message}")
	Register("settings_parity.plugin_manager.dismissMutationError", "关闭错误提示", "Dismiss error")
	Register("settings_parity.plugin_manager.openPackage", "查看 {name}", "View {name}")
	Register("settings_parity.plugin_manager.enablePackage", "启用 {name}", "Enable {name}")
	Register("settings_parity.plugin_manager.disablePackage", "停用 {name}", "Disable {name}")
	Register("settings_parity.plugin_manager.enableComponent", "启用 {name}", "Enable {name}")
	Register("settings_parity.plugin_manager.disableComponent", "停用 {name}", "Disable {name}")
	Register("settings_parity.plugin_manager.enableEntry", "启用 {name}", "Enable {name}")
	Register("settings_parity.plugin_manager.disableEntry", "停用 {name}", "Disable {name}")
	Register("settings_parity.plugin_manager.removePackage", "卸载 {name}", "Remove {name}")
	Register("settings_parity.plugin_manager.moreCount", "共 {count} 个插件", "{count} plugins")
	Register("settings_parity.plugin_manager.moreEntryTitle", "更多插件", "More")
	Register("settings_parity.plugin_manager.removeTitle", "卸载 {name}？", "Remove {name}?")
	Register("settings_parity.plugin_manager.removeBody", "将从当前 profile 卸载 {name}，并移除它的组件配置。", "This removes {name} from the current profile and deletes its component entries.")
	Register("settings_parity.plugin_manager.removing", "正在卸载…", "Removing…")
	Register("settings_parity.plugin_manager.factVersion", "版本", "Version")
	Register("settings_parity.plugin_manager.factPackage", "包名", "Package")
	Register("settings_parity.plugin_manager.factAuthor", "作者", "Author")
	Register("settings_parity.plugin_manager.factCategory", "分类", "Category")
	Register("settings_parity.plugin_manager.factLicense", "许可协议", "License")
	Register("settings_parity.plugin_manager.factAvailability", "可用范围", "Availability")
	Register("settings_parity.plugin_manager.factStatus", "当前状态", "Status")
	Register("settings_parity.plugin_manager.availabilityOfficial", "官方可选", "Official optional")
	Register("settings_parity.plugin_manager.availabilityInstalled", "已安装到当前环境", "Installed in this profile")
	Register("settings_parity.plugin_manager.availabilityInstallation", "随宿主提供", "Bundled with host")
	Register("settings_parity.plugin_manager.statusEnabled", "已启用", "Enabled")
	Register("settings_parity.plugin_manager.statusPartial", "已启用 {enabled}/{total}", "{enabled}/{total} enabled")
	Register("settings_parity.plugin_manager.statusDisabled", "未启用", "Disabled")
	Register("settings_parity.plugin_manager.componentsHint", "一个插件可以包含多个组件，可按需单独开关。", "A plugin can bundle several components; toggle them individually as needed.")
	Register("settings_parity.plugin_manager.noComponents", "这个插件没有声明可单独管理的组件。", "This plugin does not declare individually managed components.")
	Register("settings_parity.plugin_manager.componentLockedHint", "请先启用所属插件，才能调整这里的组件。", "Enable the parent plugin to change its components.")
	Register("settings_parity.plugin_manager.entryRuntimeHint", "来自同一插件的运行时扩展，不通过当前配置文件保存。", "Runtime extension from the same plugin; not stored in the current profile file.")
	Register("settings_parity.plugin_manager.entryActive", "运行中", "Active")
	Register("settings_parity.plugin_manager.entryInactive", "未运行", "Inactive")
	Register("settings_parity.plugin_manager.installTrigger", "从 npm 安装", "Install from npm")
	Register("settings_parity.plugin_manager.installSubmit", "安装", "Install")
	Register("settings_parity.plugin_manager.installing", "正在安装…", "Installing…")
	Register("settings_parity.plugin_manager.installStepResolve", "准备安装与依赖环境", "Preparing installation and dependencies")
	Register("settings_parity.plugin_manager.installStepAdd", "下载并写入插件包", "Downloading and writing package")
	Register("settings_parity.plugin_manager.installStepEnable", "注册并启用插件", "Registering and enabling plugin")
	Register("settings_parity.plugin_manager.installSuccess", "已安装并启用 {name}。", "Installed and enabled {name}.")
	Register("settings_parity.plugin_manager.installRegistryPlaceholder", "留空使用默认源，如 https://registry.npmmirror.com", "Leave blank for default, e.g. https://registry.npmmirror.com")
	Register("settings_parity.plugin_manager.installCurrentRegistry", "当前默认源：{registry}", "Current default: {registry}")
	Register("settings_parity.plugin_manager.installExitCode", "退出码 {code}", "exit {code}")
	Register("settings_parity.plugin_manager.infoDescription", "在这里配置官方插件，也可以安装和管理其他插件。内置插件列表与运行状态可在「设置 → 内置插件」中查看。", "Configure official plugins and install or manage other plugins here. View the built-in plugin list and runtime status in Settings → Built-in plugins.")
	Register("settings_parity.plugin_manager.unavailableAlt", "本部署没有可管理的 profile，无法安装或启停插件。", "This deployment runs without a manageable profile, so plugins cannot be installed or switched here.")
	Register("settings_parity.plugin_manager.empty", "尚未安装任何插件。", "No plugins are installed yet.")
	Register("settings_parity.plugin_manager.installExisting", "安装第三方插件", "Install a third-party plugin")
	Register("settings_parity.plugin_manager.partsEmpty", "此插件包没有组件。", "This plugin pack contains no components.")
	Register("settings_parity.plugin_manager.partsFilterEmpty", "没有匹配的组件。", "No component matches.")
	Register("settings_parity.plugin_manager.installDescription", "输入插件的包名、GitHub 仓库地址或本地目录路径。", "Enter the plugin's package name, GitHub repository address, or local directory path.")
	Register("settings_parity.plugin_manager.installSpecLabel", "包名或地址", "Package name or address")
	Register("settings_parity.plugin_manager.installGuideIdHint", "插件包名即 npm 包名（形如 dsh-xxx 或 @author/plugin），就是社区插件 README 安装命令里 dsh plugin add 或 pnpm add 后面的那一段。", "The plugin package name is the npm package name (like dsh-xxx or @author/plugin): the part after dsh plugin add or pnpm add in a community plugin's README install command.")
	Register("settings_parity.plugin_manager.installGuideFill", "使用示例", "Use example")
	Register("settings_parity.plugin_manager.installGuideFillAria", "填入示例 {example}", "Use the example {example}")
	Register("settings_parity.plugin_manager.registryOfficial", "npm 官方源", "Official npm registry")
	Register("settings_parity.plugin_manager.installConfirmedTitle", "安装插件", "Install plugin")
	Register("settings_parity.plugin_manager.statusMissingFiles", "包不可用", "Package unavailable")
	Register("settings_parity.plugin_manager.installPackageLabel", "插件包名", "Plugin package name")
	Register("settings_parity.plugin_manager.installCancelAndEdit", "取消安装并返回修改", "Cancel installation and return to editing")
	Register("settings_parity.plugin_manager.installDetailsHide", "收起安装详情", "Hide install details")
	Register("settings_parity.plugin_manager.installSubjectPath", "本地目录", "Local directory")
	Register("settings_parity.plugin_manager.installFailureTimeout", "安装超时", "The install timed out")
	Register("settings_parity.plugin_manager.reasonLabel", "原因", "Reason")
	Register("settings_parity.plugin_manager.sourceVersion", "当前版本", "Current version")
	Register("settings_parity.plugin_manager.partsCountOff", "{count} 个未启用", "{count} off")
	Register("settings_parity.plugin_manager.partsCountFailed", "{count} 个失败", "{count} failed")
	Register("settings_parity.plugin_manager.partToggle", "启用组件 {name}", "Enable component {name}")
	Register("settings_parity.plugin_manager.backToPackage", "返回 {name}", "Back to {name}")
	Register("settings_parity.plugin_manager.configureRow", "配置 {name}", "Configure {name}")
	Register("settings_parity.plugin_manager.uninstallLabel", "卸载 {name}", "Uninstall {name}")
	Register("settings_parity.plugin_manager.updateLabel", "更新 {name}", "Update {name}")
	Register("settings_parity.plugin_manager.installReconcile", "核对安装状态", "Check installation status")
	Register("settings_parity.plugin_manager.installVersion", "版本 {version}", "Version {version}")
	Register("settings_parity.plugin_manager.installSubjectGit", "Git 仓库", "Git repository")
	Register("settings_parity.plugin_manager.installLocation", "安装位置：{dir}", "Installs into {dir}")

	// 5. dsh-client-ui-plugin-market
	Register("settings_parity.plugin_market.factCategory", "分类", "Category")
	Register("settings_parity.plugin_market.versionsLabel", "{name} 的版本", "Versions of {name}")
	Register("settings_parity.plugin_market.removeSourceNamed", "移除 {url}", "Remove {url}")
	Register("settings_parity.plugin_market.filter", "标签", "Labels")
	Register("settings_parity.plugin_market.openEntry", "查看 {name}", "View {name}")
	Register("settings_parity.plugin_market.installTitle", "安装 {name}", "Install {name}")
	Register("settings_parity.plugin_market.installVersionTitle", "安装 {name} {version}", "Install {name} {version}")
}
