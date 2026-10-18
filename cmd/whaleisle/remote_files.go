package main

import (
	"github.com/egoist/mygo"
	remotestore "whaleisle/internal/store/remote"
	"whaleisle/internal/store/session"
)

func (s *mainState) remoteSessionFiles(win *mygo.Window, sessions *session.Store) *remotestore.Files {
	s.mu.Lock()
	if s.remoteFiles == nil {
		s.remoteFiles = remotestore.NewFiles()
	}
	files, client := s.remoteFiles, s.dshClient
	s.mu.Unlock()
	id := ""
	if sessions != nil {
		id = sessions.Snapshot().SelectedID
	}
	files.Bind(client, id, func() {
		if win != nil {
			win.Update(func() {})
		}
	})
	return files
}
