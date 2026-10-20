package terminal

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/hostterminal"
)

func hostShellView(c *ui.Context, store *hostterminal.Store, v hostterminal.View, session string, muted ui.Color) {
	shells := store.ShellSnapshot(v.Owner, session)
	if ui.Button(c, terminalLabel("选择或刷新 shell", "Choose or refresh the shell")).FontSize(11).Disabled(!v.Ready || session == "" || v.Busy || shells.Loading).Clicked() {
		store.DiscoverShells(v.Owner, session)
	}
	if shells.Loading {
		ui.Text(c, terminalLabel("正在查询 Host shell…", "Discovering Host shells…")).FontSize(11).TextColor(muted)
	}
	if len(shells.Items) > 0 {
		paths := make([]string, len(shells.Items))
		for i, item := range shells.Items {
			paths[i] = item.Path
		}
		selected := shells.SelectedPath
		if ui.Select(c, &selected, paths).FillWidth().Disabled(!v.Ready || v.Busy || shells.Loading || !shells.Loaded).Changed() {
			store.SelectShell(v.Owner, session, selected)
		}
		for _, item := range shells.Items {
			if item.Path == shells.SelectedPath {
				ui.Text(c, item.Name).FontSize(11).TextColor(muted)
				break
			}
		}
	}
	ui.Text(c, terminalLabel("选择会记在本机；仅点击「新建 Host 终端」后启动。已有终端保留自己的 shell。", "Choice is remembered on this client. Only New Host terminal starts it; existing terminals keep their own shell.")).FontSize(10).TextColor(muted)
	if shells.Error != "" {
		ui.Text(c, shells.Error).FontSize(11).TextColor(muted)
	}
	if shells.PreferenceError != "" {
		ui.Text(c, shells.PreferenceError).FontSize(11).TextColor(muted)
	}
}
