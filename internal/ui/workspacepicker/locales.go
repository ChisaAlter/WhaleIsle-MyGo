package workspacepicker

import (
	"fmt"
	"strconv"
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
	"defaultWorkspace.title": {
		ZH: "默认工作区",
		EN: "Default workspace",
	},
	"group.ungrouped": {
		ZH: "草稿",
		EN: "Scratch",
	},
	"session.new": {
		ZH: "新建会话",
		EN: "New session",
	},
	"session.untitled": {
		ZH: "新会话",
		EN: "New session",
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
	"sessions.expand": {
		ZH: "展开其余 {n} 个会话",
		EN: "Show {n} more sessions",
	},
	"sessions.collapse": {
		ZH: "收起会话",
		EN: "Show less",
	},
	"selection.error.active": {
		ZH: "会话仍有正在进行或计划中的工作，未完成此次操作。",
		EN: "This session still has running or scheduled work. The operation was not completed.",
	},
	"empty.noMatches": {
		ZH: "无匹配结果",
		EN: "No matches",
	},
	"workspace.add": {
		ZH: "添加工作区",
		EN: "Add workspace",
	},
	"workspace.reveal": {
		ZH: "在 Finder 中打开工作区",
		EN: "Open workspace in Finder",
	},
	"workspace.revealFailed": {
		ZH: "无法打开工作区文件夹",
		EN: "Couldn’t open workspace folder",
	},
	"conflict.named": {
		ZH: "已存在名为“{name}”的工作区。",
		EN: "A workspace named “{name}” already exists.",
	},
	"folderError.title": {
		ZH: "无法打开文件夹",
		EN: "Couldn’t open folder",
	},
	"folderError.retry": {
		ZH: "重新选择",
		EN: "Choose again",
	},
	"delete.desc": {
		ZH: "将把“{name}”从工作区列表中移除。文件夹与会话记录会保留，但其会话将不再显示；重新添加该工作目录后会话会回来。",
		EN: "This removes “{name}” from the workspace list. The folder and session logs will be kept, but its sessions disappear from the sidebar until you add this folder again.",
	},
	"delete.pending": {
		ZH: "正在删除工作区…",
		EN: "Deleting workspace…",
	},
	"delete.session.title": {
		ZH: "永久删除此会话？",
		EN: "Permanently delete this session?",
	},
	"delete.session.desc": {
		ZH: "将永久删除“{name}”及其子智能体历史，无法恢复。工作区文件不会变动。",
		EN: "This permanently deletes “{name}” and its subagent history. Workspace files are unchanged.",
	},
	"delete.session.pending": {
		ZH: "正在删除会话…",
		EN: "Deleting session…",
	},
	"delete.session.running": {
		ZH: "会话仍在运行，请先停止当前轮次后再删除。",
		EN: "This session is still running. Stop the active turn before deleting it.",
	},
	"delete.session.notArchived": {
		ZH: "仅已归档的会话可以永久删除。",
		EN: "Only archived sessions can be permanently deleted.",
	},
	"delete.session.notFound": {
		ZH: "未找到该会话，可能已被移除。",
		EN: "Session not found. It may have already been removed.",
	},
	"delete.session.liveUnowned": {
		ZH: "会话仍有未受管的计算活动，请等待其停止后再删除。",
		EN: "This session still has active work. Wait for it to finish before deleting.",
	},
	"delete.session.partial": {
		ZH: "部分关联的记录未能清理，请稍后重试以完成删除。",
		EN: "Some related records could not be removed. Retry to finish deleting.",
	},
	"delete.session.incomplete": {
		ZH: "会话历史尚未完全移除，仍保留在已归档列表中，可再次尝试删除。",
		EN: "Session history was not fully removed. It remains archived so you can retry.",
	},
	"delete.session.inProgress": {
		ZH: "该会话正在删除中，请稍候。",
		EN: "This session is already being deleted.",
	},
	"archive.confirm.title": {
		ZH: "停止运行并归档？",
		EN: "Stop active work and archive?",
	},
	"archive.confirm.desc": {
		ZH: "“{name}”仍有正在进行或计划中的工作。归档会先停止这些工作，会话随后可在已归档中恢复。",
		EN: "“{name}” still has running or scheduled work. Archiving will stop it first; you can unarchive the session later.",
	},
	"archive.confirm.activity": {
		ZH: "将停止的工作",
		EN: "Work that will be stopped",
	},
	"archive.confirm.turn": {
		ZH: "当前对话轮次正在运行",
		EN: "Active conversation turn",
	},
	"archive.confirm.subagents.one": {
		ZH: "1 个运行中的子智能体：{names}",
		EN: "1 running subagent: {names}",
	},
	"archive.confirm.subagents.other": {
		ZH: "{n} 个运行中的子智能体：{names}",
		EN: "{n} running subagents: {names}",
	},
	"archive.confirm.jobs.one": {
		ZH: "1 个运行中的后台任务：{names}",
		EN: "1 running background task: {names}",
	},
	"archive.confirm.jobs.other": {
		ZH: "{n} 个运行中的后台任务：{names}",
		EN: "{n} running background tasks: {names}",
	},
	"archive.confirm.schedules.one": {
		ZH: "1 个定时任务：{names}",
		EN: "1 scheduled task: {names}",
	},
	"archive.confirm.schedules.other": {
		ZH: "{n} 个定时任务：{names}",
		EN: "{n} scheduled tasks: {names}",
	},
	"archive.confirm.other.one": {
		ZH: "还有 1 项",
		EN: "1 more",
	},
	"archive.confirm.other.other": {
		ZH: "还有 {n} 项",
		EN: "{n} more",
	},
	"archive.confirm.listSeparator": {
		ZH: "、",
		EN: ", ",
	},
	"archive.confirm.action": {
		ZH: "停止并归档",
		EN: "Stop and archive",
	},
	"archive.confirm.pending": {
		ZH: "正在停止并归档…",
		EN: "Stopping and archiving…",
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
	"toast.archivedUndo": {
		ZH: "撤销",
		EN: "undo",
	},
	"toast.archivedOr": {
		ZH: "或",
		EN: " or ",
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
	"toast.openForkSource": {
		ZH: "打开会话",
		EN: "Open session",
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
	trimmed := strings.TrimSpace(key)
	if pair, ok := workspaceLocaleCatalog[trimmed]; ok {
		return resolveLocalePair(pair)
	}
	if strings.HasPrefix(trimmed, "ws.") {
		return i18n.T(trimmed)
	}
	if val := i18n.T("ws." + trimmed); val != "ws."+trimmed {
		return val
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

// FormatArchiveConfirmActivityLine formats a running-work item in the archive confirmation dialog.
func FormatArchiveConfirmActivityLine(kind string, names []string) string {
	sep := i18n.T(i18n.KeyWSArchiveConfirmListSeparator)
	joined := strings.Join(names, sep)
	count := len(names)
	if count < 1 {
		count = 1
	}
	switch kind {
	case "turn":
		return i18n.T(i18n.KeyWSArchiveConfirmTurn)
	case "subagent":
		return i18n.WSPlural(i18n.KeyWSArchiveConfirmSubagentsBase, count, "n", strconv.Itoa(count), "names", joined)
	case "job":
		return i18n.WSPlural(i18n.KeyWSArchiveConfirmJobsBase, count, "n", strconv.Itoa(count), "names", joined)
	case "schedule":
		return i18n.WSPlural(i18n.KeyWSArchiveConfirmSchedulesBase, count, "n", strconv.Itoa(count), "names", joined)
	default:
		return i18n.WSPlural(i18n.KeyWSArchiveConfirmOtherBase, count, "n", strconv.Itoa(count))
	}
}

// FormatSessionDeleteError maps a Host session-deletion error code to the localized user message.
func FormatSessionDeleteError(codeOrErr string) string {
	switch strings.TrimSpace(codeOrErr) {
	case "workspace/session-running", "session-running", "SESSION_DELETE_RUNNING":
		return i18n.T(i18n.KeyWSDeleteSessionRunning)
	case "workspace/session-not-archived", "session-not-archived", "SESSION_DELETE_NOT_ARCHIVED":
		return i18n.T(i18n.KeyWSDeleteSessionNotArchived)
	case "workspace/session-not-found", "session-not-found", "SESSION_DELETE_NOT_FOUND":
		return i18n.T(i18n.KeyWSDeleteSessionNotFound)
	case "workspace/session-live-unowned", "session-live-unowned", "SESSION_DELETE_LIVE_UNOWNED":
		return i18n.T(i18n.KeyWSDeleteSessionLiveUnowned)
	case "workspace/session-delete-partial", "session-delete-partial", "SESSION_DELETE_PARTIAL":
		return i18n.T(i18n.KeyWSDeleteSessionPartial)
	case "workspace/session-delete-incomplete", "session-delete-incomplete", "SESSION_DELETE_INCOMPLETE":
		return i18n.T(i18n.KeyWSDeleteSessionIncomplete)
	case "workspace/session-delete-in-progress", "session-delete-in-progress", "SESSION_DELETE_IN_PROGRESS":
		return i18n.T(i18n.KeyWSDeleteSessionInProgress)
	case "workspace/session-active", "session-active", "SESSION_SELECTION_ACTIVE":
		return i18n.T(i18n.KeyWSSelectionErrorActive)
	default:
		return i18n.T(i18n.KeyWSMenuDeleteSession)
	}
}
