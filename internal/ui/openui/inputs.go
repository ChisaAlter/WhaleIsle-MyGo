package openui

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/errcopy"
	"whaleisle/internal/openui/card"
	"whaleisle/internal/openui/lang"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

func init() {
	Register("Input", inputView)
	Register("TextArea", textAreaView)
	Register("Select", selectView)
	Register("SegmentedControl", segmentedView)
	Register("Slider", sliderView)
	Register("Checkbox", checkboxView)
	Register("RadioGroup", radioView)
	Register("DatePicker", dateView)
	Register("Timer", timerView)
	Register("Button", buttonView)
	Register("IconButton", iconButtonView)
	Register("Pressable", pressableView)
}

// field is an input's two-way value: its $binding, or its form field when
// the program passed a plain value (Whale viewer semantics: the field is
// named by the label, else the value prop).
type field struct {
	rt       Runtime
	el       *lang.Element
	name     string
	bind     *lang.Binding
	fallback lang.Value
}

func (r *Render) field(el *lang.Element) field {
	name, prop := card.FieldName(el)
	f := field{rt: r.opts.Runtime, el: el, name: name}
	switch v := el.Prop(prop).(type) {
	case *lang.Binding:
		f.bind = v
	default:
		f.fallback = v
	}
	return f
}

func (f field) get() lang.Value {
	if f.bind != nil {
		return f.bind.Value
	}
	if v := f.rt.FieldValue(f.el.FormName, f.name); v != nil {
		return v
	}
	return f.fallback
}

func (f field) set(v lang.Value) {
	if f.bind != nil {
		f.rt.SetBinding(f.bind, v)
		return
	}
	f.rt.SetField(f.el.FormName, f.el.Type, f.name, v)
}

// inputMemo is a control's local value between frames. synced is the
// state value it was last taken from (or written as), so a value written
// elsewhere replaces it and the user's own typing is never reformatted.
type inputMemo[T comparable] struct {
	local, shown T
	synced       lang.Value
	init         bool
}

// sync adopts the state value when it changed outside this control.
func (m *inputMemo[T]) sync(cur lang.Value, show func(lang.Value) T) {
	if !m.init || !lang.SameValue(cur, m.synced) {
		m.local, m.shown, m.synced, m.init = show(cur), show(cur), cur, true
	}
}

// commit writes the local value when the user changed it.
func (m *inputMemo[T]) commit(f field, parse func(T) lang.Value) {
	if m.local == m.shown {
		return
	}
	m.shown = m.local
	v := parse(m.local)
	m.synced = v
	f.set(v)
}

func display(v lang.Value) string {
	if v == nil {
		return ""
	}
	return lang.ToString(v)
}

func labeled(r *Render, el *lang.Element, control func()) ui.Element {
	label := str(el, "label")
	if label == "" {
		return ui.Column(r.C).FillWidth().MinWidth(0).Children(control)
	}
	return coss.Field(r.C, coss.FieldOptions{Label: label, Disabled: r.Streaming()}, control)
}

func inputView(r *Render, el *lang.Element) ui.Element {
	f := r.field(el)
	numeric := str(el, "type") == "number"
	box := ui.Column(r.C).FillWidth().MinWidth(0)
	m := ui.Local(box, "openui.input", func() inputMemo[string] { return inputMemo[string]{} })
	m.sync(f.get(), display)
	return box.Children(func() {
		labeled(r, el, func() {
			in := coss.Input(r.C, &m.local, coss.InputOptions{Placeholder: str(el, "placeholder"), Disabled: r.Streaming(), Label: str(el, "label")})
			in.Changed()
			m.commit(f, func(s string) lang.Value {
				if !numeric {
					return s
				}
				s = strings.TrimSpace(s)
				if s == "" {
					return nil
				}
				if n, err := strconv.ParseFloat(s, 64); err == nil && !math.IsInf(n, 0) {
					return n
				}
				return f.get()
			})
		})
	})
}

