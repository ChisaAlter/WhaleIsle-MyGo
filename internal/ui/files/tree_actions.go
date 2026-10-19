package files

import (
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/ui/filelink"
	"whaleisle/internal/ui/i18n"
)

type treeCopies struct{ relative, absolute, reference string }

// Match deferred native menu choices to the same active resource generation.
// The reference already scopes the row to its Host/session/workspace path.
type treeFileKey struct {
	reference filelink.Reference
	active    previewToolbarOwner
}

func treeFileRowKey(m Model, path string) treeFileKey {
	var active previewToolbarOwner
	if m.SelectedFile != nil && *m.SelectedFile == path {
		active = previewToolbarOwnerFor(m, path)
	}
	return treeFileKey{workspaceReference(m, path), active}
}

// Copying a path is lexical, like FilesPanel.absoluteOf, and works for Host
// workspaces whose filesystem is not mounted on the native client's computer.
func treePathCopies(m Model, path string) (treeCopies, bool) {
	reference, ok := filelink.Serialize(path)
	if !ok || strings.TrimSpace(m.WorkspaceRoot) == "" {
		return treeCopies{}, false
	}
	root := strings.TrimRight(strings.ReplaceAll(m.WorkspaceRoot, `\`, "/"), "/")
	return treeCopies{path, root + "/" + path, reference}, true
}

func revealTreePath(m Model, path string) {
	values, ok := treePathCopies(m, path)
	if !ok || m.OnReveal == nil {
		return
	}
	if m.ListRemoteContext != nil {
		// session/openWorkspacePath resolves relative paths against the Host
		// process, not the selected workspace, so pass the Client-resolved path.
		m.OnReveal(values.absolute)
		return
	}
	full, err := workspacefs.JoinRoot(m.WorkspaceRoot, path)
	if err != nil {
		setFileStatus(m, err.Error())
		return
	}
	m.OnReveal(full)
}

// Tree path actions share a context menu; occasional copy actions do not
// consume every file row's width. Directories do not need a preview tab.
func treeDirectoryMenu(m Model, path string) func(*ui.Menu) {
	return func(menu *ui.Menu) {
		values, ok := treePathCopies(m, path)
		if !ok {
			return
		}
		if menu.Item(label("复制引用", "Copy reference")).Chosen() {
			mygo.Clipboard.WriteText(values.reference)
		}
		if menu.Item(label("复制相对路径", "Copy relative path")).Chosen() {
			mygo.Clipboard.WriteText(values.relative)
		}
		if menu.Item(label("复制绝对路径", "Copy absolute path")).Chosen() {
			mygo.Clipboard.WriteText(values.absolute)
		}
		if m.OnReveal != nil && menu.Item(label("在文件管理器中显示", "Show in folder")).Chosen() {
			revealTreePath(m, path)
		}
	}
}

func treeFileMenu(m Model, path string) func(*ui.Menu) {
	paths := treeDirectoryMenu(m, path)
	return func(menu *ui.Menu) {
		paths(menu)
		if m.OnOpenDoc != nil && menu.Item(i18n.T(i18n.KeyFilesOpenExternal)).Chosen() {
			withToolbarPath(m, path, textPreviewPath(m, path), func(full string) { m.OnOpenDoc(full, workspacefs.ClassifyDoc(path), false) })
		}
		previewContextActions(menu, m, path)
	}
}

func paintTreeTwist(c *ui.Context, expanded bool, ink ui.Color) {
	ui.Box(c).Width(12).Height(12).Draw(func(p *ui.Painter, r ui.Rect) {
		path := &ui.Path{}
		if expanded {
			path.MoveTo(r.X+3, r.Y+4)
			path.LineTo(r.X+6, r.Y+7)
			path.LineTo(r.X+9, r.Y+4)
		} else {
			path.MoveTo(r.X+4, r.Y+3)
			path.LineTo(r.X+7, r.Y+6)
			path.LineTo(r.X+4, r.Y+9)
		}
		p.StrokePath(path, 1.2, ink)
	})
}

func paintTreeReferenceButton(c *ui.Context, m Model, row ui.Element, path string, muted ui.Color) {
	if m.OnReferenceFile == nil {
		return
	}
	button := ui.Button(c.Key("tree-reference"), "@").Label("Reference file " + path).Tooltip(label("引用此文件到对话", "Reference file in chat")).Width(20).Height(20).Padding(0).BorderWidth(0).Background(ui.Transparent).TextColor(muted).FontSize(12).Disabled(!m.CanAddToChat)
	if !row.Hovered() && !row.FocusWithin() && !button.Focused() {
		button.Opacity(0)
	}
	if button.Clicked() {
		if err := m.OnReferenceFile(workspaceReference(m, path)); err != nil {
			setFileStatus(m, err.Error())
		} else {
			setFileStatus(m, label("文件引用已插入对话草稿", "File reference inserted into chat draft"))
		}
	}
}

func paintTreeEntryIcon(c *ui.Context, folder bool, ink ui.Color) {
	ui.Box(c).Width(14).Height(14).Draw(func(p *ui.Painter, r ui.Rect) {
		path := &ui.Path{}
		if folder {
			path.MoveTo(r.X+1, r.Y+4)
			path.LineTo(r.X+5, r.Y+4)
			path.LineTo(r.X+6, r.Y+6)
			path.LineTo(r.X+13, r.Y+6)
			path.LineTo(r.X+13, r.Y+12)
			path.LineTo(r.X+1, r.Y+12)
			path.Close()
		} else {
			path.MoveTo(r.X+4, r.Y+4)
			path.LineTo(r.X+1, r.Y+7)
			path.LineTo(r.X+4, r.Y+10)
			path.MoveTo(r.X+10, r.Y+4)
			path.LineTo(r.X+13, r.Y+7)
			path.LineTo(r.X+10, r.Y+10)
			path.MoveTo(r.X+8, r.Y+3)
			path.LineTo(r.X+6, r.Y+11)
		}
		p.StrokePath(path, 1, ink)
	})
}
