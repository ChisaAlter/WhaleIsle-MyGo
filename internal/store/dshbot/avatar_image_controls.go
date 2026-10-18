package dshbot

import (
	"whaleisle/internal/desktop/avatarimage"
	"whaleisle/internal/desktop/pluginrpc"
)

type AvatarImageImportSnapshot struct {
	Busy, Ready          bool
	Name, Format         string
	Bytes, Width, Height int
	Crop                 avatarimage.Crop
	Preview, Error       string
}
type avatarImagePending struct {
	Ticket AvatarTicket
	Source *avatarimage.Source
}

// Called directly by the existing hide/minimize/close/Host/Bot/page lifecycle.
// Sent attempts retain only the reviewed output for exact unknown readback.
func (c *AvatarControls) clearImageImportLocked() {
	if e := c.entries[c.active]; e != nil {
		e.state.ImageImport = AvatarImageImportSnapshot{}
		if e.imageDraft && (e.attempt == nil || !e.attempt.Sent) {
			if before, err := pluginrpc.DshbotAvatarValue(e.state.Status); err == nil {
				e.state.Draft = before
				e.state.Dirty = false
			}
			e.state.Review = nil
			if e.attempt != nil {
				e.attempt = nil
				e.state.Busy = false
				e.state.Loaded = false
			}
			e.imageDraft = false
		}
	}
	c.imageImport = nil
}
func (c *AvatarControls) BeginImageImport(k RoutineScope) (AvatarTicket, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e == nil || e.state.Review != nil {
		return AvatarTicket{}, false
	}
	c.sequence++
	c.generation++
	t := AvatarTicket{k, c.generation, c.sequence}
	c.imageImport = &avatarImagePending{Ticket: t}
	e.state.ImageImport = AvatarImageImportSnapshot{Busy: true}
	return t, true
}
func (c *AvatarControls) imageImportCurrent(t AvatarTicket) bool {
	return c.valid(t.Scope) && c.generation == t.Generation && c.imageImport != nil && c.imageImport.Ticket == t
}
func (c *AvatarControls) ImageImportCurrent(t AvatarTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.imageImportCurrent(t)
}
func (c *AvatarControls) FinishImageImport(t AvatarTicket, source *avatarimage.Source, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.imageImportCurrent(t) {
		return false
	}
	e := c.entry(t.Scope)
	e.state.ImageImport = AvatarImageImportSnapshot{}
	c.generation++
	if err != nil || source == nil {
		c.imageImport = nil
		if err != nil {
			e.state.ImageImport.Error = err.Error()
		}
		return true
	}
	crop := avatarimage.DefaultCrop(e.state.Draft.Crop)
	preview, bakeErr := source.Bake(crop)
	if bakeErr != nil {
		c.imageImport = nil
		e.state.ImageImport.Error = bakeErr.Error()
		return true
	}
	c.imageImport.Source = source
	e.state.ImageImport = AvatarImageImportSnapshot{Ready: true, Name: source.Name, Format: source.Format, Bytes: source.Bytes, Width: source.Width, Height: source.Height, Crop: crop, Preview: preview}
	return true
}
func (c *AvatarControls) UpdateImageCrop(k RoutineScope, crop avatarimage.Crop) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) || c.imageImport == nil || c.imageImport.Source == nil {
		return
	}
	e := c.entry(k)
	if !e.state.ImageImport.Ready || e.attempt != nil || !avatarimage.ValidCrop(crop) {
		return
	}
	preview, err := c.imageImport.Source.Bake(crop)
	if err != nil {
		e.state.ImageImport.Error = err.Error()
		return
	}
	e.state.ImageImport.Crop = crop
	e.state.ImageImport.Preview = preview
	e.state.ImageImport.Error = ""
	c.generation++
}
func (c *AvatarControls) ApplyImageCrop(k RoutineScope) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) || c.imageImport == nil || c.imageImport.Source == nil {
		return false
	}
	e := c.entry(k)
	p := e.state.ImageImport
	if !p.Ready || p.Error != "" || e.attempt != nil || !e.state.Loaded || !e.state.Writable || e.state.Status.WriteError != "" || e.state.Uncertain {
		return false
	}
	if avatarimage.ValidateBaked(p.Preview) != nil {
		return false
	}
	before, err := pluginrpc.DshbotAvatarValue(e.state.Status)
	if err != nil {
		return false
	}
	e.state.Draft = pluginrpc.DshbotAvatarValues{Kind: "image", DataURL: p.Preview, Crop: p.Crop.Shape}
	e.state.Dirty = e.state.Draft != before
	e.hasDraft = true
	e.imageDraft = e.state.Dirty
	e.state.ImageImport = AvatarImageImportSnapshot{}
	c.imageImport = nil
	c.generation++
	return true
}
func (c *AvatarControls) CancelImageImport(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return
	}
	// Preserve any earlier draft: cancel affects only this local selection.
	c.imageImport = nil
	c.entry(k).state.ImageImport = AvatarImageImportSnapshot{}
	c.generation++
}
