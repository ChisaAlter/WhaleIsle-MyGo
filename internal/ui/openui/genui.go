package openui

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/errcopy"
	"whaleisle/internal/openui/lang"
	"whaleisle/internal/openui/lang/whaleui"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

// Declarative genui nodes: the specialized components (Tabs, Steps, …)
// take a genui options object, and Genui(spec) renders a whole spec (the
// dsh-ui language and the conversation panel). The node types drawn here
// are the static and locally interactive ones; host-action inputs (input,
// select, checkbox, radio, slider, switch, submit, button, copy) and the
// heavy engines (chart, echart, plot, mermaid, scene3d, diagram, svg,
// diff, audio, video) render the placeholder until they are native.

func init() {
	Register("Genui", func(r *Render, el *lang.Element) ui.Element {
		if bad := invalidElement(r, el); bad.Valid() {
			return bad
		}
		spec, _ := el.Prop("spec").(*lang.Object)
		g := &genui{r: r, el: el}
		return ui.Column(r.C).FillWidth().MinWidth(0).Gap(spacing(lang.ToNumber(spec.Value("gap"))/4 + 3)).Children(func() {
			if title, ok := spec.Value("title").(string); ok && title != "" {
				r.paragraph(title, design.TextLG, design.WeightSemibold, r.T.Foreground).Role(ui.RoleHeading)
			}
			g.nodes(spec.Value("items"), "items")
		})
	})
	for comp, typ := range map[string]string{"Tabs": "tabs", "Accordion": "accordion", "Steps": "steps", "Timeline": "timeline",
		"KeyValue": "keyvalue", "Json": "json", "FileTree": "file-tree", "Quiz": "quiz", "Blocks": "blocks"} {
		typ := typ
		Register(comp, func(r *Render, el *lang.Element) ui.Element {
			if bad := invalidElement(r, el); bad.Valid() {
				return bad
			}
			opts, _ := el.Prop("options").(*lang.Object)
			node := opts.Clone()
			node.Set("type", typ)
			g := &genui{r: r, el: el}
			return ui.Column(r.C).FillWidth().MinWidth(0).Children(func() { g.node(node, typ) })
		})
	}
}

// invalidElement shows the classified sentence for an element whose
// options failed validation.
func invalidElement(r *Render, el *lang.Element) ui.Element {
	if len(el.Issues) == 0 {
		return ui.Element{}
	}
	return coss.Alert(r.C, "", errcopy.Sentence(diagnosticError(nil)), coss.AlertOptions{Variant: coss.AlertWarning})
}

type genui struct {
	r  *Render
	el *lang.Element
}

func gstr(o *lang.Object, k string) string {
	switch v := o.Value(k).(type) {
	case string:
		return v
	case float64, bool:
		return lang.ToString(v)
	}
	return ""
}

func garr(o *lang.Object, k string) []lang.Value {
	a, _ := o.Value(k).([]lang.Value)
	return a
}

// normalize applies the schema's field aliases (text → content, …).
func normalize(typ string, o *lang.Object) *lang.Object {
	g, err := whaleui.GenuiValidator()
	if err != nil || g == nil {
		return o
	}
	s := g.Schemas[typ]
	if s == nil || len(s.Aliases) == 0 {
		return o
	}
	var out *lang.Object
	for alias, field := range s.Aliases {
		if v, ok := o.Get(alias); ok && !o.Has(field) {
			if out == nil {
				out = o.Clone()
			}
			out.Set(field, v)
		}
	}
	if out == nil {
		return o
	}
	return out
}

func (g *genui) nodes(v lang.Value, path string) {
	for i, it := range asList(v) {
		g.r.C.Key(i)
		p := path + "." + strconv.Itoa(i)
		switch t := it.(type) {
		case *lang.Object:
			g.node(t, p)
		case string:
			g.r.inlineText(t, design.TextSM, design.WeightNormal, g.r.T.Foreground, nil)
		}
	}
}

func asList(v lang.Value) []lang.Value {
	switch t := v.(type) {
	case []lang.Value:
		return t
	case *lang.Object:
		return []lang.Value{t}
	}
	return nil
}

func (g *genui) tone(name string) (ui.Color, bool) {
	if name == "accent" {
		return g.r.T.Primary, true
	}
	return g.r.tone(name)
}

