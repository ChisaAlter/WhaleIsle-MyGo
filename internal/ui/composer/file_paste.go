package composer

import (
	"errors"
	"path/filepath"
	"strings"
	"whaleisle/internal/errcopy"

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
		setFileIntakeError(m, "无法读取剪贴板文件 / Unable to read clipboard files")
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
		setFileIntakeError(m, "请等待附件准备完成 / Wait for attachment preparation")
		return
	}
	files, err := normalizedTransferPaths(paths)
	if err != nil {
		setFileIntakeError(m, errcopy.Sentence(err))
		return
	}
	if m.OnAttachFiles != nil {
		m.OnAttachFiles(files)
	} else if m.Draft != nil {
		AppendDroppedFiles(m.Draft, strings.TrimSpace(m.WorkspaceRoot), files)
	}
}

func normalizedTransferPaths(paths []string) ([]string, error) {
	if len(paths) > 128 {
		return nil, errors.New("一次最多粘贴或拖入128个文件 / At most 128 files per paste or drop")
	}
	files := make([]string, 0, len(paths))
	seen := map[string]bool{}
	for _, path := range paths {
		if !filepath.IsAbs(path) || strings.ContainsRune(path, 0) {
			return nil, errors.New("文件路径无效 / Invalid file path")
		}
		path = filepath.Clean(path)
		if !seen[path] {
			seen[path] = true
			files = append(files, path)
		}
	}
	return files, nil
}
