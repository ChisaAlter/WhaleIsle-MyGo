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

type DirectorySnapshot struct {
	InputPath, NewName string
	MkdirOpen          bool
	Path               string           `json:"path"`
	Platform           string           `json:"platform"`
	Items              []DirectoryEntry `json:"items"`
	Missing            bool             `json:"missing"`
	Loaded             bool
	Error              string
	Choosing           bool
	SelectedPath       string
	LocalMirror        string
}

func (d DirectorySnapshot) CanChoose() bool {
	return d.Loaded && !d.Missing && d.Error == "" && strings.TrimSpace(d.Path) != "" && d.Path != "/"
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
func (s *Store) Browse(path string) { s.directoryRequest(path, false) }
func (s *Store) ChooseDirectory()   { s.directoryRequest("", true) }
func (s *Store) directoryRequest(path string, choosing bool) {
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
	c, n, g := s.client, s.notify, s.generation
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
			sort.SliceStable(listing.Items, func(i, j int) bool {
				a, b := listing.Items[i], listing.Items[j]
				if a.IsDirectory() != b.IsDirectory() {
					return a.IsDirectory()
				}
				return a.Name < b.Name
			})
		}
		s.mu.Lock()
		if s.generation != g {
			s.mu.Unlock()
			return
		}
		s.view.Busy = false
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
	}()
}
