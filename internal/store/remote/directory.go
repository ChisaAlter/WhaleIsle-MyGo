package remote

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

// DirectoryEntry follows dsh-remote/ls, including the Windows drive selector.
type DirectoryEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Type  string `json:"type"`
	Drive bool   `json:"drive"`
}

func (e DirectoryEntry) IsDirectory() bool { return e.Type == "dir" || e.Drive }

func sortDirectoryEntries(items []DirectoryEntry) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].IsDirectory() != items[j].IsDirectory() {
			return items[i].IsDirectory()
		}
		return items[i].Name < items[j].Name
	})
}

type DirectorySnapshot struct {
	InputPath, NewName string
	MkdirOpen          bool
	Path               string           `json:"path"`
	Platform           string           `json:"platform"`
	Items              []DirectoryEntry `json:"items"`
	Missing            bool             `json:"missing"`
	Loaded             bool
	Error              string
	CreateError        string
	Choosing           bool
	SelectedPath       string
	LocalMirror        string
}

// DirectoryChoice is the exact successful mirror operation, not a path read
// later from whichever machine/Host happens to be current at that time.
type DirectoryChoice struct {
	LocalMirror, RemotePath, MachineID string
	Generation, RequestID              uint64
}

// DirectoryChoiceCurrent is for the consumer's actual adoption boundary.
// Navigation, another choice, current-machine changes and Host replacement
// retire it. The workspace owner also checks its own Host/session scope.
func (s *Store) DirectoryChoiceCurrent(choice DirectoryChoice) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return choice.LocalMirror != "" && !s.view.Busy && s.generation == choice.Generation && s.directoryRequestID == choice.RequestID && s.view.CurrentID == choice.MachineID && s.view.Directory.LocalMirror == choice.LocalMirror && s.view.Directory.SelectedPath == choice.RemotePath
}

func (d DirectorySnapshot) CanChoose() bool {
	return d.Loaded && !d.Missing && d.Error == "" && strings.TrimSpace(d.Path) != ""
}

// ParentDirectory uses remote path syntax, never the local host OS's separator.
func ParentDirectory(p string) string {
	p = strings.ReplaceAll(p, `\`, "/")
	p = strings.TrimRight(p, "/")
	win := len(p) >= 2 && p[1] == ':'
	if win && len(p) == 2 {
		return ""
	}
	i := strings.LastIndex(p, "/")
	if i <= 0 {
		return "/"
	}
	parent := p[:i]
	if win {
		if len(parent) == 2 {
			parent += "/"
		}
		return strings.ReplaceAll(parent, "/", `\`)
	}
	return parent
}

// Browse and ChooseDirectory only call the existing machine-scoped HTTP API.
// Browsing never adopts a workspace; only a successful mirror response does.
func (s *Store) Browse(path string) { s.directoryRequest(path, false, nil) }
func (s *Store) ChooseDirectory()   { s.directoryRequest("", true, nil) }

// ChooseDirectoryWithResult adopts only this explicit choice, never a mirror
// left over from a previous directory. Completion runs on the request worker.
func (s *Store) ChooseDirectoryWithResult(picked func(string)) {
	var deliver func(DirectoryChoice)
	if picked != nil {
		deliver = func(choice DirectoryChoice) { picked(choice.LocalMirror) }
	}
	s.directoryRequest("", true, deliver)
}

func (s *Store) ChooseDirectoryWithChoice(picked func(DirectoryChoice)) {
	s.directoryRequest("", true, picked)
}

func (s *Store) CancelDirectoryChoice() {
	s.mu.Lock()
	if s.view.Directory.Choosing {
		s.directoryRequestID++
		if s.cancel != nil {
			s.cancel()
			s.cancel = nil
		}
		s.view.Busy, s.view.Directory.Choosing = false, false
	}
	s.mu.Unlock()
}

func (s *Store) directoryRequest(path string, choosing bool, picked func(DirectoryChoice)) {
	s.mu.Lock()
	if s.view.Busy || s.view.Error != "" || s.view.CurrentID == "" || choosing && !s.view.Directory.CanChoose() {
		s.mu.Unlock()
		return
	}
	if choosing {
		path = s.view.Directory.Path
	}
	s.view.Busy = true
	s.view.Directory.Choosing = choosing
	s.view.Directory.Error = ""
	s.directoryRequestID++
	c, n, g, requestID, machine := s.client, s.notify, s.generation, s.directoryRequestID, s.view.CurrentID
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	s.cancel = cancel
	s.mu.Unlock()
	if n != nil {
		n()
	}
	go func() {
		defer cancel()
		var listing DirectorySnapshot
		var result struct {
			LocalMirror string `json:"localMirror"`
		}
		var err error
		if choosing {
			err = request(ctx, c, "POST", "mirror", map[string]string{"path": path}, &result)
			if err == nil && strings.TrimSpace(result.LocalMirror) == "" {
				err = fmt.Errorf("dsh-remote mirror: response is missing localMirror")
			}
		} else {
			err = request(ctx, c, "GET", "ls?path="+url.QueryEscape(path), nil, &listing)
			if err == nil && listing.Missing {
				err = fmt.Errorf("Remote directory is missing")
			}
			sortDirectoryEntries(listing.Items)
		}
		s.mu.Lock()
		if s.generation != g || s.directoryRequestID != requestID || s.view.CurrentID != machine {
			s.mu.Unlock()
			return
		}
		s.view.Busy = false
		s.cancel = nil
		d := &s.view.Directory
		d.Choosing = false
		if err != nil {
			d.Error = err.Error()
		} else if choosing {
			d.LocalMirror = result.LocalMirror
			d.SelectedPath = path
		} else {
			listing.Loaded = true
			listing.InputPath = listing.Path
			listing.SelectedPath = d.SelectedPath
			listing.LocalMirror = d.LocalMirror
			*d = listing
		}
		s.mu.Unlock()
		if n != nil {
			n()
		}
		if choosing && err == nil && picked != nil {
			picked(DirectoryChoice{LocalMirror: result.LocalMirror, RemotePath: path, MachineID: machine, Generation: g, RequestID: requestID})
		}
	}()
}
