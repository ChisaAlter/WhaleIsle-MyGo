package coss

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

var checkStates = []State{0, StateHover, StateFocusVisible, StateInvalid, StateDisabled}

// checkAttrs are Base UI's data attributes of a check control.
func checkAttrs(s State, on bool) string {
	a := ` data-unchecked=""`
	if on {
		a = ` data-checked=""`
	}
	if s.Has(StateFocusVisible) {
		a += ` data-force="focus,focus-visible"`
	} else if s.Has(StateHover) {
		a += ` data-force="hover"`
	}
	if s.Has(StateInvalid) {
		a += ` aria-invalid="true"`
	}
	if s.Has(StateDisabled) {
		a += ` data-disabled=""`
	}
	return a
}

func labelWrap(inner, label string, s State) string {
	dis := ""
	if s.Has(StateDisabled) {
		dis = ` style="opacity:1"`
	}
	return `<label data-g data-slot="label" class="{{part:label/label}}"` + dis + `>` + inner + label + `</label>`
}

var _ = registerSheet(func() sheet {
	var specs []specimen
	for _, kind := range []string{"unchecked", "checked", "indeterminate"} {
		for _, st := range checkStates {
			kind, st := kind, st
			on := kind == "checked"
			ind := ""
			attrs := checkAttrs(st, on)
			switch kind {
			case "checked":
				ind = `<span data-slot="checkbox-indicator" data-checked="" class="{{part:checkbox/checkbox-indicator}}">` + strings.Replace(iconHTML(IconCheckboxCheck), `<svg`, `<svg class="size-3.5 sm:size-3"`, 1) + `</span>`
			case "indeterminate":
				attrs = strings.Replace(attrs, ` data-unchecked=""`, ` data-indeterminate=""`, 1)
				ind = `<span data-slot="checkbox-indicator" data-indeterminate="" class="{{part:checkbox/checkbox-indicator}}">` + strings.Replace(iconHTML(IconCheckboxMinus), `<svg`, `<svg class="size-3.5 sm:size-3"`, 1) + `</span>`
			}
			html := labelWrap(`<span role="checkbox" data-slot="checkbox" class="{{part:checkbox/checkbox}}"`+attrs+`>`+ind+`</span>`, "Accept terms", st)
			specs = append(specs, specimen{id: "checkbox/" + kind + "/" + stateName(st), w: 150, h: 36, state: st, label: "Accept terms", html: html,
				build: func(c *ui.Context) ui.Element {
					v := on
					return Checkbox(c, &v, "Accept terms", CheckOptions{Indeterminate: kind == "indeterminate", Invalid: st.Has(StateInvalid)})
				}})
		}
	}
	for _, on := range []bool{false, true} {
		for _, st := range checkStates {
			on, st := on, st
			ind := ""
			if on {
				ind = `<span data-slot="radio-indicator" data-checked="" class="{{part:radio-group/radio-indicator}}"></span>`
			}
			html := labelWrap(`<span role="radio" data-slot="radio" class="{{part:radio-group/radio}}"`+checkAttrs(st, on)+`>`+ind+`</span>`, "Comfortable", st)
			specs = append(specs, specimen{id: fmt.Sprintf("radio/%v/%s", on, stateName(st)), w: 150, h: 36, state: st, label: "Comfortable", html: html,
				build: func(c *ui.Context) ui.Element {
					sel := "x"
					if on {
						sel = "a"
					}
					return Radio(c, &sel, "a", "Comfortable", CheckOptions{Invalid: st.Has(StateInvalid)})
				}})
		}
	}
	for _, on := range []bool{false, true} {
		for _, st := range []State{0, StateHover, StateFocusVisible, StateDisabled} {
			on, st := on, st
			ca := checkAttrs(st, on)
			thumb := `<span data-slot="switch-thumb" class="{{part:switch/switch-thumb}}"` + map[bool]string{true: ` data-checked=""`, false: ` data-unchecked=""`}[on] + `></span>`
			html := labelWrap(`<span role="switch" data-slot="switch" class="{{part:switch/switch}}"`+ca+`>`+thumb+`</span>`, "Airplane mode", st)
			specs = append(specs, specimen{id: fmt.Sprintf("switch/%v/%s", on, stateName(st)), w: 160, h: 36, state: st, label: "Airplane mode", html: html,
				build: func(c *ui.Context) ui.Element {
					v := on
					return Switch(c, &v, "Airplane mode", CheckOptions{})
				}})
		}
	}
	return sheet{comp: "check", specs: specs}
})

