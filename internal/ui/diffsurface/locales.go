package diffsurface

import (
	"fmt"
	"strings"
	"whaleisle/internal/ui/i18n"
)

// gitLocaleZH mirrors @deepseek-ai/dsh-client-ui-git/lib/types/client/locales.js (zh).
var gitLocaleZH = map[string]string{
	"menu.options":                    "Git 操作",
	"menu.detached":                   "处于分离 HEAD 状态：请先创建并检出分支，才能执行推送和拉取请求操作。",
	"menu.behind":                     "落后于上游，请先拉取或变基。",
	"commit.title":                    "提交更改",
	"commit.description":              "确认本次提交内容。提交信息留空将自动生成。",
	"commit.message":                  "提交信息（可选）",
	"commit.placeholder":              "留空则自动生成",
	"commit.submit":                   "提交",
	"commit.cancel":                   "取消",
	"commit.onNewRef":                 "在新建分支上提交",
	"commit.branch":                   "分支",
	"commit.detached":                 "（分离 HEAD）",
	"commit.defaultWarning":           "警告：目标为默认分支",
	"commit.files":                    "文件",
	"commit.edit":                     "编辑",
	"commit.done":                     "完成",
	"commit.none":                     "无",
	"commit.excluded":                 "已排除",
	"commit.selectedOf":               "{selected}/{total}",
	"confirm.abort":                   "取消",
	"confirm.push.title":              "推送到默认分支？",
	"confirm.push.description":        "此操作会将本地提交推送到“{branch}”。你可以继续在此引用上操作，或新建功能引用后再执行同一操作。",
	"confirm.push.continue":           "推送到 {branch}",
	"confirm.commitPush.title":        "提交并推送到默认分支？",
	"confirm.commitPush.description":  "此操作会提交更改并推送到“{branch}”。你可以继续在此引用上操作，或新建功能引用后再执行同一操作。",
	"confirm.commitPush.continue":     "提交并推送到 {branch}",
	"confirm.pr.title":                "从默认分支推送并创建 {kind}？",
	"confirm.pr.description":          "此操作会推送本地提交并从“{branch}”创建 {kind}。你可以继续在此引用上操作，或新建功能引用后再执行同一操作。",
	"confirm.pr.continue":             "推送并创建 {kind}",
	"confirm.commitPr.title":          "从默认分支提交、推送并创建 {kind}？",
	"confirm.commitPr.description":    "此操作会提交、推送并从“{branch}”创建 {kind}。你可以继续在此引用上操作，或新建功能引用后再执行同一操作。",
	"confirm.commitPr.continue":       "提交、推送并创建 {kind}",
	"confirm.featureContinue":         "在新功能分支上继续",
	"hint.unavailable":                "Git 状态不可用。",
	"hint.busy":                       "Git 操作进行中。",
	"publish.unavailable":             "发布仓库不可用。",
	"publish.title":                   "发布仓库",
	"publish.description":             "使用 gh 创建 GitHub 仓库，或添加已有远程 URL 并推送。",
	"publish.name":                    "仓库名",
	"publish.visibility":              "可见性",
	"publish.public":                  "公开",
	"publish.private":                 "私有",
	"publish.remoteUrl":               "远程 URL（可选）",
	"publish.remotePlaceholder":       "https://github.com/org/repo.git",
	"publish.submit":                  "发布",
	"publish.cancel":                  "取消",
	"publish.note":                    "未填写 URL 时使用 gh repo create，需要 gh auth 登录。",
	"commit.openFile":                 "打开 {path}",
	"progress.copy":                   "复制错误",
	"progress.details":                "显示详情",
	"progress.hideDetails":            "隐藏详情",
	"progress.upToDate":               "已是最新",
	"progress.pulledFrom":             "已从 {upstream} 更新 {branch}",
	"progress.synced":                 "{branch} 已同步。",
	"action.commit":                   "Commit",
	"action.commitPush":               "Commit & push",
	"action.push":                     "Push",
	"action.pull":                     "Pull",
	"action.publish":                  "Publish repository",
	"action.init":                     "初始化 Git",
	"action.init.busy":                "Initializing...",
	"settings.titlebarGit.title":       "标题栏 Git 操作",
	"settings.titlebarGit.description": "显示初始化、切换分支、提交及菜单。关闭后仍可通过进行中的进度与对话框完成操作。",
	"action.sync":                     "Sync branch",
	"action.pushCreate":               "Push & create {short}",
	"action.commitPushCreate":         "Commit, push & {short}",
	"action.view":                     "View {short}",
	"action.create":                   "Create {short}",
	"error.title":                     "操作失败",
	"error.fallback":                  "发生错误。",
	"progress.waiting":                "等待 Git…",
	"progress.dismiss":                "关闭通知",
	"progress.committed":              "已提交更改",
	"progress.pushed":                 "已推送",
	"progress.pulled":                 "已拉取",
	"progress.done":                   "完成",
	"progress.openPr":                 "打开拉取请求",
	"progress.openRepo":               "打开仓库",
	"branch.open":                     "切换分支",
	"branch.select":                   "选择分支",
	"branch.search":                   "搜索分支…",
	"branch.createHint":               "创建并检出新分支…",
	"branch.createNamed":              "创建并检出分支「{name}」",
	"branch.createTitle":              "创建并检出新分支",
	"branch.createDescription":        "基于当前 HEAD 创建一个新的本地分支，并在创建成功后立即切换过去。",
	"branch.createName":               "分支名",
	"branch.createPlaceholder":        "例如 feature/git-branch-switcher",
	"branch.createNote":               "首版只支持基于当前 HEAD 创建并切换。",
	"branch.createSubmit":             "创建并切换",
	"branch.createCancel":             "取消",
	"branch.empty":                    "没有匹配的分支",
	"branch.listFailed":               "分支列表加载失败。",
	"branch.unsupportedName":          "名称包含桌面无法安全传给 git 的字符，无法从这里切换。",
	"branch.error":                    "分支操作失败。",
	"branch.switchFailed":              "切换分支失败。",
	"branch.createFailed":              "创建并切换分支失败。",
	"commit.openFailed":               "无法打开文件",
	"commit.largeFileWarning":         "检测到超过 100 MB 的大文件",
	"commit.largeFileHint":            "GitHub 拒绝超过 100 MB 的文件。建议取消提交，将文件加入 .gitignore 或使用 Git LFS。",
}