// node draws one genui node; path keys its local state.
func (g *genui) node(o *lang.Object, path string) {
	r, t := g.r, g.r.T
	typ := gstr(o, "type")
	if typ == "" {
		switch {
		case o.Has("content") || o.Has("text"):
			typ = "text"
		case o.Has("pairs"):
			typ = "keyvalue"
		case o.Has("steps"):
			typ = "steps"
		case o.Has("tabs"):
			typ = "tabs"
		case o.Has("columns") && o.Has("rows"):
			typ = "table"
		}
	}
	o = normalize(typ, o)
	r.depth++
	defer func() { r.depth-- }()
	if r.depth >= maxDepth {
		return
	}
	switch typ {
	case "text":
		ts, w, col := design.TextSM, design.WeightNormal, t.Foreground
		switch gstr(o, "size") {
		case "display":
			ts, w = design.TextXL3, design.WeightBold
		case "h1":
			ts, w = design.TextXL2, design.WeightBold
		case "h2":
			ts, w = design.TextXL, design.WeightSemibold
		case "h3":
			ts, w = design.TextLG, design.WeightSemibold
		case "muted":
			col = t.MutedForeground
		case "caption":
			ts, col = design.TextXS, t.MutedForeground
		}
		if c, ok := g.tone(gstr(o, "tone")); ok {
			col = c
		}
		e := r.inlineText(gstr(o, "content"), ts, w, col, nil)
		if o.Value("mono") == true {
			e.Font(r.md.FontFamily).FontFeatures("tnum")
		}
		switch {
		case o.Value("center") == true || gstr(o, "align") == "center":
			e.TextAlign(ui.Center).FillWidth()
		case gstr(o, "align") == "right":
			e.TextAlign(ui.End).FillWidth()
		}
	case "badge":
		color := gstr(o, "tone")
		switch color {
		case "warn":
			color = "warning"
		case "accent":
			color = "info"
		}
		el := &lang.Element{Type: "Badge", Props: lang.ObjectFrom("text", gstr(o, "label"), "color", color, "icon", gstr(o, "icon"))}
		badgeView(r, el)
	case "callout":
		v := coss.AlertInfo
		switch gstr(o, "tone") {
		case "success":
			v = coss.AlertSuccess
		case "warning", "warn":
			v = coss.AlertWarning
		case "error", "danger":
			v = coss.AlertError
		}
		coss.Alert(r.C, gstr(o, "title"), gstr(o, "content"), coss.AlertOptions{Variant: v})
	case "card":
		card := ui.Column(r.C).FillWidth().MinWidth(0).Gap(design.Sp(3)).Padding(design.Sp(5)).Radius(design.RadiusXL)
		if o.Value("flat") == true {
			card.Background(t.Muted)
		} else {
			card.Background(t.Card).Border(design.BorderWidth, t.Border)
		}
		if c, ok := g.tone(gstr(o, "tone")); ok {
			card.BorderWidth(0, 0, 0, 3).BorderColor(c)
		}
		card.Children(func() {
			if e := gstr(o, "eyebrow"); e != "" {
				r.text(strings.ToUpper(e), design.TextXS, design.WeightMedium, t.MutedForeground)
			}
			if s := gstr(o, "title"); s != "" {
				r.paragraph(s, design.TextBase, design.WeightSemibold, t.Foreground).Role(ui.RoleHeading)
			}
			g.nodes(o.Value("items"), path+".items")
		})
	case "grid":
		cols := int(lang.ToNumber(o.Value("cols")))
		items := asList(o.Value("items"))
		if cols <= 0 {
			cols = min(max(1, len(items)), 3)
		}
		ui.Grid(r.C).FillWidth().MinWidth(0).Columns(min(cols, 12)).Gap(design.Sp(3)).Children(func() {
			for i, it := range items {
				r.C.Key(i)
				ui.Column(r.C).MinWidth(0).Children(func() { g.nodes([]lang.Value{it}, path+".items."+strconv.Itoa(i)) })
			}
		})
	case "row":
		row := ui.Row(r.C).FillWidth().MinWidth(0).Gap(design.Sp(2)).AlignItems(ui.Center).Justify(align(gstr(o, "justify"), ui.Start))
		if o.Value("wrap") == true {
			row.Wrap()
		}
		row.Children(func() { r.Within(true, func() { g.nodes(o.Value("items"), path+".items") }) })
	case "col":
		ui.Column(r.C).FillWidth().MinWidth(0).Gap(spacing(lang.ToNumber(o.Value("gap"))/4 + 3)).Children(func() { g.nodes(o.Value("items"), path+".items") })
	case "list":
		items := garr(o, "items")
		ui.Column(r.C).FillWidth().Gap(design.Sp(1.5)).Role(ui.RoleList).Children(func() {
			for i, it := range items {
				r.C.Key(i)
				ui.Row(r.C).FillWidth().Gap(design.Sp(2.5)).AlignItems(ui.Start).Role(ui.RoleListItem).Children(func() {
					r.text("•", design.TextSM, design.WeightNormal, t.MutedForeground)
					ui.Column(r.C).Grow(1).MinWidth(0).Children(func() {
						if n, ok := it.(*lang.Object); ok {
							g.node(n, path+".items."+strconv.Itoa(i))
						} else {
							r.inlineText(lang.ToString(it), design.TextSM, design.WeightNormal, t.Foreground, nil)
						}
					})
				})
			}
		})
	case "keyvalue":
		g.keyValue(garr(o, "pairs"))
	case "table":
		var cols []string
		for _, c := range garr(o, "columns") {
			cols = append(cols, lang.ToString(c))
		}
		var rows [][]lang.Value
		for _, row := range garr(o, "rows") {
			switch x := row.(type) {
			case []lang.Value:
				rows = append(rows, x)
			case *lang.Object:
				line := make([]lang.Value, len(cols))
				for i, c := range cols {
					line[i] = x.Value(c)
				}
				rows = append(rows, line)
			}
		}
		r.tableGrid(cols, rows)
	case "progress":
		v := lang.ToNumber(o.Value("value"))
		target := lang.ToNumber(o.Value("target"))
		if !(target > 0) {
			target = 100
			if v <= 1 {
				target = 1
			}
		}
		frac := max(0, min(v/target, 1))
		label := gstr(o, "valueLabel")
		if label == "" {
			label = lang.NumberToString(float64(int(frac*100+0.5))) + "%"
		}
		coss.Progress(r.C, frac, coss.ProgressOptions{Label: gstr(o, "label"), Value: label})
	case "stat", "hero":
		ui.Column(r.C).Gap(design.Sp(1)).MinWidth(0).Children(func() {
			if typ == "hero" {
				r.paragraph(gstr(o, "title"), design.TextXL, design.WeightSemibold, t.Foreground).Role(ui.RoleHeading)
				if s := gstr(o, "subtitle"); s != "" {
					r.paragraph(s, design.TextSM, design.WeightNormal, t.MutedForeground)
				}
			}
			if l := gstr(o, "label"); l != "" {
				r.text(l, design.TextSM, design.WeightNormal, t.MutedForeground)
			}
			if v := gstr(o, "value"); v != "" {
				r.text(v, design.TextXL2, design.WeightBold, t.Foreground).FontFeatures("tnum")
			}
			if d := gstr(o, "delta"); d != "" {
				col := t.Success
				if strings.HasPrefix(strings.TrimSpace(d), "-") || strings.HasPrefix(strings.TrimSpace(d), "−") {
					col = t.Destructive
				}
				r.text(d, design.TextXS, design.WeightMedium, col)
			}
		})
	case "image":
		img := &lang.Element{Type: "Image", Props: lang.ObjectFrom("src", gstr(o, "src"), "alt", gstr(o, "alt"))}
		ui.Column(r.C).FillWidth().Gap(design.Sp(1)).Children(func() {
			imageView(r, img)
			if cap := gstr(o, "caption"); cap != "" {
				r.paragraph(cap, design.TextXS, design.WeightNormal, t.MutedForeground)
			}
		})
	case "link":
		linkView(r, &lang.Element{Type: "Link", Props: lang.ObjectFrom("text", gstr(o, "label"), "url", gstr(o, "href"))})
	case "divider":
		if l := gstr(o, "label"); l != "" {
			ui.Row(r.C).FillWidth().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
				coss.Separator(r.C, coss.Horizontal).Grow(1)
				r.text(l, design.TextXS, design.WeightMedium, t.MutedForeground)
				coss.Separator(r.C, coss.Horizontal).Grow(1)
			})
		} else {
			coss.Separator(r.C, coss.Horizontal)
		}
	case "spacer":
		ui.Box(r.C).Grow(1).Height(design.Sp(2))
	case "code":
		lang_ := gstr(o, "lang")
		if lang_ == "" {
			lang_ = "text"
		}
		r.markdown(fence(lang_, gstr(o, "code")))
	case "icon":
		col := t.Foreground
		if c, ok := g.tone(gstr(o, "tone")); ok {
			col = c
		}
		ui.Row(r.C).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
			r.icon(gstr(o, "name"), design.IconMD, col)
			if l := gstr(o, "label"); l != "" {
				r.text(l, design.TextSM, design.WeightNormal, t.Foreground)
			}
		})
	case "avatar":
		coss.Avatar(r.C, gstr(o, "name"), coss.AvatarOptions{})
	case "steps":
		g.steps(o)
	case "timeline":
		items := garr(o, "items")
		ui.Column(r.C).FillWidth().Role(ui.RoleList).Children(func() {
			for i, it := range items {
				n, _ := it.(*lang.Object)
				r.C.Key(i)
				el := &lang.Element{Type: "ListItem", Props: lang.ObjectFrom("title", gstr(n, "title"))}
				desc := gstr(n, "desc")
				if tm := gstr(n, "time"); tm != "" {
					if desc != "" {
						desc = tm + " · " + desc
					} else {
						desc = tm
					}
				}
				if desc != "" {
					el.Props.Set("description", desc)
				}
				listItem(r, el, "", true, i == len(items)-1)
			}
		})
	case "tabs":
		g.tabs(o, path)
	case "accordion":
		g.accordion(o, path)
	case "json":
		g.json(o.Value("value"), path)
	case "file-tree":
		g.fileTree(garr(o, "items"), path, 0)
	case "quiz":
		g.quiz(o)
	case "blocks":
		g.blocks(o)
	default:
		placeholder(r, &lang.Element{Type: typ, Props: lang.NewObject(0)})
	}
}

