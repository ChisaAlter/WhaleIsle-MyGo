package files

import (
	"context"
	"errors"
	"time"

	"whaleisle/internal/desktop/mediapreview"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/ui/i18n"
)

func isAsyncTextPath(path string) bool {
	kind := workspacefs.ClassifyDoc(path)
	return !isPreviewImage(path) && !mediapreview.Supports(path) && (kind == workspacefs.DocNone || kind == workspacefs.DocHTML)
}

func isAsyncPreviewPath(path string) bool {
	return isAsyncTextPath(path) || isTablePreviewPath(path)
}

func textReadBlocked(m Model) bool {
	t := selectedTab(m)
	return t != nil && t.readBlocked
}

func textReadOwnerFor(m Model, t *FileTab) textReadOwner {
	return textReadOwner{binding: m.BindingID, root: m.WorkspaceRoot, session: m.SessionID, path: t.Path, tab: t.readID}
}

func (t *Tabs) stopTextRead(resume, closeQueue bool) {
	if t == nil {
		return
	}
	if t.readTab != nil {
		t.readTab.readLoading = false
		t.readTab.readResume = resume
		t.readTab = nil
	}
	if t.textReader != nil {
		t.textReader.stop(closeQueue)
	}
}

func beginTextRead(m Model) {
	tab := selectedTab(m)
	if m.Tabs == nil || tab == nil || !isAsyncPreviewPath(tab.Path) {
		return
	}
	capture(m)
	applySaveResult(m, tab)
	tab.readBlocked = true
	tab.pendingLine.begin()
	if tab.revealLine > 0 {
		tab.pendingLine.deferRequest(tab.revealLine)
		tab.revealLine = 0
	}
	if tab.save != nil && tab.save.Snapshot().Running {
		tab.readResume = true
		return // Preserve the in-flight save receipt before rereading its baseline.
	}
	disposeSave(tab)
	t := m.Tabs
	t.stopTextRead(true, false)
	if t.textReader == nil {
		t.textReader = &textReadQueue{}
	}
	if m.notify != nil {
		t.textReader.setNotify(m.notify)
	}
	if tab.readID == 0 {
		t.readSerial++
		tab.readID = t.readSerial
	}
	tab.readBlocked, tab.readLoading, tab.readResume = true, true, false
	t.readTab = tab
	root, path := m.WorkspaceRoot, tab.Path
	tableRequest := tab.tableRequest
	if isTablePreviewPath(path) && !tab.Dirty() {
		tab.Preview, tab.Draft, tab.Error = "", "", ""
		tab.Editing = false
		restore(m, tab)
	}
	remote, stat := m.ReadRemoteContext, m.StatRemoteContext
	// Capture immutable values and context-capable, frame-bound Host readers.
	// Legacy callbacks may resolve a different Host at invocation time and must
	// never run on this worker.
	t.textReader.request(textReadOwnerFor(m, tab), func(ctx context.Context) textReadResult {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		out := textReadResult{}
		if ctx.Err() != nil {
			return out
		}
		full, err := workspacefs.JoinRoot(root, path)
		out.fullPath = full
		if err == nil && ctx.Err() == nil {
			if isTablePreviewPath(path) {
				out.table, err = readTablePreview(ctx, full, tableRequest)
				if out.table != nil {
					out.text = out.table.Text
				}
			} else {
				var local workspacefs.ReadTextResult
				local, err = workspacefs.ReadFileTextPreview(full)
				out.text, out.truncated = local.Text, local.Truncated
			}
		}
		out.err = err
		if out.err != nil && remote != nil && ctx.Err() == nil && !isTablePreviewPath(path) {
			if body, remoteErr := remote(ctx, path); remoteErr == nil {
				out.text, out.err, out.remote, out.truncated = body, nil, true, false
			}
		}
		if stat != nil && ctx.Err() == nil {
			out.stat = stat(ctx, path)
		}
		if ctx.Err() != nil {
			out.err = ctx.Err()
		}
		return out
	})
}

// Runs only on the UI thread with the current frame's binding and tab pointers.
func syncTextRead(m Model) {
	if m.Tabs == nil {
		return
	}
	tab := selectedTab(m)
	if tab == nil || !isAsyncPreviewPath(tab.Path) {
		return
	}
	if tab.readResume {
		beginTextRead(m)
	}
	t := m.Tabs
	if t.textReader == nil || t.readTab != tab || t.binding != m.BindingID || t.boundRoot != m.WorkspaceRoot {
		return
	}
	t.textReader.setNotify(m.notify)
	result, ready := t.textReader.take(textReadOwnerFor(m, tab))
	if !ready {
		return
	}
	t.readTab = nil
	tab.readLoading, tab.readResume = false, false
	tab.readBlocked = result.err != nil
	applyTextRead(m, result)
}

func applyTextRead(m Model, result textReadResult) {
	capture(m)
	tab := selectedTab(m)
	if tab == nil {
		return
	}
	problem := ""
	if result.err != nil {
		problem = result.err.Error()
		if errors.Is(result.err, workspacefs.ErrBinary) {
			problem = i18n.T(i18n.KeyFilesBinary)
		}
	}
	dirty := tab.Dirty()
	next := reconcileTextRead(textReadState{preview: tab.Preview, draft: tab.Draft, truncated: tab.Truncated, remote: tab.Remote}, result)
	tab.Preview, tab.Draft = next.preview, next.draft
	tab.Truncated, tab.Remote, tab.readBlocked = next.truncated, next.remote, next.blocked
	tab.readFullPath = result.fullPath
	if isTablePreviewPath(tab.Path) && result.err == nil {
		tab.tablePage = result.table
		if result.table != nil {
			tab.tableRequest = result.table.Request
		}
		tab.Editing = false
	}
	if dirty {
		switch {
		case problem != "":
			tab.Status = label("重新载入失败，草稿保留：", "Reload failed; draft kept: ") + problem
		case result.truncated:
			tab.Status = label("磁盘文件超出完整预览上限，草稿保留且停止保存。", "File exceeds the complete preview limit. Draft kept and saving stopped.")
		case result.text != tab.Preview:
			tab.Status = label("磁盘文件已变化，草稿保留。请复制草稿并核对。", "File changed on disk. Draft kept; copy it and reconcile.")
		default:
			tab.Status = label("文件未变化，未保存草稿已保留", "File unchanged; unsaved draft kept")
		}
	} else {
		tab.Error = problem
		tab.Status = ""
		if problem == "" {
			if result.truncated {
				tab.Editing = false
			}
		}
	}
	tab.Stat = result.stat
	restore(m, tab)
	if line := tab.pendingLine.complete(problem == ""); line > 0 {
		requestLineNow(m, line) // Body/draft is now installed; only now clamp its line.
	}
}

// A text toolbar reuses only the path already authorized by its completed read.
// Revalidate on explicit filesystem actions, not on every UI paint.
func textPreviewPath(m Model, path string) string {
	if t := selectedTab(m); t != nil && isAsyncPreviewPath(path) {
		return t.readFullPath
	}
	full, _ := workspacefs.JoinRoot(m.WorkspaceRoot, path)
	return full
}

func withPreviewPath(m Model, rel, full string, action func(string)) {
	if isAsyncPreviewPath(rel) {
		var err error
		full, err = workspacefs.JoinRoot(m.WorkspaceRoot, rel)
		if err != nil {
			setFileStatus(m, err.Error())
			return
		}
	}
	action(full)
}
