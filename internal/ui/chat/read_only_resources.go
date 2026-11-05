package chat

import (
	"encoding/json"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

type ReadOnlyResourceOptions struct {
	TurnUsage   map[float64]session.TurnUsage
	Scope       string
	OnReadTable func(string, workspacefs.XLSXPageRequest)
	OnOpenFile  func(string)
	OnOpenURL   func(string)
	OnRead      func(string, bool)
	Reads       map[string]session.AgentResourceRead
}

func readOnlyResourceLink(c *ui.Context, ref string, opts ReadOnlyResourceOptions, st ToolCardStyle) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return
	}
	u, _ := url.Parse(ref)
	if u != nil && (u.Scheme == "https" || u.Scheme == "http") {
		if stdButton(c, st.Colors, ref, buttonGhost, opts.OnOpenURL == nil).Clicked() {
			opts.OnOpenURL(ref)
		}
		return
	}
	if opts.OnRead == nil {
		ui.Text(c, ref).FontSize(10).TextColor(st.Muted).Selectable()
		return
	}
	if u != nil && u.Scheme != "" && u.Scheme != "attachment" && len(u.Scheme) != 1 {
		ui.Text(c, ref).FontSize(10).TextColor(st.Muted).Selectable()
		return
	}
	read := opts.Reads[ref]
	label := i18n.T(i18n.KeyChatResourcePreview) + ": " + ref
	if read.Error != "" {
		label = i18n.T(i18n.KeyChatImageRetry) + ": " + ref
	}
	if stdButton(c, st.Colors, label, buttonGhost, read.Busy).Clicked() {
		opts.OnRead(ref, false)
	}
}
func readOnlyToolResources(c *ui.Context, msg session.Message, opts ReadOnlyResourceOptions, st ToolCardStyle) {
	seen := map[string]bool{}
	emit := func(ref string) {
		if ref != "" && !seen[ref] {
			seen[ref] = true
			readOnlyResourceLink(c, ref, opts, st)
		}
	}
	var visit func(any)
	visit = func(value any) {
		switch v := value.(type) {
		case map[string]any:
			if id, ok := v["attachmentId"].(string); ok {
				emit("attachment:" + id)
			}
			for k, x := range v {
				if k == "text" || k == "url" {
					if str, ok := x.(string); ok {
						for _, r := range readOnlyReferences(str) {
							emit(r)
						}
						if k == "url" {
							emit(str)
						}
					}
				}
				visit(x)
			}
		case []any:
			for _, x := range v {
				visit(x)
			}
		}
	}
	parse := func(raw json.RawMessage) {
		var value any
		if json.Unmarshal(raw, &value) == nil {
			visit(value)
		}
	}
	parse(msg.Raw)
	if msg.Tool != nil {
		stack := []*session.ToolCallView{msg.Tool}
		for len(stack) > 0 {
			call := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			for _, e := range call.Events {
				parse(e.Raw)
			}
			stack = append(stack, call.Children...)
		}
	}
}

// ReadOnlyResourceResultsView exposes reads started by inline references and delivery cards.
func ReadOnlyResourceResultsView(c *ui.Context, opts ReadOnlyResourceOptions, st ToolCardStyle) {
	galleryRoot := ui.Box(c.Key("child-image-gallery:" + opts.Scope))
	gallery := ui.Local(galleryRoot, "viewer", func() childGalleryState { return childGalleryState{} })
	refs := make([]string, 0, len(opts.Reads))
	for ref := range opts.Reads {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	for _, ref := range refs {
		read := opts.Reads[ref]
		ui.Text(c, ref).FontSize(11).TextColor(st.Muted).Selectable()
		if read.Error != "" {
			ui.Text(c, read.Error).FontSize(11).TextColor(st.Muted)
			if stdButton(c, st.Colors, i18n.T(i18n.KeyChatImageRetry)+": "+ref, buttonGhost, false).Clicked() {
				opts.OnRead(ref, false)
			}
		}
		if read.Path != "" {
			if opts.Scope != "" && filepath.IsAbs(read.Path) && loadChatBitmap(read.Path) != nil {
				open := func(string) { gallery.Ref = ref; gallery.Open = true; gallery.ActualSize = false }
				ImageInlineView(c, read.Path, st.FG, st.Muted, open)
				if stdButton(c, st.Colors, i18n.T(i18n.KeyChatImageEnlarge)+": "+ref, buttonGhost, false).Clicked() {
					open(read.Path)
				}
			} else {
				ImageInlineView(c, read.Path, st.FG, st.Muted, nil)
			}
		}
		if read.Table != nil {
			childWorkbookView(c, ref, read, opts, st)
		}
		if read.Text != "" {
			ui.Text(c, read.Text).FontSize(12).TextColor(st.FG).Selectable()
		}
		if !read.Busy && !read.EOF && read.Error == "" && stdButton(c, st.Colors, i18n.T(i18n.KeyChatResourceMore)+": "+ref, buttonGhost, false).Clicked() {
			opts.OnRead(ref, true)
		}
	}
	if opts.Scope != "" {
		childImageGallery(c, gallery, opts.Reads)
	}
}
