package remote

import (
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	remotestore "whaleisle/internal/store/remote"
)

type FileActions struct {
	Key, Operation, Path, Name, Error string
	Directory, Submitted bool
}

func fileRowActions(c *ui.Context, m Model, item remotestore.DirectoryEntry, s remotestore.FileSnapshot, disabled bool) {
	a := &m.FileEditor.Actions
	if ui.Button(c, label("复制路径", "Copy path")).Disabled(disabled).Clicked() { mygo.Clipboard.WriteText(item.Path) }
	if ui.Button(c, label("复制相对路径", "Copy relative path")).Disabled(disabled || s.RootPath == "").Clicked() {
		p, root := strings.ReplaceAll(item.Path, "\\", "/"), strings.TrimRight(strings.ReplaceAll(s.RootPath, "\\", "/"), "/")
		relative := p
		if p == root { relative = "." } else if strings.HasPrefix(p, root+"/") { relative = strings.TrimPrefix(p, root+"/") }
		mygo.Clipboard.WriteText(relative)
	}
	blocked := disabled || s.Dirty() || !s.Bound || !s.Loaded || item.Drive
	if ui.Button(c, label("重命名", "Rename")).Disabled(blocked).Clicked() {
		*a = FileActions{Key: s.BindingID+"\x00"+s.RemoteIdentity+"\x00"+s.Path, Operation: "rename", Path: item.Path, Name: item.Name, Directory: item.IsDirectory()}
	}
	if ui.Button(c, label("删除", "Delete")).Disabled(blocked).Clicked() {
		*a = FileActions{Key: s.BindingID+"\x00"+s.RemoteIdentity+"\x00"+s.Path, Operation: "remove", Path: item.Path, Directory: item.IsDirectory()}
	}
}

func remoteFileActionsView(c *ui.Context, m Model, s remotestore.FileSnapshot, disabled bool) {
	a := &m.FileEditor.Actions
	key := s.BindingID + "\x00" + s.RemoteIdentity + "\x00" + s.Path
	if a.Key != "" && a.Key != key { *a = FileActions{} }
	if a.Submitted && !s.Busy {
		a.Submitted = false
		if s.MutationApplied { *a = FileActions{} }
	}
	if ui.Button(c, label("新建远程文件夹", "New remote folder")).Disabled(disabled || s.Dirty() || !s.Bound || !s.Loaded).Clicked() {
		*a = FileActions{Key: key, Operation: "mkdir", Path: s.Path}
	}
	if a.Operation == "" { return }
	title := label("新建文件夹于：", "Create folder in: ")
	if a.Operation == "rename" { title = label("重命名：", "Rename: ") }
	if a.Operation == "remove" { title = label("永久删除：", "Permanently delete: ") }
	ui.Text(c, title + a.Path).Bold()
	if a.Operation == "remove" {
		warning := label("此操作直接删除远程文件，无法从应用恢复。", "This deletes the remote file directly; the app cannot restore it.")
		if a.Directory { warning = label("此操作会递归删除远程文件夹及所有内容，无法从应用恢复。", "This recursively deletes the remote folder and its contents; the app cannot restore them.") }
		ui.Text(c, warning)
	} else {
		ui.TextInput(c, &a.Name).FillWidth().Disabled(disabled).Placeholder(label("名称", "Name"))
		if a.Operation == "rename" {
			ui.Text(c, label("将改为同目录下的新名称。若远端同时创建同名文件，SSH 重命名可能覆盖它。", "Renames within the same directory. A same-named file created concurrently on the remote may be overwritten by SSH rename."))
		}
	}
	if a.Error != "" { ui.Text(c, a.Error) }
	ui.Row(c).Gap(6).Children(func() {
		if ui.Button(c, label("取消", "Cancel")).Disabled(disabled).Clicked() { *a = FileActions{} }
		title := label("确认创建", "Confirm create")
		if a.Operation == "rename" { title = label("确认重命名", "Confirm rename") }
		if a.Operation == "remove" { title = label("确认永久删除", "Confirm permanent deletion") }
		if ui.Button(c, title).Disabled(disabled || s.Dirty() || !s.Bound || !s.Loaded).Clicked() {
			var err error
			switch a.Operation {
			case "mkdir": err = m.Files.Mkdir(a.Name)
			case "rename": err = m.Files.Rename(a.Path, a.Name)
			case "remove": err = m.Files.Remove(a.Path)
			}
			if err != nil { a.Error = err.Error() } else { a.Error, a.Submitted = "", true }
		}
	})
}