func textAreaView(r *Render, el *lang.Element) ui.Element {
	f := r.field(el)
	box := ui.Column(r.C).FillWidth().MinWidth(0)
	m := ui.Local(box, "openui.textarea", func() inputMemo[string] { return inputMemo[string]{} })
	m.sync(f.get(), display)
	return box.Children(func() {
		labeled(r, el, func() {
			in := coss.Textarea(r.C, &m.local, coss.TextareaOptions{Placeholder: str(el, "placeholder"), Disabled: r.Streaming(), Label: str(el, "label"), MaxLines: int(max(4, min(numOr(el, "rows", 12), 40)))})
			in.Changed()
			m.commit(f, func(s string) lang.Value { return s })
		})
	})
}

func selectView(r *Render, el *lang.Element) ui.Element {
	f := r.field(el)
	opts := options(el.Prop("options"))
	box := ui.Column(r.C).FillWidth().MinWidth(0)
	m := ui.Local(box, "openui.select", func() inputMemo[string] { return inputMemo[string]{} })
	m.sync(f.get(), display)
	items := make([]coss.SelectOption, len(opts))
	for i, o := range opts {
		items[i] = coss.SelectOption{Value: o.value, Label: o.label}
	}
	return box.Children(func() {
		labeled(r, el, func() {
			tr := coss.Select(r.C, &m.local, items, coss.SelectOptions{Placeholder: str(el, "placeholder"), Disabled: r.Streaming(), Label: str(el, "label")})
			tr.Changed()
			m.commit(f, func(s string) lang.Value { return s })
		})
	})
}

func indexOf(opts []option, v string) int {
	for i, o := range opts {
		if o.value == v {
			return i
		}
	}
	return -1
}

func segmentedView(r *Render, el *lang.Element) ui.Element {
	f := r.field(el)
	opts := options(el.Prop("options"))
	box := ui.Column(r.C).MinWidth(0)
	if boolean(el, "block") {
		box.FillWidth()
	} else {
		r.Inline(box)
	}
	m := ui.Local(box, "openui.segmented", func() inputMemo[int] { return inputMemo[int]{} })
	m.sync(f.get(), func(v lang.Value) int { return indexOf(opts, display(v)) })
	labels := make([]string, len(opts))
	for i, o := range opts {
		labels[i] = o.label
	}
	return box.Children(func() {
		e := coss.SegmentedControl(r.C, &m.local, labels, coss.SegmentedOptions{Disabled: r.Streaming(), Label: str(el, "label")})
		if boolean(el, "block") {
			e.FillWidth()
		}
		e.Changed()
		m.commit(f, func(i int) lang.Value {
			if i >= 0 && i < len(opts) {
				return opts[i].value
			}
			return nil
		})
	})
}

func sliderView(r *Render, el *lang.Element) ui.Element {
	f := r.field(el)
	lo, hi := numOr(el, "min", 0), numOr(el, "max", 100)
	if hi <= lo {
		hi = lo + 1
	}
	step := numOr(el, "step", 0)
	t := r.T
	box := ui.Column(r.C).FillWidth().MinWidth(0).Gap(design.Sp(2))
	m := ui.Local(box, "openui.slider", func() inputMemo[float64] { return inputMemo[float64]{} })
	m.sync(f.get(), func(v lang.Value) float64 {
		n, ok := v.(float64)
		if !ok {
			n = lo
		}
		return max(lo, min(n, hi))
	})
	return box.Children(func() {
		label, unit := str(el, "label"), str(el, "unit")
		ui.Row(r.C).FillWidth().Justify(ui.SpaceBetween).Children(func() {
			r.text(label, design.TextSM, design.WeightMedium, t.Foreground)
			r.text(lang.NumberToString(roundStep(m.local, step))+unit, design.TextSM, design.WeightNormal, t.MutedForeground).FontFeatures("tnum")
		})
		e := coss.Slider(r.C, &m.local, lo, hi, coss.SliderOptions{Step: step, Disabled: r.Streaming(), Label: label})
		e.Changed()
		m.commit(f, func(v float64) lang.Value { return roundStep(v, step) })
		if marks := list(el, "marks"); len(marks) > 0 {
			ui.Row(r.C).FillWidth().Justify(ui.SpaceBetween).Children(func() {
				for i, mk := range marks {
					if o, ok := mk.(*lang.Object); ok {
						r.C.Key(i)
						r.text(lang.ToString(o.Value("label")), design.TextXS, design.WeightNormal, t.MutedForeground)
					}
				}
			})
		}
	})
}

