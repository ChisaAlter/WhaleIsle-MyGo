package i18n

const (
	KeyToolReadCard         = "chat.tool.read.card"
	KeyToolReadWindow       = "chat.tool.read.window"
	KeyToolReadExpand       = "chat.tool.read.expand"
	KeyToolReadExpandAria   = "chat.tool.read.expand_aria"
	KeyToolReadCollapseAria = "chat.tool.read.collapse_aria"
	KeyToolReadUnwrap       = "chat.tool.read.unwrap"
	KeyToolReadCodeLabel    = "chat.tool.read.code_label"
	KeyToolReadCopied       = "chat.tool.read.copied"
)

func init() {
	Register(KeyToolReadCard, "读取内容卡片", "Read content card")
	Register(KeyToolReadWindow, "显示 %d / %d 行", "Showing %d of %d lines")
	Register(KeyToolReadExpand, "… 其余 %d 行", "… %d more lines")
	Register(KeyToolReadExpandAria, "展开其余 %d 行", "Expand %d more lines")
	Register(KeyToolReadCollapseAria, "收起内容", "Collapse content")
	Register(KeyToolReadUnwrap, "取消自动换行", "Do not wrap lines")
	Register(KeyToolReadCodeLabel, "代码块", "Code block")
	Register(KeyToolReadCopied, "复制成功", "Copied")
}