// gitLocaleEN mirrors @deepseek-ai/dsh-client-ui-git/lib/types/client/locales.js (en).
var gitLocaleEN = map[string]string{
	"menu.options":                    "Git actions",
	"menu.detached":                   "Detached HEAD: create and checkout a branch to enable push and pull request actions.",
	"menu.behind":                     "Behind upstream. Pull/rebase first.",
	"commit.title":                    "Commit changes",
	"commit.description":              "Review and confirm your commit. Leave the message blank to auto-generate one.",
	"commit.message":                  "Commit message (optional)",
	"commit.placeholder":              "Leave empty to auto-generate",
	"commit.submit":                   "Commit",
	"commit.cancel":                   "Cancel",
	"commit.onNewRef":                 "Commit on new branch",
	"commit.branch":                   "Branch",
	"commit.detached":                 "(detached HEAD)",
	"commit.defaultWarning":           "Warning: default branch",
	"commit.files":                    "Files",
	"commit.edit":                     "Edit",
	"commit.done":                     "Done",
	"commit.none":                     "none",
	"commit.excluded":                 "Excluded",
	"commit.selectedOf":               "{selected} of {total}",
	"confirm.abort":                   "Abort",
	"confirm.push.title":              "Push to default ref?",
	"confirm.push.description":        "This action will push local commits on \"{branch}\". You can continue on this ref or create a feature ref and run the same action there.",
	"confirm.push.continue":           "Push to {branch}",
	"confirm.commitPush.title":        "Commit & push to default ref?",
	"confirm.commitPush.description":  "This action will commit and push changes on \"{branch}\". You can continue on this ref or create a feature ref and run the same action there.",
	"confirm.commitPush.continue":     "Commit & push to {branch}",
	"confirm.pr.title":                "Push & create {kind} from default ref?",
	"confirm.pr.description":          "This action will push local commits and create a {kind} on \"{branch}\". You can continue on this ref or create a feature ref and run the same action there.",
	"confirm.pr.continue":             "Push & create {kind}",
	"confirm.commitPr.title":          "Commit, push & create {kind} from default ref?",
	"confirm.commitPr.description":    "This action will commit, push, and create a {kind} on \"{branch}\". You can continue on this ref or create a feature ref and run the same action there.",
	"confirm.commitPr.continue":       "Commit, push & create {kind}",
	"confirm.featureContinue":         "Checkout feature branch & continue",
	"hint.unavailable":                "Git status is unavailable.",
	"hint.busy":                       "Git action in progress.",
	"publish.unavailable":             "Publish repository is unavailable.",
	"publish.title":                   "Publish repository",
	"publish.description":             "Create a GitHub repository with gh, or add an existing remote URL and push.",
	"publish.name":                    "Repository name",
	"publish.visibility":              "Visibility",
	"publish.public":                  "Public",
	"publish.private":                 "Private",
	"publish.remoteUrl":               "Remote URL (optional)",
	"publish.remotePlaceholder":       "https://github.com/org/repo.git",
	"publish.submit":                  "Publish",
	"publish.cancel":                  "Cancel",
	"publish.note":                    "Uses gh repo create when no URL is given. Requires gh auth.",
	"commit.openFile":                 "Open {path}",
	"progress.copy":                   "Copy error",
	"progress.details":                "Show details",
	"progress.hideDetails":            "Hide details",
	"progress.upToDate":               "Already up to date",
	"progress.pulledFrom":             "Updated {branch} from {upstream}",
	"progress.synced":                 "{branch} is already synchronized.",
	"action.commit":                   "Commit",
	"action.commitPush":               "Commit & push",
	"action.push":                     "Push",
	"action.pull":                     "Pull",
	"action.publish":                  "Publish repository",
	"action.init":                     "Initialize Git",
	"action.init.busy":                "Initializing...",
	"settings.titlebarGit.title":       "Titlebar Git actions",
	"settings.titlebarGit.description": "Show initialize, branch switch, commit, and the Git menu. Turning this off still keeps in-flight toasts and dialogs.",
	"action.sync":                     "Sync branch",
	"action.pushCreate":               "Push & create {short}",
	"action.commitPushCreate":         "Commit, push & {short}",
	"action.view":                     "View {short}",
	"action.create":                   "Create {short}",
	"error.title":                     "Action failed",
	"error.fallback":                  "An error occurred.",
	"progress.waiting":                "Waiting for Git...",
	"progress.dismiss":                "Dismiss notification",
	"progress.committed":              "Committed changes",
	"progress.pushed":                 "Pushed",
	"progress.pulled":                 "Pulled",
	"progress.done":                   "Done",
	"progress.openPr":                 "Open pull request",
	"progress.openRepo":               "Open repository",
	"branch.open":                     "Switch branch",
	"branch.select":                   "Select branch",
	"branch.search":                   "Search branches…",
	"branch.createHint":               "Create and checkout new branch…",
	"branch.createNamed":              "Create and checkout branch \"{name}\"",
	"branch.createTitle":              "Create and checkout new branch",
	"branch.createDescription":        "Create a new local branch from the current HEAD and switch to it immediately.",
	"branch.createName":               "Branch name",
	"branch.createPlaceholder":        "e.g. feature/git-branch-switcher",
	"branch.createNote":               "This version only creates and switches from the current HEAD.",
	"branch.createSubmit":             "Create and switch",
	"branch.createCancel":             "Cancel",
	"branch.empty":                    "No matching branches",
	"branch.listFailed":               "Failed to load branches.",
	"branch.unsupportedName":          "Name contains characters the desktop cannot pass to git safely.",
	"branch.error":                    "Branch action failed.",
	"branch.switchFailed":              "Failed to switch branch.",
	"branch.createFailed":              "Failed to create and switch branch.",
	"commit.openFailed":               "Unable to open file",
	"commit.largeFileWarning":         "Large files over 100 MB detected",
	"commit.largeFileHint":            "GitHub rejects files over 100 MB. Cancel and add them to .gitignore or use Git LFS.",
}

