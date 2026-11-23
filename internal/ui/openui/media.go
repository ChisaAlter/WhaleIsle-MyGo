package openui

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/media"
	"whaleisle/internal/openui/lang"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
)

func init() {
	Register("Image", imageView)
	Register("SearchImage", searchImageView)
}

// picture draws a media-service handle: a skeleton while it loads, the
// picture when ready, and fallback when it failed.
func (r *Render) picture(el *lang.Element, h *media.Handle, alt string, fallback func() bool) ui.Element {
	ratio := float32(numOr(el, "aspectRatio", 0))
	if ratio <= 0 || ratio > 10 {
		ratio = 0
	}
	rad := radius(el.Prop("radius"), design.RadiusLG)
	fit := ui.Cover
	if str(el, "fit") == "contain" {
		fit = ui.Contain
	}
	box := ui.Box(r.C).FillWidth().MinWidth(0).Radius(rad).Clip().Role(ui.RoleImage).Label(alt)
	if h != nil {
		if pic, ok := h.Picture(); ok {
			if ratio == 0 && pic.Height > 0 {
				ratio = max(0.2, min(pic.Width/pic.Height, 5))
			}
			box.AspectRatio(ratio)
			return box.Children(func() {
				ui.Image(r.C, pic.Image()).Fill().Fit(fit)
			})
		}
	}
	if ratio == 0 {
		ratio = 16.0 / 9
	}
	box.AspectRatio(ratio)
	if h != nil && h.State() == media.Pending {
		return box.Children(func() { coss.Skeleton(r.C).Fill() })
	}
	if fallback != nil && fallback() {
		return ui.Element{}
	}
	return box.Background(r.T.Muted).Center().Children(func() {
		ui.Column(r.C).AlignItems(ui.Center).Gap(design.Sp(1)).Padding(design.Sp(3)).Children(func() {
			r.icon("image-off", design.IconLG, r.T.MutedForeground)
			if alt != "" {
				r.paragraph(alt, design.TextXS, design.WeightNormal, r.T.MutedForeground).TextAlign(ui.Center)
			}
		})
	})
}

func imageView(r *Render, el *lang.Element) ui.Element {
	rt := r.opts.Runtime
	src, alt := str(el, "src"), str(el, "alt")
	var h *media.Handle
	if lang.ValidateExternalURL(src) == nil {
		h = rt.Image(src)
	}
	// A failed image finds a picture for its alt text, as the viewer does.
	return r.picture(el, h, alt, func() bool {
		if len([]rune(alt)) < 2 {
			return false
		}
		s := rt.SearchImage(alt)
		if s == nil || s.State() == media.Failed {
			return false
		}
		r.picture(el, s, alt, nil)
		return true
	})
}

func searchImageView(r *Render, el *lang.Element) ui.Element {
	q := str(el, "query")
	var h *media.Handle
	if len([]rune(q)) >= 2 {
		h = r.opts.Runtime.SearchImage(q)
	}
	return r.picture(el, h, q, nil)
}
