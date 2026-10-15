package remote

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
	"whaleisle/internal/dsh"
)

const filePreviewLimit = 256 * 1024

type RemoteFile struct {
	Content   string   `json:"content"`
	Mtime     *float64 `json:"mtime"`
	Binary    bool     `json:"binary"`
	Truncated bool     `json:"truncated"`
	Size      int64    `json:"size"`
}
type FileSnapshot struct {
	BindingID                                                              string
	RootPath                                                               string
	RemoteIdentity                                                         string
	MutationApplied                                                        bool
	SessionID, Path, FilePath, Draft, Error, Notice, Operation, Downloaded string
	Items                                                                  []DirectoryEntry
	File                                                                   RemoteFile
	Bound, Loaded, FileLoaded, Busy, Stale                                 bool
	Revision                                                               uint64
}

func (v FileSnapshot) Dirty() bool { return v.FileLoaded && v.Draft != v.File.Content }

type Files struct {
	mu         sync.Mutex
	client     *dsh.Client
	key        string
	generation uint64
	states     map[string]*FileSnapshot
	notify     func()
	cancel     context.CancelFunc
}

func NewFiles() *Files { return &Files{states: map[string]*FileSnapshot{}} }
func (s *Files) Bind(c *dsh.Client, sessionID string, notify func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := ""
	if c != nil && sessionID != "" {
		key = c.Endpoint.Origin + "\x00" + sessionID
	}
	s.notify = notify
	if c == s.client && key == s.key {
		return
	}
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.generation++
	if old := s.states[s.key]; old != nil {
		if old.Busy && (old.Operation == "write" || isFileMutation(old.Operation)) {
			old.Stale = true
			old.Error = "Connection or session changed during a remote mutation; outcome is uncertain. Refresh before retrying."
			old.Loaded, old.Bound = false, false
		}
		old.Busy = false
	}
	if c != s.client {
		for _, v := range s.states {
			v.Stale = v.FileLoaded
			v.Bound = false
			v.Loaded = false
			v.Busy = false
		}
	}
	s.client = c
	s.key = key
	if key != "" && s.states[key] == nil {
		s.states[key] = &FileSnapshot{BindingID: key, SessionID: sessionID}
	}
}
func (s *Files) Snapshot() FileSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.states[s.key]
	if v == nil {
		return FileSnapshot{}
	}
	out := *v
	out.Items = append([]DirectoryEntry(nil), v.Items...)
	return out
}
func (s *Files) SetDraft(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.states[s.key]
	if v != nil && !v.Busy && v.FileLoaded && !v.File.Binary && !v.File.Truncated {
		v.Draft = text
	}
}
func (s *Files) Discard() {
	s.mu.Lock()
	v := s.states[s.key]
	if v != nil && !v.Busy {
		v.Draft = v.File.Content
		v.Revision++
		v.Error = ""
	}
	n := s.notify
	s.mu.Unlock()
	if n != nil {
		n()
	}
}
func (s *Files) Browse(path string) { s.run("browse", path) }
func (s *Files) Open(path string)   { s.run("read", path) }
func (s *Files) Save()              { s.run("write", "") }
func (s *Files) Download()          { s.run("download", "") }
func (s *Files) run(action, path string) {
	s.mu.Lock()
	v := s.states[s.key]
	if v == nil || s.client == nil || v.Busy {
		s.mu.Unlock()
		return
	}
	fail := func(message string) {
		v.Error = message
		n := s.notify
		s.mu.Unlock()
		if n != nil {
			n()
		}
	}
	if (action == "browse" || action == "read") && v.Dirty() {
		fail("Unsaved remote draft retained. Save it or explicitly discard it before navigating or reloading.")
		return
	}
	if action != "browse" && !v.Bound {
		fail("This session has no verified remote workspace. Browse this session first.")
		return
	}
	if action == "read" {
		known := path != "" && path == v.FilePath
		for _, e := range v.Items {
			if e.Path == path && !e.IsDirectory() {
				known = true
			}
		}
		if !known {
			fail("Choose a file from this session's remote listing.")
			return
		}
	}
	if action == "write" {
		if !v.FileLoaded || v.File.Binary || v.File.Truncated || v.File.Mtime == nil || v.Stale || !v.Dirty() {
			fail("Save requires a complete current text read with an mtime; the draft is retained.")
			return
		}
		if len([]byte(v.Draft)) > filePreviewLimit {
			fail("Remote text exceeds the 256 KiB editing limit.")
			return
		}
	}
	if action == "download" && (!v.FileLoaded || v.Dirty()) {
		fail("Save or discard the remote draft before downloading the file.")
		return
	}
	v.Busy = true
	v.MutationApplied = false
	v.Error = ""
	v.Notice = ""
	v.Operation = action
	c, key, g, n := s.client, s.key, s.generation, s.notify
	captured := *v
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	s.cancel = cancel
	s.mu.Unlock()
	if n != nil {
		n()
	}
	go func() {
		defer cancel()
		var err error
		var directory struct {
			Path    string           `json:"path"`
			Items   []DirectoryEntry `json:"items"`
			Bound   bool             `json:"bound"`
			Missing bool             `json:"missing"`
		}
		var file RemoteFile
		rootPath, remoteIdentity := captured.RootPath, captured.RemoteIdentity
		var receipt struct {
			Mtime *float64 `json:"mtime"`
			Local string   `json:"local"`
			Bytes int64    `json:"bytes"`
		}
		switch action {
		case "browse":
			var status struct {
				SessionBound bool   `json:"sessionBound"`
				SessionMode  string `json:"sessionMode"`
				Workspace    string `json:"workspace"`
				Host         string `json:"host"`
				Username     string `json:"username"`
				Port         int    `json:"port"`
			}
			err = request(ctx, c, "GET", "status?sessionId="+url.QueryEscape(captured.SessionID), nil, &status)
			if err == nil && (!status.SessionBound || status.SessionMode != "remote") {
				err = fmt.Errorf("Selected session is not bound to a remote workspace; no machine-default fallback is used")
			}
			if err == nil {
				rootPath = status.Workspace
				remoteIdentity = fmt.Sprintf("%s\x00%s\x00%d\x00%s", status.Host, status.Username, status.Port, status.Workspace)
				err = request(ctx, c, "GET", "ls?sessionId="+url.QueryEscape(captured.SessionID)+"&path="+url.QueryEscape(path), nil, &directory)
			}
			if err == nil && (!directory.Bound || directory.Missing) {
				err = fmt.Errorf("Session-bound remote listing is unavailable or the directory is missing")
			}
		case "read":
			err = request(ctx, c, "POST", "read", map[string]any{"sessionId": captured.SessionID, "path": path, "maxBytes": filePreviewLimit}, &file)
			if err == nil && len([]byte(file.Content)) > filePreviewLimit+1024 {
				err = fmt.Errorf("Remote preview exceeded the bounded response")
			}
		case "write":
			err = request(ctx, c, "POST", "write", map[string]any{"sessionId": captured.SessionID, "path": captured.FilePath, "content": captured.Draft, "expectedMtime": *captured.File.Mtime}, &receipt)
			if err == nil && receipt.Mtime == nil {
				err = fmt.Errorf("Save response has no mtime; remote state must be re-read")
			}
		case "download":
			err = request(ctx, c, "POST", "fs", map[string]any{"sessionId": captured.SessionID, "op": "download", "path": captured.FilePath}, &receipt)
			if err == nil && strings.TrimSpace(receipt.Local) == "" {
				err = fmt.Errorf("Download response has no local mirror path")
			}
		}
		s.mu.Lock()
		if s.generation != g || s.key != key || s.client != c {
			s.mu.Unlock()
			return
		}
		v = s.states[key]
		v.Busy = false
		v.Operation = ""
		if err != nil {
			v.Error = err.Error()
			if action == "write" {
				v.Stale = true
				v.Error += "; draft retained. Do not retry automatically: the remote write may have conflicted or its outcome may be uncertain."
			}
			if action == "browse" {
				v.Bound = false
			}
		} else {
			switch action {
			case "browse":
				if v.RemoteIdentity != "" && v.RemoteIdentity != remoteIdentity {
					v.Stale = v.FileLoaded
					v.Notice = "Session remote binding changed; reload the file before editing."
				}
				v.RootPath, v.RemoteIdentity = rootPath, remoteIdentity
				v.Path = directory.Path
				v.Items = directory.Items
				v.Loaded = true
				v.Bound = true
			case "read":
				v.FilePath = path
				v.File = file
				v.FileLoaded = true
				v.Draft = file.Content
				v.Stale = false
				v.Downloaded = ""
				v.Revision++
			case "write":
				v.File.Content = captured.Draft
				v.File.Mtime = receipt.Mtime
				v.Stale = false
				v.Revision++
				v.Notice = "Saved to the session-bound remote file."
			case "download":
				v.Downloaded = receipt.Local
				v.Notice = fmt.Sprintf("Downloaded %d bytes to %s", receipt.Bytes, receipt.Local)
			}
		}
		s.mu.Unlock()
		if n != nil {
			n()
		}
	}()
}
