package settings

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"runtime"
	"sort"
	"strings"
	"whaleisle/internal/shortcuts"
	"whaleisle/internal/ui/chrome"
	fonttheme "whaleisle/internal/ui/theme"
)

// The installed build's shortcut dialog is a 480x600 reference list: a 46 DIP
// title, a 60 DIP search row, one 42 DIP row per command with its key badge,
// and a 52 DIP footer that owns the reset action and the customized count. The
// reference drops the centered card 30 DIPs down (translateY(30px)).
const (
	shortcutDialogWidth   = 480
	shortcutDialogHeight  = 600
	shortcutDialogRadius  = 24
	shortcutDialogOffsetY = 30
	shortcutHeaderHeight  = 46
	shortcutSearchHeight  = 60
	shortcutFooterHeight  = 52
	shortcutRowHeight     = 42
	shortcutBadgeHeight   = 26
	shortcutRowActionSize = 20
	shortcutRecorderWidth = 96
)

// shortcutPalette carries the resolved colors the dialog paints with.
type shortcutPalette struct {
	fg, muted, caption, border, field, hover ui.Color
}

func shortcutPaletteOf(c *ui.Context, fg, muted ui.Color) shortcutPalette {
	t := c.Theme()
	return shortcutPalette{fg: fg, muted: muted, caption: t.TextMuted, border: t.Border, field: t.Surface, hover: t.SurfaceHover}
}

// bodyShortcutPreference is the reference General page's "快捷键" Setting-Cell:
// a compact row whose right-hand button opens the shortcut reference over the
// shared modal surface.
func bodyShortcutPreference(c *ui.Context, e *shortcuts.Editor, fg, muted ui.Color) {
	if e == nil {
		return
	}
	row := ui.Column(c.Key("general-shortcuts")).FillWidth().MinWidth(0)
	open := &e.ReferenceOpen
	row.Children(func() {
		bodyPreferenceRow(c, "general-shortcuts-row",
			providerCopy("快捷键", "Keyboard shortcuts"),
			providerCopy("查看和编辑当前可用的快捷键和输入操作", "View and edit the shortcuts and input actions available now"),
			fg, muted, func() {
				ui.Row(c).FillWidth().Justify(ui.End).Children(func() {
					button := settingsPreferenceButton(c, providerCopy("编辑快捷键", "Edit shortcuts")).Label("edit-shortcuts")
					if button.Clicked() {
						*open = true
					}
				})
			})
	})
	if *open {
		bodyShortcutDialog(c, open, e, shortcutPaletteOf(c, fg, muted))
	}
}

// bodyShortcutDialog draws the reference shortcut reference: title, search,
// command rows with key badges, inline recording, and the reset footer.
func bodyShortcutDialog(c *ui.Context, open *bool, e *shortcuts.Editor, p shortcutPalette) {
	if e == nil {
		return
	}
	target := ui.Local(c.Root(), "shortcut-reference-target", func() string { return "" })
	dismiss := func() {
		if e.ResetPending {
			e.CancelReset()
			return
		}
		shortcutCloseReference(open, e, target)
	}
	ui.DialogBaseWithOptions(c, open, ui.DialogOptions{OnDismiss: dismiss}, func(backdrop, panel ui.Element) {
		ui.ShortcutModalScope(panel, "shortcuts")
		w, h := c.Size()
		box := chrome.DialogSurfaceBox{
			Width:  min(float32(shortcutDialogWidth), max(float32(0), w-32)),
			Height: min(float32(shortcutDialogHeight), max(float32(0), h-108)),
			Radius: shortcutDialogRadius, Padding: 0, OffsetY: shortcutDialogOffsetY,
		}
		chrome.DialogSurfaceSized(c, backdrop, panel, c.Theme(), box).Key("shortcut-reference-dialog").Gap(0).Children(func() {
			shortcutDialogHeader(c, open, e, p, target)
			shortcutDialogSearch(c, e, p)
			ui.Scroll(c).FillWidth().Grow(1).MinHeight(0).Children(func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(0).Padding(0, 16, 18, 16).Children(func() {
					shortcutDialogRows(c, e, p, target)
				})
			})
			shortcutDialogFooter(c, e, p, target)
			shortcutResetConfirmation(c, e, p)
		})
	})
}

