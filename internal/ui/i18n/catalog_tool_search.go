package i18n

const (
	KeyToolSearchCard             = "chat.tool.search.card"
	KeyToolSearchPaths            = "chat.tool.search.paths"
	KeyToolSearchPathsTruncated   = "chat.tool.search.paths_truncated"
	KeyToolSearchMatches          = "chat.tool.search.matches"
	KeyToolSearchMatchesTruncated = "chat.tool.search.matches_truncated"
	KeyToolSearchEmpty            = "chat.tool.search.empty"
	KeyToolSearchExpand           = "chat.tool.search.expand"
	KeyToolSearchExpandAria       = "chat.tool.search.expand_aria"
	KeyToolSearchCollapseAria     = "chat.tool.search.collapse_aria"
)

func init() {
	Register(KeyToolSearchCard, "搜索结果卡片", "Search results card")
	Register(KeyToolSearchPaths, "%d 个路径", "%d paths")
	Register(KeyToolSearchPathsTruncated, "显示 %d / 共 %s 个路径", "Showing %d of %s paths")
	Register(KeyToolSearchMatches, "%d 处匹配 · %d 个文件", "%d matches · %d files")
	Register(KeyToolSearchMatchesTruncated, "显示 %d / 共 %s 处匹配 · %d 个文件", "Showing %d of %s matches · %d files")
	Register(KeyToolSearchEmpty, "无结果", "No results")
	Register(KeyToolSearchExpand, "… 其余 %d 行", "… %d more lines")
	Register(KeyToolSearchExpandAria, "展开其余 %d 行结果", "Expand %d more result lines")
	Register(KeyToolSearchCollapseAria, "收起结果", "Collapse results")
}
