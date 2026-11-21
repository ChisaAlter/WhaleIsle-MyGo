package composer

import (
	"errors"
	"path/filepath"
	"strings"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo/transfer"
)

// readFiles is a native file-list reader, never a text-to-path heuristic.
// true consumes the paste, including invalid files, to prevent alternate text
// or image representations of the same clipboard item entering the draft.
func pasteNativeFiles(m Model, readFiles func() ([]string, error)) bool {
	paths, err := readFiles()
	if errors.Is(err, transfer.ErrFormat) {
		return false
	}
	if err != nil {
		setFileIntakeError(m, i18n.T(i18n.KeyComposerFilePasteReadFailed))
		return true
	}
	if len(paths) == 0 {
		return false
	}
	intakeTransferredFiles(m, paths)
	return true
}

func setFileIntakeError(m Model, text string) {
	if m.State != nil {
		m.State.Error = text
	}
}

// Validate only the native path representation here. The session-owned intake
// checks the file system off the UI thread before applying the complete batch.
func intakeTransferredFiles(m Model, paths []string) {
	if m.State != nil && (m.State.Sending || m.State.Preparing > 0) {
		setFileIntakeError(m, i18n.T(i18n.KeyComposerFileIntakeWaitPreparing))
		return
	}
	files, err := normalizedTransferPaths(paths)
	if err != nil {
		switch {
		case errors.Is(err, errTooManyIntakeFiles):
			setFileIntakeError(m, i18n.T(i18n.KeyComposerFileIntakeTooMany))
		case errors.Is(err, errInvalidIntakePath):
			setFileIntakeError(m, i18n.T(i18n.KeyComposerFileIntakeInvalidPath))
		default:
			setFileIntakeError(m, errcopy.Sentence(err))
		}
		return
	}
	if m.OnAttachFiles != nil {
		m.OnAttachFiles(files)
	} else if m.Draft != nil {
		AppendDroppedFiles(m.Draft, strings.TrimSpace(m.WorkspaceRoot), files)
	}
}

// Intake validation failures name their cause for the log; the intake surface
// translates each sentinel into the localized sentence the person reads.
var (
	errTooManyIntakeFiles = errors.New("composer intake: more than 128 files")
	errInvalidIntakePath  = errors.New("composer intake: invalid file path")
)

func normalizedTransferPaths(paths []string) ([]string, error) {
	if len(paths) > 128 {
		return nil, errTooManyIntakeFiles
	}
	files := make([]string, 0, len(paths))
	seen := map[string]bool{}
	for _, path := range paths {
		if !filepath.IsAbs(path) || strings.ContainsRune(path, 0) {
			return nil, errInvalidIntakePath
		}
		path = filepath.Clean(path)
		if !seen[path] {
			seen[path] = true
			files = append(files, path)
		}
	}
	return files, nil
}