func shortcutDialogHeader(c *ui.Context, open *bool, e *shortcuts.Editor, p shortcutPalette, target *string) {
	ui.Row(c).FillWidth().Height(shortcutHeaderHeight).Shrink(0).Padding(22, 14, 0, 24).AlignItems(ui.Start).Children(func() {
		ui.Text(c, providerCopy("快捷键", "Keyboard shortcuts")).FontSize(fonttheme.InterfaceSize(16)).FontWeight(500).TextColor(p.fg).Grow(1).MinWidth(0).SingleLine()
		// The reference places the 28 DIP close box at 20 from the top, two DIPs
		// above the title's 22, so the two share a center line.
		close := ui.Button(c, "").Label(providerCopy("关闭快捷键", "Close keyboard shortcuts")).
			Size(28, 28).Shrink(0).Margin(-2, 0, 0, 0).Padding(6).Radius(14).BorderWidth(0).Background(ui.Transparent).FocusRing(false)
		close.Children(func() { ui.Icon(c, settingsIcons["CloseOutline"]).Size(14, 14).TextColor(p.fg) })
		if close.Hovered() {
			close.Background(p.hover)
		}
		if close.Clicked() {
			shortcutCloseReference(open, e, target)
		}
	})
}

// shortcutCloseReference is the reference's closeReference (Reference.tsx:102):
// the close box, Escape and a press on the mask all close the armed row editor
// first and the reference itself only after it.
func shortcutCloseReference(open *bool, e *shortcuts.Editor, target *string) {
	if *target != "" {
		e.CancelRecording()
		*target = ""
		return
	}
	*open = false
}

func shortcutDialogSearch(c *ui.Context, e *shortcuts.Editor, p shortcutPalette) {
	ui.Row(c).FillWidth().Height(shortcutSearchHeight).Shrink(0).Padding(14, 20, 10, 20).Children(func() {
		field := ui.Row(c).FillWidth().Height(36).MinWidth(0).Padding(8, 14).Gap(8).Radius(12).Border(.5, p.border).Background(p.field).AlignItems(ui.Center)
		field.Children(func() {
			ui.Icon(c, bodyIcon("search")).Size(14, 14).Shrink(0).TextColor(p.caption)
			ui.TextInputBase(c, &e.Query).AutoFocus().Label("shortcut-search").Grow(1).MinWidth(0).Height(18).
				FontSize(fonttheme.InterfaceSize(13)).TextColor(p.fg).
				Placeholder(providerCopy("搜索快捷键", "Search shortcuts"))
			if e.Query == "" {
				return
			}
			clear := ui.Button(c, "").Label("clear-shortcut-search").Tooltip(providerCopy("清空搜索", "Clear search")).
				Size(16, 16).Shrink(0).Padding(0).Radius(8).BorderWidth(0).Background(ui.Transparent)
			clear.Children(func() { ui.Icon(c, settingsIcons["CloseOutline"]).Size(10, 10).TextColor(p.muted) })
			if clear.Clicked() {
				e.Query = ""
			}
		})
	})
}

