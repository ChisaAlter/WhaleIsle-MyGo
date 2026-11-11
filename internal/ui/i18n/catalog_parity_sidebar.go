package i18n

import "strings"

// 本文件镜像原版 Whale Isle 侧栏/工作区区域各功能包的文案：
//   - ui-workspace/lib/types/client/locales.js（整本词典，key 统一加 "ws." 前缀）
//   - ui-sidebar-right/lib/types/client/locales.js 中 MyGo 之前缺失的条目
//
// 措辞以原版为准（zh 为主，en 一并保留），调用点通过 WS / WSf 取词。

// WSf 取 ui-workspace 词条并把模板里的 {name} 形式占位符替换为 kv 给出的值。
// kv 以 "name", "值", "n", "3" 的形式成对给出。
func WSf(key string, kv ...string) string {
	text := T(key)
	for i := 0; i+1 < len(kv); i += 2 {
		text = strings.ReplaceAll(text, "{"+kv[i]+"}", kv[i+1])
	}
	return text
}

// WSPlural 按原版 .one / .other 复数规则取词：n == 1 取 key+".one"，其余取 key+".other"。
func WSPlural(key string, n int, kv ...string) string {
	suffix := ".other"
	if n == 1 {
		suffix = ".one"
	}
	return WSf(key+suffix, kv...)
}

