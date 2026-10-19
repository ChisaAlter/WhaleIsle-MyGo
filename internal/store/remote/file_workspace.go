package remote

import (
	"context"
	"net/url"
	"strings"
	"time"

	"whaleisle/internal/dsh"
)

// FileLevel is one retained expanded directory, including a retryable soft
// error. It is owned by the session mirror, never by the selected machine.
type FileLevel struct {
	Items                    []DirectoryEntry
	Loaded, Loading, Missing bool
	Error                    string
}

type FileResource struct {
	draftOwner                             *fileDraftOwner
	Path, Draft, Error, Notice, Downloaded string
	File                                   RemoteFile
	Loaded, Stale, Editing                 bool
	Revision                               uint64
}

func (r FileResource) Dirty() bool { return r.Loaded && r.Draft != r.File.Content }

func (s *Files) HasDirty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncDraftsLocked()
	for _, view := range s.states {
		if view.Dirty() {
			return true
		}
		for _, resource := range view.Tabs {
			if resource.Dirty() {
				return true
			}
		}
	}
	return false
}

// Hiding a native occurrence retires its pending work without throwing away
// cached levels or any resource draft. It does not disconnect an SSH machine.
func (s *Files) Suspend() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncDraftsLocked()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	if s.resolveCancel != nil {
		s.resolveCancel()
		s.resolveCancel = nil
	}
	s.generation++
	if v := s.states[s.key]; v != nil {
		if v.Busy && (v.Operation == "write" || isFileMutation(v.Operation)) {
			v.Stale = true
			v.Error = "Remote operation interrupted; its result is uncertain. Draft retained."
			stashResource(v)
		}
		v.Busy, v.Resolving = false, false
		for path, level := range v.Tree {
			level.Loading = false
			v.Tree[path] = level
		}
	}
}

func ResourceAddress(sessionID, path string) string {
	return "dsh-resource://dsh-remote/" + url.PathEscape(sessionID) + "/" + url.PathEscape(path)
}

func cloneFileTree(tree map[string]FileLevel) map[string]FileLevel {
	if tree == nil {
		return nil
	}
	copy := make(map[string]FileLevel, len(tree))
	for path, level := range tree {
		level.Items = append([]DirectoryEntry(nil), level.Items...)
		copy[path] = level
	}
	return copy
}

func resourceIndex(v *FileSnapshot, path string) int {
	for i := range v.Tabs {
		if v.Tabs[i].Path == path {
			return i
		}
	}
	return -1
}

func stashResource(v *FileSnapshot) {
	if v.FilePath == "" {
		return
	}
	i := resourceIndex(v, v.FilePath)
	if i < 0 {
		v.Tabs = append(v.Tabs, FileResource{Path: v.FilePath})
		i = len(v.Tabs) - 1
	}
	r := &v.Tabs[i]
	if r.draftOwner == nil {
		r.draftOwner = &fileDraftOwner{}
	}
	if r.draftOwner.buffer != nil {
		v.Draft = r.draftOwner.buffer.CommittedText()
	}
	r.Draft, r.Error, r.Notice, r.Downloaded = v.Draft, v.Error, v.Notice, v.Downloaded
	r.File, r.Loaded, r.Stale, r.Revision = v.File, v.FileLoaded, v.Stale, v.Revision
}

func restoreResource(v *FileSnapshot, r FileResource) {
	v.FilePath, v.Draft, v.Error, v.Notice, v.Downloaded = r.Path, r.Draft, r.Error, r.Notice, r.Downloaded
	v.File, v.FileLoaded, v.Stale, v.Revision = r.File, r.Loaded, r.Stale, r.Revision+1
}

func knownTreeEntry(v *FileSnapshot, path string, directory bool) bool {
	if directory && path == v.RootPath {
		return true
	}
	for _, level := range v.Tree {
		if !level.Loaded {
			continue
		}
		for _, entry := range level.Items {
			if entry.Path == path && (directory || !entry.IsDirectory()) {
				return true
			}
		}
	}
	return false
}

func knownTreeDirectory(v *FileSnapshot, path string) bool {
	if path == v.RootPath {
		return true
	}
	for _, level := range v.Tree {
		if level.Loaded {
			for _, entry := range level.Items {
				if entry.Path == path && entry.IsDirectory() {
					return true
				}
			}
		}
	}
	return false
}

// BindSession adds the selected cwd and selection-epoch authority to Bind.
// Each pane calls this on its own Files instance; no global resource draft is
// shared between native workspace occurrences.
func (s *Files) BindSession(c *dsh.Client, sessionID, cwd string, current func() bool, notify func()) {
	s.Bind(c, sessionID, notify)
	s.mu.Lock()
	retired := s.current != nil && !s.current()
	s.current = current
	v := s.states[s.key]
	if v != nil && (v.Cwd != cwd || retired) {
		if s.cancel != nil {
			s.cancel()
			s.cancel = nil
		}
		if s.resolveCancel != nil {
			s.resolveCancel()
			s.resolveCancel = nil
		}
		s.generation++
		stashResource(v)
		v.Cwd, v.RootPath, v.MirrorDir, v.ResolveError = cwd, "", "", ""
		v.Resolved, v.Resolving, v.Local, v.Bound, v.Loaded, v.Busy = false, false, false, false, false, false
		v.Tree, v.Expanded = nil, nil
		v.PendingClose = ""
		v.Stale = v.FileLoaded
		for i := range v.Tabs {
			v.Tabs[i].Stale = v.Tabs[i].Loaded
		}
	}
	s.mu.Unlock()
}

