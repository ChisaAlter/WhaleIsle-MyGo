package main

import (
	"log"
	"sync"

	"whaleisle/internal/desktop/editors"
	"whaleisle/internal/ui/files"
)

var probedEditors struct {
	mu   sync.Mutex
	apps []files.OpenWithApp
	done bool
}

// startEditorProbe detects the Electron editor list (editors.js) once, off the
// render path. Probing is file-system and PATH work only; it never spawns an
// editor. Doing it inline in the first frame walked PATH (a stat per extension
// per directory) while the window waited: tmp/stalls caught LookPath under
// mainView. Until it finishes, the list is empty.
func startEditorProbe() {
	go func() {
		apps := make([]files.OpenWithApp, 0, 4)
		for _, editor := range editors.Available(editors.DefaultProbe()) {
			apps = append(apps, files.OpenWithApp{ID: editor.ID, Name: editor.Label})
		}
		probedEditors.mu.Lock()
		probedEditors.apps = apps
		probedEditors.done = true
		probedEditors.mu.Unlock()
		state.mu.Lock()
		win := state.win
		state.mu.Unlock()
		if win != nil {
			win.Update(func() {})
		}
	}()
}

// editorOpenWithApps reports the probed editors for this frame. It never waits
// for the probe: a render path that blocked on it stalled the window.
func editorOpenWithApps() []files.OpenWithApp {
	probedEditors.mu.Lock()
	defer probedEditors.mu.Unlock()
	return probedEditors.apps
}

// openInEditor launches a probed external editor at line (0 opens the file
// only) and falls back to systemOpen when that editor is not installed or
// cannot be spawned, matching editors.js openInEditor + shell.openPath.
func openInEditor(absPath, editorID string, line int, systemOpen func(string) error) {
	if err := editors.Open(editorID, absPath, line, 0); err != nil {
		log.Println("open in editor:", err)
		if systemOpen != nil {
			if fallbackErr := systemOpen(absPath); fallbackErr != nil {
				log.Println("open with system default:", fallbackErr)
			}
		}
	}
}
