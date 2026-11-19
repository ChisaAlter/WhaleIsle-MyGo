package directorypicker

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"whaleisle/internal/dsh/api"
)

// The Host exposes the browse browser over directoryPicker/list. A first
// workspace is chosen before that Host exists — it cannot start without a
// workspace — so the same level has to be browsable from this process instead
// of becoming a dead end. This file mirrors the Host's browse backend:
// directories and directory links only, dot-prefixed names flagged hidden,
// every level name-ascending and cut at 1000 rows.

const maxEntries = 1000

// driveProbe bounds one volume letter, as the Host's volume picker does: a
// polled floppy or a spinning optical drive must not hold the level open.
const driveProbe = 500 * time.Millisecond

// LocalList answers directoryPicker/list from this process's filesystem.
func LocalList(path string) (api.DirectoryListing, error) {
	home, _ := os.UserHomeDir()
	path = strings.TrimSpace(path)
	if path == "" {
		path = home
	} else if runtime.GOOS == "windows" && strings.EqualFold(strings.TrimRight(path, `\/`), VolumeRoot) {
		return volumeListing(home), nil
	}
	if !FullyQualified(path) {
		return api.DirectoryListing{}, &Error{Kind: KindUnreadable, Path: path, Err: errors.New("not a fully qualified path")}
	}
	target, err := filepath.Abs(path)
	if err != nil {
		return api.DirectoryListing{}, &Error{Kind: KindUnreadable, Path: path, Err: err}
	}
	entries, truncated, err := listDirectories(target)
	if err != nil {
		return api.DirectoryListing{}, &Error{Kind: KindUnreadable, Path: target, Err: err}
	}
	crumbs := ancestryCrumbs(target)
	if runtime.GOOS == "windows" {
		crumbs = append([]api.DirectoryListingCrumbsItem{{Name: VolumeRoot, Path: VolumeRoot}}, crumbs...)
	}
	return api.DirectoryListing{Path: target, Home: home, Crumbs: crumbs, Entries: entries, Truncated: truncated}, nil
}

// LocalCreateDirectory answers directoryPicker/createDirectory from this
// process, with the Host's rules: one plain segment under an existing parent.
func LocalCreateDirectory(parent, name string) (string, error) {
	parent = strings.TrimSpace(parent)
	if runtime.GOOS == "windows" && strings.EqualFold(strings.TrimRight(parent, `\/`), VolumeRoot) {
		return "", &Error{Kind: KindCreateFailed, Path: parent, Err: errors.New("not a filesystem directory")}
	}
	if !FullyQualified(parent) {
		return "", &Error{Kind: KindCreateFailed, Path: parent, Err: errors.New("not a fully qualified parent path")}
	}
	if strings.TrimSpace(name) == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return "", &Error{Kind: KindCreateFailed, Path: filepath.Join(parent, name), Err: errors.New("not a single path segment")}
	}
	target := filepath.Join(parent, name)
	if err := os.Mkdir(target, 0o755); err != nil {
		kind := KindCreateFailed
		if errors.Is(err, fs.ErrExist) {
			kind = KindExists
		}
		return "", &Error{Kind: kind, Path: target, Err: err}
	}
	return target, nil
}

func listDirectories(target string) ([]api.DirectoryListingCrumbsItem, bool, error) {
	dir, err := os.Open(target)
	if err != nil {
		return nil, false, err
	}
	defer dir.Close()
	names, err := dir.Readdirnames(-1)
	if err != nil {
		return nil, false, err
	}
	var found []string
	for _, name := range names {
		if _, ok := directoryRow(target, name); ok {
			found = append(found, name)
		}
	}
	sort.Slice(found, func(i, j int) bool {
		left, right := strings.ToLower(found[i]), strings.ToLower(found[j])
		if left != right {
			return left < right
		}
		return found[i] < found[j]
	})
	truncated := len(found) > maxEntries
	if truncated {
		found = found[:maxEntries]
	}
	entries := make([]api.DirectoryListingCrumbsItem, 0, len(found))
	for _, name := range found {
		entries = append(entries, api.DirectoryListingCrumbsItem{Name: name, Path: filepath.Join(target, name), Hidden: strings.HasPrefix(name, ".")})
	}
	return entries, truncated, nil
}

// directoryRow follows a link to a directory; anything else, and a link that
// cannot be entered, is not a row the browser can open.
func directoryRow(parent, name string) (api.DirectoryListingCrumbsItem, bool) {
	path := filepath.Join(parent, name)
	st, err := os.Stat(path)
	if err != nil || !st.IsDir() {
		return api.DirectoryListingCrumbsItem{}, false
	}
	return api.DirectoryListingCrumbsItem{Name: name, Path: path, Hidden: strings.HasPrefix(name, ".")}, true
}

// ancestryCrumbs is the Host's: every ancestor from the filesystem root to the
// target inclusive, each one a jump target.
func ancestryCrumbs(target string) []api.DirectoryListingCrumbsItem {
	var crumbs []api.DirectoryListingCrumbsItem
	for current := target; ; {
		parent := filepath.Dir(current)
		name := filepath.Base(current)
		if parent == current {
			name = current
		}
		crumbs = append([]api.DirectoryListingCrumbsItem{{Name: name, Path: current}}, crumbs...)
		if parent == current {
			return crumbs
		}
		current = parent
	}
}

// volumeListing is the Host's Win32 volume level: one row per enterable drive.
func volumeListing(home string) api.DirectoryListing {
	type probe struct {
		row api.DirectoryListingCrumbsItem
		ok  bool
	}
	replies := make(chan probe, 26)
	for ch := byte('A'); ch <= 'Z'; ch++ {
		go func(letter byte) {
			root := string(letter) + `:\`
			st, err := os.Stat(root)
			replies <- probe{api.DirectoryListingCrumbsItem{Name: string(letter) + ":", Path: root}, err == nil && st.IsDir()}
		}(ch)
	}
	entries := make([]api.DirectoryListingCrumbsItem, 0, 26)
	deadline := time.After(driveProbe)
collect:
	for range [26]struct{}{} {
		select {
		case reply := <-replies:
			if reply.ok {
				entries = append(entries, reply.row)
			}
		case <-deadline:
			break collect
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return api.DirectoryListing{Path: VolumeRoot, Home: home, Crumbs: []api.DirectoryListingCrumbsItem{{Name: VolumeRoot, Path: VolumeRoot}}, Entries: entries}
}
