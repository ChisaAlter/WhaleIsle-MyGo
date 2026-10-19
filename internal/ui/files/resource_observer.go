package files

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/egoist/mygo/ui"
	"path"
	"strings"
	"time"
	"whaleisle/internal/desktop/workspacefs"
)

// One active resource owns one watch and one bounded latest-read queue. Watch
// signals (including ready) are invalidations, never evidence of changed bytes.
type resourceObserver struct {
	tableRequest workspacefs.XLSXPageRequest
	owner        textReadOwner
	tab          *FileTab
	cancel       context.CancelFunc
	queue        textReadQueue
	read         func(context.Context) textReadResult
	revision     uint64
}

func (t *Tabs) stopResourceObserver() {
	if t == nil || t.resourceObserver == nil {
		return
	}
	o := t.resourceObserver
	t.resourceObserver = nil
	o.cancel()
	o.queue.stop(true)
}
func observeFileResource(m Model) {
	if m.Tabs == nil {
		return
	}
	tab := selectedTab(m)
	if tab == nil || !isAsyncPreviewPath(tab.Path) || tab.document.enabled || m.WatchRemoteContext == nil {
		m.Tabs.stopResourceObserver()
		return
	}
	capture(m)
	owner := textReadOwnerFor(m, tab)
	request := tab.tableRequest
	request.Version = ""
	o := m.Tabs.resourceObserver
	if o == nil || o.owner != owner || o.tab != tab || o.tableRequest != request {
		m.Tabs.stopResourceObserver()
		ctx, cancel := context.WithCancel(context.Background())
		o = &resourceObserver{owner: owner, tab: tab, cancel: cancel, read: resourceReader(m, tab), revision: tab.saveRevision, tableRequest: request}
		m.Tabs.resourceObserver = o
		o.queue.setNotify(m.notify)
		watch := m.WatchRemoteContext
		directory := path.Dir(strings.ReplaceAll(tab.Path, "\\", "/"))
		if directory == "." {
			directory = ""
		}
		go func() {
			_ = watch(ctx, directory, func() {
				if ctx.Err() == nil {
					o.queue.request(owner, o.read)
				}
			})
		}()
		// Also close the gap before the Host subscription acknowledges readiness.
		o.queue.request(owner, o.read)
	}
	o.queue.setNotify(m.notify)
	if tab.readLoading || tab.readResume {
		return
	}
	if tab.save != nil && tab.save.Snapshot().Running {
		return
	}
	if o.revision != tab.saveRevision {
		o.revision = tab.saveRevision
		o.queue.request(owner, o.read)
		return
	}
	result, ok := o.queue.take(owner)
	if !ok {
		return
	}
	applyResourceObservation(m, tab, result)
}
func resourceReader(m Model, tab *FileTab) func(context.Context) textReadResult {
	root, name := m.WorkspaceRoot, tab.Path
	until := int64(0)
	if tab.textPaged {
		until = tab.textPage.Next
	}
	pageReader, remote, tableReader := m.ReadTextPageContext, m.ReadRemoteContext, m.ReadTableContext
	// Context callbacks are available to both desktop and Host previews. Keep
	// the source selected by the completed read: a watch is only invalidation,
	// and cannot replace a local editor's baseline with Host text or errors.
	if !isTablePreviewPath(name) && !tab.Remote {
		pageReader, remote = nil, nil
	}
	request := tab.tableRequest
	request.Version = ""
	return func(ctx context.Context) textReadResult {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		out := textReadResult{}
		if ctx.Err() != nil {
			out.err = ctx.Err()
			return out
		}
		if isTablePreviewPath(name) {
			if tableReader != nil {
				out.table, out.err = tableReader(ctx, name, request)
				out.remote = true
			} else if remote != nil || pageReader != nil {
				return out // No local fallback for a Host-owned table.
			} else {
				out.fullPath, out.err = workspacefs.JoinRoot(root, name)
				if out.err == nil {
					out.table, out.err = readTablePreview(ctx, out.fullPath, request)
				}
			}
			if out.table != nil {
				out.text = out.table.Text
			}
		} else if pageReader != nil {
			page, text, err := observeTextPages(ctx, func(offset int64) (workspacefs.TextPage, error) { return pageReader(ctx, name, offset) }, until)
			out.page, out.text, out.err, out.remote, out.truncated = &page, text, err, true, !page.EOF || until > 0
		} else if remote != nil {
			out.text, out.err = remote(ctx, name)
			out.remote = true
		} else {
			out.fullPath, out.err = workspacefs.JoinRoot(root, name)
			if out.err == nil {
				page, text, err := observeTextPages(ctx, func(offset int64) (workspacefs.TextPage, error) {
					return workspacefs.ReadFileTextPage(out.fullPath, offset)
				}, until)
				out.page, out.text, out.err, out.truncated = &page, text, err, !page.EOF || until > 0
			}
		}
		if ctx.Err() != nil {
			out.err = ctx.Err()
		}
		return out
	}
}
func resourceContentChanged(tab *FileTab, r textReadResult) bool {
	if r.err != nil || tab.Error != "" {
		return true
	}
	if isTablePreviewPath(tab.Path) {
		if r.table == nil {
			return false
		}
		// Compare the current resource page, retaining the selected worksheet/range.
		old, next := *r.table, *r.table
		if tab.tablePage == nil {
			return true
		}
		old = *tab.tablePage
		old.Version = ""
		next.Version = ""
		old.Request.Version = ""
		next.Request.Version = ""
		a, _ := json.Marshal(old)
		b, _ := json.Marshal(next)
		return string(a) != string(b)
	}
	if tab.textPaged && r.page != nil && r.page.EOF != tab.textPage.EOF {
		return true
	}

	return r.text != tab.Preview || r.truncated != tab.Truncated
}
func applyResourceObservation(m Model, tab *FileTab, r textReadResult) {
	if !resourceContentChanged(tab, r) {
		if r.page != nil && tab.textPage.Next == r.page.Next {
			tab.textPage.Version = r.page.Version
		}
		if r.table != nil && tab.tablePage != nil {
			tab.tablePage.Version = r.table.Version
			tab.tableRequest.Version = r.table.Version
		}

		if tab.externalChanged && r.err == nil {
			tab.externalChanged = false
			tab.readBlocked = false
			tab.Status = ""
			restore(m, tab)
		}
		return
	}
	tab.externalChanged = true
	if tab.Dirty() || tab.AutoRefreshDisabled {
		if tab.Dirty() {
			pauseSave(tab)
			tab.readBlocked = true
		}
		tab.Status = label("源文件已在外部更改；草稿和已加载内容已保留。", "Source changed externally; draft and loaded content were kept.")
		if r.err != nil {
			tab.Status += " " + r.err.Error()
		}
		restore(m, tab)
		return
	}
	if r.err != nil {
		tab.readBlocked = true
		pauseSave(tab)
		tab.Status = label("源文件无法重新读取；当前内容已保留：", "Source could not be reread; current content kept: ") + r.err.Error()
		restore(m, tab)
		return
	}
	disposeSave(tab)
	applyTextRead(m, r)
	tab.externalChanged = false
}
func paintResourceRefreshControls(c *ui.Context, m Model) {
	tab := selectedTab(m)
	if tab == nil || !isAsyncPreviewPath(tab.Path) || tab.document.enabled || !tab.externalChanged {
		return
	}
	ui.Row(c).Gap(6).Padding(0, 12, 4, 12).Wrap().Children(func() {
		if tab.externalChanged {
			ui.Text(c, label("外部内容已变化；未保存草稿不会被覆盖。", "External content changed; unsaved draft will not be overwritten.")).FontSize(11)
			if ui.Button(c, label("重新读取源文件", "Refresh source file")).Clicked() {
				reloadSelectedFile(m)
			}
		}
	})
}

// Probe only the already loaded prefix. This preserves paging and compares
// bytes rather than treating timestamps or noisy directory events as edits.
func observeTextPages(ctx context.Context, read func(int64) (workspacefs.TextPage, error), until int64) (workspacefs.TextPage, string, error) {
	var last workspacefs.TextPage
	var text strings.Builder
	offset := int64(0)
	version := ""
	for {
		if err := ctx.Err(); err != nil {
			return last, "", err
		}
		page, err := read(offset)
		if err != nil {
			return last, "", err
		}
		if offset > 0 && version != page.Version {
			return last, "", errors.New("file changed during resource observation")
		}
		if offset > 0 {
			text.WriteString(page.Separator)
		}
		text.WriteString(page.Text)
		last = page
		if page.EOF || page.Next >= until {
			return last, text.String(), nil
		}
		if page.Next <= offset {
			return last, "", errors.New("resource text page did not advance")
		}
		offset, version = page.Next, page.Version
	}
}
