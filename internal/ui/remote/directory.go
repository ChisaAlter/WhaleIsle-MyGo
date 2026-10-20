package remote

import (
	"strings"
	"whaleisle/internal/ui/widgets"

	"github.com/egoist/mygo/ui"
	remotestore "whaleisle/internal/store/remote"
	fonttheme "whaleisle/internal/ui/theme"
)

// viewDirectory is the file page: a breadcrumb over the current path, one row per
// entry with its kind and name, the actions that change the directory, and the
// three states (loading, empty, error) told apart instead of run together.
func viewDirectory(c *ui.Context, m Model) {
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	if m.Machines == nil {
		return
	}
	s := m.Machines.Snapshot()
	d := s.Directory
	if s.CurrentID == "" {
		return
	}
	disabled := s.Busy || s.Error != "" || !m.HarnessReady || !m.RemoteWorkspaceEnabled
	muted, border := c.Theme().TextMuted, c.Theme().Border
	ui.Column(c).FillWidth().MinWidth(0).Gap(10).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, label("远程文件", "Remote files")).Bold().Grow(1).MinWidth(0)
			if widgets.Button(c, colors, label("远程主目录", "Remote home"), widgets.ButtonSpec{Variant: widgets.ButtonToolbar, Size: widgets.ButtonSmall}).Disabled(disabled).Clicked() {
				m.Machines.BrowseHome()
			}
			if widgets.Button(c, colors, label("上级", "Up"), widgets.ButtonSpec{Variant: widgets.ButtonToolbar, Size: widgets.ButtonSmall}).Disabled(disabled || !d.Loaded || d.Path == "" || d.Path == "/").Clicked() {
				m.Machines.Browse(remotestore.ParentDirectory(d.Path))
			}
			if widgets.Button(c, colors, label("刷新", "Refresh"), widgets.ButtonSpec{Variant: widgets.ButtonToolbar, Size: widgets.ButtonSmall}).Disabled(disabled).Clicked() {
				m.Machines.Browse(d.Path)
			}
			if widgets.Button(c, colors, label("新建文件夹", "New folder"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall}).Disabled(disabled || !d.CanChoose()).Clicked() {
				m.Machines.ShowDirectoryCreate(true)
			}
		})
		directoryBreadcrumb(c, m, d, disabled, muted)
		if d.MkdirOpen {
			directoryCreate(c, m, d, disabled, muted)
		}
		if s.Busy {
			ui.Text(c, label("正在读取目录…", "Reading the directory…")).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			return
		}
		if d.Error != "" {
			ui.Column(c).FillWidth().MinWidth(0).Gap(6).Padding(14).Radius(10).Border(.5, border).Children(func() {
				ui.Text(c, label("无法读取此目录", "This directory could not be read")).FontSize(fonttheme.InterfaceSize(13)).FontWeight(500).TextColor(c.Theme().Danger)
				ui.Text(c, d.Error).FillWidth().FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				if widgets.Button(c, colors, label("重试", "Retry"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Disabled(disabled).Clicked() {
					m.Machines.Browse(d.Path)
				}
			})
			return
		}
		if !d.Loaded {
			return
		}
		if len(d.Items) == 0 {
			ui.Column(c).FillWidth().MinWidth(0).Gap(4).Padding(28, 16).Radius(10).Border(.5, border).AlignItems(ui.Center).Children(func() {
				ui.Text(c, label("此目录为空", "This directory is empty")).FontSize(fonttheme.InterfaceSize(13)).FontWeight(500)
				ui.Text(c, label("新建文件夹可在此建立第一个条目。", "Create a folder to put the first entry here.")).
					TextAlign(ui.Center).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			})
			return
		}
		ui.Column(c).FillWidth().MinWidth(0).Gap(1).Children(func() {
			for _, entry := range d.Items {
				directoryRow(c, m, entry, disabled, muted)
			}
		})
		if d.LocalMirror != "" {
			ui.Text(c, label("本地镜像：", "Local mirror: ")+d.LocalMirror).FillWidth().FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			if m.OnUseWorkspaceMirror != nil && widgets.Button(c, colors, label("使用此镜像作为工作区", "Use this mirror as workspace"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall}).Disabled(disabled).Clicked() {
				m.OnUseWorkspaceMirror(d.LocalMirror)
			}
		}
	})
}

