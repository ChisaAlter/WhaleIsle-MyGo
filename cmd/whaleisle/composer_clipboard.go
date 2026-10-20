package main

import (
	"github.com/egoist/mygo"
	"os"
	"path/filepath"
	"whaleisle/internal/desktop/clipboardimage"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/i18n"
)

// Called only from a user paste command or explicit Paste image button.
// No background clipboard polling and no screenshot capture.
func (s *mainState) pasteComposerImage(win *mygo.Window, sessions *session.Store, id string, d *composer.DraftState, explicit bool) {
	if win == nil || sessions == nil || d == nil || d.Sending || d.Preparing > 0 {
		return
	}
	raw := mygo.Clipboard.ReadImage()
	if len(raw) == 0 {
		if explicit {
			d.Error = i18n.T(i18n.KeyCmdNoImageOnTheClipboard)
		}
		return
	}
	s.mu.Lock()
	host := s.dshClient
	s.mu.Unlock()
	seq := d.EditSeq
	d.Preparing++
	go func() {
		path, err := clipboardimage.StorePNG(filepath.Join(paths.UserDataDir(), "cache", "attachments"), raw)
		win.Update(func() {
			d.Preparing--
			s.mu.Lock()
			current := s.composerDrafts[id]
			sameStore := s.sessions == sessions && s.dshClient == host
			s.mu.Unlock()
			if !sameStore || !composer.PasteStillCurrent(d, current, id, sessions.Snapshot().SelectedID, seq) {
				if path != "" {
					_ = os.Remove(path)
				}
				return
			}
			if err != nil {
				d.Error = err.Error()
				return
			}
			s.addComposerFiles(win, sessions, id, d, []string{path})
		})
	}()
}
