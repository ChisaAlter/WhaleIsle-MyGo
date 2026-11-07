package pluginui

import (
	"errors"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

type NativeOption struct {
	Value    string `json:"value"`
	Label    string `json:"label"`
	Disabled bool   `json:"disabled"`
}
type NativeNode struct {
	Kind       string  `json:"kind"`
	Handle     string  `json:"handle,omitempty"`
	Label      string  `json:"label,omitempty"`
	Text       string  `json:"text,omitempty"`
	Value      string  `json:"value,omitempty"`
	Checked    bool    `json:"checked,omitempty"`
	Disabled   bool    `json:"disabled,omitempty"`
	Password   bool    `json:"password,omitempty"`
	RichType   string  `json:"richType,omitempty"`
	Height     float64 `json:"height,omitempty"`
	Width      float64 `json:"width,omitempty"`
	X          float64 `json:"x,omitempty"`
	Y          float64 `json:"y,omitempty"`
	Inline     bool    `json:"inline,omitempty"`
	Popup      bool    `json:"popup,omitempty"`
	FullScreen bool    `json:"fullScreen,omitempty"`
	Anchor     string  `json:"anchor,omitempty"`
	Modal      bool    `json:"modal,omitempty"`
	// Ambient seats keep authored shell-overlay coordinates: X/Y are DOM-space
	// and VW/VH the viewport they were measured in, not anchor offsets.
	Ambient  bool    `json:"ambient,omitempty"`
	VW       float64 `json:"vw,omitempty"`
	VH       float64 `json:"vh,omitempty"`
	Min, Max float64
	Options  []NativeOption `json:"options,omitempty"`
	Children []NativeNode   `json:"children,omitempty"`
}
type NativeDocument struct {
	Handle     string       `json:"handle,omitempty"`
	Slot       string       `json:"slot"`
	Nodes      []NativeNode `json:"nodes"`
	Provider   string       `json:"provider,omitempty"`
	Namespace  string       `json:"namespace,omitempty"`
	Registrant string       `json:"registrant,omitempty"`
}

func validNativeNodes(nodes []NativeNode) bool {
	remaining := 512
	var check func([]NativeNode, int) bool
	check = func(rows []NativeNode, depth int) bool {
		if depth > 64 {
			return false
		}
		for _, n := range rows {
			remaining--
			if remaining < 0 || len(n.Handle) > 64 || len(n.Text) > 8192 || len(n.Label) > 8192 || len(n.Value) > 65536 || len(n.Options) > 512 {
				return false
			}
			switch n.Kind {
			case "rich":
				if !validRichType(n.RichType) || !mirrorFragmentFinite(n.Height) || n.Height < 1 || n.Height > 1200 || n.Handle == "" || !mirrorFragmentFinite(n.Width) || n.Width < 0 || n.Width > 8192 || !mirrorFragmentFinite(n.X) || !mirrorFragmentFinite(n.Y) || n.X < -8192 || n.X > 8192 || n.Y < -8192 || n.Y > 8192 || len(n.Anchor) > 64 {
					return false
				}
				if n.Ambient && (!n.Popup || !mirrorFragmentFinite(n.VW) || !mirrorFragmentFinite(n.VH) || n.VW < 1 || n.VW > 32768 || n.VH < 1 || n.VH > 32768) {
					return false
				}
			case "text", "group", "row", "button", "input", "textarea", "check", "select", "range", "unsupported":
			default:
				return false
			}
			for _, option := range n.Options {
				if len(option.Value) > 8192 || len(option.Label) > 8192 {
					return false
				}
			}
			if !check(n.Children, depth+1) {
				return false
			}
		}
		return true
	}
	return check(nodes, 0)
}

// RenderNativeSlot consumes only live occurrences in the selected page/session.
// Native text controls keep IME, keyboard and accessibility in the native UI.
// false means no elected contribution, so the caller draws its normal fallback.
func (o *Owner) RenderNativeSlot(c *ui.Context, slot, callID string, turn int) bool {
	return o.renderNativeSlot(c, slot, callID, turn, -1, "")
}

func (o *Owner) RenderNativeMessage(c *ui.Context, slot, callID string, seq float64) bool {
	return o.renderNativeSlot(c, slot, callID, -1, seq, "")
}

func (o *Owner) RenderNativeOutlet(c *ui.Context, slot, handle string) bool {
	return o.renderNativeSlot(c, slot, "", -1, -1, handle)
}

func (o *Owner) renderNativeSlot(c *ui.Context, slot, callID string, turn int, seq float64, exactHandle string) bool {
	return o.renderNativeSnapshot(c, o.Snapshot(), slot, callID, turn, seq, exactHandle)
}

func (o *Owner) RenderGlobalNativeSlot(c *ui.Context, slot string) bool {
	s := o.Snapshot()
	s.Documents = s.GlobalDocuments
	s.Outlets = nil
	return o.renderNativeSnapshot(c, s, slot, "", -1, -1, "")
}

// GlobalNativeRenderer captures one elected source snapshot for shell layout.
// The returned renderer refuses to paint after a page/revision/owner change.
func (o *Owner) GlobalNativeRenderer(slot string) func(*ui.Context) bool {
	if o == nil {
		return nil
	}
	s := o.Snapshot()
	found := false
	for _, document := range s.GlobalDocuments {
		if document.Slot == slot && len(document.Nodes) > 0 {
			found = true
			break
		}
	}
	if !found || !s.Connected || s.Subject == nil || s.Unavailable != "" {
		return nil
	}
	s.Documents = s.GlobalDocuments
	s.Outlets = nil
	return func(c *ui.Context) bool {
		o.mu.Lock()
		valid := !o.closed && !o.masked && !o.incompatible && o.state.Page == s.Page && o.state.Revision == s.Revision && o.selected.equal(*s.Subject)
		o.mu.Unlock()
		if !valid {
			return false
		}
		return o.renderNativeSnapshot(c, s, slot, "", -1, -1, "")
	}
}

func (o *Owner) renderNativeSnapshot(c *ui.Context, s Snapshot, slot, callID string, turn int, seq float64, exactHandle string) bool {
	if !s.Connected || s.Subject == nil || s.Unavailable != "" {
		return false
	}
	if s.Subject.Kind != "session" {
		for _, document := range s.Documents {
			if document.Slot == slot {
				handle := document.Handle
				if handle == "" {
					handle = slot
				}
				s.Outlets = append(s.Outlets, SessionOutlet{Handle: handle, Slot: slot, Available: true, Native: document.Nodes, Registrant: document.Registrant})
			}
		}
	}
	found := false
	for _, outlet := range s.Outlets {
		if s.Subject.Kind == "session" && nativeDefaultRegistrant(outlet.Registrant) {
			continue
		}
		if exactHandle != "" && outlet.Handle != exactHandle {
			continue
		}
		if !outlet.Available || outlet.Slot != slot || callID != "" && outlet.CallID != callID || turn >= 0 && (!outlet.HasTurn || outlet.Turn != turn) || seq >= 0 && (!outlet.HasSeq || outlet.Seq != seq) || len(outlet.Native) == 0 {
			continue
		}
		found = true
		anchors := map[string]ui.Element{}
		var render func(*ui.Context, []NativeNode)
		render = func(ctx *ui.Context, nodes []NativeNode) {
			for index, node := range nodes {
				n := node
				child := ctx.Key(outlet.Handle + ":" + n.Handle + ":" + strconv.Itoa(index))
				act := func(action, value string) { _ = o.nativeInteraction(s, outlet.Handle, n.Handle, action, value) }
				if n.Kind == "rich" {
					if n.Popup || n.FullScreen {
						o.renderRichPopup(child, s, outlet.Handle, n, anchors[n.Anchor])
					} else {
						anchors[n.Handle] = o.renderRichControl(child, s, outlet.Handle, n)
					}
					continue
				}
				if n.Modal {
					n.Modal = false
					open := true
					ui.DialogBase(child, &open, func(backdrop, panel ui.Element) {
						w, h := child.Size()
						backdrop.Background(ui.RGBA(0, 0, 0, .35))
						panel.Width(max(240, min(float32(840), w-48))).MaxHeight(max(0, h-48)).Padding(16).Background(child.Theme().Background).Radius(10).Children(func() { ui.Scroll(child).FillWidth().MinHeight(0).Children(func() { render(child, []NativeNode{n}) }) })
					})
					if !open {
						act("escape", "")
					}
					continue
				}
				switch n.Kind {
				case "group":
					ui.Column(child).FillWidth().MinWidth(0).Gap(6).Children(func() { render(child, n.Children) })
				case "row":
					ui.Row(child).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() { render(child, n.Children) })
				case "text":
					ui.Text(child, n.Text).FillWidth().MinWidth(0).Selectable()
				case "button":
					if ui.Button(child, n.Text).Label(n.Text).Disabled(n.Disabled).Clicked() {
						act("click", "")
					}
				case "check":
					v := n.Checked
					if ui.Checkbox(child, &v, n.Text).Disabled(n.Disabled).Changed() {
						act("click", "")
					}
				case "input", "textarea":
					v := n.Value
					var field ui.Element
					if n.Kind == "textarea" {
						field = ui.TextArea(child, &v).Lines(3, 16)
					} else {
						field = ui.TextInput(child, &v)
					}
					field = field.Label(n.Label).FillWidth().MinWidth(0).Disabled(n.Disabled)
					if n.Password {
						field = field.Password()
					}
					if field.Changed() {
						act("input", v)
					}
				case "select":
					labels := []string{}
					value := n.Value
					counts, seen := map[string]int{}, map[string]int{}
					for _, option := range n.Options {
						counts[option.Label]++
					}
					choices := map[string]string{}
					for _, option := range n.Options {
						label := option.Label
						seen[label]++
						if counts[label] > 1 {
							label += " [" + strconv.Itoa(seen[label]) + "]"
						}
						if !option.Disabled {
							labels = append(labels, label)
							choices[label] = option.Value
						}
						if option.Value == n.Value {
							value = label
						}
					}
					if ui.Select(child, &value, labels).Label(n.Label).Disabled(n.Disabled).Changed() {
						if selected, ok := choices[value]; ok {
							act("input", selected)
						}
					}
				case "range":
					v, _ := strconv.ParseFloat(n.Value, 64)
					if n.Max > n.Min && ui.Slider(child, &v, n.Min, n.Max).Label(n.Label).Disabled(n.Disabled).Changed() {
						act("input", strconv.FormatFloat(v, 'f', -1, 64))
					}
				case "rich":
					o.renderRichControl(child, s, outlet.Handle, n)
				case "unsupported":
					ui.Text(child, "插件原生控件尚不支持："+n.Text).FillWidth().TextColor(child.Theme().Warning)
				}
			}
		}
		render(c, outlet.Native)
	}
	return found
}

