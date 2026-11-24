package settings

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"

	"whaleisle/internal/shortcuts"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
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

var (
	shortcutStaticOnce     sync.Once
	shortcutAllCommands    []shortcuts.Command
	shortcutFixedList      []shortcuts.Command
	shortcutOrderRankMap   map[string]int
	shortcutReferenceGroup = []struct {
		zh, en string
		rows   []shortcuts.Command
	}{
		{"消息输入", "Message input", []shortcuts.Command{
			{ID: "fixed.send", ZH: "发送消息", EN: "Send message", Default: "Enter"},
			{ID: "fixed.newline", ZH: "换行", EN: "New line", Default: "Shift+Enter"},
			{ID: "fixed.complementary", ZH: "使用互补的 Queue／Steer 方式", EN: "Use the complementary Queue / Steer action", Default: "Mod+Enter"},
			{ID: "fixed.slash", ZH: "打开命令菜单", EN: "Open command menu", Default: "/"},
			{ID: "fixed.mention", ZH: "打开引用菜单", EN: "Open reference menu", Default: "@"},
			{ID: "response.stop", ZH: "停止生成", EN: "Stop generation", Default: "Esc  Esc"},
		}},
		{"菜单与弹层", "Menus and dialogs", []shortcuts.Command{
			{ID: "fixed.move", ZH: "移动菜单选择", EN: "Move menu selection", Default: "↑  ↓"},
			{ID: "fixed.select", ZH: "选择菜单项", EN: "Select menu item", Default: "Enter"},
			{ID: "fixed.dismiss", ZH: "关闭菜单或顶层弹窗", EN: "Close menu or top dialog", Default: "Esc"},
		}},
		{"审批区域", "Approval area", []shortcuts.Command{
			{ID: "approval.allow", ZH: "允许一次", EN: "Allow once", Default: "Enter"},
			{ID: "approval.reject", ZH: "拒绝", EN: "Reject", Default: "Esc"},
		}},
	}
)

func initShortcutStatic() {
	shortcutStaticOnce.Do(func() {
		shortcutAllCommands = shortcuts.Commands()
		shortcutFixedList = shortcuts.FixedCommands()
		orderList := []string{
			"shortcuts.open", "session.new", "sidebar.toggle", "session.search",
			"workspace.add", "session.rename", "session.fork", "session.archive",
			"settings.open", "fontSize.increase", "fontSize.decrease",
			"workspace.openLocal", "sidebar.right.toggle", "workspace.files",
			"browser.open", "terminal.new", "pane.split", "pane.fullscreen.toggle",
			"page.refresh", "page.close",
		}
		shortcutOrderRankMap = make(map[string]int, len(orderList))
		for i, id := range orderList {
			shortcutOrderRankMap[id] = i
		}
	})
}

func shortcutCommandRank(id string) int {
	if value, ok := shortcutOrderRankMap[id]; ok {
		return value
	}
	return len(shortcutOrderRankMap)
}

// shortcutDialogCache caches per-editor normalized defaults, issues, modified
// count and filtered row lists so steady frames never rebuild shortcuts.Defaults()
// or re-parse every chord.
type shortcutDialogCache struct {
	editor       *shortcuts.Editor
	normDefaults map[string]string
	draftsHash   uint64
	query        string
	rows         []shortcuts.Command
	fixed        map[string]bool
	issues       map[string]string
	modified     int
	valid        bool
}

