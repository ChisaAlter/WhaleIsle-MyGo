package coss

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
)

func popupHTML(inner string, popupPart string) string {
	return `<div data-g data-slot="menu-popup" class="{{part:` + popupPart + `}}"><div class="{{part:menu/MenuPopup#2}}" style="max-height:none">` + inner + `</div></div>`
}

// menuItemHTML is a menu row's markup.
func menuItemHTML(label string, ic Icon, shortcut string, highlighted, destructive bool) string {
	attrs := ""
	if highlighted {
		attrs += ` data-highlighted=""`
	}
	if destructive {
		attrs += ` data-variant="destructive"`
	}
	sc := ""
	if shortcut != "" {
		sc = `<kbd data-slot="menu-shortcut" class="{{part:menu/menu-shortcut}}">` + shortcut + `</kbd>`
	}
	return `<div role="menuitem" data-slot="menu-item" class="{{part:menu/menu-item}}"` + attrs + `>` + iconHTML(ic) + label + sc + `</div>`
}

var _ = registerSheet(func() sheet {
	var specs []specimen
	checked, radio := true, "b"
	items := []MenuItem{
		{Kind: MenuLabel, Label: "My account"},
		{ID: "profile", Label: "Profile", Icon: IconPencil, Shortcut: "⌘P"},
		{ID: "copy", Label: "Copy", Icon: IconCopy, Shortcut: "⌘C"},
		{Kind: MenuSeparator},
		{Kind: MenuCheckbox, ID: "wrap", Label: "Word wrap", Checked: &checked},
		{Kind: MenuRadio, ID: "b", Label: "Comfortable", Group: &radio, Value: "b"},
		{Kind: MenuSub, Label: "More", Items: []MenuItem{{ID: "x", Label: "Nested"}}},
		{Kind: MenuSeparator},
		{ID: "delete", Label: "Delete", Icon: IconTrash, Destructive: true},
	}
	html := popupHTML(
		`<div data-slot="menu-label" class="{{part:menu/menu-label}}">My account</div>`+
			menuItemHTML("Profile", IconPencil, "⌘P", false, false)+
			menuItemHTML("Copy", IconCopy, "⌘C", true, false)+
			`<div data-slot="menu-separator" class="{{part:menu/menu-separator}}"></div>`+
			`<div data-slot="menu-checkbox-item" data-checked="" class="{{part:menu/menu-checkbox-item@when:!(variant === &quot;switch&quot;)}}"><span class="col-start-1 -ms-0.5">`+strings.Replace(iconHTML(IconCheck), "<svg", `<svg class="size-4"`, 1)+`</span><span class="col-start-2">Word wrap</span></div>`+
			`<div data-slot="menu-radio-item" data-checked="" class="{{part:menu/menu-radio-item}}"><span class="col-start-1 -ms-0.5 inline-flex size-4 items-center justify-center"><span class="size-1.5 rounded-full bg-current"></span></span><span class="col-start-2">Comfortable</span></div>`+
			`<div data-slot="menu-sub-trigger" class="{{part:menu/menu-sub-trigger}}">More`+strings.Replace(iconHTML(IconChevronRight), "<svg", `<svg class="ms-auto -me-0.5 opacity-80"`, 1)+`</div>`+
			`<div data-slot="menu-separator" class="{{part:menu/menu-separator}}"></div>`+
			menuItemHTML("Delete", IconTrash, "", false, true), "menu/menu-popup")
	html = strings.ReplaceAll(html, "&quot;", `"`)
	specs = append(specs, specimen{id: "menu/panel", w: 240, h: 300, label: "My account", html: html,
		build: func(c *ui.Context) ui.Element {
			var p ui.Element
			ui.Column(c).Children(func() {
				p = ui.Column(c)
				t := tok(c)
				saved := forced
				forced = 0
				menuPanel(c, t, p, items, MenuOptions{}, nil, 0)
				forced = saved
			})
			return p
		}})
	// Select trigger.
	for _, st := range []State{0, StateHover, StatePressed, StateFocusVisible, StateInvalid, StateDisabled} {
		st := st
		attrs := force(st)
		if st.Has(StateInvalid) {
			attrs += ` aria-invalid="true"`
		}
		if st.Has(StateDisabled) {
			attrs += ` data-disabled=""`
		}
		html := `<button data-g type="button" data-slot="select-trigger" class="{{cva:select.selectTriggerVariants:size=default}}" style="width:180px"` + attrs + `><span data-slot="select-value" class="{{part:select/select-value}}">Apple</span>` +
			strings.Replace(iconHTML(IconChevronsUpDown), "<svg", `<svg class="-me-1 size-4.5 opacity-80 sm:size-4"`, 1) + `</button>`
		specs = append(specs, specimen{id: "select/trigger/" + stateName(st), w: 200, h: 52, state: st, label: "Apple", html: html,
			build: func(c *ui.Context) ui.Element {
				v := "apple"
				return Select(c, &v, []SelectOption{{Value: "apple", Label: "Apple"}, {Value: "pear", Label: "Pear"}}, SelectOptions{Invalid: st.Has(StateInvalid)}).Width(180)
			}})
	}
	for _, size := range []InputSize{InputSM, InputLG} {
		size := size
		sn := inputSizeNames[size]
		html := `<button data-g type="button" data-slot="select-trigger" class="{{cva:select.selectTriggerVariants:size=` + sn + `}}" style="width:180px"><span data-slot="select-value" class="{{part:select/select-value}}">Apple</span>` +
			strings.Replace(iconHTML(IconChevronsUpDown), "<svg", `<svg class="-me-1 size-4.5 opacity-80 sm:size-4"`, 1) + `</button>`
		specs = append(specs, specimen{id: "select/trigger/size-" + sn, w: 200, h: 52, label: "Apple", html: html,
			build: func(c *ui.Context) ui.Element {
				v := "apple"
				return Select(c, &v, []SelectOption{{Value: "apple", Label: "Apple"}}, SelectOptions{Size: size}).Width(180)
			}})
	}
	// Select popup list.
	selItem := func(label string, chosen, hl bool) string {
		a := ""
		if hl {
			a = ` data-highlighted=""`
		}
		ind := ""
		if chosen {
			ind = strings.Replace(iconHTML(IconCheck), "<svg", `<svg class="size-4"`, 1)
		}
		return `<div role="option" data-slot="select-item" class="{{part:select/select-item}}"` + a + `><span class="col-start-1">` + ind + `</span><span class="col-start-2 min-w-0">` + label + `</span></div>`
	}
	specs = append(specs, specimen{id: "select/popup", w: 220, h: 130, label: "Apple",
		html: `<div data-g class="{{part:select/SelectPopup#4}}" style="min-width:180px;height:auto"><div data-slot="select-list" class="{{part:select/select-list}}">` + selItem("Apple", true, false) + selItem("Banana", false, true) + selItem("Cherry", false, false) + `</div></div>`,
		build: func(c *ui.Context) ui.Element {
			t := tok(c)
			p := ui.Column(c).MinWidth(180)
			panelLook(p, t, design.RadiusLG, design.ShadowLg)
			p.Children(func() {
				ui.Column(c).Padding(4).FillWidth().Children(func() {
					for i, l := range []string{"Apple", "Banana", "Cherry"} {
						listRow(c, t, ui.Row(c).Key(i).FillWidth(), l, i == 0, i == 1, false, true)
					}
				})
			})
			return p
		}})
	// Popover and tooltip panels.
	specs = append(specs, specimen{id: "popover/panel", w: 300, h: 130, label: "Dimensions",
		html: `<div data-g data-slot="popover-popup" class="{{part:popover/popover-popup}}" style="width:260px"><div data-slot="popover-viewport" class="{{part:popover/popover-viewport@when:!(tooltipStyle)}}"><div class="mb-4 flex flex-col gap-2"><h2 data-slot="popover-title" class="{{part:popover/popover-title}}">Dimensions</h2><p data-slot="popover-description" class="{{part:popover/popover-description}}">Set the layer size.</p></div></div></div>`,
		build: func(c *ui.Context) ui.Element {
			return PopoverPanel(c, PopoverOptions{Title: "Dimensions", Description: "Set the layer size.", Width: 260}, nil)
		}})
	specs = append(specs, specimen{id: "tooltip/panel", w: 160, h: 50, label: "Add to library",
		html:  `<div data-g data-slot="tooltip-popup" class="{{part:tooltip/tooltip-popup}}"><div data-slot="tooltip-viewport" class="{{part:tooltip/tooltip-viewport}}">Add to library</div></div>`,
		build: func(c *ui.Context) ui.Element { return TooltipPanel(c, "Add to library") }})
	// Combobox and autocomplete inputs.
	for _, st := range []State{0, StateFocusVisible} {
		st := st
		specs = append(specs, specimen{id: "combobox/input/" + stateName(st), w: 240, h: 52, state: st, label: "Select a fruit",
			html: `<div data-slot="combobox-input-group" class="{{part:combobox/combobox-input-group}}" style="width:220px">` +
				`<span data-slot="input-control" class="{{part:input/input-control@when:!unstyled}} has-disabled:opacity-100"><input data-g data-slot="input" class="{{part:input/input|pe-7}}" placeholder="Select a fruit"` + inputAttrs(st) + `></span>` +
				`<button type="button" data-slot="combobox-trigger" class="{{part:combobox/ComboboxInput@base|end-0.5}}" style="position:absolute">` + iconHTML(IconChevronsUpDown) + `</button></div>`,
			build: func(c *ui.Context) ui.Element {
				v := ""
				var e ui.Element
				ui.Row(c).Width(220).Children(func() {
					e = Combobox(c, &v, []SelectOption{{Value: "apple"}}, ComboboxOptions{Placeholder: "Select a fruit"})
				})
				return e
			}})
	}
	return sheet{comp: "popup", specs: specs}
})

