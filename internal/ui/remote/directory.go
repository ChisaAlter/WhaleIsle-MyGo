package remote

import (
	"github.com/egoist/mygo/ui"
	remotestore "whaleisle/internal/store/remote"
	fonttheme "whaleisle/internal/ui/theme"
)

func viewDirectory(c *ui.Context, m Model) {
	if m.Machines == nil {
		return
	}
	s := m.Machines.Snapshot()
	d := s.Directory
	if s.CurrentID == "" {
		return
	}
	disabled := s.Busy || s.Error != "" || !m.HarnessReady || !m.RemoteWorkspaceEnabled
	ui.Column(c).Gap(8).Children(func() {
		ui.Text(c, label("远程目录 / 工作区镜像", "Remote directory / workspace mirror")).Bold()
		ui.Text(c, label("浏览当前选择的机器。创建镜像后，点击使用以切换工作区；不会自动同步文件。", "Browse the selected machine. Create a mirror, then use it to switch workspace; files are not synced automatically.")).FontSize(fonttheme.InterfaceSize(11))
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, emptyDash(d.Path)).Grow(1)
			if ui.Button(c, label("远程主目录", "Remote home")).Disabled(disabled).Clicked() {
				m.Machines.BrowseHome()
			}
			if ui.Button(c, label("机器默认目录", "Machine default directory")).Disabled(disabled).Clicked() {
				m.Machines.Browse("")
			}
			if ui.Button(c, label("上级", "Up")).Disabled(disabled || !d.Loaded || d.Path == "" || d.Path == "/").Clicked() {
				m.Machines.Browse(remotestore.ParentDirectory(d.Path))
			}
			if ui.Button(c, label("浏览 / 刷新", "Browse / refresh")).Disabled(disabled).Clicked() {
				m.Machines.Browse(d.Path)
			}
		})
		path := d.InputPath
		ui.Row(c).Gap(8).Children(func() {
			ui.TextInput(c, &path).FillWidth().Disabled(disabled).Placeholder(label("远程绝对路径", "Remote absolute path")).OnChange(func() { m.Machines.SetDirectoryPath(path) })
			if ui.Button(c, label("跳转", "Go")).Disabled(disabled || path == "").Clicked() {
				m.Machines.Browse(path)
			}
		})
		if s.Busy {
			if d.Choosing {
				ui.Text(c, label("正在创建镜像…", "Creating mirror…"))
			} else {
				ui.Text(c, label("正在加载…", "Loading…"))
			}
		}
		if d.Error != "" {
			ui.Text(c, d.Error)
		}
		if d.Loaded && len(d.Items) == 0 && d.Error == "" {
			ui.Text(c, label("此目录为空", "This directory is empty"))
		}
		for _, entry := range d.Items {
			entry := entry
			ui.Row(c).Gap(8).Children(func() {
				ui.Text(c, entry.Name).Grow(1)
				if entry.IsDirectory() && ui.Button(c, label("打开目录", "Open directory")).Disabled(disabled || entry.Path == "").Clicked() {
					m.Machines.Browse(entry.Path)
				}
			})
		}
		if ui.Button(c, label("新建文件夹", "New folder")).Disabled(disabled || !d.CanChoose()).Clicked() {
			m.Machines.ShowDirectoryCreate(true)
		}
		if d.MkdirOpen {
			ui.Text(c, label("创建于：", "Create in: ")+d.Path)
			name := d.NewName
			ui.TextInput(c, &name).FillWidth().Disabled(disabled).Placeholder(label("文件夹名称", "Folder name")).OnChange(func() { m.Machines.SetDirectoryName(name) })
			ui.Row(c).Gap(8).Children(func() {
				if ui.Button(c, label("取消创建", "Cancel creation")).Disabled(disabled).Clicked() {
					m.Machines.ShowDirectoryCreate(false)
				}
				if ui.Button(c, label("确认创建文件夹", "Create folder")).Disabled(disabled || !d.CanChoose() || name == "").Clicked() {
					if err := m.Machines.CreateDirectory(name); err != nil {
						m.Machines.SetDirectoryError(err.Error())
					}
				}
			})
		}
		if ui.Button(c, label("选择此目录并创建镜像", "Select directory and create mirror")).Disabled(disabled || !d.CanChoose()).Clicked() {
			m.Machines.ChooseDirectory()
		}
		if d.LocalMirror != "" {
			ui.Text(c, label("已选目录：", "Selected directory: ")+d.SelectedPath).FontSize(fonttheme.InterfaceSize(11))
			ui.Text(c, label("本地镜像：", "Local mirror: ")+d.LocalMirror).FontSize(fonttheme.InterfaceSize(11))
			if m.OnUseWorkspaceMirror != nil && ui.Button(c, label("使用此镜像作为工作区", "Use this mirror as workspace")).Disabled(disabled).Clicked() {
				m.OnUseWorkspaceMirror(d.LocalMirror)
			}
		}
	})
}