func (g *genui) keyValue(pairs []lang.Value) {
	r, t := g.r, g.r.T
	ui.Grid(r.C).FillWidth().MinWidth(0).ColumnTracks(ui.FitContent(), ui.Fr(1)).GapX(design.Sp(4)).GapY(design.Sp(2)).Children(func() {
		for i, p := range pairs {
			o, _ := p.(*lang.Object)
			o = normalize("keyvalue.pairs", o)
			key := gstr(o, "key")
			if key == "" {
				key = gstr(o, "label")
			}
			r.C.Key(2 * i)
			r.text(key, design.TextSM, design.WeightNormal, t.MutedForeground)
			r.C.Key(2*i + 1)
			r.inlineText(gstr(o, "value"), design.TextSM, design.WeightMedium, t.Foreground, nil)
		}
	})
}

func (g *genui) steps(o *lang.Object) {
	r, t := g.r, g.r.T
	items := garr(o, "steps")
	current := -1
	if c, ok := o.Value("current").(float64); ok {
		current = int(c)
	}
	ui.Column(r.C).FillWidth().Role(ui.RoleList).Children(func() {
		for i, it := range items {
			n, _ := it.(*lang.Object)
			r.C.Key(i)
			done, active := current >= 0 && i < current, i == current
			ui.Row(r.C).FillWidth().Gap(design.Sp(3)).AlignItems(ui.Stretch).Role(ui.RoleListItem).Children(func() {
				ui.Column(r.C).AlignItems(ui.Center).Shrink(0).Children(func() {
					dot := ui.Box(r.C).Size(design.Sp(6), design.Sp(6)).Radius(design.RadiusFull).Center().Border(design.BorderWidth, t.Border).Background(t.Background)
					fg := t.MutedForeground
					if done || active {
						dot.Background(t.Primary).Border(design.BorderWidth, t.Primary)
						fg = t.PrimaryForeground
					}
					dot.Children(func() {
						if done {
							r.icon("check", design.IconXS, fg)
						} else {
							r.text(strconv.Itoa(i+1), design.TextXS, design.WeightMedium, fg)
						}
					})
					if i < len(items)-1 {
						ui.Box(r.C).Width(design.BorderWidth * 2).Grow(1).MinHeight(design.Sp(3)).Background(t.Border)
					}
				})
				col := ui.Column(r.C).Grow(1).MinWidth(0).Gap(design.Sp(0.5)).Padding(design.Sp(0.5), 0, design.Sp(4), 0)
				col.Children(func() {
					w := design.WeightMedium
					if active {
						w = design.WeightSemibold
					}
					r.paragraph(gstr(n, "title"), design.TextSM, w, t.Foreground)
					if d := gstr(n, "desc"); d != "" {
						r.inlineText(d, design.TextSM, design.WeightNormal, t.MutedForeground, nil)
					}
				})
			})
		}
	})
}

