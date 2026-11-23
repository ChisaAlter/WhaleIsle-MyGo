package coss

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

var inputSizeNames = map[InputSize]string{InputMD: "default", InputSM: "sm", InputLG: "lg"}

var fieldStates = []State{0, StateHover, StateFocusVisible, StateInvalid, StateInvalid | StateFocusVisible, StateDisabled}

// inputAttrs returns the input element's attributes for a state.
func inputAttrs(s State) string {
	a := ""
	if s.Has(StateFocusVisible) {
		a += ` data-force="focus,focus-visible"`
	} else if s.Has(StateHover) {
		a += ` data-force="hover"`
	}
	if s.Has(StateInvalid) {
		a += ` aria-invalid="true"`
	}
	if s.Has(StateDisabled) {
		a += ` disabled`
	}
	return a
}

func sizeCfg(part string, size InputSize) string {
	if size == InputMD {
		return part
	}
	return part + `@when:size === "` + inputSizeNames[size] + `"`
}

func inputHTML(size InputSize, s State, value, ph string) string {
	return fmt.Sprintf(`<span data-slot="input-control" class="{{part:input/input-control@when:!unstyled}}"><input data-g data-slot="input" class="{{%s}}" value="%s" placeholder="%s"%s></span>`,
		"part:"+sizeCfg("input/input", size), value, ph, inputAttrs(s))
}

func textareaHTML(size InputSize, s State, value, ph string) string {
	return fmt.Sprintf(`<span data-slot="textarea-control" class="{{part:textarea/textarea-control@when:!unstyled}}"><textarea data-g data-slot="textarea" class="{{%s}}" placeholder="%s"%s>%s</textarea></span>`,
		"part:"+sizeCfg("textarea/textarea", size), ph, inputAttrs(s), value)
}

var _ = registerSheet(func() sheet {
	var specs []specimen
	for _, size := range []InputSize{InputMD, InputSM, InputLG} {
		for _, st := range fieldStates {
			for _, filled := range []bool{false, true} {
				size, st, filled := size, st, filled
				val, kind := "", "placeholder"
				if filled {
					val, kind = "Value", "value"
				}
				label := "Placeholder"
				if filled {
					label = ""
				}
				specs = append(specs, specimen{
					id: fmt.Sprintf("input/%s/%s/%s", inputSizeNames[size], kind, stateName(st)), w: 210, h: 52, state: st, label: label,
					html: inputHTML(size, st, val, "Placeholder"),
					build: func(c *ui.Context) ui.Element {
						v := val
						return Input(c, &v, InputOptions{Size: size, Placeholder: "Placeholder", Invalid: st.Has(StateInvalid), Disabled: st.Has(StateDisabled)})
					},
				})
			}
		}
	}
	return sheet{comp: "input", specs: specs}
})

var _ = registerSheet(func() sheet {
	var specs []specimen
	for _, size := range []InputSize{InputMD, InputSM, InputLG} {
		for _, st := range fieldStates {
			size, st := size, st
			specs = append(specs, specimen{
				id: fmt.Sprintf("textarea/%s/%s", inputSizeNames[size], stateName(st)), w: 250, h: 100, state: st, label: "Type your message here.",
				html: textareaHTML(size, st, "", "Type your message here."),
				build: func(c *ui.Context) ui.Element {
					v := ""
					return Textarea(c, &v, TextareaOptions{Size: size, Placeholder: "Type your message here.", Invalid: st.Has(StateInvalid), Disabled: st.Has(StateDisabled)})
				},
			})
		}
	}
	return sheet{comp: "textarea", specs: specs}
})

var _ = registerSheet(func() sheet {
	var specs []specimen
	for _, kind := range []string{"description", "error", "disabled"} {
		kind := kind
		help := `<p data-slot="field-description" class="{{part:field/field-description}}">Visible to everyone.</p>`
		attrs := ""
		if kind == "error" {
			help = `<div data-slot="field-error" class="{{part:field/field-error}}">Enter a valid email address.</div>`
			attrs = ` aria-invalid="true"`
		}
		if kind == "disabled" {
			attrs = ` disabled`
		}
		html := `<div data-g data-slot="field" class="{{part:field/field}}" style="width:240px"` + map[bool]string{true: ` data-disabled=""`}[kind == "disabled"] + `>` +
			`<label data-slot="field-label" class="{{part:field/field-label}}"` + map[bool]string{true: ` data-disabled=""`}[kind == "disabled"] + `>Email</label>` +
			strings.Replace(inputHTML(InputMD, 0, "", "you@example.com"), "<input data-g", "<input", 1) + help + `</div>`
		html = strings.Replace(html, `placeholder="you@example.com">`, `placeholder="you@example.com"`+attrs+`>`, 1)
		specs = append(specs, specimen{
			id: "field/" + kind, w: 260, h: 110, label: "Email", html: html,
			build: func(c *ui.Context) ui.Element {
				v := ""
				o := FieldOptions{Label: "Email", Description: "Visible to everyone."}
				if kind == "error" {
					o.Error = "Enter a valid email address."
				}
				o.Disabled = kind == "disabled"
				return Field(c, o, func() { Input(c, &v, InputOptions{Placeholder: "you@example.com"}) }).Width(240)
			},
		})
	}
	return sheet{comp: "field", specs: specs}
})