// diffLocaleZH mirrors @deepseek-ai/dsh-client-ui-diff/lib/types/client/locales.js (zh).
var diffLocaleZH = map[string]string{
	"type.label":            "差异",
	"guide.title":           "工作区差异",
	"guide.description":     "查看当前会话工作区的 Git 变更。",
	"empty.cwd":             "没有工作区，无法查看差异。",
	"empty.changes":         "没有净变更。",
	"unavailable":           "差异仅适用于 Git 仓库。",
	"error.load":            "无法加载差异。",
	"error.open":            "无法打开文件。",
	"error.branches":        "无法读取分支。",
	"retry":                 "重试",
	"feedback.close":        "关闭提示",
	"truncated":             "差异过长，仅显示开头。",
	"refresh":               "刷新",
	"group.staged":          "已暂存",
	"group.unstaged":        "未暂存",
	"stage":                 "暂存",
	"unstage":               "取消暂存",
	"discard":               "还原",
	"discard.title":         "还原工作区更改？",
	"discard.body":          "未暂存的更改将被丢弃，无法恢复。",
	"discard.confirm":       "还原",
	"discard.cancel":        "取消",
	"scope.worktree":        "工作区",
	"scope.branch":          "分支",
	"scope.search":          "搜索分支…",
	"collapseAll":           "全部折叠",
	"expandAll":             "全部展开",
	"view.unified":          "统一视图",
	"view.split":            "分栏视图",
	"view.splitAria":        "切换统一/分栏视图",
	"view.wrap":             "折行显示",
	"view.nowrap":           "关闭折行",
	"view.wrapAria":         "切换折行",
	"file.renamed":          "重命名自 {from}",
	"file.binary":           "二进制或过大文件，无逐行差异",
	"file.omitted":          "超出面板渲染预算，折叠其他文件后再展开",
	"file.truncated":        "仅显示前 {count} 行",
	"diff.highlightSkipped": "部分内容过大，未语法高亮",
}

