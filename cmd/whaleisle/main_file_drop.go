package main

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/frame"
)

// The caption is a known main-window region, unlike an arbitrary unconsumed
// Window.OnFileDrop event, which has coordinates but no owner/surface identity.
func (s *mainState) mainCaptionFileDrop(c *ui.Context, ready bool, m frame.Model, content func()) {
	if !ready {
		content()
		return
	}
	frame.MainCaptionFileDrop(c, s.navigationFileDropModel(m), content)
}
