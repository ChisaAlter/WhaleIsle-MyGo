package main

import (
	"log"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/config"
	"whaleisle/internal/harness"
	"whaleisle/internal/ui/chat"
	"whaleisle/internal/ui/theme"
)

// paintPreviewNoticeLayer draws the one-per-copy-version 预览版说明 notice above
// the shell. The native harness shows it once for the whole product and stores
// the acknowledgement in its welcome settings namespace
// (ui-settings-models/src/client/welcome-store.ts); MyGo keeps the same version
// string in its own config, so a fresh profile sees the notice once and a newer
// notice brings it back.
//
// The notice waits for a Ready harness, which is when the native client mounts
// its onboarding step; the boot page and the launcher never show it.
func (s *mainState) paintPreviewNoticeLayer(c *ui.Context, colors theme.Colors) {
	if s.cfgStore == nil {
		return
	}
	s.mu.Lock()
	ready := s.bootStatus.State == harness.Ready
	s.mu.Unlock()
	if !ready {
		return
	}
	ack := s.cfgStore.Get().PreviewNoticeAck
	chat.PreviewNoticeView(c, chat.PreviewNotice{
		Open:   chat.PreviewNoticeRequested(ack),
		Colors: colors,
		OnContinue: func() {
			if err := s.cfgStore.Update(func(cfg *config.Config) {
				cfg.PreviewNoticeAck = chat.PreviewNoticeVersion
			}); err != nil {
				log.Println("preview notice:", err)
				return
			}
			c.Services().Invalidate()
		},
	})
}