func (g *genui) tabs(o *lang.Object, path string) {
	r := g.r
	tabs := garr(o, "tabs")
	labels := make([]string, 0, len(tabs))
	for _, tb := range tabs {
		n, _ := tb.(*lang.Object)
		labels = append(labels, gstr(n, "label"))
	}
	box := ui.Column(r.C).FillWidth().MinWidth(0).Gap(design.Sp(3))
	sel := ui.Local(box, "openui.tabs", func() int { return 0 })
	box.Children(func() {
		coss.Tabs(r.C, sel, labels, coss.TabsOptions{Variant: coss.TabsUnderline})
		if *sel >= 0 && *sel < len(tabs) {
			n, _ := tabs[*sel].(*lang.Object)
			items := n.Value("items")
			if items == nil {
				items = n.Value("content")
			}
			r.C.Key(*sel)
			ui.Column(r.C).FillWidth().MinWidth(0).Gap(design.Sp(3)).Children(func() {
				g.nodes(items, path+".tabs."+strconv.Itoa(*sel))
			})
		}
	})
}

type openSet map[int]bool

func (g *genui) accordion(o *lang.Object, path string) {
	r := g.r
	items := garr(o, "items")
	box := ui.Column(r.C).FillWidth().MinWidth(0)
	open := ui.Local(box, "openui.accordion", func() []bool { return nil })
	if len(*open) != len(items) {
		*open = make([]bool, len(items))
	}
	box.Children(func() {
		list := make([]coss.AccordionItem, len(items))
		for i, it := range items {
			n, _ := it.(*lang.Object)
			i := i
			list[i] = coss.AccordionItem{Title: gstr(n, "title"), Open: &(*open)[i], Content: func() {
				ui.Column(r.C).FillWidth().Gap(design.Sp(3)).Children(func() { g.nodes(n.Value("items"), path+".items."+strconv.Itoa(i)) })
			}}
		}
		coss.Accordion(r.C, list)
	})
}

