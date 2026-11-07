package terminal

import (
	"fmt"
	"strings"

	"whaleisle/internal/ui/i18n"
)

// sidebarTerminalZhCatalog and sidebarTerminalEnCatalog mirror
// @deepseek-ai/dsh-client-ui-sidebar-terminal/lib/types/client/locales.js.
var sidebarTerminalZhCatalog = map[string]string{
	"shortcut.noSession": "请先选择会话",
	"recoveryFailed":     "恢复终端失败：{message}",
	"retryRecovery":      "重试恢复终端",
	"shell":              "选择 Shell",
	"shellLoading":       "正在读取 Shell…",
	"shellEmpty":         "没有可用的 Shell",
	"description":        "在会话工作区运行命令",
	"title":              "终端",
	"new":                "新建终端",
	"loading":            "正在读取终端环境…",
	"creating":           "正在启动…",
	"dismissFeedback":    "关闭提示",
	"connecting":         "正在连接…",
	"disconnected":       "连接已断开。",
	"reconnect":          "重新连接",
	"readonly":           "此页面当前只读。",
	"control":            "接管输入",
	"closed":             "终端已关闭。",
	"exited":             "进程已退出（{code}）",
	"failed":             "终端错误：{message}",
	"rename":             "终端名称",
	"unavailable":        "不可用",
	"retry":              "重试",
	"cleanupFailed":      "终端「{title}」未能结束：{message}",
	"missingTerminal":    "此终端已不存在，请新建终端。",
	"inputFull":          "输入缓冲区已满，请重新连接后重试。",
	"attachmentEnded":    "终端连接已结束，请重新连接。",
	"invalidOutput":      "终端画面传输异常，请重新连接。",
	"terminalLimit":      "终端数量已达上限，请关闭不用的终端后重试。已退出的终端也计入数量。",
}

var sidebarTerminalEnCatalog = map[string]string{
	"shortcut.noSession": "Select a session first",
	"recoveryFailed":     "Terminal recovery failed: {message}",
	"retryRecovery":      "Retry terminal recovery",
	"shell":              "Choose shell",
	"shellLoading":       "Loading shells…",
	"shellEmpty":         "No shells available",
	"description":        "Run commands in the Session workspace",
	"title":              "Terminal",
	"new":                "New terminal",
	"loading":            "Reading terminal environment…",
	"creating":           "Starting…",
	"dismissFeedback":    "Dismiss notification",
	"connecting":         "Connecting…",
	"disconnected":       "Disconnected.",
	"reconnect":          "Reconnect",
	"readonly":           "This view is read-only.",
	"control":            "Take control",
	"closed":             "Terminal closed.",
	"exited":             "Process exited ({code})",
	"failed":             "Terminal error: {message}",
	"rename":             "Terminal name",
	"unavailable":        "Unavailable",
	"retry":              "Retry",
	"cleanupFailed":      "Terminal “{title}” could not be ended: {message}",
	"missingTerminal":    "This terminal no longer exists. Open a new terminal.",
	"inputFull":          "The input buffer is full. Reconnect and try again.",
	"attachmentEnded":    "The terminal connection ended. Reconnect to continue.",
	"invalidOutput":      "The terminal screen could not be received. Reconnect to recover it.",
	"terminalLimit":      "The terminal limit has been reached. Close unused terminals and try again. Exited terminals also count toward the limit.",
}

// userTerminalZhCatalog and userTerminalEnCatalog mirror
// @deepseek-ai/dsh-client-ui-user-terminal/lib/types/client/locales.js.
var userTerminalZhCatalog = map[string]string{
	"empty.title":                 "暂无终端会话",
	"empty.unavailable":           "请先打开可访问的项目会话，再启动终端。",
	"action.splitHorizontal":       "左右分屏",
	"action.splitHorizontal.limit": "左右分屏（每组最多 4 个）",
	"action.splitVertical":         "上下分屏",
	"action.splitVertical.limit":   "上下分屏（每组最多 4 个）",
	"action.maximize":              "最大化",
	"action.restore":               "还原",
	"action.new":                   "新建终端",
	"action.close":                 "关闭终端",
	"resize":                       "调整终端高度",
	"session.label":                "终端",
	"sessions.list":                "终端会话",
	"group.label":                  "组",
	"error.create":                 "无法启动终端",
	"error.initialize":             "无法初始化终端：{message}。请关闭并重新打开终端。",
	"loading":                      "正在初始化终端…",
	"dismissFeedback":              "关闭提示",
	"action.copy":                  "复制",
	"action.addToChat":             "加入对话",
	"action.openLink":              "打开链接",
	"action.openPath":              "打开文件",
}

