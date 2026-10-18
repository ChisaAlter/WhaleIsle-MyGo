package terminal

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/hostterminal"
)

func hostRenameView(c *ui.Context, store *hostterminal.Store, v hostterminal.View, session string, muted ui.Color) {
	edit := store.RenameSnapshot(v.Owner, session, v.ActiveID)
	if !edit.Editing {
		if ui.Text(c, v.Info.Title).FontSize(12).Disabled(v.Phase == "closed" || v.Phase == "closing").DoubleClicked() {
			store.BeginRename(v.Owner, session, v.ActiveID)
		}
		if ui.Button(c, terminalLabel("重命名终端", "Rename terminal")).FontSize(11).Disabled(v.ActiveID == "" || v.Phase == "closed" || v.Phase == "closing").Clicked() {
			store.BeginRename(v.Owner, session, v.ActiveID)
		}
		return
	}
	ui.Column(c).Key(fmt.Sprintf("host-title:%d:%s:%s", v.Owner, session, v.ActiveID)).Gap(4).Children(func() {
		draft := edit.Draft
		input := ui.TextInput(c, &draft).FillWidth().Label(terminalLabel("终端名称", "Terminal name")).Placeholder(terminalLabel("最多120字符", "Up to 120 characters")).Disabled(edit.Saving).AutoFocus()
		if draft != edit.Draft {
			store.EditRename(v.Owner, session, v.ActiveID, draft)
		}
		ui.Row(c).Gap(4).Children(func() {
			save := ui.Button(c, terminalLabel("保存名称", "Save name")).FontSize(11).Disabled(edit.Saving || !v.Ready).Clicked()
			if (save || input.Submitted()) && !edit.Saving {
				store.SaveRename(v.Owner, session, v.ActiveID)
			}
			if ui.Button(c, terminalLabel("取消", "Cancel")).FontSize(11).Disabled(edit.Saving).Clicked() || input.Shortcut(0, ui.KeyEscape) {
				store.CancelRename(v.Owner, session, v.ActiveID)
			}
		})
		if edit.Saving {
			ui.Text(c, terminalLabel("正在保存名称…", "Saving name…")).FontSize(11).TextColor(muted)
		}
		if edit.Error != "" {
			ui.Text(c, edit.Error).FontSize(11).TextColor(muted)
		}
	})
}