// json is an expandable JSON tree.
func (g *genui) json(v lang.Value, path string) {
	r := g.r
	ui.Column(r.C).FillWidth().MinWidth(0).Padding(design.Sp(3)).Radius(design.RadiusLG).Background(r.T.Muted).Children(func() {
		g.jsonValue("", v, 0)
	})
}

func (g *genui) jsonValue(key string, v lang.Value, depth int) {
	r, t := g.r, g.r.T
	mono := func(s string, col ui.Color) ui.Element {
		e := r.text(s, design.TextXS, design.WeightNormal, col)
		if f := r.md.FontFamily; f != "" {
			e.Font(f)
		}
		return e
	}
	label := ""
	if key != "" {
		label = key + ": "
	}
	var kids []struct {
		k string
		v lang.Value
	}
	open, close := "", ""
	switch x := v.(type) {
	case *lang.Object:
		open, close = "{", "}"
		for _, k := range x.Keys() {
			kids = append(kids, struct {
				k string
				v lang.Value
			}{strconv.Quote(k), x.Value(k)})
		}
	case []lang.Value:
		open, close = "[", "]"
		for i, it := range x {
			kids = append(kids, struct {
				k string
				v lang.Value
			}{strconv.Itoa(i), it})
		}
	default:
		col := t.Foreground
		switch v.(type) {
		case string:
			col = t.Success
		case float64:
			col = t.Info
		case bool, nil:
			col = t.Warning
		}
		ui.Row(r.C).Gap(0).Children(func() {
			mono(label, t.MutedForeground)
			mono(string(lang.EncodeJSON(v)), col)
		})
		return
	}
	box := ui.Column(r.C).FillWidth().MinWidth(0)
	expanded := ui.Local(box, "openui.json", func() bool { return depth < 2 })
	box.Children(func() {
		head := ui.ButtonBase(r.C).Justify(ui.Start).Gap(design.Sp(1)).Cursor(ui.CursorPointer).Expanded(*expanded)
		head.Children(func() {
			rot := float32(0)
			if *expanded {
				rot = 90
			}
			r.icon("chevron-right", design.IconXS, t.MutedForeground).Rotate(rot)
			summary := open + close
			if !*expanded {
				summary = open + " " + strconv.Itoa(len(kids)) + " " + close
			}
			mono(label+summary, t.Foreground)
		})
		if head.Clicked() {
			*expanded = !*expanded
		}
		if *expanded && depth < 12 {
			ui.Column(r.C).FillWidth().Padding(0, 0, 0, design.Sp(4)).Children(func() {
				for i, k := range kids {
					if i >= 200 {
						mono("…", t.MutedForeground)
						break
					}
					r.C.Key(i)
					g.jsonValue(k.k, k.v, depth+1)
				}
			})
		}
	})
}

