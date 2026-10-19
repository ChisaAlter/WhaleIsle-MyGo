package files

import (
	"encoding/json"

	"github.com/egoist/mygo/ui"
)

// Each tab owns its editable document independently of the currently painted
// Model. Native committed input is published immediately to TextBuffer, even
// when a hidden/minimized window will not build another frame.
func (t *FileTab) editableBuffer(text *string) *ui.TextBuffer {
	if t.editBuffer == nil {
		t.Draft = *text
		t.editBuffer = ui.NewTextBuffer(t.Draft)
		t.editBufferText, t.editModelText = t.Draft, t.Draft
		t.editBufferVersion = t.editBuffer.Version()
	} else {
		t.syncDraftModel(text)
	}
	return t.editBuffer
}

// Call on the UI owner before reading a draft. Explicit tab replacement (load,
// restore or reload) takes precedence over previously committed native input.
// Ordinary frames never Set the buffer: doing so would reset native undo/IME.
func (t *FileTab) syncNativeDraft() {
	if t.editBuffer == nil {
		return
	}
	if t.Draft != t.editBufferText {
		t.editBuffer.Set(t.Draft)
		t.editBufferText = t.Draft
		t.editBufferVersion = t.editBuffer.Version()
		return
	}
	snapshot := t.editBuffer.CommittedSnapshot()
	if snapshot.Version() != t.editBufferVersion {
		t.Draft = snapshot.String()
		t.editBufferText, t.editBufferVersion = t.Draft, snapshot.Version()
	}
}

// The last Model mirror is separate from the last buffer snapshot: Export may
// consume native input before the next frame copies it into the current Model.
// Never retain that Model pointer; it may later belong to another active tab.
func (t *FileTab) syncDraftModel(text *string) {
	if t.editBuffer == nil {
		t.Draft = *text
		return
	}
	if *text != t.editModelText {
		t.Draft = *text
	}
	t.syncNativeDraft()
	*text = t.Draft
	t.editModelText = t.Draft
}

// Nested pane recovery marshals FileTabs directly, while the primary Files
// surface uses Tabs.Export. Both must capture the same committed native text.
// Runtime buffers and marked (uncommitted) IME text are never serialized.
func (t *FileTab) MarshalJSON() ([]byte, error) {
	t.syncNativeDraft()
	type persistedFileTab FileTab
	return json.Marshal((*persistedFileTab)(t))
}
