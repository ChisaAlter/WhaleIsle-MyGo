package coss

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
)

var _ = registerSheet(func() sheet {
	var specs []specimen
	segSize := map[SegmentedSize][2]string{SegmentedMD: {"default", "h-8.5 px-[calc(--spacing(2.5)-1px)] sm:h-7.5"}, SegmentedSM: {"sm", "h-7.5 px-[calc(--spacing(2)-1px)] sm:h-6.5"}, SegmentedLG: {"lg", "h-9.5 px-[calc(--spacing(3)-1px)] sm:h-8.5"}}
	layoutCls := "gap-1.5 [&_svg:not([class*='opacity-'])]:opacity-80 [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:-mx-0.5 [&_svg]:shrink-0"
	for _, v := range []TabsVariant{TabsDefault, TabsUnderline} {
		for _, st := range []State{0, StateHover, StateFocusVisible} {
			v, st := v, st
			vn := map[TabsVariant]string{TabsDefault: "default", TabsUnderline: "underline"}[v]
			listCfg := `@when:variant === "default"`
			if v == TabsUnderline {
				listCfg = `@when:!(variant === "default")`
			}
			var b strings.Builder
			fmt.Fprintf(&b, `<div data-g role="tablist" data-orientation="horizontal" data-slot="tabs-list" class="{{part:tabs/tabs-list@base%s}}">`, listCfg)
			for i, l := range []string{"Account", "Password", "Team"} {
				attrs := ` data-orientation="horizontal"`
				if i == 0 {
					attrs += ` data-active="" aria-selected="true"`
				}
				if i == 1 {
					attrs += force(st)
				}
				extra := ""
				if i == 0 && v == TabsDefault {
					extra = " bg-background shadow-sm/5 dark:bg-input"
				}
				fmt.Fprintf(&b, `<button role="tab" data-slot="tabs-tab" class="{{part:tabs/tabs-tab|%s %s%s}}"%s>%s</button>`, layoutCls, segSize[SegmentedMD][1], extra, attrs, l)
			}
			if v == TabsUnderline {
				b.WriteString(`<span data-slot="tab-indicator" data-orientation="horizontal" class="{{part:tabs/tab-indicator@base@when:variant === &quot;underline&quot;}}" style="--active-tab-width:84px;--active-tab-left:0px;--active-tab-bottom:0px;--active-tab-height:2px"></span>`)
			}
			b.WriteString(`</div>`)
			specs = append(specs, specimen{id: fmt.Sprintf("tabs/%s/%s", vn, stateName(st)), w: 300, h: 56, label: "Account", html: strings.ReplaceAll(b.String(), "&quot;", `"`),
				build: func(c *ui.Context) ui.Element {
					sel := 0
					segForce = st
					defer func() { segForce = 0 }()
					return Tabs(c, &sel, []string{"Account", "Password", "Team"}, TabsOptions{Variant: v})
				}})
		}
	}
	specs = append(specs, specimen{id: "card/full", w: 380, h: 240, label: "Create project",
		html: `<div data-g data-slot="card" class="{{part:card/card}}" style="width:350px"><div data-slot="card-header" class="{{part:card/card-header}}"><div data-slot="card-title" class="{{part:card/card-title}}">Create project</div><div data-slot="card-description" class="{{part:card/card-description}}">Deploy your new project in one click.</div></div>` +
			`<div data-slot="card-panel" class="{{part:card/card-panel}}"><p class="text-sm">Body</p></div><div data-slot="card-footer" class="{{part:card/card-footer}}"><button type="button" class="{{cva:button.buttonVariants:variant=outline,size=default}}">Cancel</button></div></div>`,
		build: func(c *ui.Context) ui.Element {
			t := tok(c)
			return Card(c, CardOptions{Title: "Create project", Description: "Deploy your new project in one click.", Footer: func() {
				Button(c, "Cancel", ButtonOptions{Variant: ButtonOutline})
			}}, func() { para(c, "Body", design.TextSM, design.WeightNormal, t.Foreground) }).Width(350)
		}})
	specs = append(specs, specimen{id: "frame/panels", w: 380, h: 200, label: "Section",
		html: `<div data-g data-slot="frame" class="{{part:frame/frame}}" style="width:350px"><header data-slot="frame-panel-header" class="{{part:frame/frame-panel-header}}"><div data-slot="frame-panel-title" class="{{part:frame/frame-panel-title}}">Section</div><div data-slot="frame-panel-description" class="{{part:frame/frame-panel-description}}">Details below.</div></header><div data-slot="frame-panel" class="{{part:frame/frame-panel}}"><p class="text-sm">Panel</p></div></div>`,
		build: func(c *ui.Context) ui.Element {
			t := tok(c)
			return Frame(c, func() {
				FrameHeader(c, "Section", "Details below.")
				FramePanel(c, func() { para(c, "Panel", design.TextSM, design.WeightNormal, t.Foreground) })
			}).Width(350)
		}})
	for _, v := range []AlertVariant{AlertDefault, AlertInfo, AlertSuccess, AlertWarning, AlertError} {
		v := v
		vn := map[AlertVariant]string{AlertDefault: "default", AlertInfo: "info", AlertSuccess: "success", AlertWarning: "warning", AlertError: "error"}[v]
		ic := map[AlertVariant]Icon{AlertDefault: IconInfo, AlertInfo: IconInfo, AlertSuccess: IconCircleCheck, AlertWarning: IconTriangleAlert, AlertError: IconCircleAlert}[v]
		specs = append(specs, specimen{id: "alert/" + vn, w: 380, h: 96, label: "Heads up",
			html: `<div data-g role="alert" data-slot="alert" class="{{cva:alert.alertVariants:variant=` + vn + `}}" style="width:350px">` + iconHTML(ic) +
				`<div data-slot="alert-title" class="{{part:alert/alert-title}}">Heads up</div><div data-slot="alert-description" class="{{part:alert/alert-description}}">The session reconnected.</div></div>`,
			build: func(c *ui.Context) ui.Element {
				return Alert(c, "Heads up", "The session reconnected.", AlertOptions{Variant: v}).Width(350)
			}})
	}
	specs = append(specs, specimen{id: "empty/icon", w: 380, h: 330, label: "No projects",
		html: `<div data-g data-slot="empty" class="{{part:empty/empty}}" style="width:350px"><div data-slot="empty-header" class="{{part:empty/empty-header}}"><div data-slot="empty-media" class="{{part:empty/empty-media}}"><div class="{{cva:empty.emptyMediaVariants:variant=icon}}">` + iconHTML(IconFile) + `</div></div>` +
			`<div data-slot="empty-title" class="{{part:empty/empty-title}}">No projects</div><div data-slot="empty-description" class="{{part:empty/empty-description}}">Create one to get started.</div></div>` +
			`<div data-slot="empty-content" class="{{part:empty/empty-content}}"><button type="button" class="{{cva:button.buttonVariants:variant=default,size=default}}">New project</button></div></div>`,
		build: func(c *ui.Context) ui.Element {
			return Empty(c, EmptyOptions{Icon: IconFile, Title: "No projects", Description: "Create one to get started."}, func() {
				Button(c, "New project", ButtonOptions{})
			}).Width(350)
		}})
	specs = append(specs, specimen{id: "accordion/open", w: 380, h: 170, label: "Is it accessible?",
		html: `<div data-g data-slot="accordion" style="width:350px"><div data-slot="accordion-item" class="{{part:accordion/accordion-item}}"><h3 class="flex"><button data-panel-open="" data-slot="accordion-trigger" class="{{part:accordion/accordion-trigger}}">Is it accessible?` +
			strings.Replace(iconHTML(IconChevronDown), "<svg", `<svg data-slot="accordion-indicator" class="pointer-events-none size-4 shrink-0 translate-y-0.5 opacity-80 rotate-180"`, 1) + `</button></h3>` +
			`<div data-slot="accordion-panel" class="{{part:accordion/accordion-panel}}" style="height:auto"><div class="pt-0 pb-4">Yes. It adheres to the WAI-ARIA pattern.</div></div></div>` +
			`<div data-slot="accordion-item" class="{{part:accordion/accordion-item}}"><h3 class="flex"><button data-slot="accordion-trigger" class="{{part:accordion/accordion-trigger}}">Is it styled?` + strings.Replace(iconHTML(IconChevronDown), "<svg", `<svg data-slot="accordion-indicator" class="pointer-events-none size-4 shrink-0 translate-y-0.5 opacity-80"`, 1) + `</button></h3></div></div>`,
		build: func(c *ui.Context) ui.Element {
			a, b := true, false
			return Accordion(c, []AccordionItem{
				{Title: "Is it accessible?", Open: &a, Content: func() {
					para(c, "Yes. It adheres to the WAI-ARIA pattern.", design.TextSM, design.WeightNormal, tok(c).MutedForeground)
				}},
				{Title: "Is it styled?", Open: &b, Content: func() {}},
			}).Width(350)
		}})
	specs = append(specs, specimen{id: "avatar/initials", w: 60, h: 50, label: "AL",
		html:  `<span data-g data-slot="avatar" class="{{part:avatar/avatar}}"><span data-slot="avatar-fallback" class="{{part:avatar/avatar-fallback}}">AL</span></span>`,
		build: func(c *ui.Context) ui.Element { return Avatar(c, "ada lovelace", AvatarOptions{}) }})
	specs = append(specs, specimen{id: "breadcrumb/three", w: 260, h: 40, label: "Home",
		html: `<nav data-g aria-label="breadcrumb" data-slot="breadcrumb"><ol data-slot="breadcrumb-list" class="{{part:breadcrumb/breadcrumb-list}}"><li data-slot="breadcrumb-item" class="{{part:breadcrumb/breadcrumb-item}}"><a data-slot="breadcrumb-link" class="{{part:breadcrumb/breadcrumb-link}}" href="#">Home</a></li>` +
			`<li data-slot="breadcrumb-separator" class="{{part:breadcrumb/breadcrumb-separator}}">` + iconHTML(IconChevronRight) + `</li><li data-slot="breadcrumb-item" class="{{part:breadcrumb/breadcrumb-item}}"><a data-slot="breadcrumb-link" class="{{part:breadcrumb/breadcrumb-link}}" href="#">Docs</a></li>` +
			`<li data-slot="breadcrumb-separator" class="{{part:breadcrumb/breadcrumb-separator}}">` + iconHTML(IconChevronRight) + `</li><li data-slot="breadcrumb-item" class="{{part:breadcrumb/breadcrumb-item}}"><span data-slot="breadcrumb-page" class="{{part:breadcrumb/breadcrumb-page}}">Button</span></li></ol></nav>`,
		build: func(c *ui.Context) ui.Element {
			var e ui.Element
			e = ui.Row(c).Children(func() { Breadcrumb(c, []string{"Home", "Docs", "Button"}) })
			return e
		}})
	specs = append(specs, specimen{id: "toolbar/basic", w: 260, h: 60, label: "",
		html: `<div data-g role="toolbar" data-slot="toolbar" class="{{part:toolbar/toolbar}}"><div data-slot="toolbar-group" class="{{part:toolbar/toolbar-group}}"><button type="button" class="{{cva:toggle.toggleVariants:variant=default,size=default}}">` + iconHTML(IconPencil) + `</button><button type="button" class="{{cva:toggle.toggleVariants:variant=default,size=default}}">` + iconHTML(IconCopy) + `</button></div>` +
			`<div data-slot="toolbar-separator" data-orientation="vertical" class="{{part:toolbar/toolbar-separator}}"></div><div data-slot="toolbar-group" class="{{part:toolbar/toolbar-group}}"><button type="button" class="{{cva:button.buttonVariants:variant=default,size=default}}">Save</button></div></div>`,
		build: func(c *ui.Context) ui.Element {
			a, b := false, false
			return Toolbar(c, "Editor", func() {
				ToolbarGroup(c, func() {
					Toggle(c, &a, "", ToggleOptions{Icon: IconPencil, Label: "Edit"})
					Toggle(c, &b, "", ToggleOptions{Icon: IconCopy, Label: "Copy"})
				})
				ToolbarSeparator(c)
				ToolbarGroup(c, func() { Button(c, "Save", ButtonOptions{}) })
			})
		}})
	specs = append(specs, specimen{id: "group/buttons", w: 260, h: 52, label: "Left",
		html: `<div data-g role="group" data-orientation="horizontal" data-slot="group" class="{{cva:group.groupVariants:orientation=horizontal}}">` +
			`<button data-slot="button" type="button" class="{{cva:button.buttonVariants:variant=outline,size=default}}">Left</button><button data-slot="button" type="button" class="{{cva:button.buttonVariants:variant=outline,size=default}}">Middle</button><button data-slot="button" type="button" class="{{cva:button.buttonVariants:variant=outline,size=default}}">Right</button></div>`,
		build: func(c *ui.Context) ui.Element {
			return Group(c, 3, Horizontal, func() {
				Button(c, "Left", ButtonOptions{Variant: ButtonOutline})
				Button(c, "Middle", ButtonOptions{Variant: ButtonOutline})
				Button(c, "Right", ButtonOptions{Variant: ButtonOutline})
			})
		}})
	specs = append(specs, specimen{id: "input-group/addons", w: 280, h: 52, label: "https://",
		html: `<div data-slot="input-group" role="group" class="{{part:input-group/input-group}}" style="width:260px"><div data-slot="input-group-addon" data-align="inline-start" class="{{cva:input-group.inputGroupAddonVariants:align=inline-start}}"><span class="{{part:input-group/InputGroupText}}">https://</span></div>` +
			`<span data-slot="input-control" class="{{part:input/input-control@when:!unstyled}} contents"><input data-g data-slot="input" class="{{part:input/input}}" placeholder="example.com"></span></div>`,
		build: func(c *ui.Context) ui.Element {
			v := ""
			var e ui.Element
			ui.Row(c).Width(260).Children(func() {
				e = InputGroup(c, &v, InputGroupOptions{InputOptions: InputOptions{Placeholder: "example.com"}, Start: func() { AddonText(c, "https://") }})
			})
			return e
		}})
	specs = append(specs, specimen{id: "table/default", w: 420, h: 170, label: "Invoice",
		html: `<div data-g data-slot="table-container" data-variant="default" class="{{part:table/table-container}}" style="width:400px"><table data-slot="table" class="{{part:table/table}}"><thead data-slot="table-header" class="{{part:table/table-header}}"><tr data-slot="table-row" class="{{part:table/table-row}}"><th data-slot="table-head" class="{{part:table/table-head}}">Invoice</th><th data-slot="table-head" class="{{part:table/table-head}}">Status</th></tr></thead>` +
			`<tbody data-slot="table-body" class="{{part:table/table-body}}"><tr data-slot="table-row" class="{{part:table/table-row}}"><td data-slot="table-cell" class="{{part:table/table-cell}}">INV001</td><td data-slot="table-cell" class="{{part:table/table-cell}}">Paid</td></tr><tr data-slot="table-row" class="{{part:table/table-row}}"><td data-slot="table-cell" class="{{part:table/table-cell}}">INV002</td><td data-slot="table-cell" class="{{part:table/table-cell}}">Pending</td></tr></tbody></table></div>`,
		build: func(c *ui.Context) ui.Element {
			var e ui.Element
			e = ui.Column(c).Width(400).Children(func() {
				Table(c, []string{"Invoice", "Status"}, [][]string{{"INV001", "Paid"}, {"INV002", "Pending"}}, TableOptions{Selected: -1})
			})
			return e
		}})
	specs = append(specs, specimen{id: "pagination/middle", w: 420, h: 52, label: "Previous",
		build: func(c *ui.Context) ui.Element {
			p := 5
			var e ui.Element
			e = ui.Row(c).Width(400).Children(func() { Pagination(c, &p, 10) })
			return e
		}})
	specs = append(specs, specimen{id: "scroll-area/hover", w: 220, h: 140, state: StateHover,
		build: func(c *ui.Context) ui.Element {
			t := tok(c)
			return ScrollArea(c, ScrollAreaOptions{}, func() {
				ui.Column(c).Padding(8).Children(func() {
					for i := 0; i < 12; i++ {
						text(c, fmt.Sprintf("Row %d", i+1), design.TextSM, design.WeightNormal, t.Foreground).Key(i)
					}
				})
			}).Size(200, 120).Border(1, t.Border).Radius(10)
		}})
	return sheet{comp: "layout", specs: specs}
})