// diffLocaleEN mirrors @deepseek-ai/dsh-client-ui-diff/lib/types/client/locales.js (en).
var diffLocaleEN = map[string]string{
	"type.label":            "Diff",
	"guide.title":           "Workspace diff",
	"guide.description":     "Review Git changes in this session workspace.",
	"empty.cwd":             "A workspace is required to review diffs.",
	"empty.changes":         "No net changes in this selection.",
	"unavailable":           "Diff is only available in Git repositories.",
	"error.load":            "Could not load the diff.",
	"error.open":            "Could not open the file.",
	"error.branches":        "Could not read the branches.",
	"retry":                 "Retry",
	"feedback.close":        "Dismiss notification",
	"truncated":             "Diff is too large; showing the beginning.",
	"refresh":               "Refresh",
	"group.staged":          "Staged",
	"group.unstaged":        "Unstaged",
	"stage":                 "Stage",
	"unstage":               "Unstage",
	"discard":               "Discard",
	"discard.title":         "Discard working-tree changes?",
	"discard.body":          "Unstaged changes will be lost and cannot be undone.",
	"discard.confirm":       "Discard",
	"discard.cancel":        "Cancel",
	"scope.worktree":        "Working tree",
	"scope.branch":          "Branch",
	"scope.search":          "Search branches…",
	"collapseAll":           "Collapse all",
	"expandAll":             "Expand all",
	"view.unified":          "Unified view",
	"view.split":            "Split view",
	"view.splitAria":        "Toggle unified/split view",
	"view.wrap":             "Wrap lines",
	"view.nowrap":           "Do not wrap lines",
	"view.wrapAria":         "Toggle line wrapping",
	"file.renamed":          "Renamed from {from}",
	"file.binary":           "Binary or oversized file; no line diff",
	"file.omitted":          "Over the panel render budget; collapse other files first",
	"file.truncated":        "Showing the first {count} lines",
	"diff.highlightSkipped": "Some content is too large to syntax-highlight",
}

// GitLocale returns the localized string from the dsh-client-ui-git dictionary.
func GitLocale(key string) string {
	if i18n.Locale() == "en" {
		if v, ok := gitLocaleEN[key]; ok {
			return v
		}
	}
	if v, ok := gitLocaleZH[key]; ok {
		return v
	}
	return key
}

// GitLocaleFormat formats a dsh-client-ui-git template with {placeholder} replacements.
func GitLocaleFormat(key string, vars map[string]string) string {
	text := GitLocale(key)
	for k, v := range vars {
		text = strings.ReplaceAll(text, "{"+k+"}", v)
	}
	return text
}

// DiffLocale returns the localized string from the dsh-client-ui-diff dictionary.
func DiffLocale(key string) string {
	if i18n.Locale() == "en" {
		if v, ok := diffLocaleEN[key]; ok {
			return v
		}
	}
	if v, ok := diffLocaleZH[key]; ok {
		return v
	}
	return key
}

// DiffLocaleFormat formats a dsh-client-ui-diff template with {placeholder} replacements.
func DiffLocaleFormat(key string, vars map[string]string) string {
	text := DiffLocale(key)
	for k, v := range vars {
		text = strings.ReplaceAll(text, "{"+k+"}", v)
	}
	return text
}

func formatSelectedCount(selected, total int) string {
	return GitLocaleFormat("commit.selectedOf", map[string]string{
		"selected": fmt.Sprintf("%d", selected),
		"total":    fmt.Sprintf("%d", total),
	})
}
