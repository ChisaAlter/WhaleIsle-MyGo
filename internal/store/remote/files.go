package remote

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/state"
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
	TreeError                                                       *state.Fault
	Cwd, MirrorDir, PendingClose                                    string
	ResolveError                                                    *state.Fault
	Resolved, Resolving, Local                                      bool
	Tree                                                            map[string]FileLevel
	Expanded                                                        map[string]bool
	Tabs                                                            []FileResource
	BindingID                                                       string
	RootPath                                                        string
	RemoteIdentity                                                  string
	MutationApplied                                                 bool
	SessionID, Path, FilePath, Draft, Notice, Operation, Downloaded string
	Error                                                           *state.Fault
	Items                                                           []DirectoryEntry
	File                                                            RemoteFile
	Bound, Loaded, FileLoaded, Busy, Stale                          bool
	Revision                                                        uint64
}

func (v FileSnapshot) Dirty() bool { return v.FileLoaded && v.Draft != v.File.Content }

// Files keeps per-session mutable state under mu; the attach lease replaces
// the old generation counter, op and resolve leases carry request contexts.
// Readers receive a published deep-copied version refreshed on every write,
// because file panes retain draft buffers and nested maps.
type Files struct {
	mu        sync.Mutex
	client    *dsh.Client
	key       string
	attach    state.LeaseOwner
	attachL   state.Lease
	op        state.LeaseOwner
	opL       state.Lease
	resolve   state.LeaseOwner
	resolveL  state.Lease
	states    map[string]*FileSnapshot
	notify    func()
	current   func() bool
	closeSave *fileCloseSave
	pub       state.Published[FileSnapshot]
}

func NewFiles() *Files { return &Files{states: map[string]*FileSnapshot{}} }
func (s *Files) Bind(c *dsh.Client, sessionID string, notify func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncDraftsLocked()
	key := ""
	if c != nil && sessionID != "" {
		key = c.Endpoint.Origin + "\x00" + sessionID
	}
	s.notify = notify
	if c == s.client && key == s.key {
		return
	}
	s.op.Retire()
	s.resolve.Retire()
	s.attachL = s.attach.Renew(context.Background())
	if old := s.states[s.key]; old != nil {
		if old.Busy && (old.Operation == "write" || isFileMutation(old.Operation)) {
			old.Stale = true
			old.Error = state.FaultText("Connection or session changed during a remote mutation; outcome is uncertain. Refresh before retrying.")
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
			v.Resolved, v.Resolving, v.Local = false, false, false
			v.Tree = nil
			v.PendingClose = ""
			for i := range v.Tabs {
				v.Tabs[i].Stale = v.Tabs[i].Loaded
			}
		}
	}
	s.client = c
	s.key = key
	if key != "" && s.states[key] == nil {
		s.states[key] = &FileSnapshot{BindingID: key, SessionID: sessionID}
	}
	s.publishLocked()
}

// publishLocked clones the current key's state; callers hold mu and have
// already synced draft buffers. Draft-owner pointers survive the clone so a
// captured resource still resolves its native buffer.
func (s *Files) publishLocked() {
	v := s.states[s.key]
	if v == nil {
		s.pub.Publish(FileSnapshot{})
		return
	}
	out := *v
	out.Items = append([]DirectoryEntry(nil), v.Items...)
	out.Tree = cloneFileTree(v.Tree)
	out.Expanded = make(map[string]bool, len(v.Expanded))
	for path, open := range v.Expanded {
		out.Expanded[path] = open
	}
	out.Tabs = append([]FileResource(nil), v.Tabs...)
	s.pub.Publish(out)
}

