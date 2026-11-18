package paths

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// DataLocationProblem classifies why a chosen data directory was refused. The
// settings surface names the reason; this package only decides it, so the copy
// stays in the i18n catalogue.
type DataLocationProblem string

// Refusal reasons for a chosen data directory.
const (
	// DataLocationEmpty means the choice was blank.
	DataLocationEmpty DataLocationProblem = "empty"
	// DataLocationRelative means the choice was not an absolute path.
	DataLocationRelative DataLocationProblem = "relative"
	// DataLocationNotDirectory means the choice names an existing file.
	DataLocationNotDirectory DataLocationProblem = "not-directory"
	// DataLocationInvalid means the path could not be resolved (a malformed
	// path, a drive that cannot be queried).
	DataLocationInvalid DataLocationProblem = "invalid"
	// DataLocationRemoteVolume means the choice is on a network drive.
	DataLocationRemoteVolume DataLocationProblem = "remote-volume"
	// DataLocationNotWritable means the choice cannot hold a profile: creating
	// the directory failed, or a probe file could not be written in it.
	DataLocationNotWritable DataLocationProblem = "not-writable"
	// DataLocationNested means one of the choice and the current directory
	// contains the other, so moving into it would recurse.
	DataLocationNested DataLocationProblem = "nested"
	// DataLocationInsideInstall means the choice overlaps the directory the
	// application is installed in.
	DataLocationInsideInstall DataLocationProblem = "inside-install"
	// DataLocationContainsRecord means the choice would contain the
	// data-location record, which cannot name a directory it lives in.
	DataLocationContainsRecord DataLocationProblem = "contains-record"
)

// DataLocationError is a refused data directory choice.
type DataLocationError struct {
	Problem DataLocationProblem
	Path    string
	// Other is the path the choice conflicted with, when there is one.
	Other string
}

func (e *DataLocationError) Error() string {
	if e.Other != "" {
		return fmt.Sprintf("data location %q refused (%s: %s)", e.Path, e.Problem, e.Other)
	}
	return fmt.Sprintf("data location %q refused (%s)", e.Path, e.Problem)
}

// ValidateDataLocationChoice decides whether chosen may replace current as the
// user data directory. installDir is the directory the running executable
// lives in; the record must stay outside the chosen directory, so ConfigRoot
// and everything under it is refused only when the choice would contain it.
//
// The choice is not required to exist: the directory is created when the
// change is applied. A choice that already exists as a file is refused, so the
// failure names the reason instead of surfacing later as a write error.
func ValidateDataLocationChoice(chosen, current, installDir string) error {
	raw := strings.TrimSpace(chosen)
	if raw == "" {
		return &DataLocationError{Problem: DataLocationEmpty}
	}
	if !filepath.IsAbs(raw) {
		return &DataLocationError{Problem: DataLocationRelative, Path: raw}
	}
	dir, err := canonicalForCompare(raw)
	if err != nil {
		return &DataLocationError{Problem: DataLocationInvalid, Path: raw, Other: err.Error()}
	}
	if info, statErr := os.Stat(dir); statErr == nil && !info.IsDir() {
		return &DataLocationError{Problem: DataLocationNotDirectory, Path: dir}
	}
	// A network volume is refused before the write probe touches it: opening a
	// path on a dropped share is what would block.
	if remoteVolume(dir) {
		return &DataLocationError{Problem: DataLocationRemoteVolume, Path: dir}
	}

	for _, conflict := range []struct {
		problem DataLocationProblem
		path    string
		nested  bool
	}{
		{DataLocationNested, current, true},
		{DataLocationInsideInstall, installDir, true},
		// The record has to stay outside the data directory it names, so a
		// choice that would contain it is refused; a choice inside ConfigRoot
		// is fine, which is where the default lives.
		{DataLocationContainsRecord, ConfigRoot(), false},
	} {
		other, ok := canonicalOrEmpty(conflict.path)
		if !ok {
			continue
		}
		if insideOrSame(dir, other) || conflict.nested && insideOrSame(other, dir) {
			return &DataLocationError{Problem: conflict.problem, Path: dir, Other: other}
		}
	}
	// Accepting a directory the profile cannot be written to would only move
	// the failure to the next launch, where the app starts in a location it
	// cannot fill.
	if err := probeDir(dir); err != nil {
		return &DataLocationError{Problem: DataLocationNotWritable, Path: dir, Other: err.Error()}
	}
	return nil
}

// remoteVolume reports whether a path is on a network drive. Tests
// substitute it so the refusal can be exercised on any machine.
var remoteVolume = volumeIsRemote

// probeDir is the write probe the validator and the startup check share.
// Tests substitute it to exercise a directory that refuses writes without
// depending on the permissions of the machine running them.
var probeDir = probeWritable

// probeWritable creates the directory when it is not there yet and writes one
// temporary file into it, so a read-only volume, a denied ACL or a cloud
// placeholder is refused here rather than after the relaunch.
//
// A missing directory is created and left in place: it is where the profile is
// about to live, and the change creates it anyway. Only the probe file is
// removed, and a failed removal is reported because a directory that keeps
// refusing deletion is not a healthy profile directory either.
func probeWritable(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	probe, err := os.CreateTemp(dir, ".whaleisle-write-probe-*")
	if err != nil {
		return err
	}
	name := probe.Name()
	if closeErr := probe.Close(); closeErr != nil {
		return errors.Join(closeErr, os.Remove(name))
	}
	return os.Remove(name)
}

func canonicalOrEmpty(path string) (string, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", false
	}
	canonical, err := canonicalForCompare(path)
	if err != nil {
		return "", false
	}
	return canonical, true
}

// canonicalForCompare resolves links over the longest existing prefix, so a
// not-yet-created directory is compared against its real parent.
func canonicalForCompare(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	dir := filepath.Clean(abs)
	var tail []string
	for {
		if resolved, err := filepath.EvalSymlinks(dir); err == nil {
			for i := len(tail) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, tail[i])
			}
			return filepath.Clean(resolved), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return dir, nil
		}
		tail = append(tail, filepath.Base(dir))
		dir = parent
	}
}

// insideOrSame reports whether candidate is root itself or below it.
func insideOrSame(root, candidate string) bool {
	root, candidate = comparablePath(root), comparablePath(candidate)
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// comparablePath folds the path the way the platform compares it: the default
// Windows and macOS file systems are case-insensitive, Linux is not.
func comparablePath(path string) string {
	clean := filepath.Clean(path)
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return strings.ToLower(clean)
	}
	return clean
}