func (sc *shortcutDialogCache) refresh(e *shortcuts.Editor) {
	initShortcutStatic()
	if sc.editor != e || sc.normDefaults == nil {
		sc.editor = e
		sc.normDefaults = make(map[string]string, len(shortcutAllCommands))
		// Probe once whether e uses shared defaults or platform defaults:
		// Editor.Default calls Defaults() on every invocation when !shared, so
		// probing a key that differs between SharedDefaults and Defaults
		// ("session.search": "Mod+K" vs "Mod+Shift+K") eliminates 40+
		// Defaults() allocations per frame.
		probe := e.Default("session.search")
		var defaults map[string]string
		if probe == "Mod+K" {
			defaults = shortcuts.SharedDefaults()
		} else {
			defaults = shortcuts.Defaults()
		}
		for _, cmd := range shortcutAllCommands {
			sc.normDefaults[cmd.ID] = shortcutBinding(defaults[cmd.ID])
		}
		sc.valid = false
	}
	h := newMemoKey(1)
	for _, cmd := range shortcutAllCommands {
		h.s(shortcutDraft(e, cmd.ID))
	}
	draftsHash := h.sum()
	if sc.valid && sc.draftsHash == draftsHash && sc.query == e.Query {
		return
	}
	if !sc.valid || sc.draftsHash != draftsHash {
		sc.issues = e.Issues()
		modified := 0
		for _, cmd := range shortcutAllCommands {
			if shortcutBinding(shortcutDraft(e, cmd.ID)) != sc.normDefaults[cmd.ID] {
				modified++
			}
		}
		sc.modified = modified
		sc.draftsHash = draftsHash
	}
	sc.query = e.Query
	rows := make([]shortcuts.Command, 0, len(shortcutAllCommands)+len(shortcutFixedList))
	fixed := make(map[string]bool, len(shortcutFixedList))
	for _, cmd := range shortcutAllCommands {
		if shortcutReferenceMatches(cmd, e.Query, shortcutDraft(e, cmd.ID)) {
			rows = append(rows, cmd)
		}
	}
	for _, cmd := range shortcutFixedList {
		if shortcutReferenceMatches(cmd, e.Query, cmd.Default) {
			rows = append(rows, cmd)
			fixed[cmd.ID] = true
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := shortcutCommandRank(rows[i].ID), shortcutCommandRank(rows[j].ID)
		if a != b {
			return a < b
		}
		return rows[i].ID < rows[j].ID
	})
	sc.rows = rows
	sc.fixed = fixed
	sc.valid = true
}

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
	cache := ui.Local(c.Root(), "shortcut-dialog-cache", func() shortcutDialogCache { return shortcutDialogCache{} })
	cache.refresh(e)
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
				ui.Column(c).FillWidth().MinWidth(0).Gap(0).Padding(0, design.Sp(4), design.Sp(4.5), design.Sp(4)).Children(func() {
					shortcutDialogRowsCached(c, e, p, target, cache)
				})
			})
			shortcutDialogFooterCached(c, e, p, target, cache)
			shortcutResetConfirmation(c, e, p)
		})
	})
}

func shortcutDialogHeader(c *ui.Context, open *bool, e *shortcuts.Editor, p shortcutPalette, target *string) {
	ui.Row(c).FillWidth().Height(shortcutHeaderHeight).Shrink(0).Padding(22, 14, 0, 24).AlignItems(ui.Start).Children(func() {
		ui.Text(c, providerCopy("快捷键", "Keyboard shortcuts")).FontSize(fonttheme.InterfaceSize(16)).FontWeight(design.WeightMedium).TextColor(p.fg).Grow(1).MinWidth(0).SingleLine()
		// The reference places the 28 DIP close box at 20 from the top, two DIPs
		// above the title's 22, so the two share a center line.
		close := coss.Button(c, "", coss.ButtonOptions{
			Variant: coss.ButtonGhost,
			Size:    coss.ButtonIconSM,
			Icon:    coss.IconX,
			Label:   providerCopy("关闭快捷键", "Close keyboard shortcuts"),
		}).Shrink(0).Margin(-2, 0, 0, 0)
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
		field := ui.Row(c).FillWidth().Height(36).MinWidth(0).Padding(design.Sp(2), design.Sp(3.5)).Gap(design.Sp(2)).Radius(design.RadiusXL).Border(design.BorderWidth, p.border).Background(p.field).AlignItems(ui.Center)
		field.Children(func() {
			ui.Icon(c, bodyIcon("search")).Size(14, 14).Shrink(0).TextColor(p.caption)
			ui.TextInputBase(c, &e.Query).AutoFocus().Label("shortcut-search").Grow(1).MinWidth(0).Height(18).
				FontSize(fonttheme.InterfaceSize(13)).TextColor(p.fg).
				Placeholder(providerCopy("搜索快捷键", "Search shortcuts"))
			if e.Query == "" {
				return
			}
			clear := coss.Button(c, "", coss.ButtonOptions{
				Variant: coss.ButtonGhost,
				Size:    coss.ButtonIconXS,
				Icon:    coss.IconX,
				Label:   "clear-shortcut-search",
			}).Tooltip(providerCopy("清空搜索", "Clear search"))
			if clear.Clicked() {
				e.Query = ""
			}
		})
	})
}

// Application entries and the read-only input/menu/approval directories are
// deliberately separate from dispatch: bare Enter/Escape are context actions.
func shortcutDialogRows(c *ui.Context, e *shortcuts.Editor, p shortcutPalette, target *string) {
	cache := &shortcutDialogCache{}
	cache.refresh(e)
	shortcutDialogRowsCached(c, e, p, target, cache)
}

