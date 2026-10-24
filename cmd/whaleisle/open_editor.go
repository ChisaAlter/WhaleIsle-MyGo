package main

import (
	"log"
	"sync"

	"whaleisle/internal/desktop/editors"
	"whaleisle/internal/ui/files"
)

var probedEditors struct {
	once sync.Once
	apps []files.OpenWithApp
}

// editorOpenWithApps probes the Electron editor list (editors.js) once per
// process. Probing is file-system and PATH work only; it never spawns an editor.
func editorOpenWithApps() []files.OpenWithApp {
	probedEditors.once.Do(func() {
		for _, editor := range editors.Available(editors.DefaultProbe()) {
			probedEditors.apps = append(probedEditors.apps, files.OpenWithApp{ID: editor.ID, Name: editor.Label})
		}
	})
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