var userTerminalEnCatalog = map[string]string{
	"empty.title":                 "No terminal sessions yet",
	"empty.unavailable":           "Open an accessible project session before starting a terminal.",
	"action.splitHorizontal":       "Split left/right",
	"action.splitHorizontal.limit": "Split left/right (max 4 per group)",
	"action.splitVertical":         "Split top/bottom",
	"action.splitVertical.limit":   "Split top/bottom (max 4 per group)",
	"action.maximize":              "Maximize",
	"action.restore":               "Restore",
	"action.new":                   "New terminal",
	"action.close":                 "Close terminal",
	"resize":                       "Resize terminal drawer",
	"session.label":                "Terminal",
	"sessions.list":                "Terminal sessions",
	"group.label":                  "Group",
	"error.create":                 "Could not start a terminal",
	"error.initialize":             "Could not initialize the terminal: {message}. Close and reopen the terminal to retry.",
	"loading":                      "Initializing terminal…",
	"dismissFeedback":              "Dismiss notification",
	"action.copy":                  "Copy",
	"action.addToChat":             "Add to chat",
	"action.openLink":              "Open link",
	"action.openPath":              "Open file",
}

func sidebarTerminalText(key string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(i18n.Locale())), "en") {
		if v, ok := sidebarTerminalEnCatalog[key]; ok {
			return v
		}
	}
	if v, ok := sidebarTerminalZhCatalog[key]; ok {
		return v
	}
	return key
}

func userTerminalText(key string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(i18n.Locale())), "en") {
		if v, ok := userTerminalEnCatalog[key]; ok {
			return v
		}
	}
	if v, ok := userTerminalZhCatalog[key]; ok {
		return v
	}
	return key
}

func formatSidebarTerminalError(raw string) string {
	trimmed := strings.TrimSpace(raw)
	switch trimmed {
	case "":
		return ""
	case "missing_terminal", "TERMINAL_NOT_FOUND":
		return sidebarTerminalText("missingTerminal")
	case "input_full", "TERMINAL_INPUT_FULL":
		return sidebarTerminalText("inputFull")
	case "attachment_ended", "TERMINAL_ATTACHMENT_ENDED":
		return sidebarTerminalText("attachmentEnded")
	case "invalid_output", "TERMINAL_INVALID_OUTPUT":
		return sidebarTerminalText("invalidOutput")
	case "terminal_limit", "TERMINAL_LIMIT_REACHED":
		return sidebarTerminalText("terminalLimit")
	}
	return trimmed
}

func formatSidebarTerminalCleanupFailed(title, message string) string {
	tpl := sidebarTerminalText("cleanupFailed")
	tpl = strings.ReplaceAll(tpl, "{title}", title)
	return strings.ReplaceAll(tpl, "{message}", message)
}

func formatSidebarTerminalRecoveryFailed(message string) string {
	return strings.ReplaceAll(sidebarTerminalText("recoveryFailed"), "{message}", message)
}

func formatSidebarTerminalFailed(message string) string {
	return strings.ReplaceAll(sidebarTerminalText("failed"), "{message}", message)
}

func formatSidebarTerminalExited(code int) string {
	return strings.ReplaceAll(sidebarTerminalText("exited"), "{code}", fmt.Sprintf("%d", code))
}

func formatUserTerminalInitError(message string) string {
	return strings.ReplaceAll(userTerminalText("error.initialize"), "{message}", message)
}
