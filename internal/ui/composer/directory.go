package composer

import (
	"errors"
	"strings"

	"whaleisle/internal/desktop/workspacefs"
)

// ErrOutsideWorkspace is returned when a picked path is not under the workspace root.
var ErrOutsideWorkspace = errors.New("composer: path outside workspace")

// DirectoryMention returns "@rel/path" for absPath under root (slash separators, no trailing space).
func DirectoryMention(root, absPath string) (string, error) {
	root = strings.TrimSpace(root)
	absPath = strings.TrimSpace(absPath)
	if root == "" || absPath == "" {
		return "", ErrOutsideWorkspace
	}
	rel, err := workspacefs.RelPath(root, absPath)
	if err != nil || rel == "" {
		return "", ErrOutsideWorkspace
	}
	return "@" + rel, nil
}

// InsertDirectoryMention appends "@rel/path " into draft for a directory under root.
func InsertDirectoryMention(draft *string, root, absPath string) error {
	mention, err := DirectoryMention(root, absPath)
	if err != nil {
		return err
	}
	if draft == nil {
		return nil
	}
	token := mention + " "
	cur := strings.TrimRight(*draft, " \t")
	if cur == "" {
		*draft = token
		return nil
	}
	*draft = cur + " " + token
	return nil
}