// Application entries and the read-only input/menu/approval directories are
// deliberately separate from dispatch: bare Enter/Escape are context actions.
func shortcutDialogRows(c *ui.Context, e *shortcuts.Editor, p shortcutPalette, target *string) {
	rows := []shortcuts.Command{}
	fixed := map[string]bool{}
	for _, command := range e.Search("") {
		if shortcutReferenceMatches(command, e.Query, shortcutDraft(e, command.ID)) {
			rows = append(rows, command)
		}
	}
	for _, command := range shortcuts.FixedCommands() {
		if shortcutReferenceMatches(command, e.Query, command.Default) {
			rows = append(rows, command)
			fixed[command.ID] = true
		}
	}
	order := map[string]int{}
	for i, id := range []string{"shortcuts.open", "session.new", "sidebar.toggle", "session.search", "workspace.add", "session.rename", "session.fork", "session.archive", "settings.open", "fontSize.increase", "fontSize.decrease", "workspace.openLocal", "sidebar.right.toggle", "workspace.files", "browser.open", "terminal.new", "pane.split", "pane.fullscreen.toggle", "page.refresh", "page.close"} {
		order[id] = i
	}
	rank := func(id string) int {
		if value, ok := order[id]; ok {
			return value
		}
		return len(order)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rank(rows[i].ID), rank(rows[j].ID)
		if a != b {
			return a < b
		}
		return rows[i].ID < rows[j].ID
	})
	issues := e.Issues()
	for _, command := range rows {
		if fixed[command.ID] {
			shortcutFixedReferenceRow(c, command, p)
			continue
		}
		if *target == command.ID {
			shortcutInlineEditor(c, e, command, p, target, issues[command.ID])
		} else {
			shortcutRow(c, e, command, p, target, issues[command.ID] != "")
		}
	}
	count := len(rows)
	groups := []struct {
		zh, en string
		rows   []shortcuts.Command
	}{
		{"输入", "Input", []shortcuts.Command{
			{ID: "fixed.send", ZH: "发送消息", EN: "Send message", Default: "Enter"},
			{ID: "fixed.newline", ZH: "换行", EN: "New line", Default: "Shift+Enter"},
			{ID: "fixed.complementary", ZH: "使用互补的 Queue／Steer 方式", EN: "Use the complementary Queue / Steer action", Default: "Mod+Enter"},
			{ID: "fixed.slash", ZH: "打开命令菜单", EN: "Open command menu", Default: "/"},
			{ID: "fixed.mention", ZH: "打开引用菜单", EN: "Open reference menu", Default: "@"},
			{ID: "response.stop", ZH: "停止生成", EN: "Stop generation", Default: "Esc  Esc"},
		}},
		{"菜单", "Menus", []shortcuts.Command{
			{ID: "fixed.move", ZH: "移动选择", EN: "Move selection", Default: "↑  ↓"},
			{ID: "fixed.select", ZH: "选择", EN: "Select", Default: "Enter"},
			{ID: "fixed.dismiss", ZH: "关闭", EN: "Dismiss", Default: "Esc"},
		}},
		{"审批", "Approval", []shortcuts.Command{
			{ID: "approval.allow", ZH: "允许一次", EN: "Allow once", Default: "Enter"},
			{ID: "approval.reject", ZH: "拒绝", EN: "Reject", Default: "Esc"},
		}},
	}
	for _, group := range groups {
		shown := false
		for _, command := range group.rows {
			if !shortcutReferenceMatches(command, e.Query, group.zh+" "+group.en) {
				continue
			}
			if !shown {
				ui.Text(c, providerCopy(group.zh, group.en)).FillWidth().Padding(16, 8, 6, 8).FontSize(fonttheme.InterfaceSize(12)).FontWeight(500).TextColor(p.caption)
				shown = true
			}
			shortcutFixedReferenceRow(c, command, p)
			count++
		}
	}
	if count == 0 {
		ui.Text(c, providerCopy("没有匹配的快捷键", "No matching shortcuts")).FontSize(fonttheme.InterfaceSize(12)).TextColor(p.caption).Padding(6, 8)
	}
}
func shortcutReferenceMatches(command shortcuts.Command, query, extra string) bool {
	haystack := strings.ToLower(command.ID + " " + command.ZH + " " + command.EN + " " + command.Default + " " + shortcutDisplay(command.Default) + " " + extra)
	for _, term := range strings.Fields(strings.ToLower(query)) {
		if !strings.Contains(haystack, term) {
			return false
		}
	}
	return true
}
func shortcutFixedReferenceRow(c *ui.Context, command shortcuts.Command, p shortcutPalette) {
	hint := providerCopy("固定快捷键，不能修改。", "Fixed shortcut; it cannot be edited.")
	if command.ID == "fontSize.increase" {
		hint += providerCopy("也支持 ", " Also accepts ") + shortcutDisplay("Mod+Shift+=")
	}
	ui.Row(c.Key("shortcut-fixed-" + command.ID)).Label("shortcut-" + command.ID).Tooltip(hint).FillWidth().MinWidth(0).MinHeight(shortcutRowHeight).Padding(8).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, shortcutCommandLabel(command)).Grow(1).MinWidth(0).FontSize(fonttheme.InterfaceSize(13)).TextColor(p.fg)
		ui.Text(c, shortcutDisplay(command.Default)).Padding(4, 12).Radius(10).Background(p.field).FontSize(fonttheme.InterfaceSize(12)).TextColor(p.muted).SingleLine()
	})
}
func shortcutFixedRows(c *ui.Context, query string, p shortcutPalette) int {
	count := 0
	for _, command := range shortcuts.FixedCommands() {
		if shortcutReferenceMatches(command, query, "") {
			shortcutFixedReferenceRow(c, command, p)
			count++
		}
	}
	return count
}