var toggleVariantNames = map[ToggleVariant]string{ToggleDefault: "default", ToggleOutline: "outline"}
var toggleSizeNames = map[ToggleSize]string{ToggleMD: "default", ToggleSM: "sm", ToggleLG: "lg"}

func toggleHTML(v ToggleVariant, size ToggleSize, s State, on bool, label string, ic Icon, extra string) string {
	attrs := force(s &^ StatePressed)
	if s.Has(StatePressed) {
		attrs += ` data-force="active"`
	}
	if on {
		attrs += ` data-pressed="" aria-pressed="true"`
	}
	if s.Has(StateDisabled) {
		attrs += ` disabled`
	}
	return fmt.Sprintf(`<button data-g type="button" data-slot="toggle" class="{{cva:toggle.toggleVariants:variant=%s,size=%s%s}}"%s>%s%s</button>`,
		toggleVariantNames[v], toggleSizeNames[size], extra, attrs, iconHTML(ic), label)
}

var _ = registerSheet(func() sheet {
	var specs []specimen
	for _, v := range []ToggleVariant{ToggleDefault, ToggleOutline} {
		for _, on := range []bool{false, true} {
			for _, st := range []State{0, StateHover, StateFocusVisible, StateDisabled} {
				v, on, st := v, on, st
				specs = append(specs, specimen{id: fmt.Sprintf("toggle/%s/%v/%s", toggleVariantNames[v], on, stateName(st)), w: 100, h: 50, state: st, label: "Bold",
					html: toggleHTML(v, ToggleMD, st, on, "Bold", IconPencil, ""),
					build: func(c *ui.Context) ui.Element {
						b := on
						return Toggle(c, &b, "Bold", ToggleOptions{Variant: v, Icon: IconPencil})
					}})
			}
		}
		for _, size := range []ToggleSize{ToggleSM, ToggleLG} {
			v, size := v, size
			specs = append(specs, specimen{id: fmt.Sprintf("toggle/%s/size-%s", toggleVariantNames[v], toggleSizeNames[size]), w: 100, h: 50, label: "Bold",
				html: toggleHTML(v, size, 0, false, "Bold", IconPencil, ""),
				build: func(c *ui.Context) ui.Element {
					b := false
					return Toggle(c, &b, "Bold", ToggleOptions{Variant: v, Size: size, Icon: IconPencil})
				}})
		}
	}
	for _, v := range []ToggleVariant{ToggleDefault, ToggleOutline} {
		v := v
		cfg := `@when:orientation === "horizontal"`
		if v == ToggleDefault {
			cfg += `@when:variant === "default"`
		} else {
			cfg += `@when:!(variant === "default") && orientation === "horizontal"`
		}
		item := func(on bool, l string) string {
			return strings.Replace(toggleHTML(v, ToggleMD, 0, on, l, IconNone, ""), "data-g ", "", 1)
		}
		html := `<div data-g data-slot="toggle-group" data-variant="` + toggleVariantNames[v] + `" class="{{part:toggle-group/toggle-group` + cfg + `}}">` + item(true, "Left") + item(false, "Center") + item(false, "Right") + `</div>`
		specs = append(specs, specimen{id: "toggle-group/" + toggleVariantNames[v], w: 220, h: 50, label: "Left", html: html,
			build: func(c *ui.Context) ui.Element {
				a, b, cc := true, false, false
				return ToggleGroup(c, []ToggleGroupItem{{On: &a, Label: "Left"}, {On: &b, Label: "Center"}, {On: &cc, Label: "Right"}}, ToggleGroupOptions{Variant: v})
			}})
	}
	segSize := map[SegmentedSize][2]string{SegmentedMD: {"default", "h-8.5 px-[calc(--spacing(2.5)-1px)] sm:h-7.5"}, SegmentedSM: {"sm", "h-7.5 px-[calc(--spacing(2)-1px)] sm:h-6.5"}, SegmentedLG: {"lg", "h-9.5 px-[calc(--spacing(3)-1px)] sm:h-8.5"}}
	for _, size := range []SegmentedSize{SegmentedMD, SegmentedSM, SegmentedLG} {
		for _, st := range []State{0, StateHover, StateFocusVisible} {
			size, st := size, st
			var b strings.Builder
			b.WriteString(`<div data-g data-slot="segmented-control" class="{{part:segmented-control/segmented-control}}">`)
			for i, l := range []string{"Day", "Week", "Month"} {
				attrs := ""
				if i == 0 {
					attrs = ` data-checked="" aria-checked="true"`
				}
				if i == 1 {
					attrs += force(st)
				}
				fmt.Fprintf(&b, `<button type="button" data-slot="segmented-control-item" class="{{part:segmented-control/segmented-control-item|%s}}"%s>%s</button>`, segSize[size][1], attrs, l)
			}
			b.WriteString(`</div>`)
			specs = append(specs, specimen{id: fmt.Sprintf("segmented/%s/%s", segSize[size][0], stateName(st)), w: 220, h: 54, label: "Day", html: b.String(),
				build: func(c *ui.Context) ui.Element {
					sel := 0
					var e ui.Element
					saved := forced
					forced = 0
					e = segmentedForced(c, &sel, size, st)
					forced = saved
					return e
				}})
		}
	}
	return sheet{comp: "toggle", specs: specs}
})