func roundStep(v, step float64) float64 {
	if step > 0 {
		v = math.Round(v/step) * step
		// Trim binary noise (0.1 steps).
		v = math.Round(v*1e9) / 1e9
	}
	return v
}

func checkboxView(r *Render, el *lang.Element) ui.Element {
	f := r.field(el)
	box := ui.Column(r.C).MinWidth(0)
	m := ui.Local(box, "openui.check", func() inputMemo[bool] { return inputMemo[bool]{} })
	m.sync(f.get(), func(v lang.Value) bool { return lang.Truthy(v) })
	return box.Children(func() {
		e := coss.Checkbox(r.C, &m.local, str(el, "label"), coss.CheckOptions{Disabled: r.Streaming()})
		if boolean(el, "lineThrough") && m.local {
			e.Opacity(design.OpacityPlaceholder)
		}
		e.Changed()
		m.commit(f, func(b bool) lang.Value { return b })
	})
}

func radioView(r *Render, el *lang.Element) ui.Element {
	f := r.field(el)
	opts := options(el.Prop("options"))
	t := r.T
	box := ui.Column(r.C).FillWidth().MinWidth(0).Gap(design.Sp(2))
	m := ui.Local(box, "openui.radio", func() inputMemo[string] { return inputMemo[string]{} })
	m.sync(f.get(), display)
	label := str(el, "label")
	return box.Children(func() {
		switch str(el, "variant") {
		case "pills":
			if label != "" {
				coss.Label(r.C, label)
			}
			ui.Row(r.C).Wrap().Gap(design.Sp(2)).Role(ui.RoleRadioGroup).Label(label).Children(func() {
				for i, o := range opts {
					variant := coss.ButtonOutline
					if o.value == m.local {
						variant = coss.ButtonDefault
					}
					if coss.Button(r.C, o.label, coss.ButtonOptions{Variant: variant, Size: coss.ButtonSM, Disabled: r.Streaming(), Pressed: o.value == m.local, Key: i}).Clicked() {
						m.local = o.value
					}
				}
			})
		case "cards":
			if label != "" {
				coss.Label(r.C, label)
			}
			ui.Column(r.C).FillWidth().Gap(design.Sp(2)).Role(ui.RoleRadioGroup).Label(label).Children(func() {
				for i, o := range opts {
					r.C.Key(i)
					on := o.value == m.local
					b := ui.ButtonBase(r.C).FillWidth().Justify(ui.Start).AlignItems(ui.Start).Gap(design.Sp(3)).Padding(design.Sp(3), design.Sp(3.5)).
						Radius(design.RadiusLG).Border(design.BorderWidth, t.Border).Cursor(ui.CursorPointer).Role(ui.RoleRadio).Checked(on)
					if on {
						b.Border(design.BorderWidth, t.Primary).Background(t.Primary.Alpha(0.06))
					} else if b.Hovered() {
						b.Background(t.Accent)
					}
					if r.Streaming() {
						b.Disabled(true)
					}
					b.Children(func() {
						ring := ui.Box(r.C).Size(design.Sp(4), design.Sp(4)).Radius(design.RadiusFull).Border(design.BorderWidth, t.Input).Center().Shrink(0).Margin(design.Sp(0.5), 0, 0, 0)
						if on {
							ring.Background(t.Primary).Border(design.BorderWidth, t.Primary)
						}
						ring.Children(func() {
							if on {
								ui.Box(r.C).Size(design.Sp(1.5), design.Sp(1.5)).Radius(design.RadiusFull).Background(t.PrimaryForeground)
							}
						})
						ui.Column(r.C).Grow(1).MinWidth(0).Gap(design.Sp(0.5)).Children(func() {
							r.paragraph(o.label, design.TextSM, design.WeightMedium, t.Foreground)
							if o.description != "" {
								r.paragraph(o.description, design.TextXS, design.WeightNormal, t.MutedForeground)
							}
						})
					})
					if b.Clicked() {
						m.local = o.value
					}
				}
			})
		default:
			if label != "" {
				coss.Label(r.C, label)
			}
			coss.RadioGroup(r.C, label, func() {
				for i, o := range opts {
					e := coss.Radio(r.C, &m.local, o.value, o.label, coss.CheckOptions{Disabled: r.Streaming(), Key: i})
					e.Changed()
					if o.description != "" {
						r.paragraph(o.description, design.TextXS, design.WeightNormal, t.MutedForeground).Margin(0, 0, 0, design.Sp(6))
					}
				}
			})
		}
		m.commit(f, func(s string) lang.Value { return s })
	})
}

