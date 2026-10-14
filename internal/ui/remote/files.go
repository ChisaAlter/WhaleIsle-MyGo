package remote

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	remotestore "whaleisle/internal/store/remote"
)

// FileEditor keeps the native text input pointer stable while Files owns per-session drafts.
type FileEditor struct {
	Key                      string
	Revision                 uint64
	Draft                    string
	Editing, ConfirmDownload bool
}

func viewSessionFiles(c *ui.Context, m Model) {
	if m.Files == nil || m.FileEditor == nil {
		return
	}
	s := m.Files.Snapshot()
	e := m.FileEditor
	key := s.BindingID + "\x00" + s.FilePath
	if e.Key != key || e.Revision != s.Revision {
		e.Key = key
		e.Revision = s.Revision
		e.Draft = s.Draft
		e.Editing = s.Dirty()
		e.ConfirmDownload = false
	}
	ui.Column(c).FillWidth().Gap(8).Children(func() {
		ui.Text(c, label("当前会话的远程文件", "Current session's remote files")).Bold()
		if s.SessionID == "" {
			ui.Text(c, label("先选择已绑定远程工作区的会话。", "Select a session bound to a remote workspace first.")).FontSize(12)
			return
		}
		ui.Text(c, s.SessionID).FontSize(11)
		ui.Text(c, label("所有读取、保存和下载都绑定此会话，不使用当前机器默认目录。", "Reads, saves and downloads use this session binding, never the active machine default.")).FontSize(11)
		disabled := s.Busy || !m.HarnessReady || !m.RemoteWorkspaceEnabled
		ui.Row(c).Gap(6).Children(func() {
			if ui.Button(c, label("浏览 / 刷新会话目录", "Browse / refresh session directory")).Disabled(disabled || s.Dirty()).Clicked() {
				m.Files.Browse(s.Path)
			}
			if ui.Button(c, label("上级", "Up")).Disabled(disabled || s.Dirty() || !s.Loaded || s.Path == "" || s.Path == "/").Clicked() {
				m.Files.Browse(remotestore.ParentDirectory(s.Path))
			}
		})
		if s.Busy {
			ui.Text(c, label("正在处理：", "Working: ")+s.Operation).FontSize(12)
		}
		if s.Error != "" {
			ui.Text(c, s.Error).FontSize(12)
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(12)
		}
		ui.Text(c, s.Path).FontSize(11)
		if s.Loaded && s.Bound && len(s.Items) == 0 {
			ui.Text(c, label("此远程目录为空。", "This remote directory is empty.")).FontSize(12)
		}
		ui.Scroll(c).Height(160).Children(func() {
			ui.Column(c).FillWidth().Gap(3).Children(func() {
				for _, item := range s.Items {
					item := item
					caption := item.Name
					if item.IsDirectory() {
						caption = "▸ " + caption
					}
					if ui.Button(c, caption).FillWidth().Disabled(disabled || s.Dirty() || !s.Bound).Clicked() {
						if item.IsDirectory() {
							m.Files.Browse(item.Path)
						} else {
							m.Files.Open(item.Path)
						}
					}
				}
			})
		})
		if !s.FileLoaded {
			return
		}
		ui.Text(c, s.FilePath).FontSize(12)
		ui.Text(c, fmt.Sprintf("%d bytes", s.File.Size)).FontSize(11)
		if s.Stale {
			ui.Text(c, label("原连接或保存状态已变化。草稿保留；请复制草稿，明确放弃后重读远端再编辑。", "Connection or save state changed. Draft retained; copy it, explicitly discard, then re-read the remote file before editing.")).FontSize(12)
		}
		if s.File.Binary {
			ui.Text(c, label("二进制文件，只提供下载。", "Binary file; download is available.")).FontSize(12)
		} else if s.File.Truncated {
			ui.Text(c, label("预览已截断，禁止保存不完整文件。", "Preview is truncated; saving incomplete contents is disabled.")).FontSize(12)
		}
		ui.Row(c).Gap(6).Children(func() {
			if ui.Button(c, label("重新读取", "Reload")).Disabled(disabled || s.Dirty() || !s.Bound).Clicked() {
				m.Files.Open(s.FilePath)
			}
			if ui.Button(c, label("编辑", "Edit")).Disabled(disabled || s.Stale || s.File.Binary || s.File.Truncated || s.File.Mtime == nil).Clicked() {
				e.Editing = true
			}
			if ui.Button(c, label("保存到远端", "Save to remote")).Disabled(disabled || s.Stale || !s.Dirty()).Clicked() {
				m.Files.Save()
			}
			if ui.Button(c, label("放弃未保存草稿", "Discard unsaved draft")).Disabled(disabled || !s.Dirty()).Clicked() {
				m.Files.Discard()
				e.Editing = false
			}
		})
		if !s.File.Binary {
			if e.Editing {
				area := ui.TextArea(c, &e.Draft).FillWidth().Height(180).Disabled(disabled)
				if area.Changed() {
					m.Files.SetDraft(e.Draft)
				}
			} else {
				ui.Scroll(c).Height(180).Children(func() { ui.Text(c, s.File.Content).FontSize(12) })
			}
		}
		if ui.Button(c, label("下载到会话镜像…", "Download to session mirror…")).Disabled(disabled || s.Dirty() || !s.Bound).Clicked() {
			e.ConfirmDownload = true
		}
		if e.ConfirmDownload {
			ui.Text(c, label("下载会写入此会话的本地镜像，并可能覆盖同名文件。远端未保存草稿不会被上传。", "Download writes into this session's local mirror and may overwrite the same-named file. Unsaved remote drafts are not uploaded.")).FontSize(12)
			ui.Row(c).Gap(6).Children(func() {
				if ui.Button(c, label("取消", "Cancel")).Disabled(disabled).Clicked() {
					e.ConfirmDownload = false
				}
				if ui.Button(c, label("确认下载", "Confirm download")).Disabled(disabled || s.Dirty()).Clicked() {
					e.ConfirmDownload = false
					m.Files.Download()
				}
			})
		}
	})
}