// directoryBreadcrumb is the clickable path above the listing; each segment
// returns to that ancestor.
func directoryBreadcrumb(c *ui.Context, m Model, d remotestore.DirectorySnapshot, disabled bool, muted ui.Color) {
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(2).AlignItems(ui.Center).Children(func() {
		path := strings.TrimSpace(d.Path)
		if path == "" {
			ui.Text(c, label("机器默认目录", "Machine default directory")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			return
		}
		segments := splitRemotePath(path)
		current := ""
		for index, segment := range segments {
			if index > 0 {
				ui.Text(c, "/").FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			}
			if strings.HasPrefix(path, "/") && index == 0 {
				current = "/"
				segment = "/"
			} else if current == "" || current == "/" {
				current += segment
			} else {
				current += "/" + segment
			}
			target := current
			caption := segment
			if target == path {
				ui.Text(c, caption).FontSize(fonttheme.InterfaceSize(12)).FontWeight(500)
				continue
			}
			if ui.Button(c, caption).FontSize(fonttheme.InterfaceSize(12)).BorderWidth(0).Padding(0, 4).Background(ui.Transparent).TextColor(muted).Disabled(disabled).Clicked() {
				m.Machines.Browse(target)
			}
		}
	})
}

// splitRemotePath breaks a POSIX path into breadcrumb segments.
func splitRemotePath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return []string{"/"}
	}
	return strings.Split(trimmed, "/")
}

// directoryRow is one entry: a kind icon, the name, and the single action that
// applies to it.
func directoryRow(c *ui.Context, m Model, entry remotestore.DirectoryEntry, disabled bool, muted ui.Color) {
	dir := entry.IsDirectory()
	row := ui.Row(c.Key("remote-entry-"+entry.Path)).FillWidth().MinWidth(0).Gap(10).Height(30).Padding(0, 8).Radius(6).AlignItems(ui.Center)
	if row.Hovered() {
		row.Background(c.Theme().SurfaceHover)
	}
	row.Children(func() {
		art := remoteFileIcon
		if dir {
			art = remoteFolderIcon
		}
		ui.Icon(c, art).Size(13, 13).TextColor(muted)
		name := entry.Name
		if name == "" {
			name = entry.Path
		}
		if dir {
			if ui.Button(c, name).Label(name).BorderWidth(0).Padding(0).Justify(ui.Start).Background(ui.Transparent).Grow(1).MinWidth(0).Disabled(disabled || entry.Path == "").Clicked() {
				m.Machines.Browse(entry.Path)
			}
		} else {
			ui.Text(c, name).Grow(1).MinWidth(0).SingleLine().Ellipsis("…").Tooltip(name).FontSize(fonttheme.InterfaceSize(12))
		}
		if dir {
			ui.Text(c, label("目录", "Folder")).FontSize(fonttheme.InterfaceSize(10)).TextColor(muted)
		} else {
			ui.Text(c, label("文件", "File")).FontSize(fonttheme.InterfaceSize(10)).TextColor(muted)
		}
	})
}

// directoryCreate is the inline new-folder form.
func directoryCreate(c *ui.Context, m Model, d remotestore.DirectorySnapshot, disabled bool, muted ui.Color) {
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	ui.Column(c).FillWidth().MinWidth(0).Gap(6).Padding(12).Radius(10).Border(.5, c.Theme().Border).Children(func() {
		ui.Text(c, label("新建文件夹于：", "Create a folder in: ")+d.Path).FillWidth().FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		name := d.NewName
		ui.TextInput(c, &name).FillWidth().Disabled(disabled).Placeholder(label("文件夹名称", "Folder name")).OnChange(func() { m.Machines.SetDirectoryName(name) })
		ui.Row(c).FillWidth().Gap(8).Children(func() {
			if widgets.Button(c, colors, label("取消", "Cancel"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Disabled(disabled).Clicked() {
				m.Machines.ShowDirectoryCreate(false)
			}
			if ui.PrimaryButton(c, label("创建", "Create")).Disabled(disabled || !d.CanChoose() || strings.TrimSpace(name) == "").Clicked() {
				if err := m.Machines.CreateDirectory(name); err != nil {
					m.Machines.SetDirectoryError(err.Error())
				}
			}
		})
	})
}

// Entry-kind artwork for the file page, in the 16px outline set the rest of the
// surfaces use.
var (
	remoteFolderIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2"><path d="M2 12.5V4.2c0-.4.3-.7.7-.7h3.1l1.4 1.6h5.3c.4 0 .7.3.7.7v6.7c0 .4-.3.7-.7.7H2.7a.7.7 0 0 1-.7-.7Z"/></svg>`))
	remoteFileIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2"><path d="M4 2.2h4.7L12.5 6v7.8H4Z"/><path d="M8.7 2.2V6h3.8"/></svg>`))
)