package remote

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

func isFileMutation(op string) bool { return op == "mkdir" || op == "rename" || op == "remove" }

func remoteChild(parent, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\:\x00\r\n") {
		return "", fmt.Errorf("Enter one file or folder name without path separators")
	}
	parent = strings.TrimSpace(parent)
	if parent == "" {
		return "", fmt.Errorf("Browse a remote directory first")
	}
	sep := "/"
	if strings.Contains(parent, "\\") || len(parent) >= 2 && parent[1] == ':' {
		sep = "\\"
	}
	return strings.TrimRight(parent, "/\\") + sep + name, nil
}

func (s *Files) Mkdir(name string) error         { return s.mutate("mkdir", "", name) }
func (s *Files) MkdirAt(path, name string) error { return s.mutate("mkdir", path, name) }
func (s *Files) Rename(path, name string) error  { return s.mutate("rename", path, name) }
func (s *Files) Remove(path string) error        { return s.mutate("remove", path, "") }

func (s *Files) mutate(operation, path, name string) error {
	s.mu.Lock()
	s.syncDraftsLocked()
	v := s.states[s.key]
	if v == nil || s.client == nil || v.Busy || !v.Bound || !v.Loaded || s.current != nil && !s.current() {
		s.mu.Unlock()
		return fmt.Errorf("Refresh this session's verified remote directory before changing files")
	}
	if v.Dirty() {
		s.mu.Unlock()
		return fmt.Errorf("Save or explicitly discard the remote draft before changing files")
	}
	target, dest := path, ""
	var err error
	if operation == "mkdir" {
		parent := path
		if parent == "" {
			parent = v.Path
		}
		if parent != v.Path && !knownTreeDirectory(v, parent) {
			err = fmt.Errorf("Choose a directory from this session's tree")
		} else {
			target, err = remoteChild(parent, name)
		}
	} else {
		known := false
		for _, entry := range v.Items {
			if entry.Path == path && !entry.Drive && path != "/" {
				known = true
				break
			}
		}
		if !known {
			known = knownTreeEntry(v, path, true) && path != v.RootPath
		}
		if !known {
			err = fmt.Errorf("Choose a file or folder from the current session listing")
		}
		if operation == "rename" && err == nil {
			dest, err = remoteChild(ParentDirectory(path), name)
			if err == nil && dest == path {
				err = fmt.Errorf("The new name is unchanged")
			}
		}
	}
	if err == nil && operation != "remove" {
		candidate := target
		if operation == "rename" {
			candidate = dest
		}
		for _, entry := range v.Items {
			if entry.Path == candidate {
				err = fmt.Errorf("That name already exists; overwriting is not supported")
				break
			}
		}
	}
	if err != nil {
		s.mu.Unlock()
		return err
	}
	c, key, g, n := s.client, s.key, s.generation, s.notify
	owner := s.current
	captured := *v
	if operation == "mkdir" {
		captured.Path = ParentDirectory(target)
	} else {
		captured.Path = ParentDirectory(path)
	}
	v.Busy, v.MutationApplied = true, false
	v.Operation, v.Error, v.Notice = operation, "", ""
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	s.cancel = cancel
	s.mu.Unlock()
	if n != nil {
		n()
	}
	go func() {
		defer cancel()
		var status struct {
			SessionBound bool   `json:"sessionBound"`
			SessionMode  string `json:"sessionMode"`
			Workspace    string `json:"workspace"`
			Host         string `json:"host"`
			Username     string `json:"username"`
			Port         int    `json:"port"`
		}
		err := request(ctx, c, "GET", "status?sessionId="+url.QueryEscape(captured.SessionID), nil, &status)
		if err == nil && (!status.SessionBound || status.SessionMode != "remote") {
			err = fmt.Errorf("This session no longer has a remote workspace binding")
		}
		identity := fmt.Sprintf("%s\x00%s\x00%d\x00%s", status.Host, status.Username, status.Port, status.Workspace)
		if err == nil && captured.RemoteIdentity != "" && identity != captured.RemoteIdentity {
			err = fmt.Errorf("Session workspace binding changed; refresh before changing files")
		}
		applied, attempted := false, false
		if err == nil {
			if owner != nil && !owner() {
				err = context.Canceled
			}
		}
		if err == nil {
			body := map[string]any{"sessionId": captured.SessionID, "op": operation, "path": target}
			if operation == "rename" {
				body["dest"] = dest
			}
			attempted = true
			err = request(ctx, c, "POST", "fs", body, nil)
			applied = err == nil
		}
		var listing struct {
			Path    string           `json:"path"`
			Items   []DirectoryEntry `json:"items"`
			Bound   bool             `json:"bound"`
			Missing bool             `json:"missing"`
		}
		if err == nil {
			err = request(ctx, c, "GET", "ls?sessionId="+url.QueryEscape(captured.SessionID)+"&path="+url.QueryEscape(captured.Path), nil, &listing)
			if err == nil && (!listing.Bound || listing.Missing) {
				err = fmt.Errorf("The changed directory could not be refreshed")
			}
		}
		s.mu.Lock()
		if s.generation != g || s.key != key || s.client != c || owner != nil && !owner() {
			s.mu.Unlock()
			return
		}
		v = s.states[key]
		syncResourceDrafts(v)
		v.Busy, v.Operation, v.MutationApplied = false, "", applied
		s.cancel = nil
		if applied {
			stashResource(v)
			v.Notice = operation + " completed: " + target
			if operation == "rename" {
				v.Notice += " → " + dest
			}
			if operation == "remove" || operation == "rename" {
				for i := len(v.Tabs) - 1; i >= 0; i-- {
					if !pathAffected(v.Tabs[i].Path, target) {
						continue
					}
					if v.Tabs[i].Dirty() {
						v.Tabs[i].Stale = true
						v.Tabs[i].Error = "Remote resource was renamed or deleted; draft retained."
					} else {
						closeResource(v, v.Tabs[i].Path)
					}
				}
				for dir := range v.Tree {
					if pathAffected(dir, target) {
						delete(v.Tree, dir)
						delete(v.Expanded, dir)
					}
				}
			}
		}
		if err != nil {
			v.Loaded, v.Bound = false, false
			v.Error = err.Error()
			v.TreeError = err.Error()
			if attempted && !applied {
				v.Error += "; outcome may be uncertain. Refresh before retrying."
			}
			if applied {
				v.Error += "; the operation succeeded but the listing refresh failed."
			}
		} else {
			sortDirectoryEntries(listing.Items)
			v.Path, v.Items = listing.Path, listing.Items
			if v.Tree == nil {
				v.Tree = map[string]FileLevel{}
			}
			v.Tree[listing.Path] = FileLevel{Loaded: true, Items: append([]DirectoryEntry(nil), listing.Items...)}
			v.Loaded, v.Bound = true, true
		}
		s.mu.Unlock()
		if n != nil {
			n()
		}
	}()
	return nil
}
