package remote

import (
	"github.com/egoist/mygo/ui"
	remotestore "whaleisle/internal/store/remote"
)

type DirectoryPickerModel struct {
	Store          *remotestore.Store
	Ready, Enabled bool
	OnPicked       func(localMirror string)
	OnChoice       func(remotestore.DirectoryChoice)
	OnCancel       func()
}

// DirectoryPicker is the same remote directory flow for the composer and
// sidebar workspace entry points. It never adopts a local path before /mirror
// acknowledges the explicitly selected remote directory.
func DirectoryPicker(c *ui.Context, m DirectoryPickerModel) {
	if m.Store == nil || !m.Ready || !m.Enabled {
		ui.Text(c, label("启用远程工作区并连接 Host 后可选择远程目录。", "Enable remote workspaces and connect the Host to choose a remote directory."))
		return
	}
	s := m.Store.Snapshot()
	if !s.Loaded && !s.Busy && s.Error == "" {
		m.Store.Refresh()
	}
	ui.Column(c).FillWidth().Grow(1).MinWidth(0).MinHeight(0).Gap(10).Children(func() {
		ui.Row(c).FillWidth().Shrink(0).MinWidth(0).Wrap().Gap(8).Children(func() {
			ui.Text(c, label("远程机器", "Remote machine"))
			choices, ids, selected := []string{}, map[string]string{}, ""
			for _, machine := range s.Machines {
				name := machine.Name
				if name == "" {
					name = machine.Host
				}
				caption := name + " · " + machineAddress(machine)
				if _, exists := ids[caption]; exists {
					caption += " · " + machine.ID
				}
				choices = append(choices, caption)
				ids[caption] = machine.ID
				if machine.ID == s.CurrentID {
					selected = caption
				}
			}
			if len(choices) > 0 && ui.Select(c, &selected, choices).Label(label("选择远程机器", "Select remote machine")).Grow(1).Disabled(s.Busy).Changed() {
				m.Store.Select(ids[selected])
			}
			if ui.Button(c, label("刷新机器", "Refresh machines")).Disabled(s.Busy).Clicked() {
				m.Store.Refresh()
			}
		})
		if s.Error != "" {
			ui.Text(c, s.Error).FillWidth().Selectable()
		}
		if s.Loaded && len(s.Machines) == 0 {
			ui.Text(c, label("先在设置 → 远程工作区添加机器。", "Add a machine in Settings → Remote workspace first."))
		}
		if s.CurrentID != "" && !s.Busy && !s.Directory.Loaded && s.Directory.Error == "" && s.Error == "" {
			m.Store.Browse("")
		}
		model := Model{Machines: m.Store, HarnessReady: m.Ready, RemoteWorkspaceEnabled: m.Enabled, OnUseWorkspaceMirror: m.OnPicked, OnUseWorkspaceChoice: m.OnChoice}
		if s.CurrentID != "" {
			directoryHeader(c, model, s)
		}
		ui.Scroll(c).FillWidth().Grow(1).MinHeight(0).Children(func() { directoryContents(c, model, s) })
		ui.Column(c).FillWidth().Shrink(0).MinWidth(0).Gap(8).Children(func() {
			ui.Box(c).FillWidth().Height(.5).Background(c.Theme().Border)
			ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
				if ui.Button(c, label("新建文件夹", "New folder")).Disabled(directoryBlocked(model, s) || !s.Directory.CanChoose()).Clicked() {
					m.Store.ShowDirectoryCreate(true)
				}
				ui.Box(c).Grow(1)
				if ui.Button(c, label("取消", "Cancel")).Clicked() {
					m.Store.CancelDirectoryChoice()
					if m.OnCancel != nil {
						m.OnCancel()
					}
				}
				directoryChoose(c, model, s)
			})
		})
		directoryCreate(c, model, s.Directory, s.Busy, c.Theme().TextMuted)
	})
}