const dateLayout = "2006-01-02"

type dateMemo struct {
	inputMemo[string]
	open bool
	day  time.Time
}

func dateView(r *Render, el *lang.Element) ui.Element {
	f := r.field(el)
	box := ui.Column(r.C).FillWidth().MinWidth(0)
	m := ui.Local(box, "openui.date", func() dateMemo { return dateMemo{} })
	m.sync(f.get(), display)
	lo, _ := time.ParseInLocation(dateLayout, str(el, "min"), time.Local)
	hi, _ := time.ParseInLocation(dateLayout, str(el, "max"), time.Local)
	return box.Children(func() {
		labeled(r, el, func() {
			ui.Row(r.C).FillWidth().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
				in := coss.Input(r.C, &m.local, coss.InputOptions{Placeholder: "YYYY-MM-DD", Disabled: r.Streaming(), Label: str(el, "label")})
				in.Changed()
				anchor := coss.Button(r.C, "", coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonIcon, Label: i18n.Pick("选择日期", "Choose date"), Disabled: r.Streaming(),
					Content: func() { r.icon("calendar", design.IconSM, r.T.Foreground) }})
				if anchor.Clicked() {
					m.open = !m.open
					if d, err := time.ParseInLocation(dateLayout, m.local, time.Local); err == nil {
						m.day = d
					} else if m.day.IsZero() {
						m.day = r.C.Now()
					}
				}
				coss.Popover(r.C, anchor, &m.open, coss.PopoverOptions{Align: coss.AlignEnd}, func() {
					before := m.day
					cal := ui.Calendar(r.C, &m.day)
					cal.Changed()
					if !m.day.Equal(before) {
						d := m.day
						if !lo.IsZero() && d.Before(lo) {
							d = lo
						}
						if !hi.IsZero() && d.After(hi) {
							d = hi
						}
						m.local = d.Format(dateLayout)
						m.open = false
					}
				})
			})
		})
		m.commit(f, func(s string) lang.Value {
			s = strings.TrimSpace(s)
			if s == "" {
				return ""
			}
			if _, err := time.ParseInLocation(dateLayout, s, time.Local); err != nil {
				return f.get()
			}
			return s
		})
	})
}

// timerClock is the viewer's persisted Timer clock: baseline seconds, the
// running deadline (epoch ms) and the last shown seconds.
type timerClock struct {
	baseline, last float64
	deadline       float64 // 0 when paused
}

func readClock(v lang.Value, secs float64) timerClock {
	o, ok := v.(*lang.Object)
	if !ok {
		return timerClock{baseline: secs, last: secs}
	}
	c := timerClock{baseline: lang.ToNumber(o.Value("baseline")), last: lang.ToNumber(o.Value("last"))}
	if d, ok := o.Value("deadline").(float64); ok {
		c.deadline = d
	}
	return c
}

func (c timerClock) value() lang.Value {
	var d lang.Value
	if c.deadline > 0 {
		d = c.deadline
	}
	return lang.ObjectFrom("baseline", c.baseline, "deadline", d, "last", c.last)
}