// segmentedForced renders a segmented control whose second item is in
// state st, as the reference forces it on that item only.
func segmentedForced(c *ui.Context, sel *int, size SegmentedSize, st State) ui.Element {
	segForce = st
	defer func() { segForce = 0 }()
	return SegmentedControl(c, sel, []string{"Day", "Week", "Month"}, SegmentedOptions{Size: size})
}

var _ = registerSheet(func() sheet {
	var specs []specimen
	for _, st := range []State{0, StateFocusVisible, StatePressed, StateDisabled} {
		st := st
		thumbExtra := force(st &^ StatePressed)
		if st.Has(StatePressed) {
			thumbExtra = ` data-dragging=""`
		}
		if st.Has(StateFocusVisible) {
			thumbExtra = ``
		}
		inner := `<input type="range" style="position:absolute;opacity:0;width:1px;height:1px"` + map[bool]string{true: ` data-force="focus,focus-visible"`}[st.Has(StateFocusVisible)] + `>`
		dis := ""
		if st.Has(StateDisabled) {
			dis = ` data-disabled=""`
		}
		html := `<div data-g style="display:flex;align-items:center;height:16px;width:100%"><div data-orientation="horizontal" class="{{part:slider/Slider}}" style="width:100%">` +
			`<div data-slot="slider-control" data-orientation="horizontal" class="{{part:slider/slider-control}}"` + dis + `>` +
			`<div data-slot="slider-track" data-orientation="horizontal" class="{{part:slider/slider-track}}">` +
			`<div data-slot="slider-indicator" data-orientation="horizontal" class="{{part:slider/slider-indicator}}" style="position:absolute;left:0;top:0;height:100%;width:calc(8px + (100% - 16px) * 0.4 - 2px)"></div>` +
			`<div data-slot="slider-thumb" class="{{part:slider/slider-thumb}}"` + thumbExtra + ` style="position:absolute;left:calc(8px + (100% - 16px) * 0.4);top:50%;translate:-50% -50%">` + inner + `</div>` +
			`</div></div></div></div>`
		specs = append(specs, specimen{id: "slider/" + stateName(st), w: 240, h: 40, state: st, html: html,
			build: func(c *ui.Context) ui.Element {
				v := 40.0
				return Slider(c, &v, 0, 100, SliderOptions{})
			}})
	}
	for _, size := range []InputSize{InputMD, InputSM, InputLG} {
		for _, st := range []State{0, StateFocusVisible, StateDisabled} {
			size, st := size, st
			ds := map[InputSize]string{InputMD: "default", InputSM: "sm", InputLG: "lg"}[size]
			btn := func(slot string, ic Icon) string {
				return `<button type="button" data-slot="number-field-` + slot + `" class="{{part:number-field/number-field-` + slot + `}}">` + strings.Replace(iconHTML(ic), "<svg", `<svg class="size-4"`, 1) + `</button>`
			}
			dis := ""
			if st.Has(StateDisabled) {
				dis = ` data-disabled=""`
			}
			html := `<div data-slot="number-field" data-size="` + ds + `" class="{{part:number-field/number-field}}" style="width:160px"><div data-g data-slot="number-field-group" class="{{part:number-field/number-field-group}}"` + dis + map[bool]string{true: ` data-force="focus-within"`}[st.Has(StateFocusVisible)] + `>` +
				btn("decrement", IconMinus) + `<input data-slot="number-field-input" class="{{part:number-field/number-field-input}}" value="42"` + map[bool]string{true: ` data-force="focus,focus-visible"`}[st.Has(StateFocusVisible)] + `>` + btn("increment", IconPlus) + `</div></div>`
			specs = append(specs, specimen{id: fmt.Sprintf("number-field/%s/%s", ds, stateName(st)), w: 180, h: 54, state: st, label: "42", html: html,
				build: func(c *ui.Context) ui.Element {
					v := 42.0
					return NumberField(c, &v, NumberFieldOptions{Size: size, Disabled: st.Has(StateDisabled)}).Width(160)
				}})
		}
	}
	for _, kind := range []string{"progress", "meter"} {
		kind := kind
		var html string
		if kind == "progress" {
			html = `<div data-g data-slot="progress" class="{{part:progress/progress}}" style="width:240px"><div class="flex items-center justify-between gap-2"><span data-slot="progress-label" class="{{part:progress/progress-label}}">Uploading</span><span data-slot="progress-value" class="{{part:progress/progress-value}}">60%</span></div><div data-slot="progress-track" class="{{part:progress/progress-track}}"><div data-slot="progress-indicator" class="{{part:progress/progress-indicator}}" style="height:100%;width:60%"></div></div></div>`
		} else {
			html = `<div data-g class="{{part:meter/Meter}}" style="width:240px"><div class="flex items-center justify-between gap-2"><span data-slot="meter-label" class="{{part:meter/meter-label}}">Storage</span><span data-slot="meter-value" class="{{part:meter/meter-value}}">60%</span></div><div data-slot="meter-track" class="{{part:meter/meter-track}}"><div data-slot="meter-indicator" class="{{part:meter/meter-indicator}}" style="height:100%;width:60%"></div></div></div>`
		}
		specs = append(specs, specimen{id: kind + "/labelled", w: 260, h: 60, label: map[string]string{"progress": "Uploading", "meter": "Storage"}[kind], html: html,
			build: func(c *ui.Context) ui.Element {
				o := ProgressOptions{Label: map[string]string{"progress": "Uploading", "meter": "Storage"}[kind], Value: "60%"}
				if kind == "progress" {
					return Progress(c, 0.6, o).Width(240)
				}
				return Meter(c, 0.6, o).Width(240)
			}})
	}
	return sheet{comp: "range", specs: specs}
})
