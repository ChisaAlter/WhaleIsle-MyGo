package chat

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"path/filepath"
	"sort"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

type childGalleryState struct {
	Ref              string
	Open, ActualSize bool
}

// Only already-loaded local cache paths owned by this resource snapshot enter
// the gallery. Never resolve attachment IDs via the parent/global image loader.
func childGalleryImages(c *ui.Context, reads map[string]session.AgentResourceRead) []string {
	refs := []string{}
	invalidate := c.Services().Invalidate
	for ref, r := range reads {
		if !r.Busy && r.Error == "" && filepath.IsAbs(r.Path) {
			if bmp, _ := cachedChatBitmap(r.Path, invalidate); bmp != nil {
				refs = append(refs, ref)
			}
		}
	}
	sort.Strings(refs)
	return refs
}

func childImageGallery(c *ui.Context, state *childGalleryState, reads map[string]session.AgentResourceRead) {
	if !state.Open {
		return
	}
	refs := childGalleryImages(c, reads)
	index := -1
	for i, ref := range refs {
		if ref == state.Ref {
			index = i
			break
		}
	}
	if index < 0 {
		state.Open = false
		state.Ref = ""
		return
	}
	t := design.For(c)
	ui.DialogBase(c, &state.Open, func(backdrop, panel ui.Element) {
		backdrop.Background(design.TW.Black.Alpha(0.6))
		w, h := c.Size()
		pw, ph := max(float32(0), min(float32(1000), w-32)), max(float32(0), h-32)
		chatPopupSurface(c, panel.Width(pw).Height(ph).Padding(design.Sp(4)).Clip(), design.RadiusXL2+design.Sp(1)).Background(t.Background).Children(func() {
			ui.Row(c).Gap(design.Sp(2)).Children(func() {
				if coss.Button(c, i18n.T(i18n.KeyChatImagePrev), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS, Disabled: index == 0}).Clicked() {
					index--
					state.Ref = refs[index]
				}
				if coss.Button(c, i18n.T(i18n.KeyChatImageNext), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS, Disabled: index == len(refs)-1}).Clicked() {
					index++
					state.Ref = refs[index]
				}
				if coss.Button(c, i18n.T(i18n.KeyChatImageFit), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS}).Clicked() {
					state.ActualSize = false
				}
				if coss.Button(c, "1:1", coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS}).Clicked() {
					state.ActualSize = true
				}
				if coss.Button(c, i18n.T(i18n.KeyChatImageCloseOne), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS}).Clicked() {
					state.Open = false
				}
			})
			ui.Text(c, fmt.Sprintf("%d / %d · %s", index+1, len(refs), state.Ref)).Selectable()
			bmp, pending := cachedChatBitmap(reads[state.Ref].Path, c.Services().Invalidate)
			if pending {
				ui.Text(c, i18n.T(i18n.KeyChatImageLoading)).FontSize(12)
				return
			}
			if bmp == nil {
				state.Open = false
				return
			}
			ui.ScrollBoth(c).Grow(1).MinHeight(0).MinWidth(0).Children(func() {
				if state.ActualSize {
					bw, bh := bmp.Size()
					ui.Image(c, bmp).Width(float32(bw)).Height(float32(bh))
				} else {
					ui.Image(c, bmp).Width(max(0, pw-24)).Height(max(0, ph-110)).Fit(ui.Contain)
				}
			})
		})
	})
}