func shortcutCommandLabel(command shortcuts.Command) string {
	return providerCopy(command.ZH, command.EN)
}

// shortcutDisplay names the portable Mod modifier the way each platform shows
// it; the stored value keeps Mod so the binding reads the same everywhere.
func shortcutDisplay(value string) string {
	switch runtime.GOOS {
	case "windows":
		return strings.Replace(value, "Mod", "Ctrl", 1)
	case "darwin":
		return strings.Replace(value, "Mod", "Cmd", 1)
	}
	return value
}

func shortcutDraft(e *shortcuts.Editor, id string) string {
	if draft := e.Draft(id); draft != nil {
		return *draft
	}
	return ""
}

func shortcutRow(c *ui.Context, e *shortcuts.Editor, command shortcuts.Command, p shortcutPalette, target *string, invalid bool) {
	label := shortcutCommandLabel(command)
	row := ui.Button(c, "").Label("shortcut-" + command.ID).FillWidth().Shrink(0).MinHeight(shortcutRowHeight).
		Padding(8).Gap(8).Radius(10).BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).
		Tooltip(providerCopy("修改"+label+"快捷键", "Edit shortcut for "+label))
	hovered := row.Hovered()
	row.Children(func() {
		ui.Text(c, label).FontSize(fonttheme.InterfaceSize(13)).TextColor(p.fg).Grow(1).MinWidth(0).SingleLine()
		shortcutRowActions(c, command.ID, p, hovered)
		shortcutKeyBadge(c, e, command.ID, p, invalid)
	})
	if hovered {
		row.Background(p.hover)
	}
	if row.Clicked() && e.ReadError == "" && !e.ResetPending {
		*target = command.ID
		e.BeginRecording(command.ID)
	}
}

// shortcutRowActions is the reference row's hover affordance: a 20x26 slot left
// of the key badge that shows the 12 DIP pencil while the pointer is over the
// row. The slot stays in the layout while hidden, so the badge never shifts as
// the pointer enters and leaves.
func shortcutRowActions(c *ui.Context, id string, p shortcutPalette, hovered bool) {
	slot := ui.Box(c).Label("row-edit-"+id).Size(shortcutRowActionSize, shortcutBadgeHeight).Shrink(0).
		AlignItems(ui.Center).Justify(ui.Center).PassThrough()
	slot.Children(func() {
		icon := ui.Icon(c, shortcutEditIcon).Size(12, 12).TextColor(p.caption)
		if !hovered {
			icon.Opacity(0)
		}
	})
}