var badgeVariantNames = map[BadgeVariant]string{BadgeDefault: "default", BadgeSecondary: "secondary", BadgeOutline: "outline", BadgeDestructive: "destructive", BadgeError: "error", BadgeInfo: "info", BadgeSuccess: "success", BadgeWarning: "warning"}

var badgeSizeNames = map[BadgeSize]string{BadgeMD: "default", BadgeSM: "sm", BadgeLG: "lg"}

var _ = registerSheet(func() sheet {
	var specs []specimen
	for v := BadgeDefault; v <= BadgeWarning; v++ {
		for _, size := range []BadgeSize{BadgeMD, BadgeSM, BadgeLG} {
			for _, ic := range []Icon{IconNone, IconCheck} {
				v, size, ic := v, size, ic
				specs = append(specs, specimen{
					id: fmt.Sprintf("badge/%s/%s/%v", badgeVariantNames[v], badgeSizeNames[size], ic != IconNone), w: 90, h: 40, label: "Badge",
					html: fmt.Sprintf(`<span data-g data-slot="badge" class="{{cva:badge.badgeVariants:variant=%s,size=%s}}">%sBadge</span>`, badgeVariantNames[v], badgeSizeNames[size], iconHTML(ic)),
					build: func(c *ui.Context) ui.Element {
						return Badge(c, "Badge", BadgeOptions{Variant: v, Size: size, Icon: ic})
					},
				})
			}
		}
	}
	for _, st := range []State{StateHover, StateFocusVisible} {
		st := st
		specs = append(specs, specimen{
			id: "badge/outline/interactive/" + stateName(st), w: 90, h: 40, state: st, label: "Badge",
			html: `<button data-g type="button" data-slot="badge" class="{{cva:badge.badgeVariants:variant=outline,size=default}}"` + force(st) + `>Badge</button>`,
			build: func(c *ui.Context) ui.Element {
				return Badge(c, "Badge", BadgeOptions{Variant: BadgeOutline, Interactive: true})
			},
		})
	}
	return sheet{comp: "badge", specs: specs}
})

var _ = registerSheet(func() sheet {
	specs := []specimen{
		{id: "kbd/single", w: 60, h: 40, label: "K", html: `<kbd data-g data-slot="kbd" class="{{part:kbd/kbd}}">K</kbd>`,
			build: func(c *ui.Context) ui.Element { return Kbd(c, "K") }},
		{id: "kbd/word", w: 80, h: 40, label: "Ctrl", html: `<kbd data-g data-slot="kbd" class="{{part:kbd/kbd}}">Ctrl</kbd>`,
			build: func(c *ui.Context) ui.Element { return Kbd(c, "Ctrl") }},
		{id: "kbd/group", w: 120, h: 40, label: "Ctrl", html: `<kbd data-g data-slot="kbd-group" class="{{part:kbd/kbd-group}}"><kbd data-slot="kbd" class="{{part:kbd/kbd}}">Ctrl</kbd><kbd data-slot="kbd" class="{{part:kbd/kbd}}">K</kbd></kbd>`,
			build: func(c *ui.Context) ui.Element { return KbdGroup(c, "Ctrl", "K") }},
		{id: "separator/horizontal", w: 200, h: 30, html: `<div data-g data-slot="separator" data-orientation="horizontal" class="{{part:separator/separator}}"></div>`,
			build: func(c *ui.Context) ui.Element { return Separator(c, Horizontal) }},
		{id: "separator/vertical", w: 60, h: 60, html: `<div style="display:flex;height:40px"><div data-g data-slot="separator" data-orientation="vertical" class="{{part:separator/separator}}"></div></div>`,
			build: func(c *ui.Context) ui.Element {
				var s ui.Element
				ui.Row(c).Height(40).Children(func() { s = Separator(c, Vertical) })
				return s
			}},
		{id: "skeleton/line", w: 220, h: 40, html: `<div data-g data-slot="skeleton" class="{{part:skeleton/skeleton|h-4 w-48}}"></div>`,
			build: func(c *ui.Context) ui.Element { return Skeleton(c).Size(192, 16) }},
		{id: "skeleton/avatar", w: 60, h: 60, html: `<div data-g data-slot="skeleton" class="{{part:skeleton/skeleton|size-10 rounded-full}}"></div>`,
			build: func(c *ui.Context) ui.Element { return Skeleton(c).Size(40, 40).Radius(9999) }},
		{id: "spinner/default", w: 40, h: 40, html: `<span data-g class="animate-spin" role="status" style="display:inline-flex;width:16px;height:16px">` + strings.Replace(iconHTML(IconLoader), `width="24" height="24"`, `width="16" height="16"`, 1) + `</span>`,
			build: func(c *ui.Context) ui.Element { return Spinner(c, SpinnerOptions{}) }},
	}
	return sheet{comp: "misc", specs: specs}
})