// Resolve asks the same original mirror producer as RemoteExplorerBody. An
// unknown or local session never claims the ordinary native files pane.
func (s *Files) Resolve() {
	s.mu.Lock()
	v := s.states[s.key]
	if v == nil || s.client == nil || v.Resolving || s.current != nil && !s.current() {
		s.mu.Unlock()
		return
	}
	v.Resolving, v.ResolveError = true, ""
	c, key, generation, owner, notify := s.client, s.key, s.generation, s.current, s.notify
	sid := v.SessionID
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	s.resolveCancel = cancel
	s.mu.Unlock()
	go func() {
		defer cancel()
		var result struct {
			RemotePath string `json:"remotePath"`
			MirrorDir  string `json:"mirrorDir"`
			Mode       string `json:"mode"`
		}
		err := request(ctx, c, "GET", "resolve-mirror?sessionId="+url.QueryEscape(sid), nil, &result)
		s.mu.Lock()
		if s.client != c || s.key != key || s.generation != generation || owner != nil && !owner() {
			s.mu.Unlock()
			return
		}
		v = s.states[key]
		v.Resolving = false
		if err != nil {
			v.ResolveError = err.Error()
		} else {
			v.Resolved, v.Local = true, result.Mode != "remote" || result.RemotePath == ""
			v.RootPath, v.MirrorDir = result.RemotePath, result.MirrorDir
		}
		remote, root := err == nil && !v.Local, v.RootPath
		s.mu.Unlock()
		if notify != nil {
			notify()
		}
		if remote {
			s.Browse(root)
		}
	}()
}

func (s *Files) ToggleDirectory(path string) {
	s.mu.Lock()
	v := s.states[s.key]
	if v == nil || v.Busy || !v.Bound || !knownTreeDirectory(v, path) {
		s.mu.Unlock()
		return
	}
	if v.Expanded == nil {
		v.Expanded = map[string]bool{}
	}
	v.Expanded[path] = !v.Expanded[path]
	// Reopening an expanded level refreshes its retained entries, as the original tree does.
	load := v.Expanded[path]
	n := s.notify
	s.mu.Unlock()
	if load {
		s.Browse(path)
	} else if n != nil {
		n()
	}
}

func (s *Files) openResource(path string) {
	s.mu.Lock()
	s.syncDraftsLocked()
	v := s.states[s.key]
	if v == nil || v.Busy {
		s.mu.Unlock()
		return
	}
	i := resourceIndex(v, path)
	if i < 0 && !v.Bound {
		s.mu.Unlock()
		return
	}
	if i < 0 && !knownTreeEntry(v, path, false) {
		// Legacy panel callers may have a listing before the tree is mounted.
		known := false
		for _, entry := range v.Items {
			if entry.Path == path && !entry.IsDirectory() {
				known = true
				break
			}
		}
		if !known {
			s.mu.Unlock()
			return
		}
	}
	stashResource(v)
	i = resourceIndex(v, path)
	if i < 0 {
		v.Tabs = append(v.Tabs, FileResource{Path: path})
		i = len(v.Tabs) - 1
	}
	r := v.Tabs[i]
	restoreResource(v, r)
	n := s.notify
	s.mu.Unlock()
	if !r.Loaded {
		s.run("read", path)
	} else if n != nil {
		n()
	}
}

func (s *Files) SetEditing(editing bool) {
	s.mu.Lock()
	s.syncDraftsLocked()
	if v := s.states[s.key]; v != nil && !v.Busy {
		stashResource(v)
		if i := resourceIndex(v, v.FilePath); i >= 0 {
			v.Tabs[i].Editing = editing
		}
	}
	s.mu.Unlock()
}

func closeResource(v *FileSnapshot, path string) {
	i := resourceIndex(v, path)
	if i < 0 {
		return
	}
	v.Tabs = append(v.Tabs[:i], v.Tabs[i+1:]...)
	if v.FilePath == path {
		next := FileResource{}
		if len(v.Tabs) > 0 {
			if i >= len(v.Tabs) {
				i = len(v.Tabs) - 1
			}
			next = v.Tabs[i]
		}
		restoreResource(v, next)
	}
	v.PendingClose = ""
}

func (s *Files) RequestClose(path string) {
	s.mu.Lock()
	s.syncDraftsLocked()
	if v := s.states[s.key]; v != nil && !v.Busy {
		stashResource(v)
		if i := resourceIndex(v, path); i >= 0 {
			if v.Tabs[i].Dirty() {
				v.PendingClose = path
			} else {
				closeResource(v, path)
			}
		}
	}
	n := s.notify
	s.mu.Unlock()
	if n != nil {
		n()
	}
}

func (s *Files) ConfirmClose(discard bool) {
	s.mu.Lock()
	if v := s.states[s.key]; v != nil && !v.Busy {
		if discard {
			closeResource(v, v.PendingClose)
		} else {
			v.PendingClose = ""
		}
	}
	n := s.notify
	s.mu.Unlock()
	if n != nil {
		n()
	}
}

func (s *Files) DownloadEntry(path string) { s.run("download-entry", path) }

func pathAffected(path, parent string) bool {
	path, parent = strings.ReplaceAll(path, "\\", "/"), strings.TrimRight(strings.ReplaceAll(parent, "\\", "/"), "/")
	return path == parent || strings.HasPrefix(path, parent+"/")
}
