package main

import (
	"context"
	"errors"
	"github.com/egoist/mygo"
	"os"
	"path/filepath"
	"whaleisle/internal/desktop/clipboardimage"
	"whaleisle/internal/errcopy"
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
	s.dshMu.Lock()
	host := s.dshClient
	s.dshMu.Unlock()
	seq := d.EditSeq
	d.Preparing++
	ctx := s.lifecycleContext()
	go func() {
		var path string
		err := withComposerFileWorker(ctx, func() error {
			var err error
			path, err = clipboardimage.StorePNG(filepath.Join(paths.UserDataDir(), "cache", "attachments"), raw)
			return err
		})
		if err == nil {
			err = ctx.Err()
		}
		win.Update(func() {
			d.Preparing--
			s.mu.Lock()
			current := s.composerDrafts[id]
			sameStore := s.sessions == sessions && !s.quitting
			s.mu.Unlock()
			s.dshMu.Lock()
			sameStore = sameStore && s.dshClient == host
			s.dshMu.Unlock()
			if !sameStore || !composer.PasteStillCurrent(d, current, id, id, seq) || errors.Is(err, context.Canceled) {
				if path != "" {
					go os.Remove(path)
				}
				return
			}
			if err != nil {
				d.Error = errcopy.Sentence(err)
				return
			}
			s.addComposerFiles(win, sessions, id, d, []string{path})
		})
	}()
}
