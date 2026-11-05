package remote

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"time"
	remotestore "whaleisle/internal/store/remote"
	fonttheme "whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

// FileEditor belongs to one native pane occurrence, while Files retains every
// resource draft for that pane's Host/session owners.
type FileEditor struct {
	CopyBinding, CopiedPath, CopyError string
	CopiedUntil                        time.Time
	Actions                            FileActions
	Key                                string
	Revision                           uint64
	Editing, ConfirmDownload           bool
}

func viewSessionFiles(c *ui.Context, m Model) {
	if m.Files == nil || m.FileEditor == nil {
		return
	}
	s := m.Files.Snapshot()
	ui.Column(c).FillWidth().Gap(8).Children(func() {
		ui.Text(c, label("当前会话的远程文件", "Current session's remote files")).Bold()
		if s.SessionID == "" {
			ui.Text(c, label("先选择会话。", "Select a session first."))
			return
		}
		if !s.Loaded && !s.Busy && s.Error == "" {
			m.Files.Browse("")
		}
		fileWorkspaceBody(c, m)
	})
}

func fileResourceView(c *ui.Context, m Model, s remotestore.FileSnapshot) {
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	e := m.FileEditor
	key := s.BindingID + "\x00" + s.RemoteIdentity + "\x00" + remotestore.ResourceAddress(s.SessionID, s.FilePath)
	if e.Key != key || e.Revision != s.Revision {
		e.Key, e.Revision = key, s.Revision
		e.Editing = s.Dirty()
		for _, tab := range s.Tabs {
			if tab.Path == s.FilePath {
				e.Editing = e.Editing || tab.Editing
			}
		}
		e.ConfirmDownload = false
	}
	disabled := s.Busy || !m.HarnessReady || !m.RemoteWorkspaceEnabled || !s.Bound
	if s.FilePath == "" {
		ui.Text(c, label("从远程目录打开文件。", "Open a file from the remote tree."))
		return
	}
	ui.Text(c, s.FilePath).FontSize(fonttheme.InterfaceSize(12)).Selectable()
	if s.Error != "" {
		ui.Text(c, s.Error).FillWidth().Selectable()
	}
	widgets.Notice(c, "files:65", "", s.Notice != "", func() string { return s.Notice })
	if !s.FileLoaded {
		if s.Busy {
			ui.Text(c, label("正在读取…", "Reading…"))
		} else if widgets.Button(c, colors, label("重试读取", "Retry read"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Clicked() {
			m.Files.Reload()
		}
		return
	}
	ui.Text(c, fmt.Sprintf("%d bytes", s.File.Size)).FontSize(fonttheme.InterfaceSize(11))
	if s.Stale {
		ui.Text(c, label("连接或远程资源已变化，草稿保留；重新读取前先保存草稿副本或明确放弃。", "Connection or resource changed. Draft retained; copy it or explicitly discard before re-reading.")).FillWidth()
	}
	if s.File.Binary {
		ui.Text(c, label("二进制文件，只提供下载。", "Binary file; download is available."))
	} else if s.File.Truncated {
		ui.Text(c, label("预览已截断，禁止保存不完整文件。", "Preview truncated; saving incomplete contents is disabled."))
	}
	ui.Row(c).FillWidth().Gap(6).Wrap().Children(func() {
		if widgets.Button(c, colors, label("重新读取", "Reload"), widgets.ButtonSpec{Variant: widgets.ButtonToolbar, Size: widgets.ButtonSmall}).Disabled(disabled || s.Dirty() || !s.Bound).Clicked() {
			m.Files.Reload()
		}
		if !e.Editing && widgets.Button(c, colors, label("编辑", "Edit"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Disabled(disabled || s.Stale || s.File.Binary || s.File.Truncated || s.File.Mtime == nil).Clicked() {
			e.Editing = true
			m.Files.SetEditing(true)
		}
		if e.Editing {
			if widgets.Button(c, colors, label("保存到远端", "Save to remote"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall}).Disabled(disabled || s.Stale || !s.Dirty()).Clicked() {
				m.Files.Save()
			}
			if widgets.Button(c, colors, label("取消编辑", "Cancel editing"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Disabled(disabled).Clicked() {
				m.Files.Discard()
				m.Files.SetEditing(false)
				e.Editing = false
			}
		}
		if widgets.Button(c, colors, label("下载到会话镜像…", "Download to session mirror…"), widgets.ButtonSpec{Variant: widgets.ButtonToolbar, Size: widgets.ButtonSmall}).Disabled(disabled || s.Dirty() || !s.Bound).Clicked() {
			e.ConfirmDownload = true
		}
	})
	if !s.File.Binary {
		if e.Editing {
			source, ok := m.Files.EditBuffer(s, newFileTextBuffer).(*fileTextBuffer)
			if !ok {
				return
			}
			area := ui.TextAreaBuffer(c.Key(key), source.buffer).FillWidth().MinHeight(180).Grow(1).Disabled(disabled)
			save := func() {
				if !disabled && !s.Stale {
					m.Files.Save()
				}
			}
			area.OnShortcut(ui.Ctrl, ui.KeyS, save)
			area.OnShortcut(ui.Super, ui.KeyS, save)
			area.OnShortcut(0, ui.KeyEscape, func() {
				if !disabled {
					m.Files.Discard()
					m.Files.SetEditing(false)
					e.Editing = false
				}
			})
		} else {
			ui.Scroll(c).FillWidth().MinHeight(180).Grow(1).Children(func() { ui.Text(c, s.File.Content).FillWidth().Selectable().FontSize(fonttheme.InterfaceSize(12)) })
		}
	}
	if e.ConfirmDownload {
		ui.Text(c, label("下载会写入此会话的本地镜像，并可能覆盖同名文件。", "Download writes into this session's local mirror and may overwrite a same-named file.")).FillWidth()
		ui.Row(c).Gap(6).Children(func() {
			if widgets.Button(c, colors, label("取消", "Cancel"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Disabled(disabled).Clicked() {
				e.ConfirmDownload = false
			}
			if widgets.Button(c, colors, label("确认下载", "Confirm download"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall}).Disabled(disabled || s.Dirty()).Clicked() {
				e.ConfirmDownload = false
				m.Files.Download()
			}
		})
	}
}
