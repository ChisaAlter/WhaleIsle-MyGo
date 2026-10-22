package remote

import (
	"path/filepath"
	"strings"
	"time"
	"whaleisle/internal/ui/widgets"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
	remotestore "whaleisle/internal/store/remote"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

type FileWorkspaceState struct {
	Files   *remotestore.Files
	Editor  FileEditor
	visible bool
}

func NewFileWorkspaceState() *FileWorkspaceState {
	return &FileWorkspaceState{Files: remotestore.NewFiles()}
}
func (s *FileWorkspaceState) Dirty() bool { return s != nil && s.Files != nil && s.Files.HasDirty() }
func (s *FileWorkspaceState) PreflightDirtyForClose() error {
	if s == nil || s.Files == nil {
		return nil
	}
	return s.Files.PreflightDirtyForClose()
}
func (s *FileWorkspaceState) SaveDirtyForClose() error {
	if s == nil || s.Files == nil {
		return nil
	}
	return s.Files.SaveDirtyForClose()
}
func (s *FileWorkspaceState) CheckDirtyForClose() (bool, error) {
	if s == nil || s.Files == nil {
		return true, nil
	}
	return s.Files.CheckDirtyForClose()
}
func (s *FileWorkspaceState) Suspend() {
	if s != nil && s.Files != nil {
		s.Files.Suspend()
		s.visible = false
	}
}

// FileWorkspace contributes only an authoritatively resolved remote session.
// State is retained by the native pane occurrence, so two panes opening
// the same path do not share edit pointers, close dialogs or resource selection.
func FileWorkspace(c *ui.Context, m Model) bool {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	anchor := ui.Column(c.Key("remote-file-workspace"))
	state := m.FileWorkspace
	if state == nil || state.Files == nil {
		return false
	}
	wasVisible := state.visible
	state.visible = true
	state.Files.BindSession(m.FileClient, m.FileSessionID, m.FileCwd, m.FileCurrent, m.OnInvalidate)
	if m.FileClient == nil || m.FileSessionID == "" || !m.HarnessReady || !m.RemoteWorkspaceEnabled {
		state.Suspend()
		return false
	}
	// A scratch session has no folder to mirror. Asking the Host to resolve one
	// returned a 404 JSON error and left the right pane with a retry button; show
	// the real reason instead of a failing remote call.
	cwd := strings.TrimSpace(m.FileCwd)
	if cwd == "" || strings.EqualFold(filepath.Base(filepath.Clean(cwd)), "no-workspace") {
		anchor.Grow(1).FillWidth().MinHeight(0).Padding(12, 16, 12, 16).Gap(4).Children(func() {
			ui.Text(c, label("此会话没有工作区文件夹", "This session has no workspace folder")).FontSize(theme.InterfaceSize(13)).TextColor(chrome.Color(colors.FG))
			ui.Text(c, label("在左侧选择或添加项目目录后，这里会显示工作区文件。", "Choose or add a project folder on the left to see its files here.")).FontSize(theme.InterfaceSize(11)).TextColor(chrome.Color(colors.Muted))
		})
		return true
	}
	s := state.Files.Snapshot()
	if !s.Resolved && !s.Resolving && s.ResolveError == "" {
		state.Files.Resolve()
	}
	if !wasVisible && s.Resolved && !s.Local && !s.Busy {
		state.Files.Browse(s.RootPath)
	}
	if !s.Resolved || s.Local || s.RootPath == "" {
		if s.ResolveError != "" {
			anchor.FillWidth().Gap(4).Children(func() {
				ui.Text(c, s.ResolveError).FillWidth()
				if widgets.Button(c, colors, label("重试远程工作区解析", "Retry remote workspace resolution"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Disabled(s.Resolving).Clicked() {
					state.Files.Resolve()
				}
			})
		}
		return false
	}
	m.Files, m.FileEditor = state.Files, &state.Editor
	anchor.Grow(1).FillWidth().MinHeight(0).MinWidth(0).Gap(8).Children(func() { fileWorkspaceBody(c, m) })
	return true
}

func fileWorkspaceBody(c *ui.Context, m Model) {
	s := m.Files.Snapshot()
	if m.FileEditor.CopyBinding != remoteFileActionKey(s) {
		m.FileEditor.CopyBinding = remoteFileActionKey(s)
		m.FileEditor.CopiedPath, m.FileEditor.CopyError = "", ""
	}
	disabled := s.Busy || !m.HarnessReady || !m.RemoteWorkspaceEnabled
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	ui.Row(c).FillWidth().Gap(6).Children(func() {
		ui.Text(c, label("远程文件", "Remote files")).Bold().Grow(1).MinWidth(0)
		if widgets.Button(c, colors, label("刷新", "Refresh"), widgets.ButtonSpec{Variant: widgets.ButtonToolbar, Size: widgets.ButtonSmall}).Disabled(disabled).Clicked() {
			m.Files.Browse(s.RootPath)
		}
	})
	if s.RootPath != "" {
		ui.Text(c, s.RootPath).FillWidth().FontSize(theme.InterfaceSize(11)).TextColor(chrome.Color(colors.Muted))
	}
	if s.MirrorDir != "" {
		ui.Text(c, s.MirrorDir).FillWidth().FontSize(theme.InterfaceSize(11)).TextColor(chrome.Color(colors.Muted))
	}
	if s.Busy {
		ui.Text(c, label("正在处理：", "Working: ")+s.Operation).FontSize(theme.InterfaceSize(11))
	}
	if s.TreeError != "" {
		ui.Text(c, s.TreeError).FillWidth().Selectable()
	}
	if m.FileEditor.CopyError != "" {
		ui.Text(c, m.FileEditor.CopyError).FillWidth()
	}
	if s.ResolveError != "" {
		ui.Text(c, s.ResolveError).FillWidth()
		if widgets.Button(c, colors, label("重试解析", "Retry resolution"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Disabled(s.Resolving).Clicked() {
			m.Files.Resolve()
		}
	}
	_, height := c.Size()
	treeHeight := min(float32(300), max(float32(120), height*.4))
	ui.Scroll(c.Key("remote-tree")).FillWidth().Height(treeHeight).Children(func() {
		ui.Column(c).FillWidth().MinWidth(0).Gap(0).Children(func() {
			if s.RootPath == "" {
				if s.Error != "" {
					ui.Text(c, s.Error).FillWidth()
				} else {
					ui.Text(c, label("正在解析远程工作区…", "Resolving remote workspace…"))
				}
				return
			}
			root := remotestore.DirectoryEntry{Name: label("远程工作区", "Remote workspace"), Path: s.RootPath, Type: "dir"}
			fileTreeRow(c, m, s, root, 0, disabled)
			var level func(string, int)
			level = func(path string, depth int) {
				if !s.Expanded[path] {
					return
				}
				d := s.Tree[path]
				if d.Error != "" {
					ui.Text(c, d.Error).FillWidth().FontSize(theme.InterfaceSize(11))
					if widgets.Button(c, colors, label("重试目录", "Retry directory"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Disabled(disabled).Clicked() {
						m.Files.Browse(path)
					}
				}
				if d.Missing {
					ui.Text(c, label("远程目录不存在。", "Remote directory is missing."))
				}
				if d.Loading && !d.Loaded {
					ui.Text(c, label("正在读取…", "Loading…"))
				}
				if d.Loaded && len(d.Items) == 0 {
					ui.Text(c, label("空目录", "Empty directory")).FontSize(theme.InterfaceSize(11))
				}
				for _, entry := range d.Items {
					fileTreeRow(c, m, s, entry, depth, disabled)
					if entry.IsDirectory() {
						level(entry.Path, depth+1)
					}
				}
			}
			level(s.RootPath, 1)
		})
	})
	remoteFileActionsView(c, m, s, disabled)
	ui.Row(c.Key("remote-resources")).FillWidth().MinWidth(0).Wrap().Gap(4).Role(ui.RoleTabList).Children(func() {
		for _, resource := range s.Tabs {
			resource := resource
			title := strings.ReplaceAll(resource.Path, "\\", "/")
			if slash := strings.LastIndex(title, "/"); slash >= 0 {
				title = title[slash+1:]
			}
			if resource.Dirty() {
				title += " •"
			}
			ui.Row(c.Key(remotestore.ResourceAddress(s.SessionID, resource.Path))).Gap(2).Children(func() {
				if ui.Button(c, title).Role(ui.RoleTab).Selected(s.FilePath == resource.Path).Disabled(s.Busy).Clicked() {
					m.Files.Open(resource.Path)
				}
				if ui.Button(c, "×").Label(label("关闭远程文件 ", "Close remote file ") + resource.Path).Disabled(s.Busy).Clicked() {
					m.Files.RequestClose(resource.Path)
				}
			})
		}
	})
	ui.Column(c.Key("remote-resource:" + s.FilePath)).FillWidth().Grow(1).MinWidth(0).MinHeight(0).Gap(6).Children(func() { fileResourceView(c, m, s) })
	if s.PendingClose != "" {
		open := true
		ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
			width, _ := c.Size()
			chrome.DialogSurface(c, backdrop, panel, colors, min(float32(480), max(float32(0), width-32)), 20).Gap(12).Children(func() {
				ui.Text(c, label("关闭未保存的远程文件？", "Close an unsaved remote file?")).Bold()
				ui.Text(c, s.PendingClose).FillWidth()
				ui.Row(c).Gap(8).Wrap().Children(func() {
					if widgets.Button(c, colors, label("取消", "Cancel"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
						m.Files.ConfirmClose(false)
					}
					if widgets.Button(c, colors, label("放弃草稿并关闭", "Discard draft and close"), widgets.ButtonSpec{Variant: widgets.ButtonDanger, Size: widgets.ButtonSmall}).Clicked() {
						m.Files.ConfirmClose(true)
					}
				})
			})
		})
		if !open {
			m.Files.ConfirmClose(false)
		}
	}
}

func fileTreeRow(c *ui.Context, m Model, s remotestore.FileSnapshot, item remotestore.DirectoryEntry, depth int, disabled bool) {
	caption := item.Name
	if item.IsDirectory() {
		prefix := "▸ "
		if s.Expanded[item.Path] {
			prefix = "▾ "
		}
		if s.Tree[item.Path].Loading {
			prefix = "◌ "
		}
		caption = prefix + caption
	} else {
		caption = "  " + caption
	}
	open := func() {
		if item.IsDirectory() {
			m.Files.ToggleDirectory(item.Path)
		} else {
			m.Files.Open(item.Path)
		}
	}
	ui.Row(c.Key(s.BindingID+"\x00"+s.RemoteIdentity+"\x00"+item.Path)).FillWidth().MinWidth(0).Padding(0, 0, 0, float32(depth)*14).Children(func() {
		row := ui.Button(c, caption).Grow(1).MinWidth(0).Height(26).FontSize(theme.InterfaceSize(12)).BorderWidth(0).Disabled(disabled || !s.Bound)
		row.ContextMenu(func(menu *ui.Menu) {
			if menu.Item(label("打开 / 展开", "Open / expand")).Disabled(disabled).Chosen() {
				open()
			}
			if menu.Item(label("下载到会话镜像", "Download to session mirror")).Disabled(disabled || !s.Bound).Chosen() {
				m.Files.DownloadEntry(item.Path)
			}
			if menu.Item(label("复制相对路径", "Copy relative path")).Chosen() {
				path, root := strings.ReplaceAll(item.Path, "\\", "/"), strings.TrimRight(strings.ReplaceAll(s.RootPath, "\\", "/"), "/")
				if strings.HasPrefix(path, root+"/") {
					path = strings.TrimPrefix(path, root+"/")
				}
				copyRemoteTreePath(m.FileEditor, path, item.Path)
			}
			if menu.Item(label("复制完整路径", "Copy absolute path")).Chosen() {
				copyRemoteTreePath(m.FileEditor, item.Path, item.Path)
			}
			menu.Separator()
			blocked := disabled || !s.Bound || !s.Loaded || item.Drive
			if menu.Item(label("重命名", "Rename")).Disabled(blocked || item.Path == s.RootPath).Chosen() {
				m.FileEditor.Actions = FileActions{Key: remoteFileActionKey(s), Operation: "rename", Path: item.Path, Name: item.Name, Directory: item.IsDirectory()}
			}
			if menu.Item(label("新建文件夹", "New folder")).Disabled(blocked).Chosen() {
				base := item.Path
				if !item.IsDirectory() {
					base = remotestore.ParentDirectory(base)
				}
				m.FileEditor.Actions = FileActions{Key: remoteFileActionKey(s), Operation: "mkdir", Path: base}
			}
			menu.Separator()
			if menu.Item(label("删除", "Delete")).Disabled(blocked || item.Path == s.RootPath).Chosen() {
				m.FileEditor.Actions = FileActions{Key: remoteFileActionKey(s), Operation: "remove", Path: item.Path, Directory: item.IsDirectory()}
			}
		})
		if row.Clicked() {
			open()
		}
		if m.FileEditor.CopiedPath == item.Path && time.Now().Before(m.FileEditor.CopiedUntil) {
			ui.Text(c, label("已复制", "Copied")).FontSize(theme.InterfaceSize(11))
			c.After(time.Until(m.FileEditor.CopiedUntil))
		}
	})
}

func copyRemoteTreePath(editor *FileEditor, text, path string) {
	if err := mygo.Clipboard.Write(transfer.TextData(text), mygo.ClipboardOptions{}); err != nil {
		editor.CopyError = err.Error()
		return
	}
	editor.CopyError, editor.CopiedPath = "", path
	editor.CopiedUntil = time.Now().Add(1200 * time.Millisecond)
}
