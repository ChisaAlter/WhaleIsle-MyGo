package chat

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"path/filepath"
	"sort"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

type childGalleryState struct {
	Ref              string
	Open, ActualSize bool
}

// Only already-loaded local cache paths owned by this resource snapshot enter
// the gallery. Never resolve attachment IDs via the parent/global image loader.
func childGalleryImages(reads map[string]session.AgentResourceRead) []string {
	refs := []string{}
	for ref, r := range reads {
		if !r.Busy && r.Error == "" && filepath.IsAbs(r.Path) && loadChatBitmap(r.Path) != nil {
			refs = append(refs, ref)
		}
	}
	sort.Strings(refs)
	return refs
}

func childImageGallery(c *ui.Context, state *childGalleryState, reads map[string]session.AgentResourceRead) {
	if !state.Open {
		return
	}
	refs := childGalleryImages(reads)
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
	ui.DialogBase(c, &state.Open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, .6))
		w, h := c.Size()
		pw, ph := max(float32(0), min(float32(1000), w-32)), max(float32(0), h-32)
		panel.Width(pw).Height(ph).Padding(16).Radius(20).Clip().Background(c.Theme().Background).Children(func() {
			ui.Row(c).Gap(8).Children(func() {
				if ui.Button(c, i18n.T(i18n.KeyChatImagePrev)).Disabled(index == 0).Clicked() {
					index--
					state.Ref = refs[index]
				}
				if ui.Button(c, i18n.T(i18n.KeyChatImageNext)).Disabled(index == len(refs)-1).Clicked() {
					index++
					state.Ref = refs[index]
				}
				if ui.Button(c, i18n.T(i18n.KeyChatImageFit)).Clicked() {
					state.ActualSize = false
				}
				if ui.Button(c, "1:1").Clicked() {
					state.ActualSize = true
				}
				if ui.Button(c, i18n.T(i18n.KeyChatImageCloseOne)).Clicked() {
					state.Open = false
				}
			})
			ui.Text(c, fmt.Sprintf("%d / %d · %s", index+1, len(refs), state.Ref)).Selectable()
			bmp := loadChatBitmap(reads[state.Ref].Path)
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
