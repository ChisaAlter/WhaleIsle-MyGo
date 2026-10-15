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
	if parent == "" { return "", fmt.Errorf("Browse a remote directory first") }
	sep := "/"
	if strings.Contains(parent, "\\") || len(parent) >= 2 && parent[1] == ':' { sep = "\\" }
	return strings.TrimRight(parent, "/\\") + sep + name, nil
}

func (s *Files) Mkdir(name string) error { return s.mutate("mkdir", "", name) }
func (s *Files) Rename(path, name string) error { return s.mutate("rename", path, name) }
func (s *Files) Remove(path string) error { return s.mutate("remove", path, "") }

func (s *Files) mutate(operation, path, name string) error {
	s.mu.Lock()
	v := s.states[s.key]
	if v == nil || s.client == nil || v.Busy || !v.Bound || !v.Loaded {
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
		target, err = remoteChild(v.Path, name)
	} else {
		known := false
		for _, entry := range v.Items {
			if entry.Path == path && !entry.Drive && path != "/" { known = true; break }
		}
		if !known { err = fmt.Errorf("Choose a file or folder from the current session listing") }
		if operation == "rename" && err == nil {
			dest, err = remoteChild(ParentDirectory(path), name)
			if err == nil && dest == path { err = fmt.Errorf("The new name is unchanged") }
		}
	}
	if err == nil && operation != "remove" {
		candidate := target
		if operation == "rename" { candidate = dest }
		for _, entry := range v.Items {
			if entry.Path == candidate { err = fmt.Errorf("That name already exists; overwriting is not supported"); break }
		}
	}
	if err != nil { s.mu.Unlock(); return err }
	c, key, g, n := s.client, s.key, s.generation, s.notify
	captured := *v
	v.Busy, v.MutationApplied = true, false
	v.Operation, v.Error, v.Notice = operation, "", ""
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	s.cancel = cancel
	s.mu.Unlock()
	if n != nil { n() }
	go func() {
		defer cancel()
		var status struct {
			SessionBound bool `json:"sessionBound"`
			SessionMode string `json:"sessionMode"`
			Workspace string `json:"workspace"`
			Host string `json:"host"`
			Username string `json:"username"`
			Port int `json:"port"`
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
			body := map[string]any{"sessionId": captured.SessionID, "op": operation, "path": target}
			if operation == "rename" { body["dest"] = dest }
			attempted = true
			err = request(ctx, c, "POST", "fs", body, nil)
			applied = err == nil
		}
		var listing struct {
			Path string `json:"path"`
			Items []DirectoryEntry `json:"items"`
			Bound bool `json:"bound"`
			Missing bool `json:"missing"`
		}
		if err == nil {
			err = request(ctx, c, "GET", "ls?sessionId="+url.QueryEscape(captured.SessionID)+"&path="+url.QueryEscape(captured.Path), nil, &listing)
			if err == nil && (!listing.Bound || listing.Missing) { err = fmt.Errorf("The changed directory could not be refreshed") }
		}
		s.mu.Lock()
		if s.generation != g || s.key != key || s.client != c { s.mu.Unlock(); return }
		v = s.states[key]
		v.Busy, v.Operation, v.MutationApplied = false, "", applied
		s.cancel = nil
		if applied {
			v.Notice = operation + " completed: " + target
			if operation == "rename" { v.Notice += " → " + dest }
			normalized, normalizedTarget := strings.ReplaceAll(v.FilePath, "\\", "/"), strings.ReplaceAll(target, "\\", "/")
			affected := normalized == normalizedTarget || strings.HasPrefix(normalized, strings.TrimRight(normalizedTarget, "/")+"/")
			if affected && (operation == "remove" || operation == "rename") {
				// Do not retain an editor bound to a deleted/renamed resource.
				v.FilePath, v.Draft, v.Downloaded = "", "", ""
				v.File, v.FileLoaded, v.Stale = RemoteFile{}, false, false
				v.Revision++
			}
		}
		if err != nil {
			v.Loaded, v.Bound = false, false
			v.Error = err.Error()
			if attempted && !applied { v.Error += "; outcome may be uncertain. Refresh before retrying." }
			if applied { v.Error += "; the operation succeeded but the listing refresh failed." }
		} else {
			v.Path, v.Items = listing.Path, listing.Items
			v.Loaded, v.Bound = true, true
		}
		s.mu.Unlock()
		if n != nil { n() }
	}()
	return nil
}
