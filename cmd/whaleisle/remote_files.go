package main

import (
	"github.com/egoist/mygo"
	"whaleisle/internal/dsh"
	remotestore "whaleisle/internal/store/remote"
	"whaleisle/internal/store/session"
	remoteui "whaleisle/internal/ui/remote"
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

// The UI owns one Files/editor per pane occurrence. This producer supplies only
// the selected session's authenticated client, cwd and selection-epoch fence.
func (s *mainState) remoteFileWorkspaceModel(win *mygo.Window, sessions *session.Store, themeID string, dark, ready, enabled bool, owners ...*remoteui.FileWorkspaceState) remoteui.Model {
	m := remoteui.Model{ThemeID: themeID, SystemDark: dark, HarnessReady: ready, RemoteWorkspaceEnabled: enabled}
	if len(owners) > 0 {
		m.FileWorkspace = owners[0]
	}
	if sessions == nil {
		return m
	}
	snapshot := sessions.Snapshot()
	reader, current, err := sessions.MediaPreviewSource(snapshot.SelectedID)
	if err != nil {
		return m
	}
	client, ok := reader.(*dsh.Client)
	if !ok || !current() {
		return m
	}
	m.FileClient, m.FileSessionID, m.FileCwd, m.FileCurrent = client, snapshot.SelectedID, snapshot.SelectedCwd, current
	m.OnInvalidate = func() {
		if win != nil {
			win.Invalidate()
		}
	}
	return m
}
