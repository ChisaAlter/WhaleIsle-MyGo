package main

import (
	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/avatarimage"
	dshbotstore "whaleisle/internal/store/dshbot"
	assistantui "whaleisle/internal/ui/assistant"
)

func (s *mainState) bindBotAvatarImageActions(m *assistantui.BotAvatarModel, k dshbotstore.RoutineScope, current, viewCurrent func() bool) {
	controls := &s.assistantRoutines.Avatar
	m.OnChooseImage = func() {
		if !viewCurrent() {
			return
		}
		ticket, ok := controls.BeginImageImport(k)
		if !ok {
			return
		}
		s.mu.Lock()
		win := s.win
		s.mu.Unlock()
		s.updateAssistantTasks()
		go func() {
			live := func() bool { return current() && controls.ImageImportCurrent(ticket) }
			source, err := avatarimage.Select(func() ([]string, error) {
				// MyGo dialog methods may be called from any goroutine. No path entry,
				// default path, enumeration, directory selection or remote fetch is added.
				return mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: win, Title: "选择本地头像图片 / Choose local avatar image", Filters: []mygo.FileFilter{{Name: "JPEG / PNG", Extensions: []string{"jpg", "jpeg", "png"}}}})
			}, live, avatarimage.ReadPickedFile)
			if !live() {
				return
			}
			if controls.FinishImageImport(ticket, source, err) {
				s.updateAssistantTasks()
			}
		}()
	}
	m.OnImageCropChange = func(crop avatarimage.Crop) {
		if viewCurrent() {
			controls.UpdateImageCrop(k, crop)
			s.updateAssistantTasks()
		}
	}
	m.OnApplyImageCrop = func() {
		if viewCurrent() {
			controls.ApplyImageCrop(k)
			s.updateAssistantTasks()
		}
	}
	m.OnCancelImageCrop = func() {
		if viewCurrent() {
			controls.CancelImageImport(k)
			s.updateAssistantTasks()
		}
	}
}