var _ = registerSheet(func() sheet {
	var specs []specimen
	footer := func(c *ui.Context) {
		Button(c, "Cancel", ButtonOptions{Variant: ButtonGhost})
		Button(c, "Save", ButtonOptions{})
	}
	btn := func(v, label string) string {
		return fmt.Sprintf(`<button type="button" class="{{cva:button.buttonVariants:variant=%s,size=default}}">%s</button>`, v, label)
	}
	dlg := func(part string, closeBtn bool, footerVariant string) string {
		cb := ""
		if closeBtn {
			cb = `<button type="button" aria-label="Close" class="{{cva:button.buttonVariants:variant=ghost,size=icon|absolute end-2 top-2}}">` + iconHTML(IconX) + `</button>`
		}
		return `<div data-g data-slot="` + part + `-popup" class="{{part:` + part + `/` + part + `-popup}}" style="width:512px">` +
			`<div data-slot="` + part + `-header" class="{{part:` + part + `/` + part + `-header}}"><h2 data-slot="` + part + `-title" class="{{part:` + part + `/` + part + `-title}}">Edit profile</h2><p data-slot="` + part + `-description" class="{{part:` + part + `/` + part + `-description}}">Make changes to your profile here.</p></div>` +
			`<div data-slot="` + part + `-footer" class="{{part:` + part + `/` + part + `-footer@when:variant === &quot;` + footerVariant + `&quot;}}">` + btn("ghost", "Cancel") + btn("default", "Save") + `</div>` + cb + `</div>`
	}
	fix := func(s string) string { return strings.ReplaceAll(s, "&quot;", `"`) }
	specs = append(specs, specimen{id: "dialog/panel", w: 540, h: 200, label: "Edit profile", html: fix(dlg("dialog", true, "default")),
		build: func(c *ui.Context) ui.Element {
			return DialogPanel(c, DialogOptions{Title: "Edit profile", Description: "Make changes to your profile here.", Footer: func() { footer(c) }}, nil)
		}})
	specs = append(specs, specimen{id: "dialog/bare", w: 540, h: 200, label: "Edit profile", html: fix(dlg("dialog", true, "bare")),
		build: func(c *ui.Context) ui.Element {
			return DialogPanel(c, DialogOptions{Title: "Edit profile", Description: "Make changes to your profile here.", Footer: func() { footer(c) }, FooterVariant: FooterBare}, nil)
		}})
	specs = append(specs, specimen{id: "alert-dialog/panel", w: 540, h: 200, label: "Edit profile", html: fix(dlg("alert-dialog", false, "default")),
		build: func(c *ui.Context) ui.Element {
			t := tok(c)
			p := ui.Column(c)
			modalPanel(c, t, p, DialogOptions{Title: "Edit profile", Description: "Make changes to your profile here.", Footer: func() { footer(c) }, NoClose: true}, nil, kindAlert, 1280)
			return p
		}})
	specs = append(specs, specimen{id: "sheet/right", w: 480, h: 220, label: "Edit profile",
		html: fix(`<div data-g data-slot="sheet-popup" class="{{part:sheet/sheet-popup@base@when:side === &quot;right&quot;}}" style="width:448px;height:200px">` +
			`<div data-slot="sheet-header" class="{{part:sheet/sheet-header}}"><h2 data-slot="sheet-title" class="{{part:sheet/sheet-title}}">Edit profile</h2><p data-slot="sheet-description" class="{{part:sheet/sheet-description}}">Make changes to your profile here.</p></div>` +
			`<button type="button" aria-label="Close" class="{{cva:button.buttonVariants:variant=ghost,size=icon|absolute end-2 top-2}}">` + iconHTML(IconX) + `</button></div>`),
		build: func(c *ui.Context) ui.Element {
			return SheetPanel(c, DialogOptions{Title: "Edit profile", Description: "Make changes to your profile here."}, nil).Height(200)
		}})
	specs = append(specs, specimen{id: "drawer/bottom", w: 540, h: 200, label: "Edit profile",
		build: func(c *ui.Context) ui.Element {
			return DrawerPanel(c, DialogOptions{Title: "Edit profile", Description: "Make changes to your profile here.", NoClose: true}, nil).Width(512)
		}})
	q := ""
	cmds := []CommandItem{{ID: "new", Label: "New chat", Group: "Suggestions", Icon: IconPlus, Shortcut: "⌘N"}, {ID: "search", Label: "Search files", Group: "Suggestions", Icon: IconSearch}, {ID: "settings", Label: "Settings", Group: "General", Icon: IconPencil, Shortcut: "⌘,"}}
	specs = append(specs, specimen{id: "command/panel", w: 600, h: 300, label: "New chat",
		build: func(c *ui.Context) ui.Element {
			return ui.Column(c).Children(func() {
				CommandPanel(c, &q, cmds, CommandOptions{Placeholder: "Type a command or search…", Footer: func() {
					text(c, "↑↓ to navigate", design.TextXS, design.WeightMedium, tok(c).MutedForeground)
					text(c, "↵ to open", design.TextXS, design.WeightMedium, tok(c).MutedForeground)
				}})
			})
		}})
	return sheet{comp: "dialog", specs: specs}
})
