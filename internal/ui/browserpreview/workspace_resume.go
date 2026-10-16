package browserpreview

// WorkspaceSnapshot persists resource addresses and native profile ownership.
// Temporary document URLs, bytes, leases and native handles are never serialized.
type WorkspaceSnapshot struct {
	Tabs []WorkspaceTabSnapshot `json:"tabs,omitempty"`
	Active int `json:"active"`
}

type WorkspaceTabSnapshot struct {
	ID int `json:"id"`
	URL string `json:"url,omitempty"`
	Draft string `json:"draft,omitempty"`
	Partition string `json:"partition,omitempty"`
	Label string `json:"label,omitempty"`
	Document bool `json:"document,omitempty"`
	Source *DocumentSource `json:"source,omitempty"`
}

func (b *Controller) WorkspaceSnapshot() *WorkspaceSnapshot {
	if b == nil { return nil }
	saved := &WorkspaceSnapshot{Active: b.Active}
	for _, tab := range b.Tabs {
		item := WorkspaceTabSnapshot{ID: tab.ID, Draft: tab.Draft, Partition: tab.workspace.key, Label: tab.workspace.label}
		item.Document = tab.source != nil || tab.Document != nil || tab.HTML != nil || tab.Media != nil
		if tab.source != nil {
			source := *tab.source
			item.Source = &source
			item.Draft = source.Path
		}
		if !item.Document {
			item.URL = tab.resumeURL
			if tab.Host != nil { item.URL = NormalizeURL(tab.Host.State().URL) }
		}
		saved.Tabs = append(saved.Tabs, item)
	}
	return saved
}

// RestoreWorkspace does not create a native guest or navigate during recovery.
// The owning pane lazily reopens the selected resource when it is displayed.
func (b *Controller) RestoreWorkspace(saved *WorkspaceSnapshot) {
	if saved == nil || len(b.Tabs) != 0 { return }
	for _, item := range saved.Tabs {
		tab := &Tab{ID: item.ID, Draft: item.Draft, workspace: browserWorkspace{key: item.Partition, label: item.Label}}
		if item.Source != nil {
			source := *item.Source
			tab.source, tab.sourcePending, tab.Draft = &source, true, source.Path
		} else if item.Document {
			tab.Draft = ""
			tab.ActionMessage = "文档快照已结束，请从原文件重新打开预览。"
		} else {
			tab.resumeURL = NormalizeURL(item.URL)
		}
		b.Tabs = append(b.Tabs, tab)
		if tab.ID > b.next { b.next = tab.ID }
	}
	b.Active = saved.Active
	if b.Active < 0 || b.Active >= len(b.Tabs) { b.Active = 0 }
}

func (b *Controller) resumeCurrent() {
	tab := b.current()
	if tab == nil { return }
	if tab.source != nil { b.resumeDocument(tab); return }
	if tab.resumeURL == "" { return }
	workspace, err := b.browserWorkspace()
	if err != nil || tab.workspace.needsNewTab(workspace) {
		tab.ActionMessage = "此标签属于另一个工作区分区；切回原工作区后继续浏览，或点击前往新建当前分区页面。"
		return
	}
	raw := tab.resumeURL
	tab.resumeURL = ""
	b.Navigate(raw)
}

func (b *Controller) SetOnFocus(focus func()) { b.onFocus = focus }
func (b *Controller) focus() {
	if b.onFocus != nil { b.onFocus(); b.update() }
}
