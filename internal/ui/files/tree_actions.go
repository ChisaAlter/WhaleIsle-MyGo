package files

import (
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/ui/filelink"
)

type treeCopies struct{ relative, absolute, reference string }

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

// Directories cannot open a preview tab, so their path actions must live on
// the tree row itself. This mirrors FileTree's directory context menu.
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
