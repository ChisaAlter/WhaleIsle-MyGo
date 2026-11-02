package pluginui

import (
	_ "embed"
	"github.com/egoist/mygo/ui"
)

//go:embed mirror_portal_guard.js
var mirrorPortalGuardSource []byte

type MirrorPortal struct {
	Token       string     `json:"token"`
	Handle      string     `json:"handle"`
	Entry       string     `json:"entry"`
	Label       string     `json:"label"`
	Turn        int        `json:"turn"`
	Seq         uint64     `json:"seq"`
	Rect        MirrorRect `json:"rect"`
	VisibleRect MirrorRect `json:"visibleRect"`
	Visible     bool       `json:"visible"`
	Text        string     `json:"text"`
}

// MirrorPortals returns data from the same single original bitmap. The original
// nodes and registration tokens remain owned by the document-start bridge.
func (f *MirrorFrame) MirrorPortals() []MirrorPortal {
	if f == nil {
		return nil
	}
	return append([]MirrorPortal(nil), f.geometry.Portals...)
}
func (p MirrorPortal) Occurrence() MirrorOccurrence {
	return MirrorOccurrence{Handle: p.Handle, Entry: p.Entry, Label: p.Label, Turn: p.Turn, Seq: p.Seq, Rect: p.Rect, VisibleRect: p.VisibleRect, Visible: p.Visible, Connected: true, Text: p.Text}
}
func (o *Owner) MirrorPortalPointer(f *MirrorFrame, portal MirrorPortal, event ui.InputEvent, width, height float32) bool {
	return o.mirrorPointer(f, portal.Occurrence(), event, width, height, portal.Token)
}