func shortcutDialogRowsCached(c *ui.Context, e *shortcuts.Editor, p shortcutPalette, target *string, cache *shortcutDialogCache) {
	rows := cache.rows
	fixed := cache.fixed
	issues := cache.issues
	if len(rows) > 0 {
		ui.Text(c, providerCopy("应用操作", "Application")).FillWidth().Padding(design.Sp(4), design.Sp(2), design.Sp(1.5), design.Sp(2)).FontSize(fonttheme.InterfaceSize(12)).FontWeight(design.WeightMedium).TextColor(p.caption)
	}
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
	for _, group := range shortcutReferenceGroup {
		shown := false
		for _, command := range group.rows {
			if !shortcutReferenceMatches(command, e.Query, group.zh+" "+group.en) {
				continue
			}
			if !shown {
				ui.Text(c, providerCopy(group.zh, group.en)).FillWidth().Padding(design.Sp(4), design.Sp(2), design.Sp(1.5), design.Sp(2)).FontSize(fonttheme.InterfaceSize(12)).FontWeight(design.WeightMedium).TextColor(p.caption)
				shown = true
			}
			shortcutFixedReferenceRow(c, command, p)
			count++
		}
	}
	if count == 0 {
		ui.Text(c, providerCopy("没有匹配的快捷键", "No matching shortcuts")).FontSize(fonttheme.InterfaceSize(12)).TextColor(p.caption).Padding(design.Sp(1.5), design.Sp(2))
	}
}
func shortcutReferenceMatches(command shortcuts.Command, query, extra string) bool {
	if strings.TrimSpace(query) == "" {
		return true
	}
	haystack := strings.ToLower(command.ID + " " + command.ZH + " " + command.EN + " " + command.Default + " " + shortcutDisplay(command.Default) + " " + extra)
	for _, term := range strings.Fields(strings.ToLower(query)) {
		if !strings.Contains(haystack, term) {
			return false
		}
	}
	return true
}
func shortcutFixedReferenceRow(c *ui.Context, command shortcuts.Command, p shortcutPalette) {
	row := ui.Row(c.Key("shortcut-fixed-" + command.ID)).Label("shortcut-" + command.ID).FillWidth()
	if command.ID == "fontSize.increase" {
		row.Tooltip(providerCopy("也支持 ", "Also accepts ") + shortcutDisplay("Mod+Shift+="))
	}
	row.MinWidth(0).MinHeight(shortcutRowHeight).Padding(design.Sp(2)).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
		ui.Text(c, shortcutCommandLabel(command)).Grow(1).MinWidth(0).FontSize(fonttheme.InterfaceSize(13)).TextColor(p.fg)
		ui.Text(c, shortcutDisplay(command.Default)).Padding(design.Sp(1), design.Sp(3)).Radius(design.RadiusLG).Background(p.field).FontSize(fonttheme.InterfaceSize(12)).TextColor(p.muted).SingleLine()
	})
}
func shortcutFixedRows(c *ui.Context, query string, p shortcutPalette) int {
	initShortcutStatic()
	count := 0
	for _, command := range shortcutFixedList {
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
	row := ui.ButtonBase(c).Label("shortcut-" + command.ID).FillWidth().Shrink(0).MinHeight(shortcutRowHeight).
		Padding(design.Sp(2)).Gap(design.Sp(2)).Radius(design.RadiusLG).BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).
		Cursor(ui.CursorPointer).Tooltip(providerCopy("修改"+label+"快捷键", "Edit shortcut for "+label))
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
		color = tk(c).DestructiveForeground
	}
	badge := ui.Box(c).Height(shortcutBadgeHeight).Shrink(0).Radius(design.RadiusLG).Background(p.field).AlignItems(ui.Center).Justify(ui.Center).PassThrough()
	if value == "" {
		badge.Padding(0, design.Sp(4)).Children(func() {
			ui.Text(c, providerCopy("暂无快捷键", "No shortcut")).FontSize(fonttheme.InterfaceSize(12)).TextColor(p.caption).SingleLine().PassThrough()
		})
		return
	}
	badge.Padding(0, design.Sp(3)).Children(func() {
		ui.Text(c, shortcutDisplay(value)).FontSize(fonttheme.InterfaceSize(12)).TextColor(color).SingleLine().PassThrough()
	})
}

