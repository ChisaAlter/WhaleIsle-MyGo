package terminal

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/hostterminal"
	"whaleisle/internal/ui/coss"
)

// BeginHostTitleRename captures the selected terminal in this occurrence. The
// Store enforces the Host epoch, visible session and active terminal identity.
// A hidden occurrence must be activated and shown before calling this helper.
func BeginHostTitleRename(store *hostterminal.Store, session string) bool {
	if store == nil {
		return false
	}
	v := store.Snapshot(session).Value
	if v.ActiveID == "" {
		return false
	}
	store.BeginRename(v.Owner, session, v.ActiveID)
	return store.RenameSnapshot(v.Owner, session, v.ActiveID).Editing
}

// HostTitleEditor paints only an active rename. Call from the DockKit title
// seat; true means the seat must not also paint its normal title or drag target.
// A failed rename retains its draft and error in this same fixed-height input.
func HostTitleEditor(c *ui.Context, store *hostterminal.Store, session string) bool {
	if store == nil {
		return false
	}
	v := store.Snapshot(session).Value
	edit := store.RenameSnapshot(v.Owner, session, v.ActiveID)
	if !edit.Editing {
		return false
	}
	draft := edit.Draft
	var input ui.Element
	ui.Box(c).Grow(1).MinWidth(0).MaxWidth(160).Children(func() {
		input = coss.Input(c, &draft, coss.InputOptions{
			Key:      fmt.Sprintf("host-chip-title:%d:%s:%s", v.Owner, session, v.ActiveID),
			Size:     coss.InputSM,
			Label:    terminalLabel("终端名称", "Terminal name"),
			Disabled: edit.Saving,
			Invalid:  edit.Error != "",
			Error:    edit.Error,
		}).Height(24).AutoFocus()
	})
	selected := ui.Local(input, "selected-once", func() bool { return false })
	if !*selected && !edit.Saving {
		input.SetTextSelection(0, len([]rune(draft))).Focus()
		*selected = true
	}
	if edit.Error != "" {
		input.Tooltip(edit.Error)
	} else {
		input.Tooltip(terminalLabel("Enter 保存，Esc 取消", "Enter saves, Esc cancels"))
	}
	if input.Changed() {
		store.EditRename(v.Owner, session, v.ActiveID, draft)
	}
	wasFocused := ui.Local(input, "focused", func() bool { return false })
	focused, blurred := input.Focused(), *wasFocused && !input.Focused()
	*wasFocused = focused
	if edit.Saving || input.Composing() {
		return true
	}
	if input.Shortcut(0, ui.KeyEscape) {
		store.CancelRename(v.Owner, session, v.ActiveID)
	} else if (input.Submitted() && !input.Composing()) || blurred {
		store.SaveRename(v.Owner, session, v.ActiveID)
	}
	return true
}

func hostRenameView(c *ui.Context, store *hostterminal.Store, v hostterminal.View, session string, muted ui.Color) {
	edit := store.RenameSnapshot(v.Owner, session, v.ActiveID)
	if !edit.Editing {
		if ui.Text(c, v.Info.Title).FillWidth().MinWidth(0).SingleLine().Ellipsis("…").Tooltip(v.Info.Title).FontSize(12).Disabled(v.Phase == "closed" || v.Phase == "closing").DoubleClicked() {
			store.BeginRename(v.Owner, session, v.ActiveID)
		}
		if coss.Button(c, terminalLabel("重命名终端", "Rename terminal"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS, Disabled: v.ActiveID == "" || v.Phase == "closed" || v.Phase == "closing"}).Clicked() {
			store.BeginRename(v.Owner, session, v.ActiveID)
		}
		return
	}
	ui.Column(c).Key(fmt.Sprintf("host-title:%d:%s:%s", v.Owner, session, v.ActiveID)).Gap(4).Children(func() {
		draft := edit.Draft
		input := coss.Input(c, &draft, coss.InputOptions{
			Size:        coss.InputSM,
			Label:       terminalLabel("终端名称", "Terminal name"),
			Placeholder: terminalLabel("最多120字符", "Up to 120 characters"),
			Disabled:    edit.Saving,
		}).AutoFocus()
		if input.Changed() {
			store.EditRename(v.Owner, session, v.ActiveID, draft)
		}
		ui.Row(c).Gap(4).Children(func() {
			save := coss.Button(c, terminalLabel("保存名称", "Save name"), coss.ButtonOptions{Variant: coss.ButtonDefault, Size: coss.ButtonXS, Disabled: edit.Saving || !v.Ready}).Clicked()
			if (save || (input.Submitted() && !input.Composing())) && !edit.Saving {
				store.SaveRename(v.Owner, session, v.ActiveID)
			}
			if coss.Button(c, terminalLabel("取消", "Cancel"), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS, Disabled: edit.Saving}).Clicked() || input.Shortcut(0, ui.KeyEscape) {
				store.CancelRename(v.Owner, session, v.ActiveID)
			}
		})
		if edit.Error != "" {
			ui.Text(c, edit.Error).FontSize(11).TextColor(muted)
		}
	})
}
