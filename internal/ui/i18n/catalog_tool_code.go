package i18n

const (
	KeyToolCodeProgram = "chat.tool.code.program"
	KeyToolCodeCard    = "chat.tool.code.card"
)

func init() {
	Register(KeyToolCodeProgram, "工具代码", "Tool program")
	Register(KeyToolCodeCard, "工具代码卡片", "Tool code card")
}
