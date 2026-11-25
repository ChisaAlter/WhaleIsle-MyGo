package remote

import (
	"github.com/egoist/mygo/ui"
	remotestore "whaleisle/internal/store/remote"
	"whaleisle/internal/ui/coss"
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
	s := m.Store.Snapshot().Value
	if !s.Loaded && !s.Busy && s.Error == nil {
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
			opts := make([]coss.SelectOption, len(choices))
			for i, choice := range choices {
				opts[i] = coss.SelectOption{Value: choice, Label: choice}
			}
			if len(opts) > 0 && coss.Select(c, &selected, opts, coss.SelectOptions{Label: label("选择远程机器", "Select remote machine"), Disabled: s.Busy}).Grow(1).Changed() {
				m.Store.Select(ids[selected])
			}
			if coss.Button(c, label("刷新机器", "Refresh machines"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonMD, Disabled: s.Busy}).Clicked() {
				m.Store.Refresh()
			}
		})
		if s.Error != nil {
			ui.Text(c, s.Error.SentenceText()).FillWidth().Selectable()
		}
		if s.Loaded && len(s.Machines) == 0 {
			ui.Text(c, label("先在设置 → 远程工作区添加机器。", "Add a machine in Settings → Remote workspace first."))
		}
		if s.CurrentID != "" && !s.Busy && !s.Directory.Loaded && s.Directory.Error == nil && s.Error == nil {
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
				if coss.Button(c, label("新建文件夹", "New folder"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: directoryBlocked(model, s) || !s.Directory.CanChoose()}).Clicked() {
					m.Store.ShowDirectoryCreate(true)
				}
				ui.Box(c).Grow(1)
				if coss.Button(c, label("取消", "Cancel"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM}).Clicked() {
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
