package workspacepicker

import (
	"fmt"
	"strings"

	"whaleisle/internal/ui/i18n"
)

type localePair struct {
	ZH string
	EN string
}

func resolveLocalePair(pair localePair) string {
	if i18n.Locale() == i18n.LocaleZH {
		if pair.ZH != "" {
			return pair.ZH
		}
		return pair.EN
	}
	if pair.EN != "" {
		return pair.EN
	}
	return pair.ZH
}

func formatLocaleTemplate(tmpl string, vars map[string]string) string {
	out := tmpl
	for k, v := range vars {
		out = strings.ReplaceAll(out, "{"+k+"}", v)
	}
	return out
}

// workspaceLocaleCatalog mirrors @deepseek-ai/dsh-client-ui-workspace locales.js.
var workspaceLocaleCatalog = map[string]localePair{
	"draft.initializationFailed": {
		ZH: "暂时无法填入草稿，请稍后重试",
		EN: "Could not fill the draft. Please try again shortly.",
	},
	"defaultWorkspace.failed": {
		ZH: "无法创建默认工作区，请通过“选择工作区”选择文件夹",
		EN: "Unable to create default workspace. Use Choose workspace to select a folder.",
	},
	"shortcut.noSession": {
		ZH: "请先选择一个会话",
		EN: "Select a session first",
	},
	"shortcut.noPicker": {
		ZH: "目录选择器不可用",
		EN: "Directory picker unavailable",
	},
	"shortcut.directoryBusy": {
		ZH: "正在选择或添加工作区",
		EN: "Selecting or adding a workspace",
	},
	"shortcut.noCompletedTurn": {
		ZH: "当前会话没有已结束的轮次",
		EN: "This session has no completed turn",
	},
	"shortcut.forkFailed": {
		ZH: "无法分叉会话，请重试",
		EN: "Could not fork the session. Try again.",
	},
	"shortcut.projectActionUnavailable": {
		ZH: "项目操作暂不可用，请在项目侧栏中操作",
		EN: "Project actions are unavailable. Use the Project sidebar.",
	},
	"shortcut.projectForkUnavailable": {
		ZH: "项目需要持续使用同一主对话，不能创建分支",
		EN: "A Project keeps one continuing main conversation and cannot be branched.",
	},
	"selection.error.active": {
		ZH: "会话仍有正在进行或计划中的工作，未完成此次操作。",
		EN: "This session still has running or scheduled work. The operation was not completed.",
	},
	"empty.noMatches": {
		ZH: "无匹配结果",
		EN: "No matches",
	},
	"folderError.title": {
		ZH: "无法打开文件夹",
		EN: "Couldn’t open folder",
	},
	"delete.desc": {
		ZH: "将把“{name}”从工作区列表中移除。文件夹与会话记录会保留，但其会话将不再显示；重新添加该工作目录后会话会回来。",
		EN: "This removes “{name}” from the workspace list. The folder and session logs will be kept, but its sessions disappear from the sidebar until you add this folder again.",
	},
	"delete.pending": {
		ZH: "正在删除工作区…",
		EN: "Deleting workspace…",
	},
	"toast.sessionIdCopied": {
		ZH: "Session ID 已复制",
		EN: "Session ID copied",
	},
	"toast.sessionIdCopyFailed": {
		ZH: "复制失败，请重试",
		EN: "Copy failed. Try again.",
	},
	"toast.archivedNotOpenable": {
		ZH: "已归档对话暂时无法查看，请取消归档后查看",
		EN: "Archived sessions cannot be opened. Unarchive it to view.",
	},
	"toast.archived": {
		ZH: "会话已归档，可",
		EN: "Session archived. You can ",
	},
	"toast.stoppedAndArchived": {
		ZH: "已停止并归档，可",
		EN: "Session stopped and archived. You can ",
	},
	"toast.archivedFilter": {
		ZH: "查看全部会话",
		EN: "view all sessions",
	},
	"toast.pinFailed": {
		ZH: "置顶失败，请稍后重试",
		EN: "Pin failed. Try again later.",
	},
	"toast.unpinFailed": {
		ZH: "取消置顶失败，请稍后重试",
		EN: "Unpin failed. Try again later.",
	},
	"toast.createFailed": {
		ZH: "新建会话失败：{message}",
		EN: "New session failed: {message}",
	},
	"toast.forkRequiresOpen": {
		ZH: "请先打开原会话完成迁移，再创建分支",
		EN: "Open the original session to complete migration before creating a branch.",
	},
	"status.subagentsRunning.one": {
		ZH: "{n} 个子智能体运行中",
		EN: "{n} subagent running",
	},
	"status.subagentsRunning.other": {
		ZH: "{n} 个子智能体运行中",
		EN: "{n} subagents running",
	},
	"date.ymd": {
		ZH: "{y}年{m}月{d}日",
		EN: "{y}-{m}-{d}",
	},
	"time.days": {
		ZH: "{n}天",
		EN: "{n}d",
	},
	"time.years": {
		ZH: "{n}年",
		EN: "{n}y",
	},
	"time.ago": {
		ZH: "{t}前",
		EN: "{t} ago",
	},
}

// WorkspaceLocaleText returns the localized text for a dsh-client-ui-workspace key.
func WorkspaceLocaleText(key string) string {
	if pair, ok := workspaceLocaleCatalog[strings.TrimSpace(key)]; ok {
		return resolveLocalePair(pair)
	}
	return ""
}

// FormatWorkspaceLocale formats a dsh-client-ui-workspace locale template with named variables.
func FormatWorkspaceLocale(key string, vars map[string]string) string {
	return formatLocaleTemplate(WorkspaceLocaleText(key), vars)
}

// FormatWorkspaceDeleteDescription formats the workspace removal confirmation body for a named workspace.
func FormatWorkspaceDeleteDescription(name string) string {
	return FormatWorkspaceLocale("delete.desc", map[string]string{
		"name": strings.TrimSpace(name),
	})
}

// FormatWorkspaceCreateFailed formats the new-session creation failure toast message.
func FormatWorkspaceCreateFailed(message string) string {
	return FormatWorkspaceLocale("toast.createFailed", map[string]string{
		"message": strings.TrimSpace(message),
	})
}

// FormatWorkspaceSubagentsRunning formats the running subagents badge text.
func FormatWorkspaceSubagentsRunning(count int) string {
	if count < 1 {
		count = 1
	}
	key := "status.subagentsRunning.other"
	if count == 1 {
		key = "status.subagentsRunning.one"
	}
	return FormatWorkspaceLocale(key, map[string]string{
		"n": fmt.Sprintf("%d", count),
	})
}

// FormatWorkspaceDateYMD formats a calendar date using the workspace locale template.
func FormatWorkspaceDateYMD(year, month, day int) string {
	return FormatWorkspaceLocale("date.ymd", map[string]string{
		"y": fmt.Sprintf("%04d", year),
		"m": fmt.Sprintf("%d", month),
		"d": fmt.Sprintf("%d", day),
	})
}

// FormatWorkspaceRelativeAge formats a relative age in days or years with the "ago" suffix.
func FormatWorkspaceRelativeAge(days int) string {
	if days < 1 {
		days = 1
	}
	var token string
	if days >= 365 {
		years := days / 365
		if years < 1 {
			years = 1
		}
		token = FormatWorkspaceLocale("time.years", map[string]string{"n": fmt.Sprintf("%d", years)})
	} else {
		token = FormatWorkspaceLocale("time.days", map[string]string{"n": fmt.Sprintf("%d", days)})
	}
	return FormatWorkspaceLocale("time.ago", map[string]string{"t": token})
}
