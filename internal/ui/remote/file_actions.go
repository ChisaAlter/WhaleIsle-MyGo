package remote

import (
	"strings"
	"whaleisle/internal/ui/widgets"

	"github.com/egoist/mygo/ui"
	remotestore "whaleisle/internal/store/remote"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

type FileActions struct {
	Key, Operation, Path, Name, Error string
	Directory, Submitted              bool
}

func remoteFileActionKey(s remotestore.FileSnapshot) string {
	return s.BindingID + "\x00" + s.RemoteIdentity
}

func remoteFileActionsView(c *ui.Context, m Model, s remotestore.FileSnapshot, disabled bool) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	a := &m.FileEditor.Actions
	key := remoteFileActionKey(s)
	if a.Key != "" && a.Key != key {
		*a = FileActions{}
	}
	if a.Submitted && !s.Busy {
		a.Submitted = false
		if s.MutationApplied {
			*a = FileActions{}
		} else if s.Error != "" {
			a.Error = s.Error
		}
	}
	if widgets.Button(c, colors, label("新建远程文件夹", "New remote folder"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall}).Disabled(disabled || !s.Bound || !s.Loaded).Clicked() {
		*a = FileActions{Key: key, Operation: "mkdir", Path: s.RootPath}
	}
	if a.Operation == "" {
		return
	}
	confirm := func() {
		if disabled || !s.Bound || !s.Loaded || a.Operation != "remove" && strings.TrimSpace(a.Name) == "" {
			return
		}
		var err error
		switch a.Operation {
		case "mkdir":
			err = m.Files.MkdirAt(a.Path, a.Name)
		case "rename":
			err = m.Files.Rename(a.Path, a.Name)
		case "remove":
			err = m.Files.Remove(a.Path)
		}
		if err != nil {
			a.Error = err.Error()
		} else {
			a.Error, a.Submitted = "", true
		}
	}
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		width, _ := c.Size()
		chrome.DialogSurface(c, backdrop, panel, colors, min(float32(480), max(float32(0), width-32)), 20).Gap(10).Children(func() {
			title := label("新建文件夹于：", "Create folder in: ")
			if a.Operation == "rename" {
				title = label("重命名：", "Rename: ")
			}
			if a.Operation == "remove" {
				title = label("永久删除：", "Permanently delete: ")
			}
			ui.Text(c, title+a.Path).FillWidth().Bold()
			if a.Operation == "remove" {
				warning := label("此操作直接删除远程文件，无法从应用恢复。", "This deletes the remote file directly; the app cannot restore it.")
				if a.Directory {
					warning = label("此操作会递归删除远程文件夹及所有内容，无法从应用恢复。", "This recursively deletes the remote folder and its contents; the app cannot restore them.")
				}
				ui.Text(c, warning).FillWidth()
			} else {
				ui.TextInput(c, &a.Name).FillWidth().Disabled(disabled).Placeholder(label("名称", "Name")).AutoFocus().OnSubmit(confirm)
			}
			if a.Error != "" {
				ui.Text(c, a.Error).FillWidth()
			}
			if a.Submitted && s.Error != "" {
				ui.Text(c, s.Error).FillWidth()
			}
			ui.Row(c).Gap(8).Wrap().Children(func() {
				if widgets.Button(c, colors, label("取消", "Cancel"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Disabled(disabled).Clicked() {
					*a = FileActions{}
				}
				title := label("确认创建", "Confirm create")
				if a.Operation == "rename" {
					title = label("确认重命名", "Confirm rename")
				}
				if a.Operation == "remove" {
					title = label("确认永久删除", "Confirm permanent deletion")
				}
				if ui.Button(c, title).Disabled(disabled || !s.Bound || !s.Loaded || a.Operation != "remove" && strings.TrimSpace(a.Name) == "").Clicked() {
					confirm()
				}
			})
		})
	})
	if !open && !disabled {
		*a = FileActions{}
	}
}
