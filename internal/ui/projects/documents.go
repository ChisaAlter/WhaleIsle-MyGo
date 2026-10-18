package projects

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	fonttheme "whaleisle/internal/ui/theme"
)

func documentsView(c *ui.Context, m Model) {
	d := m.Documents
	action := func(kind, path string) {
		if m.OnDocumentAction != nil {
			m.OnDocumentAction(kind, path)
		}
	}
	ui.Text(c, tr("项目资料", "Project documents")+" · "+d.Title).FontSize(fonttheme.InterfaceSize(13))
	ui.Text(c, d.Scope.ProjectID).FontSize(fonttheme.InterfaceSize(11)).Selectable()
	// Closing and Back stay enabled during a read and cancel only this reader.
	if ui.Button(c, tr("返回 Projects", "Back to Projects")).Clicked() {
		action("close", "")
	}
	if d.Error != "" {
		ui.Text(c, d.Error).FontSize(fonttheme.InterfaceSize(12)).Selectable()
		if d.TextLoaded {
			ui.Text(c, tr("读取失败；仍显示本资料上次读取的正文。", "Read failed; showing this document's previously fetched contents.")).FontSize(fonttheme.InterfaceSize(11))
		}
	}
	if d.SelectedPath != "" {
		ui.Row(c).Gap(4).Children(func() {
			if ui.Button(c, tr("返回资料清单", "Back to documents")).Clicked() {
				action("back", "")
			}
			if ui.Button(c, tr("重新读取正文", "Reload document")).Disabled(m.Busy).Clicked() {
				action("read", d.SelectedPath)
			}
		})
		ui.Text(c, d.SelectedPath).FontSize(fonttheme.InterfaceSize(12)).Selectable()
		ui.Text(c, tr("同一 Host 的当前只读正文。内容和链接仅按文字显示。", "Current read-only text from this Host. Content and links are displayed literally.")).FontSize(fonttheme.InterfaceSize(11))
		if d.TextLoaded {
			if d.Text == "" {
				ui.Text(c, tr("此资料正文为空。", "This document is empty.")).FontSize(fonttheme.InterfaceSize(12))
			}
			// Native read-only text keeps the entire body selectable and scrollable;
			// there is no Markdown link handler, external opener or file execution.
			text := d.Text
			ui.TextArea(c.Key("project-document-"+d.Scope.ProjectID+"-"+d.SelectedPath), &text).FillWidth().Height(320).ReadOnly(true).FontSize(fonttheme.InterfaceSize(12))
		}
		return
	}
	if ui.Button(c, tr("刷新资料清单", "Refresh documents")).Disabled(m.Busy).Clicked() {
		action("list", "")
	}
	ui.Text(c, tr("从 Project 的 docs 读取资料正文，最多 512000 字节。", "Read Project docs as text, up to 512000 bytes.")).FontSize(fonttheme.InterfaceSize(11))
	if d.Loaded && len(d.Docs) == 0 {
		ui.Text(c, tr("此项目还没有独立资料。", "This Project has no standalone documents yet.")).FontSize(fonttheme.InterfaceSize(12))
	}
	ui.Scroll(c).Height(300).Children(func() {
		ui.Column(c).FillWidth().Gap(6).Children(func() {
			for _, entry := range d.Docs {
				entry := entry
				readErr := projectflow.DocumentReadError(entry)
				if ui.Button(c, entry.Path).FillWidth().Disabled(m.Busy || readErr != nil).Clicked() {
					action("read", entry.Path)
				}
				ui.Text(c, fmt.Sprintf("%d bytes", entry.Size)).FontSize(fonttheme.InterfaceSize(11))
				if readErr != nil {
					ui.Text(c, readErr.Error()).FontSize(fonttheme.InterfaceSize(11))
				}
			}
		})
	})
}
