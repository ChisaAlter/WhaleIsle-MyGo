// Package workspacefs reads the local workspace directory tree for M7 file UI.
package workspacefs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// DefaultMaxReadBytes is the maximum file size read for text preview.
const DefaultMaxReadBytes = 512 * 1024

// Entry is one directory listing item (name relative to the listed directory).
type Entry struct {
	Name  string
	IsDir bool
}

// ListDir lists immediate children using the original visibility policy.
// Call ListWorkspaceDir when the workspace root and relative path are known.
func ListDir(dirPath string) ([]Entry, error) {
	return ListWorkspaceDir(context.Background(), dirPath, "")
}

// ReadTextResult is the outcome of ReadFileText.
type ReadTextResult struct {
	Text      string
	Truncated bool
	Size      int64
}

var (
	// ErrNotFile is returned when path is not a regular file.
	ErrNotFile = errors.New("workspacefs: not a file")
	// ErrTooLarge is returned when the file exceeds maxBytes before read.
	ErrTooLarge = errors.New("workspacefs: file too large")
	// ErrBinary is returned when the file looks binary (NUL byte in sample).
	ErrBinary = errors.New("workspacefs: binary file")
)

// ReadFileText reads up to maxBytes of a UTF-8 text file at filePath.
// When maxBytes is zero, DefaultMaxReadBytes is used.
func ReadFileText(filePath string, maxBytes int64) (ReadTextResult, error) {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxReadBytes
	}
	filePath = filepath.Clean(filePath)
	st, err := os.Stat(filePath)
	if err != nil {
		return ReadTextResult{}, err
	}
	if st.IsDir() {
		return ReadTextResult{}, ErrNotFile
	}
	if !st.Mode().IsRegular() {
		return ReadTextResult{}, ErrNotFile
	}
	size := st.Size()
	if size > maxBytes {
		return ReadTextResult{Size: size}, ErrTooLarge
	}
	f, err := os.Open(filePath)
	if err != nil {
		return ReadTextResult{}, err
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		return ReadTextResult{}, err
	}
	if int64(len(data)) > maxBytes {
		return ReadTextResult{Size: size}, ErrTooLarge
	}
	if bytesContainsNUL(data) {
		return ReadTextResult{Size: size}, ErrBinary
	}
	if !utf8.Valid(data) {
		return ReadTextResult{Size: size}, ErrBinary
	}
	return ReadTextResult{
		Text:      string(data),
		Truncated: false,
		Size:      size,
	}, nil
}

// WriteFileText writes UTF-8 text to filePath under the workspace (caller must
// JoinRoot first). Refuses binary-looking content (NUL) and empty path.
func WriteFileText(filePath, text string) error {
	filePath = filepath.Clean(filePath)
	if filePath == "" || filePath == "." {
		return fmt.Errorf("workspacefs: empty path")
	}
	data := []byte(text)
	if bytesContainsNUL(data) {
		return ErrBinary
	}
	if !utf8.Valid(data) {
		return ErrBinary
	}
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if st, err := os.Lstat(filePath); err == nil {
		if !st.Mode().IsRegular() {
			return ErrNotFile
		}
		mode = st.Mode().Perm()
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	f, err := os.CreateTemp(dir, ".whaleisle-save-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err = f.Chmod(mode); err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = os.Rename(tmp, filePath); err != nil {
		return err
	}
	return nil
}

// RelPath returns path relative to root, using slash separators for UI keys.
func RelPath(root, path string) (string, error) {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("workspacefs: path outside root")
	}
	return filepath.ToSlash(rel), nil
}

// JoinRoot validates both lexical and existing symlink ancestry. The root is
// the authorized boundary; .git metadata and aliases to it are never allowed.
func JoinRoot(root, rel string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("workspacefs: no workspace root")
	}
	if err := ValidateRelativePath(rel); err != nil {
		return "", err
	}
	root = filepath.Clean(root)
	rel = filepath.ToSlash(rel)
	full := filepath.Clean(filepath.Join(root, filepath.FromSlash(rel)))
	if _, err := RelPath(root, full); err != nil {
		return "", err
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	// Nonexistent leaf names are allowed for saves. Resolve the nearest existing
	// ancestor so a symlinked parent cannot escape the workspace on creation.
	node := full
	for {
		real, resolveErr := filepath.EvalSymlinks(node)
		if resolveErr == nil {
			relative, boundaryErr := RelPath(realRoot, real)
			if boundaryErr != nil {
				return "", boundaryErr
			}
			for _, part := range strings.Split(relative, "/") {
				if strings.EqualFold(part, ".git") {
					return "", fmt.Errorf("workspacefs: git metadata is outside file access")
				}
			}
			break
		}
		if !errors.Is(resolveErr, os.ErrNotExist) {
			return "", resolveErr
		}
		// Broken links must not become writable paths after ancestor fallback.
		if _, statErr := os.Lstat(node); statErr == nil {
			return "", resolveErr
		}
		parent := filepath.Dir(node)
		if node == root || parent == node {
			return "", resolveErr
		}
		node = parent
	}
	return full, nil
}

func bytesContainsNUL(b []byte) bool {
	for _, c := range b {
		if c == 0 {
			return true
		}
	}
	return false
}

// IsNotExist reports whether err is fs.ErrNotExist or wraps it.
func IsNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}

// ValidateRelativePath checks recovered/offline tab names without touching disk.
// The complete canonical authority check still occurs before each local access.
func ValidateRelativePath(rel string) error {
	rel = filepath.ToSlash(rel)
	if filepath.IsAbs(rel) || filepath.VolumeName(rel) != "" || strings.ContainsAny(rel, "\\\x00") {
		return fmt.Errorf("workspacefs: invalid relative path")
	}
	for _, part := range strings.Split(rel, "/") {
		if part == ".." || strings.EqualFold(part, ".git") {
			return fmt.Errorf("workspacefs: invalid relative path")
		}
	}
	return nil
}
