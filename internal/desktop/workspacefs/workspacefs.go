// Package workspacefs reads the local workspace directory tree for M7 file UI.
package workspacefs

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
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

// ListDir reads immediate children of dirPath. Hidden dot entries are omitted.
func ListDir(dirPath string) ([]Entry, error) {
	dirPath = filepath.Clean(dirPath)
	ents, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(ents))
	for _, e := range ents {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		out = append(out, Entry{Name: name, IsDir: e.IsDir()})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
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
	tmp := filePath + ".whaleisle-tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, filePath); err != nil {
		_ = os.Remove(tmp)
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

// JoinRoot joins root with a slash-relative rel path and ensures it stays under root.
func JoinRoot(root, rel string) (string, error) {
	root = filepath.Clean(root)
	rel = strings.TrimPrefix(filepath.ToSlash(rel), "/")
	if rel == "" || rel == "." {
		return root, nil
	}
	if strings.Contains(rel, "..") {
		return "", fmt.Errorf("workspacefs: invalid relative path")
	}
	full := filepath.Clean(filepath.Join(root, filepath.FromSlash(rel)))
	rootPrefix := root + string(os.PathSeparator)
	if full != root && !strings.HasPrefix(full, rootPrefix) {
		return "", fmt.Errorf("workspacefs: path outside root")
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
