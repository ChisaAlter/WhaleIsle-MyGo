package coss

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

var buttonVariantNames = map[ButtonVariant]string{ButtonDefault: "default", ButtonOutline: "outline", ButtonSecondary: "secondary", ButtonGhost: "ghost", ButtonDestructive: "destructive", ButtonDestructiveOutline: "destructive-outline", ButtonLink: "link"}

var buttonSizeNames = map[ButtonSize]string{ButtonMD: "default", ButtonXS: "xs", ButtonSM: "sm", ButtonLG: "lg", ButtonXL: "xl", ButtonIconXS: "icon-xs", ButtonIconSM: "icon-sm", ButtonIcon: "icon", ButtonIconLG: "icon-lg", ButtonIconXL: "icon-xl"}

func buttonHTML(v ButtonVariant, size ButtonSize, s State, label string, ic Icon) string {
	attrs := force(s)
	if s.Has(StateDisabled) {
		attrs += " disabled"
	}
	body := iconHTML(ic)
	if size < ButtonIconXS {
		body += label
	}
	return fmt.Sprintf(`<button data-g type="button" class="{{cva:button.buttonVariants:variant=%s,size=%s}}"%s>%s</button>`, buttonVariantNames[v], buttonSizeNames[size], attrs, body)
}

var _ = registerSheet(func() sheet {
	var specs []specimen
	for v := ButtonDefault; v <= ButtonLink; v++ {
		for _, st := range baseStates {
			v, st := v, st
			specs = append(specs, specimen{
				id: fmt.Sprintf("button/%s/default/%s", buttonVariantNames[v], stateName(st)), w: 130, h: 52, state: st, label: "Button",
				html:  buttonHTML(v, ButtonMD, st, "Button", IconPlus),
				build: func(c *ui.Context) ui.Element { return Button(c, "Button", ButtonOptions{Variant: v, Icon: IconPlus}) },
			})
		}
	}
	for size := ButtonMD; size <= ButtonIconXL; size++ {
		for _, v := range []ButtonVariant{ButtonDefault, ButtonOutline} {
			size, v := size, v
			specs = append(specs, specimen{
				id: fmt.Sprintf("button/%s/size-%s", buttonVariantNames[v], buttonSizeNames[size]), w: 130, h: 60, label: "Button",
				html: buttonHTML(v, size, 0, "Button", IconPlus),
				build: func(c *ui.Context) ui.Element {
					return Button(c, "Button", ButtonOptions{Variant: v, Size: size, Icon: IconPlus})
				},
			})
		}
	}
	specs = append(specs, specimen{id: "button/default/default/loading", w: 130, h: 52, state: StateLoading, label: "",
		html:  `<button data-g type="button" aria-disabled="true" data-loading="" disabled class="{{cva:button.buttonVariants:variant=default,size=default}}">Loading<span data-slot="button-loading-indicator" class="pointer-events-none absolute inset-0 flex items-center justify-center">` + iconHTML(IconLoader) + `</span></button>`,
		build: func(c *ui.Context) ui.Element { return Button(c, "Loading", ButtonOptions{Loading: true}) }})
	return sheet{comp: "button", specs: specs}
})
