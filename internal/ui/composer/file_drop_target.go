package composer

// NativeFileDropTarget is implemented by a MyGo element. Its native file event
// channel is distinct from application-local, typed Dock drag payloads.
type NativeFileDropTarget interface {
	DroppedFiles() []string
	FileDragOver() bool
}

// FileDropTarget extends the same paste/drop intake beyond the input field.
// The caller owns the target's session identity and modal hit-test scope.
func FileDropTarget(target NativeFileDropTarget, m Model) bool {
	if m.Draft == nil {
		return false
	}
	if paths := target.DroppedFiles(); len(paths) > 0 {
		intakeTransferredFiles(m, paths)
	}
	return target.FileDragOver()
}
