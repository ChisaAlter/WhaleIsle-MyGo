package i18n

// Keys for the functional surfaces domain: settings shell and rows, workspace
// files, the diff workbench and the usage dashboard. Feature packages in that
// domain resolve copy through i18n.T so no screen prints two languages at once.
const (
	KeySurfacesSettingsSave         = "surfaces.settings.save"
	KeySurfacesSettingsSaving       = "surfaces.settings.saving"
	KeySurfacesSettingsSaveFailed   = "surfaces.settings.save_failed"
	KeySurfacesSettingsUnsaved      = "surfaces.settings.unsaved"
	KeySurfacesSettingsReset        = "surfaces.settings.reset"
	KeySurfacesSettingsInherit      = "surfaces.settings.inherit"
	KeySurfacesSettingsReplace      = "surfaces.settings.replace"
	KeySurfacesSettingsDiscard      = "surfaces.settings.discard"
	KeySurfacesSettingsAdvanced     = "surfaces.settings.advanced"
	KeySurfacesSettingsFilter       = "surfaces.settings.filter"
	KeySurfacesSettingsFilterEmpty  = "surfaces.settings.filter_empty"
	KeySurfacesSettingsNamespace    = "surfaces.settings.namespace"
	KeySurfacesSettingsRefresh      = "surfaces.settings.refresh"
	KeySurfacesSettingsSignIn       = "surfaces.settings.sign_in"
	KeySurfacesSettingsWritable     = "surfaces.settings.writable"
	KeySurfacesSettingsReadOnly     = "surfaces.settings.read_only"
	KeySurfacesSettingsRevision     = "surfaces.settings.revision"
	KeySurfacesSettingsReveal       = "surfaces.settings.reveal_secrets"
	KeySurfacesSettingsPick         = "surfaces.settings.pick"
	KeySurfacesSettingsDanger       = "surfaces.settings.danger"
	KeySurfacesSettingsConfirmTitle = "surfaces.settings.confirm_title"
	KeySurfacesSettingsConfirmBody  = "surfaces.settings.confirm_body"
	KeySurfacesSettingsConfirmOK    = "surfaces.settings.confirm_ok"
	KeySurfacesSettingsCancel       = "surfaces.settings.cancel"
	KeySurfacesSettingsVisualHint   = "surfaces.settings.visual_replies_hint"
	KeySurfacesSettingsSecretsHint  = "surfaces.settings.secrets_hint"
	KeySurfacesSettingsSaveHint     = "surfaces.settings.save_hint"
	KeySurfacesSettingsReplaceHint  = "surfaces.settings.replace_hint"

	KeySurfacesUsageTotal      = "surfaces.usage.total"
	KeySurfacesUsageInput      = "surfaces.usage.input"
	KeySurfacesUsageOutput     = "surfaces.usage.output"
	KeySurfacesUsageSessions   = "surfaces.usage.sessions"
	KeySurfacesUsageHit        = "surfaces.usage.hit"
	KeySurfacesUsageCost       = "surfaces.usage.cost"
	KeySurfacesUsageAllTime    = "surfaces.usage.all_time"
	KeySurfacesUsageCalendar   = "surfaces.usage.calendar"
	KeySurfacesUsageLess       = "surfaces.usage.less"
	KeySurfacesUsageMore       = "surfaces.usage.more"
	KeySurfacesUsageModels     = "surfaces.usage.models"
	KeySurfacesUsageRanking    = "surfaces.usage.ranking"
	KeySurfacesUsageModelRank  = "surfaces.usage.model_ranking"
	KeySurfacesUsageEmpty      = "surfaces.usage.empty"
	KeySurfacesUsageDiagnostic = "surfaces.usage.diagnostics"
	KeySurfacesUsageDays       = "surfaces.usage.days"
	KeySurfacesUsageScan       = "surfaces.usage.scan"
	KeySurfacesUsageExport     = "surfaces.usage.export"
	KeySurfacesUsageShowMore   = "surfaces.usage.show_more"
	KeySurfacesUsageLoading    = "surfaces.usage.loading"
	KeySurfacesUsageUpdated    = "surfaces.usage.updated"
	KeySurfacesUsageCoverage   = "surfaces.usage.coverage"
	KeySurfacesUsageCoverageOK = "surfaces.usage.coverage_ok"
	KeySurfacesUsageTokens     = "surfaces.usage.tokens"
	KeySurfacesUsageStale      = "surfaces.usage.stale"

	KeySurfacesDiffUnified     = "surfaces.diff.unified"
	KeySurfacesDiffSplit       = "surfaces.diff.split"
	KeySurfacesDiffWrap        = "surfaces.diff.wrap"
	KeySurfacesDiffExpandAll   = "surfaces.diff.expand_all"
	KeySurfacesDiffCollapseAll = "surfaces.diff.collapse_all"
	KeySurfacesDiffBefore      = "surfaces.diff.before"
	KeySurfacesDiffAfter       = "surfaces.diff.after"
	KeySurfacesDiffEmpty       = "surfaces.diff.empty_line"

	KeySurfacesFilesOpen  = "surfaces.files.open"
	KeySurfacesFilesClose = "surfaces.files.close"
	KeySurfacesFilesDirty = "surfaces.files.dirty"
)