func (g *genui) fileTree(items []lang.Value, path string, depth int) {
	r, t := g.r, g.r.T
	ui.Column(r.C).FillWidth().MinWidth(0).Role(ui.RoleTree).Children(func() {
		for i, it := range items {
			n, _ := it.(*lang.Object)
			kids := garr(n, "children")
			dir := gstr(n, "type") == "dir" || len(kids) > 0
			r.C.Key(i)
			box := ui.Column(r.C).FillWidth().MinWidth(0)
			open := ui.Local(box, "openui.tree", func() bool { return depth < 1 })
			box.Children(func() {
				row := ui.ButtonBase(r.C).FillWidth().Justify(ui.Start).Gap(design.Sp(1.5)).PaddingY(design.Sp(0.5)).Role(ui.RoleTreeItem)
				if dir {
					row.Cursor(ui.CursorPointer).Expanded(*open)
				}
				row.Children(func() {
					switch {
					case dir && *open:
						r.icon("folder-open", design.IconSM, t.MutedForeground)
					case dir:
						r.icon("folder", design.IconSM, t.MutedForeground)
					default:
						r.icon("file", design.IconSM, t.MutedForeground)
					}
					r.text(gstr(n, "name"), design.TextSM, design.WeightNormal, t.Foreground)
				})
				if dir && row.Clicked() {
					*open = !*open
				}
				if dir && *open && depth < 8 {
					ui.Column(r.C).FillWidth().Padding(0, 0, 0, design.Sp(5)).Children(func() {
						g.fileTree(kids, path+"."+strconv.Itoa(i), depth+1)
					})
				}
			})
		}
	})
}

type quizOption struct {
	label, feedback string
	correct         bool
}

func quizOptions(o *lang.Object) []quizOption {
	var out []quizOption
	for _, it := range garr(o, "options") {
		switch x := it.(type) {
		case string:
			out = append(out, quizOption{label: x})
		case *lang.Object:
			out = append(out, quizOption{label: gstr(x, "label"), feedback: gstr(x, "feedback"), correct: x.Value("correct") == true})
		}
	}
	any := false
	for _, q := range out {
		any = any || q.correct
	}
	if !any {
		switch a := o.Value("answer").(type) {
		case float64:
			if i := int(a); i >= 0 && i < len(out) {
				out[i].correct = true
			}
		case string:
			for i := range out {
				if out[i].label == a {
					out[i].correct = true
				}
			}
		}
	}
	return out
}

