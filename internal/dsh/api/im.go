package api

import (
	"sort"
	"strings"
)

// dsh-im (@xmanrui/dsh-im 3.0.1) has no typert.remote-client schemas — codegen
// Registry therefore contains zero im/* endpoints. The plugin speaks host
// connection.rpc on per-channel paths (/feishu, /telegram, …) with endpoints
// such as connection.status / bot.bind-credentials, not Typert Remote im/*.
//
// Preferred* helpers below document the Typert-style names WhaleIsle will
// probe via dsh.Client.Call when (a) they appear in Registry after a future
// upstream codegen, or (b) a harness build starts serving them. Until then
// store/im falls back to pluginInventory/list diagnostics for @xmanrui/dsh-im.

const (
	// PluginInventoryListMethod is the typed host RPC used when im/* is absent.
	PluginInventoryListMethod = "pluginInventory/list"

	// DshIMModuleName is the npm / inventory module name for dsh-im.
	DshIMModuleName = "@xmanrui/dsh-im"
	// DshIMShortName is the vendor folder / overlay alias.
	DshIMShortName = "dsh-im"
)

// IMCordisChannels are dsh-im host prefixes for POST /{channel}/connection.status.
func IMCordisChannels() []string {
	return []string{"feishu", "telegram", "discord", "slack", "dingtalk", "wecom", "qq", "weixin"}
}

// IMCordisBotEndpoints lists Cordis method names exposed under each channel prefix
// (POST /{channel}/{method}) in vendored dsh-im plugin-src. There is no
// listMessages/getHistory/inbox RPC — bot roster comes from connection.status only.
//
// Shared (token-api / shared/rpc): connection.status, bot.bind-credentials,
// bot.reconnect (+ optional sendTest → internal sendConnectionTest, not free text),
// bot.delete, bot.workspace.set, bot.preset.set.
//
// Per-channel extras include provision.begin|poll|cancel, bot.sender.approve|revoke
// (dingtalk), bot.access-policy.set (telegram/whatsapp), bot.callback-repair.begin,
// bot.group-message-permission.begin, bot.group-response-mode.set, connection.test
// (feishu legacy), provision.verify (weixin), /office connector.* endpoints.
func IMCordisBotEndpoints() []string {
	return []string{
		"connection.status",
		"bot.bind-credentials",
		"bot.reconnect",
		"bot.delete",
		"bot.workspace.set",
		"bot.preset.set",
		"provision.begin",
		"provision.poll",
		"provision.cancel",
	}
}

// IMListMethodCandidates are Typert-style list RPCs to probe with Client.Call.
//
// Request args: {} or omitted (nil → {}).
// Result JSON shapes accepted by store/im:
//
//	{"items"|"conversations"|"channels"|"threads"|"list":[{id,title|name|label,…}]}
//	or a bare array of the same objects.
func IMListMethodCandidates() []string {
	return []string{
		"im/list",
		"im/listConversations",
		"im/listChannels",
	}
}

// IMHistoryMethodCandidates are Typert-style history RPCs to probe.
//
// Request args:
//
//	{"channelId":"<id>","limit":50}
//
// Result JSON shapes accepted by store/im:
//
//	{"messages"|"items"|"history":[{id,text|content|body,at|createdAt|timestamp,outbound|fromMe|direction,…}]}
//	or a bare array of message objects.
func IMHistoryMethodCandidates() []string {
	return []string{
		"im/history",
		"im/listMessages",
		"im/getHistory",
	}
}

// IMSendMethodCandidates are Typert-style send RPCs to probe.
//
// Request args:
//
//	{"channelId":"<id>","text":"<body>"}
//
// Result: any JSON object (ignored beyond ok); errors surface via Client.Call.
func IMSendMethodCandidates() []string {
	return []string{
		"im/send",
		"im/sendMessage",
	}
}

// IMMethods returns every registered "<namespace>/<method>" whose namespace is im.
func IMMethods() []string {
	var out []string
	for method := range Registry {
		if strings.HasPrefix(method, "im/") {
			out = append(out, method)
		}
	}
	sort.Strings(out)
	return out
}

// PreferredIMListMethod picks a list-style im/* RPC when the codegen registry
// includes it; empty when none are registered (current dsh-im vendor).
func PreferredIMListMethod() string {
	return firstRegistered(IMListMethodCandidates(), "list")
}

// PreferredIMHistoryMethod picks a history-style im/* RPC from the registry,
// or "" when none are registered.
func PreferredIMHistoryMethod() string {
	return firstRegistered(IMHistoryMethodCandidates(), "history", "message")
}

// PreferredIMSendMethod picks a send-style im/* RPC from the registry, or "".
func PreferredIMSendMethod() string {
	return firstRegistered(IMSendMethodCandidates(), "send")
}

// ResolveIMListMethod prefers a registry hit, else the first candidate string
// for live Client.Call probing.
func ResolveIMListMethod() string {
	if m := PreferredIMListMethod(); m != "" {
		return m
	}
	cands := IMListMethodCandidates()
	if len(cands) == 0 {
		return ""
	}
	return cands[0]
}

// ResolveIMHistoryMethod prefers a registry hit, else the first history candidate.
func ResolveIMHistoryMethod() string {
	if m := PreferredIMHistoryMethod(); m != "" {
		return m
	}
	cands := IMHistoryMethodCandidates()
	if len(cands) == 0 {
		return ""
	}
	return cands[0]
}

// ResolveIMSendMethod prefers a registry hit, else the first send candidate.
func ResolveIMSendMethod() string {
	if m := PreferredIMSendMethod(); m != "" {
		return m
	}
	cands := IMSendMethodCandidates()
	if len(cands) == 0 {
		return ""
	}
	return cands[0]
}

func firstRegistered(prefs []string, fallbackSubstrings ...string) string {
	for _, p := range prefs {
		if _, ok := Registry[p]; ok {
			return p
		}
	}
	methods := IMMethods()
	set := map[string]bool{}
	for _, m := range methods {
		set[m] = true
	}
	for _, p := range prefs {
		if set[p] {
			return p
		}
	}
	for _, m := range methods {
		lower := strings.ToLower(m)
		for _, sub := range fallbackSubstrings {
			if strings.Contains(lower, sub) {
				return m
			}
		}
	}
	return ""
}

// IsDshIMInventoryEntry reports whether a pluginInventory entry refers to dsh-im.
func IsDshIMInventoryEntry(moduleName, entryID string) bool {
	name := strings.ToLower(strings.TrimSpace(moduleName))
	id := strings.ToLower(strings.TrimSpace(entryID))
	if name == strings.ToLower(DshIMModuleName) || name == strings.ToLower(DshIMShortName) {
		return true
	}
	if strings.Contains(name, "dsh-im") || strings.Contains(id, "dsh-im") {
		return true
	}
	if strings.Contains(id, "xmanrui-dsh-im") {
		return true
	}
	return false
}
