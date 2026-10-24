package i18n

const (
	KeyToolTerminalSignal   = "chat.tool.terminal.signal"
	KeyToolTerminalExit     = "chat.tool.terminal.exit"
	KeyToolTerminalNoExit   = "chat.tool.terminal.no_exit"
	KeyToolTerminalNoOutput = "chat.tool.terminal.no_output"
	KeyToolTerminalLine     = "chat.tool.terminal.line"
	KeyToolTerminalSend     = "chat.tool.terminal.send"
	KeyToolTerminalSession  = "chat.tool.terminal.session"
	KeyToolTerminalOutput   = "chat.tool.terminal.output_group"
	KeyToolTerminalBanner   = "chat.tool.terminal.banner"
	KeyToolTerminalStatus   = "chat.tool.terminal.status"
	KeyToolTerminalDone     = "chat.tool.terminal.done"
)

func init() {
	Register(KeyToolTerminalSignal, "信号 %s", "signal %s")
	Register(KeyToolTerminalExit, "退出码 %s", "exit code %s")
	Register(KeyToolTerminalNoExit, "未正常退出", "no exit code")
	Register(KeyToolTerminalNoOutput, "无输出", "No output")
	Register(KeyToolTerminalLine, "命令第 %d 行", "Command line %d")
	Register(KeyToolTerminalSend, "（发送输入）", "(send input)")
	Register(KeyToolTerminalSession, "终端 %s", "Terminal %s")
	Register(KeyToolTerminalOutput, "终端输出", "Terminal output")
	Register(KeyToolTerminalBanner, "终端命令", "Terminal command banner")
	Register(KeyToolTerminalStatus, "命令退出状态", "Terminal status")
	Register(KeyToolTerminalDone, "已完成", "Done")
}