func init() {
	Register(KeySurfacesSettingsSave, "保存", "Save")
	Register(KeySurfacesSettingsSaving, "正在保存…", "Saving…")
	Register(KeySurfacesSettingsSaveFailed, "保存未被接受", "The save was not accepted")
	Register(KeySurfacesSettingsUnsaved, "未保存的更改", "Unsaved changes")
	Register(KeySurfacesSettingsReset, "恢复默认", "Reset to default")
	Register(KeySurfacesSettingsInherit, "保存后使用继承值", "Inherits its default after save")
	Register(KeySurfacesSettingsReplace, "完整替换", "Replace all")
	Register(KeySurfacesSettingsDiscard, "放弃草稿", "Discard drafts")
	Register(KeySurfacesSettingsAdvanced, "高级", "Advanced")
	Register(KeySurfacesSettingsFilter, "按名称筛选", "Filter by name")
	Register(KeySurfacesSettingsFilterEmpty, "没有匹配的分区", "No matching sections")
	Register(KeySurfacesSettingsNamespace, "配置分区", "Configuration section")
	Register(KeySurfacesSettingsRefresh, "刷新", "Refresh")
	Register(KeySurfacesSettingsSignIn, "登录账号", "Sign in")
	Register(KeySurfacesSettingsWritable, "可写", "Writable")
	Register(KeySurfacesSettingsReadOnly, "只读", "Read-only")
	Register(KeySurfacesSettingsRevision, "修订", "Rev")
	Register(KeySurfacesSettingsReveal, "显示密钥", "Show secrets")
	Register(KeySurfacesSettingsPick, "选择左侧分区以查看或修改设置。", "Choose a section to view or change settings.")
	Register(KeySurfacesSettingsDanger, "危险操作", "Dangerous action")
	Register(KeySurfacesSettingsConfirmTitle, "完整替换此分区", "Replace this section")
	Register(KeySurfacesSettingsConfirmBody, "完整替换会写入当前草稿中的每个字段，并清空未编辑的密钥；此操作无法撤销。", "A full replace writes every field in the draft and clears unedited secrets. It cannot be undone.")
	Register(KeySurfacesSettingsConfirmOK, "确认完整替换", "Replace all")
	Register(KeySurfacesSettingsCancel, "取消", "Cancel")
	Register(KeySurfacesSettingsVisualHint, "启用 HTML 视觉回复前会检查本机隔离渲染；失败时保持关闭并显示原因。", "Enabling HTML visual replies checks isolated rendering first; on failure the setting stays off and the reason is shown.")
	Register(KeySurfacesSettingsSecretsHint, "此分区包含隐藏密钥：普通保存会保留未编辑的密钥，完整替换不可用。", "This section holds hidden secrets: Save keeps unedited secrets, full replacement is unavailable.")
	Register(KeySurfacesSettingsSaveHint, "保存只提交已修改字段；恢复默认会删除覆盖值并继承默认设置。", "Save submits changed fields only. Reset removes an override and inherits its default.")
	Register(KeySurfacesSettingsReplaceHint, "完整替换以当前草稿覆盖整个分区，包含字段的新增与删除。", "Replace all overwrites the whole section with the current draft, adding and removing fields.")

	Register(KeySurfacesUsageTotal, "总 Token", "Total tokens")
	Register(KeySurfacesUsageInput, "输入（含缓存）", "Input (incl. cache)")
	Register(KeySurfacesUsageOutput, "输出", "Output")
	Register(KeySurfacesUsageSessions, "会话", "Sessions")
	Register(KeySurfacesUsageHit, "缓存命中", "Cache hit")
	Register(KeySurfacesUsageCost, "估算费用", "Estimated cost")
	Register(KeySurfacesUsageAllTime, "全部历史", "All time")
	Register(KeySurfacesUsageCalendar, "每日用量", "Daily usage")
	Register(KeySurfacesUsageLess, "少", "Less")
	Register(KeySurfacesUsageMore, "多", "More")
	Register(KeySurfacesUsageModels, "模型占比", "By model")
	Register(KeySurfacesUsageRanking, "会话排行", "Session ranking")
	Register(KeySurfacesUsageModelRank, "模型排行", "Model ranking")
	Register(KeySurfacesUsageEmpty, "此范围暂无用量。", "No usage in this range.")
	Register(KeySurfacesUsageDiagnostic, "统计诊断", "Statistics diagnostics")
	Register(KeySurfacesUsageDays, "近 %d 天", "Last %d days")
	Register(KeySurfacesUsageScan, "重新扫描", "Rescan")
	Register(KeySurfacesUsageExport, "导出", "Export")
	Register(KeySurfacesUsageShowMore, "显示更多", "Show more")
	Register(KeySurfacesUsageLoading, "正在读取用量…", "Loading usage…")
	Register(KeySurfacesUsageUpdated, "最近更新", "Last updated")
	Register(KeySurfacesUsageCoverage, "统计覆盖", "Coverage")
	Register(KeySurfacesUsageCoverageOK, "%d/%d 个会话可读", "%d/%d sessions readable")
	Register(KeySurfacesUsageTokens, "tokens", "tokens")
	Register(KeySurfacesUsageStale, "正在显示上次结果，可能已过期。", "Showing the previous result; it may be stale.")

	Register(KeySurfacesDiffUnified, "统一视图", "Unified")
	Register(KeySurfacesDiffSplit, "分栏视图", "Split")
	Register(KeySurfacesDiffWrap, "折行", "Wrap")
	Register(KeySurfacesDiffExpandAll, "全部展开", "Expand all")
	Register(KeySurfacesDiffCollapseAll, "全部折叠", "Collapse all")
	Register(KeySurfacesDiffBefore, "原始", "Before")
	Register(KeySurfacesDiffAfter, "修改后", "After")
	Register(KeySurfacesDiffEmpty, " ", " ")

	Register(KeySurfacesFilesOpen, "打开文件", "Open a file")
	Register(KeySurfacesFilesClose, "关闭标签", "Close tab")
	Register(KeySurfacesFilesDirty, "有未保存的更改", "Has unsaved changes")
}
