package main

import (
	"testing"
	"whaleisle/internal/ui/files"
)

func TestStartupGlobalFileEditorUsesSingleEditor(t *testing.T) {
	if state.fileEditViewMode != files.EditViewEdit {
		t.Fatalf("production startup file mode = %v", state.fileEditViewMode)
	}
}