// Snapshot returns the published version for the bound session. Committed
// draft text arrives outside the mutex, so a read republishes only when a
// buffer produced new text; an idle reader gets the shared immutable view.
func (s *Files) Snapshot() *state.Version[FileSnapshot] {
	s.mu.Lock()
	if s.syncDraftsLocked() || s.pub.Load() == nil {
		s.publishLocked()
	}
	s.mu.Unlock()
	return s.pub.Load()
}
func (s *Files) SetDraft(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncDraftsLocked()
	v := s.states[s.key]
	if v != nil && !v.Busy && v.FileLoaded && !v.File.Binary && !v.File.Truncated {
		replaceResourceDraft(v, text)
		stashResource(v)
		s.publishLocked()
	}
}
func (s *Files) Discard() {
	s.mu.Lock()
	v := s.states[s.key]
	if v != nil && !v.Busy {
		replaceResourceDraft(v, v.File.Content)
		v.Revision++
		v.Error = nil
		stashResource(v)
		s.publishLocked()
	}
	n := s.notify
	s.mu.Unlock()
	if n != nil {
		n()
	}
}
func (s *Files) Browse(path string) { s.run("browse", path) }
func (s *Files) Open(path string)   { s.openResource(path) }
func (s *Files) Reload()            { s.run("read", s.Snapshot().Value.FilePath) }
func (s *Files) Save()              { s.run("write", "") }
func (s *Files) Download()          { s.run("download", "") }
func (s *Files) run(action, path string) {
	s.mu.Lock()
	s.syncDraftsLocked()
	v := s.states[s.key]
	if v == nil || s.client == nil || v.Busy || s.current != nil && !s.current() {
		s.mu.Unlock()
		return
	}
	fail := func(message string) {
		v.Error = state.FaultText(message)
		n := s.notify
		s.publishLocked()
		s.mu.Unlock()
		if n != nil {
			n()
		}
	}
	if action == "read" && v.Dirty() {
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
		known = known || knownTreeEntry(v, path, false)
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
	if action == "download-entry" && !knownTreeEntry(v, path, true) {
		fail("Choose a file or folder from this session's tree.")
		return
	}
	if action == "browse" {
		if v.Tree == nil {
			v.Tree = map[string]FileLevel{}
		}
		level := v.Tree[path]
		level.Loading, level.Error = true, nil
		v.Tree[path] = level
	}
	v.Busy = true
	if action == "read" {
		v.FileLoaded = false
	}
	v.MutationApplied = false
	if action != "browse" && action != "download-entry" {
		v.Error, v.Notice = nil, ""
	} else {
		v.TreeError = nil
	}
	v.Operation = action
	c, key, attachL, n := s.client, s.key, s.attachL, s.notify
	owner := s.current
	captured := *v
	parent, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	s.opL = s.op.Renew(parent)
	op := s.opL
	s.publishLocked()
	s.mu.Unlock()
	if n != nil {
		n()
	}
	go func() {
		defer cancel()
		ctx := op.Context()
		if owner != nil && !owner() {
			s.mu.Lock()
			if s.attachL == attachL && s.key == key && s.client == c {
				s.states[key].Busy = false
				s.publishLocked()
			}
			s.mu.Unlock()
			if n != nil {
				n()
			}
			return
		}
		var err error
		var directory struct {
			Path    string           `json:"path"`
			Items   []DirectoryEntry `json:"items"`
			Bound   bool             `json:"bound"`
			Missing bool             `json:"missing"`
		}
		var file RemoteFile
		localSession := false
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
				localSession = true
				err = fmt.Errorf("Selected session is not bound to a remote workspace; no machine-default fallback is used")
			}
			if err == nil {
				rootPath = status.Workspace
				remoteIdentity = fmt.Sprintf("%s\x00%s\x00%d\x00%s", status.Host, status.Username, status.Port, status.Workspace)
				err = request(ctx, c, "GET", "ls?sessionId="+url.QueryEscape(captured.SessionID)+"&path="+url.QueryEscape(path), nil, &directory)
			}
			if err == nil && !directory.Bound {
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
		case "download", "download-entry":
			target := captured.FilePath
			if action == "download-entry" {
				target = path
			}
			err = request(ctx, c, "POST", "fs", map[string]any{"sessionId": captured.SessionID, "op": "download", "path": target}, &receipt)
			if err == nil && strings.TrimSpace(receipt.Local) == "" {
				err = fmt.Errorf("Download response has no local mirror path")
			}
		}
		s.mu.Lock()
		if s.attachL != attachL || s.key != key || s.client != c || owner != nil && !owner() {
			s.mu.Unlock()
			return
		}
		v = s.states[key]
		v.Busy = false
		v.Operation = ""
		if err != nil {
			if action == "browse" || action == "download-entry" {
				v.TreeError = state.FaultOf(err)
				if v.FilePath == "" {
					v.Error = state.FaultOf(err)
				}
			} else {
				v.Error = state.FaultOf(err)
			}
			if action == "write" {
				v.Stale = true
				v.Notice = "Draft retained. Do not retry automatically: the remote write may have conflicted or its outcome may be uncertain."
			}
			if action == "browse" {
				if localSession {
					v.Resolved, v.Local, v.Bound, v.RootPath = true, true, false, ""
					v.Tree, v.Expanded = nil, nil
				}
				if v.Tree != nil {
					level := v.Tree[path]
					level.Loading, level.Error = false, state.FaultOf(err)
					v.Tree[path] = level
				}
			}
		} else {
			switch action {
			case "browse":
				if v.RemoteIdentity != "" && v.RemoteIdentity != remoteIdentity {
					v.Stale = v.FileLoaded
					v.Notice = "Session remote binding changed; reload the file before editing."
					v.Tree, v.Expanded = nil, nil
					for i := range v.Tabs {
						v.Tabs[i].Stale = true
					}
				}
				v.RootPath, v.RemoteIdentity = rootPath, remoteIdentity
				v.Path = directory.Path
				sortDirectoryEntries(directory.Items)
				v.Items = directory.Items
				v.Loaded = true
				v.Bound = true
				if v.Tree == nil {
					v.Tree = map[string]FileLevel{}
				}
				if v.Expanded == nil {
					v.Expanded = map[string]bool{}
				}
				v.Tree[directory.Path] = FileLevel{Loaded: true, Missing: directory.Missing, Items: append([]DirectoryEntry(nil), directory.Items...)}
				v.Expanded[directory.Path] = true
			case "read":
				v.FilePath = path
				v.File = file
				v.FileLoaded = true
				replaceResourceDraft(v, file.Content)
				v.Stale = false
				v.Downloaded = ""
				v.Revision++
				if i := resourceIndex(v, path); i >= 0 {
					v.Tabs[i].Editing = false
				}
			case "write":
				v.File.Content = captured.Draft
				v.File.Mtime = receipt.Mtime
				v.Stale = false
				v.Revision++
				v.Notice = "Saved to the session-bound remote file."
				if i := resourceIndex(v, v.FilePath); i >= 0 {
					v.Tabs[i].Editing = false
				}
			case "download", "download-entry":
				v.Downloaded = receipt.Local
				v.Notice = fmt.Sprintf("Downloaded %d bytes to %s", receipt.Bytes, receipt.Local)
			}
		}
		if action != "browse" && action != "download-entry" {
			stashResource(v)
		}
		s.publishLocked()
		s.mu.Unlock()
		if n != nil {
			n()
		}
	}()
}
