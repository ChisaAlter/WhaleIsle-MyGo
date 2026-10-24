package i18n

const (
	KeyToolDiffCard         = "chat.tool.diff.card"
	KeyToolDiffExpand       = "chat.tool.diff.expand"
	KeyToolDiffExpandAria   = "chat.tool.diff.expand_aria"
	KeyToolDiffCollapseAria = "chat.tool.diff.collapse_aria"
)

func init() {
	Register(KeyToolDiffCard, "差异内容卡片", "Diff content card")
	Register(KeyToolDiffExpand, "… 其余 %d 行", "… %d more lines")
	Register(KeyToolDiffExpandAria, "展开其余 %d 行差异", "Expand %d more diff lines")
	Register(KeyToolDiffCollapseAria, "收起差异", "Collapse diff")
}
