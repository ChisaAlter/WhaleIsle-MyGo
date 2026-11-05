package terminal

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/hostterminal"
)

func hostShellView(c *ui.Context, store *hostterminal.Store, v hostterminal.View, session string, muted ui.Color) {
	shells := store.ShellSnapshot(v.Owner, session)
	if ui.Button(c, terminalLabel("选择或刷新 shell", "Choose or refresh the shell")).FontSize(12).Disabled(!v.Ready || session == "" || v.Busy || shells.Loading).Clicked() {
		store.DiscoverShells(v.Owner, session)
	}
	if len(shells.Items) > 0 {
		selected := terminalLabel("选择 shell", "Choose shell")
		for _, item := range shells.Items {
			if item.Path == shells.SelectedPath {
				selected = item.Name
				break
			}
		}
		ui.MenuButton(c, selected, func(menu *ui.Menu) {
			for _, item := range shells.Items {
				name := item.Name
				if name == "" {
					name = item.Path
				}
				if menu.Item(name).Checked(item.Path == shells.SelectedPath).Chosen() {
					store.SelectShell(v.Owner, session, item.Path)
				}
			}
		}).FillWidth().Disabled(!v.Ready || v.Busy || shells.Loading || !shells.Loaded).Tooltip(shells.SelectedPath)
	}
	ui.Text(c, terminalLabel("选择会记在本机；仅点击「新建 Host 终端」后启动。已有终端保留自己的 shell。", "Choice is remembered on this client. Only New Host terminal starts it; existing terminals keep their own shell.")).FontSize(12).TextColor(muted)
	if shells.Error != "" {
		ui.Text(c, shells.Error).FontSize(12).TextColor(muted)
	}
	if shells.PreferenceError != "" {
		ui.Text(c, shells.PreferenceError).FontSize(12).TextColor(muted)
	}
}
