package remote

import (
	"github.com/egoist/mygo/ui"
	"strings"
	remotestore "whaleisle/internal/store/remote"
	"whaleisle/internal/ui/chrome"
	fonttheme "whaleisle/internal/ui/theme"
)

func viewDirectory(c *ui.Context, m Model) {
	if m.Machines == nil {
		return
	}
	s := m.Machines.Snapshot()
	if s.CurrentID == "" {
		return
	}
	ui.Column(c).FillWidth().MinWidth(0).Gap(10).Children(func() {
		directoryHeader(c, m, s)
		ui.Scroll(c).FillWidth().Height(280).MinHeight(0).Children(func() { directoryContents(c, m, s) })
		ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
			if ui.Button(c, label("新建文件夹", "New folder")).Disabled(directoryBlocked(m, s) || !s.Directory.CanChoose()).Clicked() {
				m.Machines.ShowDirectoryCreate(true)
			}
			directoryChoose(c, m, s)
		})
		directoryCreate(c, m, s.Directory, s.Busy, c.Theme().TextMuted)
	})
}

func directoryBlocked(m Model, s remotestore.Snapshot) bool {
	return s.Busy || s.Error != "" || !m.HarnessReady || !m.RemoteWorkspaceEnabled
}

// Header and footer never join the scrolling directory contents.
func directoryHeader(c *ui.Context, m Model, s remotestore.Snapshot) {
	d, muted, disabled := s.Directory, c.Theme().TextMuted, directoryBlocked(m, s)
	ui.Column(c).FillWidth().MinWidth(0).Shrink(0).Gap(8).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, label("远程文件", "Remote files")).Bold().Grow(1).MinWidth(0)
			if ui.Button(c, label("主目录", "Home")).Disabled(disabled).Clicked() {
				m.Machines.BrowseHome()
			}
			parent := remotestore.ParentDirectory(d.Path)
			if ui.Button(c, label("上级", "Up")).Disabled(disabled || !d.Loaded || d.Path == "" || parent == d.Path).Clicked() {
				m.Machines.Browse(parent)
			}
			if ui.Button(c, label("刷新", "Refresh")).Disabled(disabled).Clicked() {
				m.Machines.Browse(d.Path)
			}
		})
		directoryBreadcrumb(c, m, d, disabled, muted)
		path := d.InputPath
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
			input := ui.TextInput(c, &path).Label(label("远程目录路径", "Remote directory path")).Grow(1).MinWidth(0).Disabled(disabled)
			input.OnChange(func() { m.Machines.SetDirectoryPath(path) })
			input.OnShortcut(0, ui.KeyEnter, func() {
				if !disabled {
					m.Machines.Browse(path)
				}
			})
			if ui.Button(c, label("前往", "Go")).Disabled(disabled).Clicked() {
				m.Machines.Browse(path)
			}
		})
	})
}

func directoryContents(c *ui.Context, m Model, s remotestore.Snapshot) {
	d, muted, disabled := s.Directory, c.Theme().TextMuted, directoryBlocked(m, s)
	ui.Column(c).FillWidth().MinWidth(0).Gap(4).Children(func() {
		if s.Busy {
			text := label("正在读取目录…", "Reading the directory…")
			if d.Choosing {
				text = label("正在建立本地镜像…", "Preparing the local mirror…")
			}
			ui.Text(c, text).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			return
		}
		if d.Error != "" {
			ui.Text(c, label("无法读取此目录", "This directory could not be read")).FillWidth().Bold().TextColor(c.Theme().Danger)
			ui.Text(c, d.Error).FillWidth().Selectable().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			if ui.Button(c, label("重试", "Retry")).Disabled(disabled).Clicked() {
				m.Machines.Browse(d.Path)
			}
			return
		}
		if !d.Loaded {
			return
		}
		if len(d.Items) == 0 {
			ui.Text(c, label("此目录为空", "This directory is empty")).FillWidth().Padding(24).TextAlign(ui.Center).TextColor(muted)
		}
		for _, entry := range d.Items {
			directoryRow(c, m, entry, disabled, muted)
		}
	})
}

func directoryChoose(c *ui.Context, m Model, s remotestore.Snapshot) {
	d, disabled := s.Directory, directoryBlocked(m, s)
	if (m.OnUseWorkspaceChoice != nil || m.OnUseWorkspaceMirror != nil) && ui.PrimaryButton(c, label("选择此目录作为工作区", "Use this directory as workspace")).Disabled(disabled || !d.CanChoose()).Clicked() {
		if m.OnUseWorkspaceChoice != nil {
			m.Machines.ChooseDirectoryWithChoice(m.OnUseWorkspaceChoice)
		} else {
			m.Machines.ChooseDirectoryWithResult(m.OnUseWorkspaceMirror)
		}
	}
	if d.LocalMirror != "" && d.SelectedPath == d.Path {
		ui.Text(c, label("本地镜像：", "Local mirror: ")+d.LocalMirror).FillWidth().FontSize(fonttheme.InterfaceSize(11)).TextColor(c.Theme().TextMuted)
	}
}

type remoteCrumb struct{ name, path string }

