package i18n

const (
	KeyToolWebCard             = "chat.tool.web.card"
	KeyToolWebEmpty            = "chat.tool.web.empty"
	KeyToolWebSourcesTruncated = "chat.tool.web.sources_truncated"
	KeyToolWebHTTP             = "chat.tool.web.http"
	KeyToolWebContentTruncated = "chat.tool.web.content_truncated"
)

func init() {
	Register(KeyToolWebCard, "网页结果卡片", "Web results card")
	Register(KeyToolWebEmpty, "未找到结果", "No results found")
	Register(KeyToolWebSourcesTruncated, "来源列表已截断", "Source list truncated")
	Register(KeyToolWebHTTP, "HTTP", "HTTP")
	Register(KeyToolWebContentTruncated, "内容已截断", "Content truncated")
}
