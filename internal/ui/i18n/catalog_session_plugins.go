package i18n

const (
	KeySessionPluginsStop              = "session.plugins.stop"
	KeySessionPluginsFullConversation  = "session.plugins.full_conversation"
	KeySessionPluginsIncompatible      = "session.plugins.incompatible"
	KeySessionPluginsOutletTurn        = "session.plugins.outlet_turn"
	KeySessionPluginsOutletCall        = "session.plugins.outlet_call"
	KeySessionPluginsOutletUnknown     = "session.plugins.outlet_unknown"
	KeySessionPluginsOutletInstance    = "session.plugins.outlet_instance"
	KeySessionPluginsOutletUntitled    = "session.plugins.outlet_untitled"
	KeySessionPluginsOutletUnavailable = "session.plugins.outlet_unavailable"
	KeySessionPluginsOpenTurnContent   = "session.plugins.open_turn_content"
	KeySessionPluginsTurnExpired       = "session.plugins.turn_expired"
)

func init() {
	Register(KeySessionPluginsStop, "关闭会话插件", "Turn off Session plugins")
	Register(KeySessionPluginsFullConversation, "打开完整会话视图", "Open full conversation")
	Register(KeySessionPluginsIncompatible, "当前版本不支持此会话视图。", "This version does not support the Session view.")
	Register(KeySessionPluginsOutletTurn, "第 %d 轮", "Turn %d")
	Register(KeySessionPluginsOutletCall, "工具调用 %s", "Tool call %s")
	Register(KeySessionPluginsOutletUnknown, "归属未知", "Location unknown")
	Register(KeySessionPluginsOutletInstance, "实例 %d", "Instance %d")
	Register(KeySessionPluginsOutletUntitled, "未命名插件内容", "Untitled plugin content")
	Register(KeySessionPluginsOutletUnavailable, "不可用", "Unavailable")
	Register(KeySessionPluginsOpenTurnContent, "打开此轮插件：%s", "Open turn plugin: %s")
	Register(KeySessionPluginsTurnExpired, "此轮插件已变化，请从列表重新选择。", "This turn's plugin content changed. Select it again from the list.")
}