// Compute targets in the remote platform's syntax, not this desktop's OS.
func remoteBreadcrumbs(path string) []remoteCrumb {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	windows := strings.Contains(path, `\`) || len(path) >= 2 && path[1] == ':'
	separator := "/"
	normalized := path
	if windows {
		separator = `\`
		normalized = strings.ReplaceAll(path, "/", separator)
	}
	parts := strings.FieldsFunc(normalized, func(r rune) bool { return r == '/' || windows && r == '\\' })
	crumbs := []remoteCrumb{}
	current := ""
	if windows && strings.HasPrefix(normalized, `\\`) {
		if len(parts) >= 2 {
			current = `\\` + parts[0] + separator + parts[1]
			crumbs = append(crumbs, remoteCrumb{current, current + separator})
			parts = parts[2:]
		}
	} else if windows && len(parts) > 0 && strings.HasSuffix(parts[0], ":") {
		current = parts[0] + separator
		crumbs = append(crumbs, remoteCrumb{current, current})
		parts = parts[1:]
	} else if strings.HasPrefix(normalized, "/") {
		current = "/"
		crumbs = append(crumbs, remoteCrumb{"/", "/"})
	}
	for _, part := range parts {
		if current != "" && !strings.HasSuffix(current, separator) {
			current += separator
		}
		current += part
		crumbs = append(crumbs, remoteCrumb{part, current})
	}
	return crumbs
}

func directoryBreadcrumb(c *ui.Context, m Model, d remotestore.DirectorySnapshot, disabled bool, muted ui.Color) {
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(2).AlignItems(ui.Center).Children(func() {
		crumbs := remoteBreadcrumbs(d.Path)
		if len(crumbs) == 0 {
			ui.Text(c, label("机器默认目录", "Machine default directory")).TextColor(muted)
			return
		}
		for index, crumb := range crumbs {
			if index > 0 {
				ui.Text(c, "›").TextColor(muted)
			}
			if index == len(crumbs)-1 {
				ui.Text(c, crumb.name).FontSize(fonttheme.InterfaceSize(12)).FontWeight(500)
				continue
			}
			if ui.Button(c, crumb.name).FontSize(fonttheme.InterfaceSize(12)).BorderWidth(0).Padding(0, 4).Background(ui.Transparent).TextColor(muted).Disabled(disabled).Clicked() {
				m.Machines.Browse(crumb.path)
			}
		}
	})
}

func splitRemotePath(path string) []string {
	crumbs := remoteBreadcrumbs(path)
	names := make([]string, len(crumbs))
	for index, crumb := range crumbs {
		names[index] = crumb.name
	}
	return names
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

// New-folder work is a modal; its draft and error stay in the Store until an
// acknowledged create or an explicit cancel.
func directoryCreate(c *ui.Context, m Model, d remotestore.DirectorySnapshot, disabled bool, muted ui.Color) {
	if !d.MkdirOpen {
		return
	}
	open := true
	ui.DialogBaseWithOptions(c.Key("remote-new-folder"), &open, ui.DialogOptions{DismissDisabled: disabled, OnDismiss: func() { m.Machines.ShowDirectoryCreate(false) }}, func(backdrop, panel ui.Element) {
		_, height := c.Size()
		chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), 420, 20).MaxHeight(height - 24).Gap(12).Children(func() {
			ui.Text(c, label("新建文件夹", "New folder")).Bold()
			ui.Text(c, d.Path).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted).Selectable()
			name := d.NewName
			input := ui.TextInput(c, &name).Label(label("文件夹名称", "Folder name")).FillWidth().Disabled(disabled).AutoFocus()
			input.OnChange(func() { m.Machines.SetDirectoryName(name) })
			create := func() {
				if !disabled && d.CanChoose() && strings.TrimSpace(name) != "" {
					if err := m.Machines.CreateDirectory(name); err != nil {
						m.Machines.SetDirectoryCreateError(err.Error())
					}
				}
			}
			input.OnShortcut(0, ui.KeyEnter, create)
			if d.CreateError != "" {
				ui.Text(c, d.CreateError).FillWidth().Selectable().TextColor(c.Theme().Danger)
			}
			ui.Row(c).FillWidth().Justify(ui.End).Gap(8).Children(func() {
				if ui.Button(c, label("取消", "Cancel")).Disabled(disabled).Clicked() {
					m.Machines.ShowDirectoryCreate(false)
				}
				if ui.PrimaryButton(c, label("创建", "Create")).Disabled(disabled || !d.CanChoose() || strings.TrimSpace(name) == "").Clicked() {
					create()
				}
			})
		})
	})
}

// Entry-kind artwork for the file page, in the 16px outline set the rest of the
// surfaces use.
var (
	remoteFolderIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2"><path d="M2 12.5V4.2c0-.4.3-.7.7-.7h3.1l1.4 1.6h5.3c.4 0 .7.3.7.7v6.7c0 .4-.3.7-.7.7H2.7a.7.7 0 0 1-.7-.7Z"/></svg>`))
	remoteFileIcon   = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2"><path d="M4 2.2h4.7L12.5 6v7.8H4Z"/><path d="M8.7 2.2V6h3.8"/></svg>`))
)