// quiz grades locally; the choice persists with the card state.
func (g *genui) quiz(o *lang.Object) {
	r, t := g.r, g.r.T
	rt := r.opts.Runtime
	opts := quizOptions(o)
	id := g.el.Identity()
	key := "selected:" + gstr(o, "id")
	if gstr(o, "id") == "" {
		key = "selected:" + gstr(o, "question")
	}
	chosen := -1
	if v, ok := rt.ComponentState(id, key).(float64); ok {
		chosen = int(v)
	}
	answered := chosen >= 0 && chosen < len(opts)
	ui.Column(r.C).FillWidth().Gap(design.Sp(3)).Children(func() {
		r.inlineText(gstr(o, "question"), design.TextBase, design.WeightSemibold, t.Foreground, nil)
		ui.Column(r.C).FillWidth().Gap(design.Sp(2)).Role(ui.RoleRadioGroup).Children(func() {
			for i, q := range opts {
				r.C.Key(i)
				b := ui.ButtonBase(r.C).FillWidth().Justify(ui.Start).Gap(design.Sp(2.5)).Padding(design.Sp(2.5), design.Sp(3)).
					Radius(design.RadiusLG).Border(design.BorderWidth, t.Border).Role(ui.RoleRadio).Checked(i == chosen)
				marker, col := "○", t.MutedForeground
				switch {
				case answered && q.correct:
					b.Border(design.BorderWidth, t.Success).Background(t.Success.Alpha(0.08))
					marker, col = "✓", t.Success
				case answered && i == chosen:
					b.Border(design.BorderWidth, t.Destructive).Background(t.Destructive.Alpha(0.08))
					marker, col = "✗", t.Destructive
				case !answered:
					b.Cursor(ui.CursorPointer)
					if b.Hovered() {
						b.Background(t.Accent)
					}
				}
				if answered || r.Streaming() {
					b.Disabled(true)
				}
				b.Children(func() {
					r.text(marker, design.TextSM, design.WeightMedium, col)
					r.inlineText(q.label, design.TextSM, design.WeightNormal, t.Foreground, nil)
				})
				if !answered && b.Clicked() {
					rt.SetComponentState(id, key, float64(i))
				}
			}
		})
		if answered {
			c := opts[chosen]
			msg, col := i18n.Pick("回答正确。", "Correct."), t.Success
			if !c.correct {
				msg, col = i18n.Pick("回答错误。", "Not quite."), t.Destructive
			}
			r.paragraph(msg, design.TextSM, design.WeightMedium, col)
			if c.feedback != "" {
				r.inlineText(c.feedback, design.TextSM, design.WeightNormal, t.Foreground, nil)
			}
			if e := gstr(o, "explanation"); e != "" {
				r.inlineText(e, design.TextSM, design.WeightNormal, t.MutedForeground, nil)
			}
			if coss.Button(r.C, i18n.Pick("再试一次", "Try again"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM}).Clicked() {
				rt.SetComponentState(id, key, nil)
			}
		}
	})
}

// blocks draws labelled rows of cells (a simple block diagram).
func (g *genui) blocks(o *lang.Object) {
	r, t := g.r, g.r.T
	rows := garr(o, "rows")
	ui.Column(r.C).FillWidth().MinWidth(0).Gap(design.Sp(2)).Children(func() {
		if s := gstr(o, "title"); s != "" {
			r.paragraph(s, design.TextBase, design.WeightSemibold, t.Foreground).Role(ui.RoleHeading)
		}
		for i, it := range rows {
			n, _ := it.(*lang.Object)
			n = normalize("blocks.rows", n)
			r.C.Key(i)
			col, ok := r.color(gstr(n, "color"))
			if !ok {
				col = t.Primary
			}
			variant := gstr(n, "variant")
			ui.Row(r.C).FillWidth().MinWidth(0).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
				if l := gstr(n, "label"); l != "" {
					r.text(l, design.TextXS, design.WeightMedium, t.MutedForeground).Width(design.Sp(24)).Shrink(0)
				}
				ui.Row(r.C).Grow(1).MinWidth(0).Gap(design.Sp(1)).Wrap().Children(func() {
					for j, cell := range garr(n, "cells") {
						r.C.Key(j)
						text := lang.ToString(cell)
						if co, ok := cell.(*lang.Object); ok {
							text = gstr(co, "label")
						}
						b := ui.Box(r.C).Padding(design.Sp(1), design.Sp(2)).Radius(design.RadiusSM).MinWidth(design.Sp(8)).Center()
						fg := t.Foreground
						switch variant {
						case "outline":
							b.Border(design.BorderWidth, col)
						case "plain":
						default:
							b.Background(col.Alpha(0.16))
						}
						b.Children(func() { r.text(text, design.TextXS, design.WeightMedium, fg) })
					}
				})
			})
		}
		if s := gstr(o, "caption"); s != "" {
			r.paragraph(s, design.TextXS, design.WeightNormal, t.MutedForeground)
		}
	})
}
