package composer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// Validate the whole drop/paste before invoking the existing session-owned
// intake. Explicit folder references use their separate picker/reference path.
func intakeTransferredFiles(m Model, paths []string) {
	if m.State != nil && (m.State.Sending || m.State.Preparing > 0) {
		setFileIntakeError(m, "请等待附件准备完成 / Wait for attachment preparation")
		return
	}
	files, err := regularTransferFiles(paths)
	if err != nil {
		setFileIntakeError(m, err.Error())
		return
	}
	if m.OnAttachFiles != nil {
		m.OnAttachFiles(files)
	} else if m.Draft != nil {
		AppendDroppedFiles(m.Draft, strings.TrimSpace(m.WorkspaceRoot), files)
	}
}

func regularTransferFiles(paths []string) ([]string, error) {
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
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("无法读取文件 %s / File unavailable", filepath.Base(path))
		}
		if !info.Mode().IsRegular() {
			return nil, errors.New("粘贴或拖入仅接受文件；文件夹请使用显式引用入口 / Paste or drop accepts files only; use the folder reference picker")
		}
		if !seen[path] {
			seen[path] = true
			files = append(files, path)
		}
	}
	return files, nil
}
