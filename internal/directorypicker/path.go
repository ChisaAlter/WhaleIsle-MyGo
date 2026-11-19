// Package directorypicker distinguishes filesystem paths from the host's virtual volume root.
package directorypicker

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// VolumeRoot is the virtual chooser that lists the machine's drives. It is
// never a workspace: only a level the browser passes through.
const VolumeRoot = `\\.\dsh-computer`

// SelectablePath excludes the virtual Windows volume chooser from workspace
// selection and directory creation. This does not probe or enumerate any disk.
func SelectablePath(path string) bool {
	path = strings.TrimSpace(path)
	normalized := strings.TrimRight(strings.ReplaceAll(path, "/", `\`), `\`)
	return path != "" && !strings.EqualFold(normalized, VolumeRoot)
}

// qualified is the Host's fullyQualified: a Windows path that names one fixed
// location regardless of the process's current drive.
var qualified = regexp.MustCompile(`^(?:[A-Za-z]:[\\/]|[\\/]{2}[^\\/]+[\\/]+[^\\/]+)`)

func FullyQualified(path string) bool {
	if runtime.GOOS != "windows" {
		return filepath.IsAbs(path)
	}
	return filepath.IsAbs(path) && qualified.MatchString(path)
}

// UsableDirectory resolves a picked folder to the absolute path a workspace can
// be committed under, without a Host: a first workspace has to be persisted
// before the Host that would validate it can start.
func UsableDirectory(path string) (string, error) {
	path = strings.TrimSpace(path)
	if !SelectablePath(path) || !FullyQualified(path) {
		return "", &Error{Kind: KindUnreadable, Path: path, Err: errors.New("not a selectable absolute path")}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", &Error{Kind: KindUnreadable, Path: path, Err: err}
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", &Error{Kind: KindUnreadable, Path: abs, Err: err}
	}
	if !st.IsDir() {
		return "", &Error{Kind: KindUnreadable, Path: abs, Err: errors.New("not a directory")}
	}
	return abs, nil
}

// Directory failure kinds, mirroring the Host's DirectoryPickerError codes so
// the copy classifier has one vocabulary for both backends.
const (
	KindUnreadable   = "directory-unreadable"
	KindExists       = "directory-exists"
	KindCreateFailed = "directory-create-failed"
)

// Error is a browse failure that names the operation, the path and the cause.
type Error struct {
	Kind string
	Path string
	Err  error
}

func (e *Error) Error() string { return e.Kind + ": " + e.Path + ": " + e.Err.Error() }

func (e *Error) Unwrap() error { return e.Err }
