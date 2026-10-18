package chat

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/store/session"
)

// Bounded per navigation occurrence, not a background history prefetch queue.
type childImageRequests map[string]bool

func (requests childImageRequests) request(ref string, opts ReadOnlyResourceOptions) {
	if opts.Scope == "" || opts.OnRead == nil || requests[ref] || len(requests) >= 32 {
		return
	}
	if _, exists := opts.Reads[ref]; exists {
		return
	} // errors retain the explicit retry action
	active := 0
	for id := range requests {
		r, known := opts.Reads[id]
		if !known || r.Busy {
			active++
		}
	}
	if active >= 2 {
		return
	}
	requests[ref] = true
	opts.OnRead(ref, false)
}

func childDeclaredImages(msg session.Message) []string {
	refs := session.ImageRefs(msg.Body)
	if msg.ImageRef != "" {
		refs = append(refs, msg.ImageRef)
	}
	refs = append(refs, ParseToolCard(msg.EventType, msg.Raw).Images...)
	if msg.Tool != nil {
		stack := []*session.ToolCallView{msg.Tool}
		seen := map[*session.ToolCallView]bool{}
		for len(stack) > 0 {
			call := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if call == nil || seen[call] {
				continue
			}
			seen[call] = true
			for _, ev := range call.Events {
				refs = append(refs, ParseToolCard(ev.EventType, ev.Raw).Images...)
			}
			stack = append(stack, call.Children...)
		}
	}
	out := []string{}
	seen := map[string]bool{}
	for _, ref := range refs {
		id, ok := strings.CutPrefix(ref, "attachment:")
		if ok && strings.TrimSpace(id) != "" && !seen[ref] {
			seen[ref] = true
			out = append(out, ref)
		}
	}
	return out
}
func childImageVisible(r, clip ui.Rect) bool {
	return r.W > 0 && r.H > 0 && clip.W > 0 && clip.H > 0 && r.X < clip.X+clip.W && r.X+r.W > clip.X && r.Y < clip.Y+clip.H && r.Y+r.H > clip.Y
}

func childAutoImages(c *ui.Context, msg session.Message, opts ReadOnlyResourceOptions, requests childImageRequests, st ToolCardStyle) {
	if opts.Scope == "" {
		return
	}
	for _, ref := range childDeclaredImages(msg) {
		read := opts.Reads[ref]
		seat := ui.Column(c.Key("child-auto-image:" + ref)).FillWidth().MinHeight(40)
		gallery := ui.Local(seat, "viewer", func() childGalleryState { return childGalleryState{} })
		seat.DrawOver(func(p *ui.Painter, r ui.Rect) {
			if childImageVisible(r, p.NativeSurfaceClip()) {
				requests.request(ref, opts)
			}
		})
		seat.Children(func() {
			if read.Path != "" {
				ImageInlineView(c, read.Path, st.FG, st.Muted, func(string) { gallery.Ref = ref; gallery.Open = true; gallery.ActualSize = false })
			} else if read.Busy {
				ui.Text(c, "加载图片 / Loading image: "+ref).FontSize(11)
			} else {
				readOnlyResourceLink(c, ref, opts, st)
			}
		})
		childImageGallery(c, gallery, opts.Reads)
	}
}