// shortcutInlineEditor is the reference inline command editor: the row keeps
// its label, and the binding column holds 恢复默认, 移除 and the recorder. The
// label stays inert: a press on it disarms the editor, as the reference's
// pointer-down handler does, but it is not a second control in the row.
func shortcutInlineEditor(c *ui.Context, e *shortcuts.Editor, command shortcuts.Command, p shortcutPalette, target *string, issue string) {
	label := shortcutCommandLabel(command)
	ui.Row(c).Key("shortcut-row-" + command.ID).FillWidth().Shrink(0).MinHeight(shortcutRowHeight).Padding(design.Sp(2)).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
		labelArea := ui.Box(c).Label("shortcut-" + command.ID).
			Grow(1).MinWidth(0).Height(shortcutRowHeight).AlignItems(ui.Start).Justify(ui.Center)
		labelArea.Children(func() {
			ui.Text(c, label).FontSize(fonttheme.InterfaceSize(13)).TextColor(p.fg).Grow(1).MinWidth(0).SingleLine()
		})
		if labelArea.Clicked() {
			e.CancelRecording()
			*target = ""
		}
		ui.Column(c).Gap(design.Sp(1)).Shrink(0).AlignItems(ui.End).Children(func() {
			ui.Row(c).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
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
				kError(c, issue).MaxWidth(220)
			}
			if e.Err != "" && e.Recording == "" {
				settingsFailureLine(c, "shortcut:save", providerCopy("没能保存这个快捷键，请重试。", "This shortcut was not saved. Try again."), e.Err)
			}
		})
	})
}

func shortcutTextAction(c *ui.Context, text, label string, color ui.Color, disabled bool) bool {
	button := coss.Button(c, text, coss.ButtonOptions{
		Variant:  coss.ButtonGhost,
		Size:     coss.ButtonXS,
		Disabled: disabled,
	}).Label(label).Shrink(0)
	return button.Clicked()
}

func shortcutDialogFooter(c *ui.Context, e *shortcuts.Editor, p shortcutPalette, target *string) {
	cache := &shortcutDialogCache{}
	cache.refresh(e)
	shortcutDialogFooterCached(c, e, p, target, cache)
}

func shortcutDialogFooterCached(c *ui.Context, e *shortcuts.Editor, p shortcutPalette, target *string, cache *shortcutDialogCache) {
	modified := cache.modified
	ui.Row(c).FillWidth().Height(shortcutFooterHeight).Shrink(0).Padding(0, design.Sp(6)).Gap(design.Sp(3)).AlignItems(ui.Center).
		BorderWidth(design.BorderWidth, 0, 0, 0).BorderColor(p.border).Children(func() {
		reset := coss.Button(c, "", coss.ButtonOptions{
			Variant:  coss.ButtonGhost,
			Size:     coss.ButtonXS,
			Label:    "reset-shortcuts",
			Disabled: modified == 0 || e.ReadError != "" || e.ResetPending,
			Content: func() {
				ui.Row(c).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, bodyIcon("refresh")).Size(12, 12).TextColor(p.muted).PassThrough()
					ui.Text(c, providerCopy("恢复全部默认", "Restore all defaults")).FontSize(fonttheme.InterfaceSize(12)).TextColor(p.muted).SingleLine().PassThrough()
				})
			},
		}).Shrink(0).Tooltip(providerCopy("恢复全部默认", "Restore all defaults"))
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
	cache := &shortcutDialogCache{}
	cache.refresh(e)
	return cache.modified
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
		chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), min(float32(420), max(float32(0), w-32)), design.Sp(5)).Gap(design.Sp(3)).Children(func() {
			kHeading(c, providerCopy("恢复全部默认快捷键？", "Restore all default shortcuts?"))
			kBody(c, providerCopy("将恢复当前平台的默认快捷键。你修改或移除的快捷键都会恢复，其他平台不受影响。", "Restore the default shortcuts for this platform. All modified or removed shortcuts will be restored. Other platforms are unaffected.")).TextColor(p.muted)
			ui.Box(c).FillWidth().MinHeight(36).Children(func() {
				if e.Err != "" {
					settingsFailureLine(c, "shortcut:restore", providerCopy("没能恢复默认快捷键，请重试。", "The default shortcuts could not be restored. Try again."), e.Err)
				}
			})
			ui.Row(c).Gap(design.Sp(2)).Children(func() {
				if kButton(c, providerCopy("取消", "Cancel")).Label("cancel-reset-shortcuts").AutoFocus().Clicked() {
					e.CancelReset()
				}
				if kDanger(c, providerCopy("恢复默认", "Restore default"), false).Label("confirm-reset-shortcuts").Clicked() {
					e.ConfirmReset()
				}
			})
		})
	})
	if !open {
		e.CancelReset()
	}
}
