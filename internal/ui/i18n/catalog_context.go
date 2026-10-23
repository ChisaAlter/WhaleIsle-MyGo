package i18n

// Context body copy follows the durable context presentations in the reference
// client. Producer values, paths and section names are displayed verbatim.
const (
	KeyContextInstructionsLoaded  = "message.context.instructions.loaded"
	KeyContextInstructionsAdded   = "message.context.instructions.added"
	KeyContextInstructionsUpdated = "message.context.instructions.updated"
	KeyContextInstructionsRemoved = "message.context.instructions.removed"
	KeyContextCatalogReplaced     = "message.context.catalog.replaced"
	KeyContextCatalogMore         = "message.context.catalog.more"
	KeyContextSnapshotSupersedes  = "message.context.snapshot.supersedes"
	KeyContextRelayFrom           = "message.context.relay.from"
	KeyContextRecallCounts        = "message.context.recall.counts"
	KeyContextRecallTruncated     = "message.context.recall.truncated"
	KeyContextUnknownBlock        = "message.unknownBlock"
	KeyContextJSONTruncated       = "json.truncated"
	KeyChatClockMD                = "clock.md"
	KeyChatClockYMD               = "clock.ymd"
)

func init() {
	Register(KeyContextInstructionsLoaded, "已载入", "loaded")
	Register(KeyContextInstructionsAdded, "已新增", "added")
	Register(KeyContextInstructionsUpdated, "已更新", "updated")
	Register(KeyContextInstructionsRemoved, "已移除", "removed")
	Register(KeyContextCatalogReplaced, "替换目录", "Replacement catalog")
	Register(KeyContextCatalogMore, "…还有 %d 条", "… %d more")
	Register(KeyContextSnapshotSupersedes, "取代先前的快照", "Supersedes earlier snapshots")
	Register(KeyContextRelayFrom, "来自会话 %s", "From session %s")
	Register(KeyContextRecallCounts, "保留 %s 条 · 省略 %s 条", "%s kept · %s omitted")
	Register(KeyContextRecallTruncated, "已截断", "truncated")
	Register(KeyContextUnknownBlock, "未知内容块", "Unknown content block")
	Register(KeyContextJSONTruncated, "… 已截断，共 %d 字符", "… truncated, %d characters total")
	Register(KeyChatClockMD, "%d月%d日", "%d/%d")
	Register(KeyChatClockYMD, "%d年%d月%d日", "%d-%d-%d")
}