func (c timerClock) remaining(nowMs float64) float64 {
	return max(0, math.Ceil((c.deadline-nowMs)/1000))
}

func clockText(secs float64) string {
	s := int(secs)
	h, m, sec := s/3600, s/60%60, s%60
	two := func(n int) string {
		if n < 10 {
			return "0" + strconv.Itoa(n)
		}
		return strconv.Itoa(n)
	}
	if h > 0 {
		return strconv.Itoa(h) + ":" + two(m) + ":" + two(sec)
	}
	return two(m) + ":" + two(sec)
}

func timerView(r *Render, el *lang.Element) ui.Element {
	f := r.field(el)
	rt := r.opts.Runtime
	id := el.Identity()
	t := r.T
	secs := max(0, math.Ceil(lang.ToNumber(f.get())))
	if math.IsNaN(secs) {
		secs = 0
	}
	clock := readClock(rt.ComponentState(id, "clock"), secs)
	if secs != clock.last {
		clock = timerClock{baseline: secs, last: secs}
		rt.SetComponentState(id, "clock", clock.value())
	}
	now := float64(r.C.Now().UnixMilli())
	running := clock.deadline > 0
	if running && !r.Streaming() {
		p := clock.remaining(now)
		if p != clock.last || p == 0 {
			clock.last = p
			if p == 0 {
				clock.deadline = 0
			}
			rt.SetComponentState(id, "clock", clock.value())
			f.set(p)
			secs = p
		}
		if clock.deadline > 0 {
			wait := math.Mod(clock.deadline-now, 1000)
			if wait <= 0 {
				wait = 1000
			}
			r.C.After(time.Duration(wait+1) * time.Millisecond)
		}
	}
	running = clock.deadline > 0
	label := str(el, "label")
	if label == "" {
		label = i18n.Pick("计时器", "Timer")
	}
	return r.Inline(ui.Column(r.C)).Gap(design.Sp(2)).Role(ui.RoleStatus).Label(label).Children(func() {
		r.text(label, design.TextSM, design.WeightMedium, t.MutedForeground)
		r.text(clockText(secs), design.TextXL3, design.WeightSemibold, t.Foreground).FontFeatures("tnum")
		ui.Row(r.C).Gap(design.Sp(2)).Children(func() {
			icon, text := "play", i18n.Pick("开始", "Start")
			if running {
				icon, text = "pause", i18n.Pick("暂停", "Pause")
			}
			if coss.Button(r.C, text, coss.ButtonOptions{Size: coss.ButtonSM, Disabled: r.Streaming() || secs == 0, Content: func() {
				r.icon(icon, design.IconSM, t.PrimaryForeground)
				r.text(text, design.TextSM, design.WeightMedium, t.PrimaryForeground)
			}}).Clicked() {
				if running {
					p := clock.remaining(now)
					clock.deadline, clock.last = 0, p
					rt.SetComponentState(id, "clock", clock.value())
					f.set(p)
				} else {
					clock.last, clock.deadline = secs, now+secs*1000
					rt.SetComponentState(id, "clock", clock.value())
				}
			}
			reset := i18n.Pick("重置", "Reset")
			if coss.Button(r.C, reset, coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: r.Streaming(), Content: func() {
				r.icon("rotate-ccw", design.IconSM, t.Foreground)
				r.text(reset, design.TextSM, design.WeightMedium, t.Foreground)
			}}).Clicked() {
				clock = timerClock{baseline: clock.baseline, last: clock.baseline}
				rt.SetComponentState(id, "clock", clock.value())
				f.set(clock.baseline)
			}
		})
	})
}

// ── actions ──

// act dispatches an element's action for a real click on this frame.
func (r *Render) act(el *lang.Element, label string) {
	r.opts.Runtime.Dispatch(r.Frame.Revision, el.Identity(), label, el.FormName, el.Prop("action"))
}

// actionError shows why an element's last action stopped.
func (r *Render) actionError(el *lang.Element) {
	if err := r.opts.Runtime.Failure(el.Identity()); err != nil {
		coss.FieldError(r.C, errcopy.Sentence(err))
	}
}