const (
	KeyWSDraftInitializationFailed        = "ws.draft.initializationFailed"
	KeyWSDefaultWorkspaceFailed           = "ws.defaultWorkspace.failed"
	KeyWSDefaultWorkspaceTitle            = "ws.defaultWorkspace.title"
	KeyWSGroupUngrouped                   = "ws.group.ungrouped"
	KeyWSSessionNew                       = "ws.session.new"
	KeyWSSessionUntitled                  = "ws.session.untitled"
	KeyWSShortcutNoSession                = "ws.shortcut.noSession"
	KeyWSShortcutNoPicker                 = "ws.shortcut.noPicker"
	KeyWSShortcutDirectoryBusy            = "ws.shortcut.directoryBusy"
	KeyWSShortcutNoCompletedTurn          = "ws.shortcut.noCompletedTurn"
	KeyWSShortcutForkFailed               = "ws.shortcut.forkFailed"
	KeyWSShortcutProjectActionUnavailable = "ws.shortcut.projectActionUnavailable"
	KeyWSShortcutProjectForkUnavailable   = "ws.shortcut.projectForkUnavailable"
	KeyWSSectionWorkspaces                = "ws.section.workspaces"
	KeyWSSectionSessions                  = "ws.section.sessions"
	KeyWSSectionPinned                    = "ws.section.pinned"
	KeyWSSectionArchived                  = "ws.section.archived"
	KeyWSArchivedMissingTitle             = "ws.archived.missingTitle"
	KeyWSSettingsShowArchivedTitle        = "ws.settings.showArchived.title"
	KeyWSSettingsShowArchivedDescription  = "ws.settings.showArchived.description"
	KeyWSViewOptionsLabel                 = "ws.viewOptions.label"
	KeyWSGroupByLabel                     = "ws.groupBy.label"
	KeyWSGroupByWorkspace                 = "ws.groupBy.workspace"
	KeyWSGroupByWorkspaceTree             = "ws.groupBy.workspaceTree"
	KeyWSGroupByFlat                      = "ws.groupBy.flat"
	KeyWSOrderByLabel                     = "ws.orderBy.label"
	KeyWSOrderByManual                    = "ws.orderBy.manual"
	KeyWSOrderByUpdated                   = "ws.orderBy.updated"
	KeyWSFilterByLabel                    = "ws.filterBy.label"
	KeyWSViewOptionsHideArchived          = "ws.viewOptions.hideArchived"
	KeyWSViewOptionsShowArchived          = "ws.viewOptions.showArchived"
	KeyWSViewOptionsOnlyArchived          = "ws.viewOptions.onlyArchived"
	KeyWSSessionsExpand                   = "ws.sessions.expand"
	KeyWSSessionsCollapse                 = "ws.sessions.collapse"
	KeyWSSelectionEnter                   = "ws.selection.enter"
	KeyWSSelectionExit                    = "ws.selection.exit"
	KeyWSSelectionToolbarAria             = "ws.selection.toolbar.aria"
	KeyWSSelectionSessionAria             = "ws.selection.session.aria"
	KeyWSSelectionCount                   = "ws.selection.count"
	KeyWSSelectionAllVisible              = "ws.selection.allVisible"
	KeyWSSelectionClear                   = "ws.selection.clear"
	KeyWSSelectionArchive                 = "ws.selection.archive"
	KeyWSSelectionDelete                  = "ws.selection.delete"
	KeyWSSelectionConfirmTargets          = "ws.selection.confirm.targets"
	KeyWSSelectionConfirmArchiveTitle     = "ws.selection.confirm.archive.title"
	KeyWSSelectionConfirmArchiveDesc      = "ws.selection.confirm.archive.desc"
	KeyWSSelectionConfirmArchiveAction    = "ws.selection.confirm.archive.action"
	KeyWSSelectionConfirmDeleteTitle      = "ws.selection.confirm.delete.title"
	KeyWSSelectionConfirmDeleteDesc       = "ws.selection.confirm.delete.desc"
	KeyWSSelectionConfirmDeleteAction     = "ws.selection.confirm.delete.action"
	KeyWSSelectionResultArchived          = "ws.selection.result.archived"
	KeyWSSelectionResultDeleted           = "ws.selection.result.deleted"
	KeyWSSelectionResultFailed            = "ws.selection.result.failed"
	KeyWSSelectionErrorActive             = "ws.selection.error.active"
	KeyWSEmptyNone                        = "ws.empty.none"
	KeyWSEmptyNoneArchived                = "ws.empty.noneArchived"
	KeyWSEmptyViewOthers                  = "ws.empty.viewOthers"
	KeyWSEmptyNoMatches                   = "ws.empty.noMatches"
	KeyWSWorkspaceAdd                     = "ws.workspace.add"
	KeyWSWorkspaceReveal                  = "ws.workspace.reveal"
	KeyWSWorkspaceRevealFailed            = "ws.workspace.revealFailed"
	KeyWSSearchSessionsAria               = "ws.search.sessions.aria"
	KeyWSSearchPlaceholder                = "ws.search.placeholder"
	KeyWSSearchClear                      = "ws.search.clear"
	KeyWSSearchResultsAria                = "ws.search.results.aria"
	KeyWSSearchPending                    = "ws.search.pending"
	KeyWSSearchNoMatches                  = "ws.search.noMatches"
	KeyWSSearchHasMore                    = "ws.search.hasMore"
	KeyWSMenuAddWorkspace                 = "ws.menu.addWorkspace"
	KeyWSMenuNoDirectory                  = "ws.menu.noDirectory"
	KeyWSPickerLoading                    = "ws.picker.loading"
	KeyWSConflictNamed                    = "ws.conflict.named"
	KeyWSFolderErrorTitle                 = "ws.folderError.title"
	KeyWSFolderErrorRetry                 = "ws.folderError.retry"
	KeyWSRename                           = "ws.rename"
	KeyWSRenameWorkspaceTitle             = "ws.rename.workspace.title"
	KeyWSRenameSessionTitle               = "ws.rename.session.title"
	KeyWSFieldWorkspaceName               = "ws.field.workspaceName"
	KeyWSFieldSessionName                 = "ws.field.sessionName"
	KeyWSDeleteWorkspace                  = "ws.delete.workspace"
	KeyWSDeleteDesc                       = "ws.delete.desc"
	KeyWSDeletePending                    = "ws.delete.pending"
	KeyWSMenuFork                         = "ws.menu.fork"
	KeyWSMenuArchiveSession               = "ws.menu.archiveSession"
	KeyWSMenuUnarchiveSession             = "ws.menu.unarchiveSession"
	KeyWSMenuDeleteSession                = "ws.menu.deleteSession"
	KeyWSDeleteSessionTitle               = "ws.delete.session.title"
	KeyWSDeleteSessionDesc                = "ws.delete.session.desc"
	KeyWSDeleteSessionPending             = "ws.delete.session.pending"
	KeyWSDeleteSessionRunning             = "ws.delete.session.running"
	KeyWSDeleteSessionNotArchived         = "ws.delete.session.notArchived"
	KeyWSDeleteSessionNotFound            = "ws.delete.session.notFound"
	KeyWSDeleteSessionLiveUnowned         = "ws.delete.session.liveUnowned"
	KeyWSDeleteSessionPartial             = "ws.delete.session.partial"
	KeyWSDeleteSessionIncomplete          = "ws.delete.session.incomplete"
	KeyWSDeleteSessionInProgress          = "ws.delete.session.inProgress"
	KeyWSMenuPinSession                   = "ws.menu.pinSession"
	KeyWSMenuUnpinSession                 = "ws.menu.unpinSession"
	KeyWSMenuSessionId                    = "ws.menu.sessionId"
	KeyWSToastSessionIdCopied             = "ws.toast.sessionIdCopied"
	KeyWSToastSessionIdCopyFailed         = "ws.toast.sessionIdCopyFailed"
	KeyWSRowArchived                      = "ws.row.archived"
	KeyWSRowPinned                        = "ws.row.pinned"
	KeyWSToastArchivedNotOpenable         = "ws.toast.archivedNotOpenable"
	KeyWSToastArchived                    = "ws.toast.archived"
	KeyWSToastStoppedAndArchived          = "ws.toast.stoppedAndArchived"
	KeyWSArchiveConfirmTitle              = "ws.archive.confirm.title"
	KeyWSArchiveConfirmDesc               = "ws.archive.confirm.desc"
	KeyWSArchiveConfirmActivity           = "ws.archive.confirm.activity"
	KeyWSArchiveConfirmTurn               = "ws.archive.confirm.turn"
	KeyWSArchiveConfirmSubagentsOne       = "ws.archive.confirm.subagents.one"
	KeyWSArchiveConfirmSubagentsOther     = "ws.archive.confirm.subagents.other"
	KeyWSArchiveConfirmJobsOne            = "ws.archive.confirm.jobs.one"
	KeyWSArchiveConfirmJobsOther          = "ws.archive.confirm.jobs.other"
	KeyWSArchiveConfirmSchedulesOne       = "ws.archive.confirm.schedules.one"
	KeyWSArchiveConfirmSchedulesOther     = "ws.archive.confirm.schedules.other"
	KeyWSArchiveConfirmOtherOne           = "ws.archive.confirm.other.one"
	KeyWSArchiveConfirmOtherOther         = "ws.archive.confirm.other.other"
	KeyWSArchiveConfirmListSeparator      = "ws.archive.confirm.listSeparator"
	KeyWSArchiveConfirmAction             = "ws.archive.confirm.action"
	KeyWSArchiveConfirmPending            = "ws.archive.confirm.pending"
	KeyWSToastArchivedUndo                = "ws.toast.archivedUndo"
	KeyWSToastArchivedOr                  = "ws.toast.archivedOr"
	KeyWSToastArchivedFilter              = "ws.toast.archivedFilter"
	KeyWSToastPinFailed                   = "ws.toast.pinFailed"
	KeyWSToastUnpinFailed                 = "ws.toast.unpinFailed"
	KeyWSToastCreateFailed                = "ws.toast.createFailed"
	KeyWSToastForkRequiresOpen            = "ws.toast.forkRequiresOpen"
	KeyWSToastOpenForkSource              = "ws.toast.openForkSource"
	KeyWSSessionsCountOne                 = "ws.sessions.count.one"
	KeyWSSessionsCountOther               = "ws.sessions.count.other"
	KeyWSActionsWorkspaceAria             = "ws.actions.workspace.aria"
	KeyWSActionsSessionAria               = "ws.actions.session.aria"
	KeyWSActionsArchive                   = "ws.actions.archive"
	KeyWSActionsUnarchive                 = "ws.actions.unarchive"
	KeyWSActionsPin                       = "ws.actions.pin"
	KeyWSActionsUnpin                     = "ws.actions.unpin"
	KeyWSActionsNewSession                = "ws.actions.newSession"
	KeyWSActionsNewSessionAria            = "ws.actions.newSession.aria"
	KeyWSStatusRunning                    = "ws.status.running"
	KeyWSStatusSubagentsRunningOne        = "ws.status.subagentsRunning.one"
	KeyWSStatusSubagentsRunningOther      = "ws.status.subagentsRunning.other"
	KeyWSStatusIdle                       = "ws.status.idle"
	KeyWSStatusWaitingApproval            = "ws.status.waitingApproval"
	KeyWSStatusPlanReview                 = "ws.status.planReview"
	KeyWSStatusWaitingAnswer              = "ws.status.waitingAnswer"
	KeyWSStatusCompactApproval            = "ws.status.compact.approval"
	KeyWSStatusCompactPlanReview          = "ws.status.compact.planReview"
	KeyWSStatusCompactAnswer              = "ws.status.compact.answer"
	KeyWSStatusCompleted                  = "ws.status.completed"
	KeyWSHoverCreated                     = "ws.hover.created"
	KeyWSHoverCopied                      = "ws.hover.copied"
	KeyWSDateYmd                          = "ws.date.ymd"
	KeyWSTimeNow                          = "ws.time.now"
	KeyWSTimeMinutes                      = "ws.time.minutes"
	KeyWSTimeHours                        = "ws.time.hours"
	KeyWSTimeDays                         = "ws.time.days"
	KeyWSTimeMonths                       = "ws.time.months"
	KeyWSTimeYears                        = "ws.time.years"
	KeyWSTimeAgo                          = "ws.time.ago"
	// 复数词条的基底 key（WSPlural 追加 .one / .other）。
	KeyWSSessionsCountBase           = "ws.sessions.count"
	KeyWSArchiveConfirmSubagentsBase = "ws.archive.confirm.subagents"
	KeyWSArchiveConfirmJobsBase      = "ws.archive.confirm.jobs"
	KeyWSArchiveConfirmSchedulesBase = "ws.archive.confirm.schedules"
	KeyWSArchiveConfirmOtherBase     = "ws.archive.confirm.other"
	KeyWSStatusSubagentsRunningBase  = "ws.status.subagentsRunning"

	// ui-sidebar-right：tab.guide.more
	KeySidebarRightTabGuideMore = "sidebar_right.tab.guide.more"
)

