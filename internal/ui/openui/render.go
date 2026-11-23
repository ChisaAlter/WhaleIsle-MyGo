package openui

import (
	"sort"
	"sync"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/media"
	"whaleisle/internal/openui/lang"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/widgets/markdown"
)

// Runtime is the card a view renders: the last evaluated frame and the
// writes and clicks the view sends back. *card.Card implements it; its
// methods never block on I/O.
type Runtime interface {
	Frame() *lang.Frame
	Streaming() bool
	Panel() bool
	SetBinding(b *lang.Binding, v lang.Value)
	SetField(form, componentType, name string, v lang.Value)
	FieldValue(form, name string) lang.Value
	ComponentState(identity, key string) lang.Value
	SetComponentState(identity, key string, v lang.Value)
	// Dispatch runs an element's action for a real click on the frame of
	// revision.
	Dispatch(revision uint64, identity, label, form string, action lang.Value)
	Busy(identity string) bool
	Failure(identity string) error
	Notice() error
	// Image and SearchImage request pictures from the media service; nil
	// when the surface has no image source.
	Image(src string) *media.Handle
	SearchImage(query string) *media.Handle
}

// Options configure a View.
type Options struct {
	Runtime Runtime
	// Markdown is the prose style of Text, Markdown and Code (the
	// transcript's fonts and code highlighting). Zero colors take the
	// design tokens.
	Markdown markdown.Style
	// OpenURL opens a validated http(s) link after a real click (Link and
	// inline links); nil shows links as text.
	OpenURL func(url string)
	// OpenCompat opens this card in the A2 compatibility workbench (the
	// web viewer, in its own window). Placeholders offer it; nil hides it.
	OpenCompat func()
	// Width is the width the card lays out in (0: unknown).
	Width float32
}

// Component builds the native view of one element. It returns the element
// it built (callers may size it, as Grid does), or the zero Element.
type Component func(r *Render, el *lang.Element) ui.Element

var (
	registryMu sync.RWMutex
	registry   = map[string]Component{}
)

// Register makes fn the native view of component name, replacing any
// earlier registration. Components without one render a placeholder that
// opens the card in the compatibility workbench.
func Register(name string, fn Component) {
	registryMu.Lock()
	registry[name] = fn
	registryMu.Unlock()
}

// Lookup returns the registered view of name.
func Lookup(name string) (Component, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	fn, ok := registry[name]
	return fn, ok
}

// Native lists the components with a native view, sorted.
func Native() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Render is the state of one build of a card: the window context, the
// frame being shown and the design tokens. Components receive it.
type Render struct {
	C     *ui.Context
	T     *design.Tokens
	Frame *lang.Frame
	opts  *Options
	md    markdown.Style
	depth int
	// row is set while the children of a horizontal container build.
	row bool
}

// Runtime is the card being rendered.
func (r *Render) Runtime() Runtime { return r.opts.Runtime }

// Options are the view options.
func (r *Render) Options() *Options { return r.opts }

// Streaming reports whether the program is still being written; actions
// are disabled until it settles.
func (r *Render) Streaming() bool { return r.Frame.Streaming }

// maxDepth bounds native nesting independently of the language limits.
const maxDepth = 64

// Node builds el with its registered component, or a placeholder.
func (r *Render) Node(el *lang.Element) ui.Element {
	if el == nil {
		return ui.Element{}
	}
	if r.depth >= maxDepth {
		return placeholder(r, el)
	}
	fn, ok := Lookup(el.Type)
	if !ok {
		return placeholder(r, el)
	}
	r.depth++
	defer func() { r.depth-- }()
	r.C.Key(el.Identity())
	return fn(r, el)
}

// Content builds a prop that holds text or components (a string, an
// element, or an array of strings and elements).
func (r *Render) Content(v lang.Value) {
	switch t := v.(type) {
	case nil:
	case string:
		r.paragraph(t, design.TextSM, design.WeightNormal, r.T.Foreground)
	case *lang.Element:
		r.Node(t)
	case []lang.Value:
		for i, it := range t {
			r.C.Key(i)
			r.Content(it)
		}
	default:
		r.paragraph(lang.ToString(t), design.TextSM, design.WeightNormal, r.T.Foreground)
	}
}

// Children builds the element-valued children of prop in a vertical
// container.
func (r *Render) Children(el *lang.Element, prop string) {
	r.Within(false, func() { r.Content(el.Prop(prop)) })
}

// Within builds fn as the children of a horizontal (row) or vertical
// container, so inline components know how to align.
func (r *Render) Within(row bool, fn func()) {
	saved := r.row
	r.row = row
	fn()
	r.row = saved
}

// Inline keeps a compact component (badge, button, link) at its own width
// in a vertical container instead of stretching across it.
func (r *Render) Inline(e ui.Element) ui.Element {
	if !r.row && e.Valid() {
		e.AlignSelf(ui.Start)
	}
	return e
}

// View builds a card. It reads the card's last frame; it never parses or
// evaluates the program.
func View(c *ui.Context, o Options) ui.Element {
	rt := o.Runtime
	f := rt.Frame()
	r := &Render{C: c, T: design.For(c), Frame: f, opts: &o}
	r.md = r.markdownStyle()
	root := ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(3)).TextColor(r.T.Foreground)
	root.Children(func() {
		switch {
		case rt.Panel() && !f.Streaming:
			panelNote(r)
		case f.Root != nil:
			r.Node(f.Root)
		case f.Streaming || len(f.Errors) == 0:
			loading(r)
		}
		if !f.Streaming {
			frameErrors(r, f)
		}
		if err := rt.Notice(); err != nil {
			notice(r, err)
		}
	})
	return root
}