func (o *Owner) nativeInteraction(s Snapshot, outlet, handle, action, value string) error {
	o.mu.Lock()
	valid := !o.closed && o.state.Page == s.Page && o.state.Revision == s.Revision && s.Subject != nil && o.state.Subject != nil && o.state.Subject.equal(*s.Subject) && o.selected.equal(*s.Subject)
	page := o.page
	o.mu.Unlock()
	if !valid {
		return errors.New("plugin UI: native control owner changed")
	}
	return o.sendToPage(page, map[string]any{"type": "nativeInput", "revision": s.Revision, "subject": s.Subject, "outlet": outlet, "handle": handle, "action": action, "value": value})
}

// BundledRegistrant is how the page reports entries registered by the Harness
// web bundle itself, which all share one minified registrant (see
// whaleBundledRegistrant in public_slots.js).
const BundledRegistrant = "whaleisle:bundled"

// These original default renderers already have native equivalents. Only an
// elected extension from another registrant replaces that native fallback.
func nativeDefaultRegistrant(registrant string) bool {
	switch registrant {
	case BundledRegistrant, "@deepseek-ai/dsh-client-ui-conversation", "@deepseek-ai/dsh-client-ui-chat", "@deepseek-ai/dsh-client-ui-tool":
		return true
	}
	return false
}

// CanonicalNativeSlot names the exact installed renderer surface consumed by
// each existing native UI delegate. toolview/body is the native card as a whole;
// the installed ui-tool owner dispatches its keyed body as tool.call.toolview.
func CanonicalNativeSlot(slot string) string {
	switch slot {
	case "composer":
		return "conversation.composer"
	case "user.content":
		return "conversation.chat.user.content"
	case "reasoning.body":
		return "conversation.chat.reasoning.body"
	case "assistant-actions":
		return "conversation.chat.assistant-actions"
	case "turnTail":
		return "conversation.chat.turnTail"
	case "toolview/body":
		return "tool.call.toolview"
	}
	return slot
}

// AdditiveNativeSlot reports the composer seats the original UI fills with every
// registered entry (renderSlot), unlike the chain slots where one elected
// contribution replaces the seat. Plugin entries join the native controls there.
func AdditiveNativeSlot(slot string) bool {
	return slot == "conversation.composer.dock" || strings.HasPrefix(slot, "conversation.input.")
}
