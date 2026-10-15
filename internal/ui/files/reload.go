package files

// reloadSelectedFile follows the original Files activation/Refresh contract:
// reread clean tabs while preserving both the draft and its original save base
// when an unsaved tab encounters new, missing, binary or truncated disk bytes.
func reloadSelectedFile(m Model) {
	if m.SelectedFile == nil || *m.SelectedFile == "" {
		return
	}
	capture(m)
	if tab := selectedTab(m); tab != nil {
		applySaveResult(m, tab)
		if tab.save != nil && tab.save.Snapshot().Running {
			setFileStatus(m, label("保存正在进行，暂不重新载入。草稿已保留。", "Save in progress; reload paused and draft kept."))
			capture(m)
			return
		}
		disposeSave(tab)
	}
	rel := *m.SelectedFile
	if tab := selectedTab(m); tab != nil {
		tab.image = nil
		tab.selectionText = ""
		tab.selectionStart, tab.selectionEnd = 0, 0
	}
	if tab := selectedTab(m); tab != nil && isTablePreviewPath(rel) {
		tab.tableRequest.Version = "" // Explicit Reload permits a new file version.
	}
	if m.Tabs != nil && isAsyncPreviewPath(rel) {
		beginTextRead(m)
		return
	}
	dirty := m.EditDraft != nil && m.Preview != nil && *m.EditDraft != *m.Preview
	remotePreview := false
	text, errMsg, truncated := loadPreview(m.WorkspaceRoot, rel)
	if truncated && m.Tabs == nil {
		errMsg = truncatedPreviewNotice()
	}
	if errMsg != "" && m.ReadRemote != nil && !isTablePreviewPath(rel) {
		if remote, remoteErr := LoadPreviewRemote(m.ReadRemote, rel); remoteErr == "" {
			text, errMsg = remote, ""
			remotePreview = true
			truncated = false
		}
	}
	if tab := selectedTab(m); tab != nil && errMsg == "" {
		tab.Truncated = truncated
	}
	if dirty {
		if m.EditStatus != nil {
			switch {
			case errMsg != "":
				*m.EditStatus = label("重新载入失败，草稿保留：", "Reload failed; draft kept: ") + errMsg
			case truncated:
				*m.EditStatus = label("磁盘文件已超出完整预览上限，草稿保留且停止保存。请复制草稿并在外部编辑器核对。", "File now exceeds the complete preview limit. Draft kept and saving stopped. Copy it and reconcile in an external editor.")
			case text != *m.Preview:
				*m.EditStatus = label("磁盘文件已变化，草稿保留。请复制草稿后关闭并重新打开文件核对。", "File changed on disk; draft kept. Copy the draft, then close and reopen to reconcile.")
			default:
				*m.EditStatus = label("文件未变化，未保存草稿已保留", "File unchanged; unsaved draft kept")
			}
		}
		capture(m)
		return
	}
	if (truncated || isPreviewImage(rel) || mediaMode(m, rel)) && m.Editing != nil {
		*m.Editing = false
	}
	if m.PreviewErr != nil {
		*m.PreviewErr = errMsg
	}
	if errMsg == "" {
		if tab := selectedTab(m); tab != nil {
			tab.Remote = remotePreview
		}
		if m.Preview != nil {
			*m.Preview = text
		}
		if m.EditDraft != nil {
			*m.EditDraft = text
		}
	}
	if m.EditStatus != nil {
		*m.EditStatus = ""
	}
	if m.StatNote != nil && m.StatRemote != nil {
		*m.StatNote = m.StatRemote(rel)
	}
	capture(m)
}
