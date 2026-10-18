package files

import "errors"

func previewTruncated(m Model) bool {
	t := selectedTab(m)
	return t != nil && t.Truncated
}

func truncatedPreviewNotice() string {
	return label("文件较大，仅显示前 1 MiB（末尾字符可能省略）。这是不完整的只读预览，不能保存覆盖原文件；可用外部编辑器打开完整文件。", "Large file: showing only the first 1 MiB (a final partial character may be omitted). This incomplete preview is read-only and cannot overwrite the file. Open it in an external editor for the full content.")
}

func checkPreviewComplete(m Model) error {
	if textReadBlocked(m) {
		return errors.New(label("读取尚未成功完成，不能保存；草稿已保留。", "Read has not completed successfully; cannot save. Draft kept."))
	}
	if previewTruncated(m) {
		return errors.New(previewTruncationNotice(m))
	}
	return nil
}

func previewTruncationNotice(m Model) string {
	if t := selectedTab(m); t != nil && t.textPaged && !t.Dirty() {
		return label("分页资源只读；可继续读取下一页，不能保存覆盖原文件。", "Paged resource is read-only. Load the next page to continue; saving over the source is disabled.")
	}
	if m.Preview != nil && m.EditDraft != nil && *m.Preview != *m.EditDraft {
		return label("磁盘文件超出 1 MiB 完整预览上限；当前显示的是保留的未保存草稿，不是完整磁盘内容。保存已停止，请复制草稿并在外部编辑器核对。", "The disk file exceeds the 1 MiB complete preview limit. Showing your retained unsaved draft, not the complete disk content. Saving is stopped; copy the draft and reconcile in an external editor.")
	}
	return truncatedPreviewNotice()
}
