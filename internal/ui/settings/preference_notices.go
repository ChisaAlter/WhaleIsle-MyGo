package settings

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	store "whaleisle/internal/store/settings"
)

// UploadPreference's shell-level receipt survives closing the Settings panel.
func PreferenceNotices(c *ui.Context, s *store.Store) {
	if s == nil {
		return
	}
	sequence, notice := s.UploadNotice()
	if notice == "" {
		return
	}
	anchor := ui.Box(c.Key("settings-preference-notices")).Size(0, 0)
	shown := ui.Local(anchor, "shown", func() uint64 { return 0 })
	if *shown == sequence {
		return
	}
	*shown = sequence
	text, tone := providerCopy("已保存会话日志上传设置。", "Session log upload preference saved."), "success"
	if notice == "failed" {
		text, tone = providerCopy("未能保存会话日志上传设置。", "Could not save the session log upload preference."), "error"
	}
	c.AddToast(ui.Toast{ID: fmt.Sprintf("session-log-upload-%d", sequence), Title: text, Type: tone, OnClose: func() { s.DismissUploadNotice(sequence) }})
}
