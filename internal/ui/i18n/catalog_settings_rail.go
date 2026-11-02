package i18n

// Keys for the 0.3.13 settings rail: the search box, the group headers and the
// two sections that were split out of General and Interface.
const (
	KeySettingsNavSearch           = "settings.nav.search"
	KeySettingsNavSearchClear      = "settings.nav.search_clear"
	KeySettingsNavSearchEmpty      = "settings.nav.search_empty"
	KeySettingsNavGroupAccount     = "settings.nav.group.account"
	KeySettingsNavGroupGeneral     = "settings.nav.group.general"
	KeySettingsNavGroupAgents      = "settings.nav.group.agents"
	KeySettingsNavGroupFeatures    = "settings.nav.group.features"
	KeySettingsNavGroupOther       = "settings.nav.group.other"
	KeySettingsSectionConversation = "settings.section.conversation"
	KeySettingsSectionFeatures     = "settings.section.features"
)

func init() {
	Register(KeySettingsNavSearch, "搜索设置", "Search settings")
	Register(KeySettingsNavSearchClear, "清除搜索", "Clear search")
	Register(KeySettingsNavSearchEmpty, "没有找到「%s」相关的设置", "No settings match “%s”")
	Register(KeySettingsNavGroupAccount, "账户", "Accounts")
	Register(KeySettingsNavGroupGeneral, "通用", "General")
	Register(KeySettingsNavGroupAgents, "智能体", "Agents")
	Register(KeySettingsNavGroupFeatures, "功能", "Features")
	Register(KeySettingsNavGroupOther, "其他", "Other")
	Register(KeySettingsSectionConversation, "对话", "Conversation")
	Register(KeySettingsSectionFeatures, "内置功能", "Built-in features")
}