func shortcutKeyBadge(c *ui.Context, e *shortcuts.Editor, id string, p shortcutPalette, invalid bool) {
	value := shortcutDraft(e, id)
	color := p.muted
	if invalid {
		color = c.Theme().Danger
	}
	badge := ui.Box(c).Height(shortcutBadgeHeight).Shrink(0).Radius(10).Background(p.field).AlignItems(ui.Center).Justify(ui.Center).PassThrough()
	if value == "" {
		badge.Padding(0, 16).Children(func() {
			ui.Text(c, providerCopy("暂无快捷键", "No shortcut")).FontSize(fonttheme.InterfaceSize(12)).TextColor(p.caption).SingleLine().PassThrough()
		})
		return
	}
	badge.Padding(0, 12).Children(func() {
		ui.Text(c, shortcutDisplay(value)).FontSize(fonttheme.InterfaceSize(12)).TextColor(color).SingleLine().PassThrough()
	})
}

// shortcutInlineEditor is the reference inline command editor: the row keeps
// its label, and the binding column holds 恢复默认, 移除 and the recorder. The
// label stays inert: a press on it disarms the editor, as the reference's
// pointer-down handler does, but it is not a second control in the row.
func shortcutInlineEditor(c *ui.Context, e *shortcuts.Editor, command shortcuts.Command, p shortcutPalette, target *string, issue string) {
	label := shortcutCommandLabel(command)
	ui.Row(c).Key("shortcut-row-" + command.ID).FillWidth().Shrink(0).MinHeight(shortcutRowHeight).Padding(8).Gap(8).AlignItems(ui.Center).Children(func() {
		labelArea := ui.Box(c).Label("shortcut-" + command.ID).
			Grow(1).MinWidth(0).Height(shortcutRowHeight).AlignItems(ui.Start).Justify(ui.Center)
		labelArea.Children(func() {
			ui.Text(c, label).FontSize(fonttheme.InterfaceSize(13)).TextColor(p.fg).Grow(1).MinWidth(0).SingleLine()
		})
		if labelArea.Clicked() {
			e.CancelRecording()
			*target = ""
		}
		ui.Column(c).Gap(4).Shrink(0).AlignItems(ui.End).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				disabled := e.ReadError != "" || e.ResetPending
				if shortcutTextAction(c, providerCopy("恢复默认", "Restore default"), "default-"+command.ID, p.muted, disabled) {
					e.ResetDraft(command.ID)
					e.SaveRow(command.ID, true)
					*target = ""
				}
				if shortcutDraft(e, command.ID) != "" {
					if shortcutTextAction(c, providerCopy("移除", "Remove"), "unbind-"+command.ID, p.muted, disabled) {
						*e.Draft(command.ID) = ""
						e.SaveRow(command.ID, false)
						*target = ""
					}
				}
				shortcutRecorder(c, e, command.ID, func() { *target = "" })
			})
			if issue != "" && e.Recording != command.ID {
				ui.Text(c, issue).FontSize(fonttheme.InterfaceSize(11)).TextColor(c.Theme().Danger).MaxWidth(220)
			}
			if e.Err != "" && e.Recording == "" {
				ui.Text(c, e.Err).FontSize(fonttheme.InterfaceSize(11)).TextColor(c.Theme().Danger).MaxWidth(220)
			}
		})
	})
}

func shortcutTextAction(c *ui.Context, text, label string, color ui.Color, disabled bool) bool {
	button := ui.Button(c, text).Label(label).Shrink(0).Height(shortcutBadgeHeight).Padding(0).BorderWidth(0).Background(ui.Transparent).
		FontSize(fonttheme.InterfaceSize(12)).TextColor(color).Disabled(disabled)
	if button.Hovered() && !disabled {
		button.TextColor(c.Theme().Text)
	}
	return button.Clicked()
}

