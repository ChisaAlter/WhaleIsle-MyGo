package settings

import "whaleisle/internal/ui/i18n"

// Pass 4A keys live here rather than in internal/ui/i18n/catalog_misc.go: another
// agent edits that file in the same round, and this batch only needs settings-side
// copy. Registration is init-only, like every other surface catalog.
const (
	KeyPass4ABuiltinHint        = "surfaces.pass4a.builtin_hint"
	KeyPass4ABuiltinLoading     = "surfaces.pass4a.builtin_loading"
	KeyPass4AMCPHintManaged     = "surfaces.pass4a.mcp_hint_managed"
	KeyPass4AMCPHintHTTP        = "surfaces.pass4a.mcp_hint_http"
	KeyPass4AMCPHintComposition = "surfaces.pass4a.mcp_hint_composition"
	KeyPass4AMCPEmptyPrimary    = "surfaces.pass4a.mcp_empty_primary"
	KeyPass4AMCPEmptySecondary  = "surfaces.pass4a.mcp_empty_secondary"
	KeyPass4AGlowSettings       = "surfaces.pass4a.glow_settings"
	KeyPass4AGlowUnavailable    = "surfaces.pass4a.glow_unavailable"
)

func init() {
	i18n.Register(KeyPass4ABuiltinHint, "查看内置部署的插件列表", "Built-in plugins deployed with this app.")
	i18n.Register(KeyPass4ABuiltinLoading, "正在读取内置插件", "Loading built-in plugins")
	i18n.Register(KeyPass4AMCPHintManaged, "受管条目会写入 $DSH_HOME/mcp-servers.yaml。", "Managed entries are written to $DSH_HOME/mcp-servers.yaml.")
	i18n.Register(KeyPass4AMCPHintHTTP, "HTTP 服务器可在本页登录后于对话里使用工具。", "Sign in on an HTTP server here to use its tools in chat.")
	i18n.Register(KeyPass4AMCPHintComposition, "组成配置条目只读。", "Composition entries are read-only.")
	i18n.Register(KeyPass4AMCPEmptyPrimary, "还没有 MCP 服务器。", "No MCP servers yet.")
	i18n.Register(KeyPass4AMCPEmptySecondary, "添加一个服务器以连接本地工具或远程服务。", "Add one to reach local tools or a remote service.")
	i18n.Register(KeyPass4AGlowSettings, "思考炫光设置", "Thinking glow settings")
	i18n.Register(KeyPass4AGlowUnavailable, "此构建未接入思考炫光设置", "Thinking glow settings are not wired in this build")
}