func buttonView(r *Render, el *lang.Element) ui.Element {
	label := str(el, "label")
	o := coss.ButtonOptions{Disabled: boolean(el, "disabled") || r.Streaming(), Loading: r.opts.Runtime.Busy(el.Identity())}
	solid := true
	switch str(el, "variant") {
	case "outline":
		o.Variant, solid = coss.ButtonOutline, false
	case "soft":
		o.Variant, solid = coss.ButtonSecondary, false
	case "ghost":
		o.Variant, solid = coss.ButtonGhost, false
	}
	if str(el, "color") == "danger" {
		if solid {
			o.Variant = coss.ButtonDestructive
		} else {
			o.Variant = coss.ButtonDestructiveOutline
		}
	}
	switch str(el, "size") {
	case "sm":
		o.Size = coss.ButtonSM
	case "lg":
		o.Size = coss.ButtonLG
	}
	if name := str(el, "icon"); name != "" {
		fg := r.T.Foreground
		if solid {
			fg = r.T.PrimaryForeground
		}
		o.Content = func() {
			r.icon(name, design.IconSM, fg)
			r.text(label, design.TextSM, design.WeightMedium, fg)
		}
	}
	box := ui.Column(r.C).Gap(design.Sp(1)).MinWidth(0)
	if boolean(el, "block") {
		box.FillWidth()
	} else {
		r.Inline(box)
	}
	return box.Children(func() {
		b := coss.Button(r.C, label, o)
		if boolean(el, "block") {
			b.FillWidth()
		}
		if b.Clicked() {
			r.act(el, label)
		}
		r.actionError(el)
	})
}

func iconButtonView(r *Render, el *lang.Element) ui.Element {
	name := str(el, "icon")
	label := str(el, "label")
	if label == "" {
		label = name
	}
	o := coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonIcon, Label: label, Disabled: r.Streaming(), Loading: r.opts.Runtime.Busy(el.Identity())}
	fg := r.T.Foreground
	switch str(el, "variant") {
	case "ghost":
		o.Variant = coss.ButtonGhost
	case "solid":
		o.Variant, fg = coss.ButtonDefault, r.T.PrimaryForeground
	}
	o.Content = func() { r.icon(name, design.IconSM, fg) }
	return r.Inline(ui.Column(r.C)).Gap(design.Sp(1)).Children(func() {
		b := coss.Button(r.C, "", o).Radius(design.RadiusFull).Tooltip(str(el, "label"))
		if b.Clicked() {
			r.act(el, label)
		}
		r.actionError(el)
	})
}

func pressableView(r *Render, el *lang.Element) ui.Element {
	t := r.T
	b := ui.ButtonBase(r.C).FillWidth().MinWidth(0).Justify(ui.Start).AlignItems(ui.Stretch).Cursor(ui.CursorPointer).
		Radius(radius(el.Prop("radius"), design.RadiusLG)).Padding(spacing(numOr(el, "padding", 3)))
	if bg, ok := r.softColor(str(el, "background")); ok {
		b.Background(bg)
	}
	selected := boolean(el, "selected")
	if selected {
		b.Border(design.BorderWidth, t.Primary).Background(t.Primary.Alpha(0.06)).Selected(true)
	} else {
		b.Border(design.BorderWidth, t.Border)
		if b.Hovered() {
			b.Background(t.Accent)
		}
	}
	busy := r.opts.Runtime.Busy(el.Identity())
	if r.Streaming() || busy {
		b.Disabled(true)
	}
	b.Children(func() {
		ui.Column(r.C).FillWidth().MinWidth(0).Gap(design.Sp(2)).Children(func() { r.Children(el, "children") })
	})
	if b.Clicked() {
		r.act(el, "")
	}
	if err := r.opts.Runtime.Failure(el.Identity()); err != nil {
		return ui.Column(r.C).FillWidth().Gap(design.Sp(1)).Children(func() { coss.FieldError(r.C, errcopy.Sentence(err)) })
	}
	return b
}