func shortcutDialogFooter(c *ui.Context, e *shortcuts.Editor, p shortcutPalette, target *string) {
	modified := shortcutModifiedCount(e)
	ui.Row(c).FillWidth().Height(shortcutFooterHeight).Shrink(0).Padding(0, 24).Gap(12).AlignItems(ui.Center).
		BorderWidth(1, 0, 0, 0).BorderColor(p.border).Children(func() {
		reset := ui.Button(c, "").Label("reset-shortcuts").Height(shortcutBadgeHeight).Shrink(0).Padding(6, 0).
			BorderWidth(0).Background(ui.Transparent).Disabled(modified == 0 || e.ReadError != "" || e.ResetPending).
			Tooltip(providerCopy("恢复全部默认", "Restore all defaults"))
		reset.Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, bodyIcon("refresh")).Size(12, 12).TextColor(p.muted)
				ui.Text(c, providerCopy("恢复全部默认", "Restore all defaults")).FontSize(fonttheme.InterfaceSize(12)).TextColor(p.muted).SingleLine()
			})
		})
		if reset.Clicked() {
			// The reference's Reset All clears the armed row editor as it opens
			// its confirmation (Reference.tsx:206).
			*target = ""
			e.RequestReset()
		}
		ui.Box(c).Grow(1)
		if e.Dirty() || e.ReadError != "" {
			if shortcutTextAction(c, providerCopy("重新加载", "Reload"), "reload-shortcuts", p.muted, false) {
				e.Reload()
			}
		}
		if modified > 0 {
			ui.Text(c, fmt.Sprintf(providerCopy("%d 项已自定义", "%d customized"), modified)).FontSize(fonttheme.InterfaceSize(12)).TextColor(p.caption).SingleLine()
		}
	})
}

// shortcutModifiedCount is the reference footer's count: commands whose
// binding differs from this platform's default. Saved values and the table's
// defaults are compared in the normalized chord spelling Save writes.
func shortcutModifiedCount(e *shortcuts.Editor) int {
	count := 0
	for _, command := range e.Search("") {
		if shortcutBinding(shortcutDraft(e, command.ID)) != shortcutBinding(e.Default(command.ID)) {
			count++
		}
	}
	return count
}

// shortcutBinding compares bindings in the normalized chord spelling Save
// writes, so "Mod+Alt+Enter" and the stored "Mod+Alt+ENTER" count as the same.
func shortcutBinding(raw string) string {
	if chord, err := shortcuts.Parse(raw); err == nil {
		return chord.String()
	}
	return raw
}

// shortcutResetConfirmation mirrors the reference reset dialog's copy.
func shortcutResetConfirmation(c *ui.Context, e *shortcuts.Editor, p shortcutPalette) {
	if !e.ResetPending {
		return
	}
	open := true
	ui.DialogBaseWithOptions(c, &open, ui.DialogOptions{OnDismiss: e.CancelReset}, func(backdrop, panel ui.Element) {
		ui.ShortcutModalScope(panel, "shortcuts")
		w, _ := c.Size()
		chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), min(float32(420), max(float32(0), w-32)), 20).Gap(12).Children(func() {
			ui.Text(c, providerCopy("恢复全部默认快捷键？", "Restore all default shortcuts?")).FontSize(fonttheme.InterfaceSize(16)).FontWeight(500).TextColor(p.fg)
			ui.Text(c, providerCopy("将恢复当前平台的默认快捷键。你修改或移除的快捷键都会恢复，其他平台不受影响。", "Restore the default shortcuts for this platform. All modified or removed shortcuts will be restored. Other platforms are unaffected.")).FontSize(fonttheme.InterfaceSize(13)).TextColor(p.muted)
			ui.Box(c).FillWidth().Height(36).Children(func() {
				if e.Err != "" {
					ui.Text(c, e.Err).FillWidth().TextColor(c.Theme().Danger).FontSize(fonttheme.InterfaceSize(12))
				}
			})
			ui.Row(c).Gap(8).Children(func() {
				if ui.Button(c, providerCopy("取消", "Cancel")).Label("cancel-reset-shortcuts").AutoFocus().Clicked() {
					e.CancelReset()
				}
				if ui.Button(c, providerCopy("恢复默认", "Restore default")).Label("confirm-reset-shortcuts").Clicked() {
					e.ConfirmReset()
				}
			})
		})
	})
	if !open {
		e.CancelReset()
	}
}