func init() {
	Register(KeyWSDraftInitializationFailed, "暂时无法填入草稿，请稍后重试", "Could not fill the draft. Please try again shortly.")
	Register(KeyWSDefaultWorkspaceFailed, "无法创建默认工作区，请通过“选择工作区”选择文件夹", "Unable to create default workspace. Use Choose workspace to select a folder.")
	Register(KeyWSDefaultWorkspaceTitle, "默认工作区", "Default workspace")
	Register(KeyWSGroupUngrouped, "无工作目录", "No workspace folder")
	Register(KeyWSSessionNew, "新会话", "New Session")
	Register(KeyWSSessionUntitled, "未命名", "Untitled")
	Register(KeyWSShortcutNoSession, "请先选择一个会话", "Select a session first")
	Register(KeyWSShortcutNoPicker, "目录选择器不可用", "Directory picker unavailable")
	Register(KeyWSShortcutDirectoryBusy, "正在选择或添加工作区", "Selecting or adding a workspace")
	Register(KeyWSShortcutNoCompletedTurn, "当前会话没有已结束的轮次", "This session has no completed turn")
	Register(KeyWSShortcutForkFailed, "无法分叉会话，请重试", "Could not fork the session. Try again.")
	Register(KeyWSShortcutProjectActionUnavailable, "项目操作暂不可用，请在项目侧栏中操作", "Project actions are unavailable. Use the Project sidebar.")
	Register(KeyWSShortcutProjectForkUnavailable, "项目需要持续使用同一主对话，不能创建分支", "A Project keeps one continuing main conversation and cannot be branched.")
	Register(KeyWSSectionWorkspaces, "工作区", "Workspaces")
	Register(KeyWSSectionSessions, "会话", "Sessions")
	Register(KeyWSSectionPinned, "已置顶", "Pinned")
	Register(KeyWSSectionArchived, "已归档", "Archived")
	Register(KeyWSArchivedMissingTitle, "缺失会话", "Missing session")
	Register(KeyWSSettingsShowArchivedTitle, "显示底部归档管理区", "Show bottom archive section")
	Register(KeyWSSettingsShowArchivedDescription, "选择“隐藏已归档”时，在侧栏底部显示可折叠的“已归档”管理区。关闭后仍可从视图选项查看全部或仅归档会话，并取消归档或删除。", "Show the collapsible Archived section at the bottom of the sidebar when Hide archived is selected. When off, use View options to show all or only archived sessions and restore or delete them.")
	Register(KeyWSViewOptionsLabel, "视图选项", "View options")
	Register(KeyWSGroupByLabel, "分组方式", "Group by")
	Register(KeyWSGroupByWorkspace, "按工作区", "WorkSpace")
	Register(KeyWSGroupByWorkspaceTree, "按工作区树", "Workspace Tree")
	Register(KeyWSGroupByFlat, "单列表", "In one list")
	Register(KeyWSOrderByLabel, "排序方式", "Order by")
	Register(KeyWSOrderByManual, "手动排序", "Manual")
	Register(KeyWSOrderByUpdated, "最近更新", "Last updated")
	Register(KeyWSFilterByLabel, "筛选列表与搜索", "Filter list and search")
	Register(KeyWSViewOptionsHideArchived, "隐藏已归档", "Hide archived")
	Register(KeyWSViewOptionsShowArchived, "全部对话（显示已归档）", "All conversations (show archived)")
	Register(KeyWSViewOptionsOnlyArchived, "仅显示已归档", "Archived only")
	Register(KeyWSSessionsExpand, "展开其余 {n} 个会话", "Show {n} more sessions")
	Register(KeyWSSessionsCollapse, "收起", "Show less")
	Register(KeyWSSelectionEnter, "多选会话", "Select sessions")
	Register(KeyWSSelectionExit, "退出多选", "Exit selection")
	Register(KeyWSSelectionToolbarAria, "会话批量操作", "Batch session actions")
	Register(KeyWSSelectionSessionAria, "选择会话“{name}”", "Select session “{name}”")
	Register(KeyWSSelectionCount, "已选 {n} 项", "{n} selected")
	Register(KeyWSSelectionAllVisible, "全选可见", "Select visible")
	Register(KeyWSSelectionClear, "清空", "Clear")
	Register(KeyWSSelectionArchive, "归档", "Archive")
	Register(KeyWSSelectionDelete, "删除", "Delete")
	Register(KeyWSSelectionConfirmTargets, "已选会话", "Selected sessions")
	Register(KeyWSSelectionConfirmArchiveTitle, "归档 {n} 个会话？", "Archive {n} sessions?")
	Register(KeyWSSelectionConfirmArchiveDesc, "将归档所选会话，并停止其中正在进行的回合、子智能体、后台任务和定时任务。已归档的会话会保留；之后可取消归档，但被停止的工作不会自动继续。", "This archives the selected sessions and stops their running turns, subagents, background jobs, and scheduled tasks. Already archived sessions are kept. You can unarchive later, but stopped work will not resume automatically.")
	Register(KeyWSSelectionConfirmArchiveAction, "停止并归档", "Stop and archive")
	Register(KeyWSSelectionConfirmDeleteTitle, "永久删除 {n} 个会话？", "Permanently delete {n} sessions?")
	Register(KeyWSSelectionConfirmDeleteDesc, "将永久删除所选会话及其关联子会话的对话记录，无法恢复。工作区文件不受影响。运行中的回合须先停止；无法删除的会话会保留，并在结果中列出原因。", "This permanently deletes the selected conversation logs and their linked subagent sessions. This cannot be undone. Workspace files are unchanged. Running turns must be stopped first. Sessions that cannot be deleted will be kept with their reasons listed in the results.")
	Register(KeyWSSelectionConfirmDeleteAction, "永久删除", "Delete permanently")
	Register(KeyWSSelectionResultArchived, "已归档 {n} 个会话", "Archived {n} sessions")
	Register(KeyWSSelectionResultDeleted, "已删除 {n} 个会话", "Deleted {n} sessions")
	Register(KeyWSSelectionResultFailed, "{n} 个会话未完成", "{n} sessions unfinished")
	Register(KeyWSSelectionErrorActive, "会话仍有正在进行或计划中的工作，未完成此次操作。", "This session still has running or scheduled work. The operation was not completed.")
	Register(KeyWSEmptyNone, "暂无会话", "No sessions yet")
	Register(KeyWSEmptyNoneArchived, "暂无已归档会话", "No archived sessions yet")
	Register(KeyWSEmptyViewOthers, "查看其他会话", "View other sessions")
	Register(KeyWSEmptyNoMatches, "无匹配结果", "No matches")
	Register(KeyWSWorkspaceAdd, "添加工作区", "Add workspace")
	Register(KeyWSWorkspaceReveal, "在 Finder 中打开工作区", "Open workspace in Finder")
	Register(KeyWSWorkspaceRevealFailed, "无法打开工作区文件夹，请确认目录仍存在后重试", "Could not open the workspace folder. Check that it still exists and try again.")
	Register(KeyWSSearchSessionsAria, "搜索会话", "Search sessions")
	Register(KeyWSSearchPlaceholder, "搜索会话名称", "Search session names")
	Register(KeyWSSearchClear, "清除搜索", "Clear search")
	Register(KeyWSSearchResultsAria, "搜索结果", "Search results")
	Register(KeyWSSearchPending, "正在搜索会话历史…", "Searching session history…")
	Register(KeyWSSearchNoMatches, "无匹配会话", "No matching sessions")
	Register(KeyWSSearchHasMore, "仅显示前 {n} 条结果，请缩小搜索范围。", "Showing the first {n} results. Narrow your search.")
	Register(KeyWSMenuAddWorkspace, "添加工作区…", "Add workspace…")
	Register(KeyWSMenuNoDirectory, "无工作目录", "No workspace folder")
	Register(KeyWSPickerLoading, "正在加载工作区…", "Loading workspaces…")
	Register(KeyWSConflictNamed, "已存在名为“{name}”的工作区。", "A workspace named “{name}” already exists.")
	Register(KeyWSFolderErrorTitle, "无法打开文件夹", "Couldn’t open folder")
	Register(KeyWSFolderErrorRetry, "重新选择", "Choose again")
	Register(KeyWSRename, "重命名", "Rename")
	Register(KeyWSRenameWorkspaceTitle, "重命名工作区", "Rename workspace")
	Register(KeyWSRenameSessionTitle, "重命名会话", "Rename session")
	Register(KeyWSFieldWorkspaceName, "工作区名称", "Workspace name")
	Register(KeyWSFieldSessionName, "会话名称", "Session name")
	Register(KeyWSDeleteWorkspace, "删除工作区", "Delete workspace")
	Register(KeyWSDeleteDesc, "将把“{name}”从工作区列表中移除。文件夹与会话记录会保留，但其会话将不再显示；重新添加该工作目录后会话会回来。", "This removes “{name}” from the workspace list. The folder and session logs will be kept, but its sessions disappear from the sidebar until you add this folder again.")
	Register(KeyWSDeletePending, "正在删除工作区…", "Deleting workspace…")
	Register(KeyWSMenuFork, "分叉会话", "Fork session")
	Register(KeyWSMenuArchiveSession, "归档会话", "Archive session")
	Register(KeyWSMenuUnarchiveSession, "取消归档", "Unarchive session")
	Register(KeyWSMenuDeleteSession, "删除会话", "Delete session")
	Register(KeyWSDeleteSessionTitle, "删除会话", "Delete session")
	Register(KeyWSDeleteSessionDesc, "将永久删除「{name}」的对话记录，无法恢复。工作区文件夹不受影响。", "This permanently deletes the conversation log for \"{name}\". It cannot be recovered. The workspace folder is unchanged.")
	Register(KeyWSDeleteSessionPending, "正在删除会话…", "Deleting session…")
	Register(KeyWSDeleteSessionRunning, "会话仍在运行，请先停止后再删除。", "The session is still running. Stop it before deleting.")
	Register(KeyWSDeleteSessionNotArchived, "只能删除已归档的会话。", "Only archived sessions can be deleted.")
	Register(KeyWSDeleteSessionNotFound, "找不到该会话。", "That session could not be found.")
	Register(KeyWSDeleteSessionLiveUnowned, "会话仍在运行且无法安全结束，请重试或重启后再删。", "The session is still live and could not be shut down safely. Retry or restart, then delete.")
	Register(KeyWSDeleteSessionPartial, "部分子会话已删除，但根会话仍在。请重试删除。", "Some child sessions were deleted, but the root remains. Retry the delete.")
	Register(KeyWSDeleteSessionIncomplete, "会话日志已删除，但归档状态未能完全更新。请刷新后检查。", "The conversation log was deleted, but archive state did not fully update. Refresh and check.")
	Register(KeyWSDeleteSessionInProgress, "该会话正在删除中，请稍后再试。", "That session is already being deleted. Try again in a moment.")
	Register(KeyWSMenuPinSession, "置顶会话", "Pin session")
	Register(KeyWSMenuUnpinSession, "取消置顶", "Unpin session")
	Register(KeyWSMenuSessionId, "复制 Session ID", "Copy Session ID")
	Register(KeyWSToastSessionIdCopied, "Session ID 已复制", "Session ID copied")
	Register(KeyWSToastSessionIdCopyFailed, "复制失败，请重试", "Copy failed. Try again.")
	Register(KeyWSRowArchived, "已归档", "Archived")
	Register(KeyWSRowPinned, "已置顶", "Pinned")
	Register(KeyWSToastArchivedNotOpenable, "已归档对话暂时无法查看，请取消归档后查看", "Archived sessions cannot be opened. Unarchive it to view.")
	Register(KeyWSToastArchived, "会话已归档，可", "Session archived. You can ")
	Register(KeyWSToastStoppedAndArchived, "已停止并归档，可", "Session stopped and archived. You can ")
	Register(KeyWSArchiveConfirmTitle, "停止并归档此会话？", "Stop and archive this session?")
	Register(KeyWSArchiveConfirmDesc, "“{title}”仍有正在进行的工作。归档会先停止这些工作；之后可在侧栏筛选“全部对话（显示已归档）”中恢复会话，被停止的工作不会自动继续。", "“{title}” still has work in progress. Archiving stops it first; you can restore the session later from the “All conversations (show archived)” filter in the sidebar, and the stopped work will not resume on its own.")
	Register(KeyWSArchiveConfirmActivity, "将被停止的工作", "Work that will be stopped")
	Register(KeyWSArchiveConfirmTurn, "进行中的回合", "The turn in progress")
	Register(KeyWSArchiveConfirmSubagentsOne, "{n} 个运行中的子智能体：{names}", "{n} running subagent: {names}")
	Register(KeyWSArchiveConfirmSubagentsOther, "{n} 个运行中的子智能体：{names}", "{n} running subagents: {names}")
	Register(KeyWSArchiveConfirmJobsOne, "{n} 个后台任务：{names}", "{n} background job: {names}")
	Register(KeyWSArchiveConfirmJobsOther, "{n} 个后台任务：{names}", "{n} background jobs: {names}")
	Register(KeyWSArchiveConfirmSchedulesOne, "{n} 条定时提醒：{names}", "{n} scheduled reminder: {names}")
	Register(KeyWSArchiveConfirmSchedulesOther, "{n} 条定时提醒：{names}", "{n} scheduled reminders: {names}")
	Register(KeyWSArchiveConfirmOtherOne, "{n} 项其他工作（{kind}）", "{n} other item of work ({kind})")
	Register(KeyWSArchiveConfirmOtherOther, "{n} 项其他工作（{kind}）", "{n} other items of work ({kind})")
	Register(KeyWSArchiveConfirmListSeparator, "、", ", ")
	Register(KeyWSArchiveConfirmAction, "停止并归档", "Stop and archive")
	Register(KeyWSArchiveConfirmPending, "正在停止并归档…", "Stopping and archiving…")
	Register(KeyWSToastArchivedUndo, "撤销", "undo")
	Register(KeyWSToastArchivedOr, "或", " or ")
	Register(KeyWSToastArchivedFilter, "查看全部会话", "view all sessions")
	Register(KeyWSToastPinFailed, "置顶失败，请稍后重试", "Pin failed. Try again later.")
	Register(KeyWSToastUnpinFailed, "取消置顶失败，请稍后重试", "Unpin failed. Try again later.")
	Register(KeyWSToastCreateFailed, "新建会话失败：{message}", "New session failed: {message}")
	Register(KeyWSToastForkRequiresOpen, "请先打开原会话完成迁移，再创建分支", "Open the original session to complete migration before creating a branch.")
	Register(KeyWSToastOpenForkSource, "打开原会话", "Open original session")
	Register(KeyWSSessionsCountOne, "{n} 个会话", "{n} session")
	Register(KeyWSSessionsCountOther, "{n} 个会话", "{n} sessions")
	Register(KeyWSActionsWorkspaceAria, "工作区“{name}”的操作", "Workspace actions for {name}")
	Register(KeyWSActionsSessionAria, "会话“{name}”的操作", "Session actions for {name}")
	Register(KeyWSActionsArchive, "归档会话", "Archive")
	Register(KeyWSActionsUnarchive, "取消归档", "Unarchive")
	Register(KeyWSActionsPin, "置顶会话", "Pin")
	Register(KeyWSActionsUnpin, "取消置顶", "Unpin")
	Register(KeyWSActionsNewSession, "新会话", "New session")
	Register(KeyWSActionsNewSessionAria, "在“{name}”中新建会话", "New session in {name}")
	Register(KeyWSStatusRunning, "进行中", "Running")
	Register(KeyWSStatusSubagentsRunningOne, "{n} 个子智能体运行中", "{n} subagent running")
	Register(KeyWSStatusSubagentsRunningOther, "{n} 个子智能体运行中", "{n} subagents running")
	Register(KeyWSStatusIdle, "空闲", "Idle")
	Register(KeyWSStatusWaitingApproval, "等待审批", "Waiting for approval")
	Register(KeyWSStatusPlanReview, "计划待审", "Plan awaiting review")
	Register(KeyWSStatusWaitingAnswer, "等待回答", "Waiting for answer")
	Register(KeyWSStatusCompactApproval, "待审批", "Approval")
	Register(KeyWSStatusCompactPlanReview, "计划待审", "Plan review")
	Register(KeyWSStatusCompactAnswer, "待回答", "Answer")
	Register(KeyWSStatusCompleted, "已完成", "Completed")
	Register(KeyWSHoverCreated, "创建于 {time}", "Created {time}")
	Register(KeyWSHoverCopied, "已复制", "Copied")
	Register(KeyWSDateYmd, "{y}年{m}月{d}日", "{y}-{m}-{d}")
	Register(KeyWSTimeNow, "刚刚", "now")
	Register(KeyWSTimeMinutes, "{n}分钟", "{n}min")
	Register(KeyWSTimeHours, "{n}小时", "{n}h")
	Register(KeyWSTimeDays, "{n}天", "{n}d")
	Register(KeyWSTimeMonths, "{n}个月", "{n}mo")
	Register(KeyWSTimeYears, "{n}年", "{n}y")
	Register(KeyWSTimeAgo, "{t}前", "{t} ago")
	Register(KeySidebarRightTabGuideMore, "更多工具", "More tools")
}
