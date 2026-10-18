package main

import (
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"whaleisle/internal/desktop/workspaceauth"
	"whaleisle/internal/harness"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/store/session"
	remoteui "whaleisle/internal/ui/remote"
)

type phoneGitWorkspaceScope struct {
	Owner string
	Roots map[string]bool
}

func phoneGitRegisteredRoots(workspaces []session.WorkspaceRow, projects []projectflow.Project) []remoteui.PhoneGitWorkspace {
	rows := []remoteui.PhoneGitWorkspace{}
	add := func(id, title, path string) {
		if id == "" || !filepath.IsAbs(path) || workspaceauth.Validate(path) != nil {
			return
		}
		root, err := filepath.EvalSymlinks(path)
		if err != nil {
			return
		}
		root, err = filepath.Abs(root)
		if err != nil {
			return
		}
		if strings.TrimSpace(title) == "" {
			title = filepath.Base(root)
		}
		rows = append(rows, remoteui.PhoneGitWorkspace{ID: id, Title: title, Root: root})
	}
	for _, w := range workspaces {
		if w.WorkspaceID != "" {
			add("host:"+w.WorkspaceID, w.Title, w.Path)
		}
	}
	for _, p := range projects {
		if p.ID != "" {
			add("project:"+p.ID, p.Title, p.CanonicalWorkingDirectory)
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows
}

// Local-process ownership is essential: a registered path on a remote Host
// must never be interpreted as a path on this desktop.
func (s *mainState) phoneGitWorkspaceCatalog() (string, []remoteui.PhoneGitWorkspace) {
	s.mu.Lock()
	sup, status, sessions := s.sup, s.bootStatus, s.sessions
	s.mu.Unlock()
	if sup == nil || sessions == nil {
		return "", nil
	}
	live := sup.Status()
	if live.State != harness.Ready || live.Generation == 0 || live.Generation != status.Generation || live.Endpoint != status.Endpoint {
		return "", nil
	}
	u, err := url.Parse(live.Endpoint.Origin)
	if err != nil || (u.Hostname() != "localhost" && !net.ParseIP(u.Hostname()).IsLoopback()) {
		return "", nil
	}
	s.dshMu.Lock()
	client, endpoint := s.dshClient, s.dshEndpoint
	s.dshMu.Unlock()
	if client == nil || endpoint != live.Endpoint.Origin+"\x00"+live.Endpoint.Cookie {
		return "", nil
	}
	binding, id, _ := sessions.WorkspaceFileReferenceOrigin()
	if id == "" || binding != fmt.Sprintf("%p\x00%s", client, id) {
		return "", nil
	}
	snap := sessions.Snapshot()
	if !snap.WorkspaceReady {
		return "", nil
	}
	s.mu.Lock()
	var projects []projectflow.Project
	if s.projectFlow.client == client && s.projectFlow.loaded && s.projectFlow.listing.Available {
		projects = append(projects, s.projectFlow.listing.Projects...)
	}
	s.mu.Unlock()
	return fmt.Sprintf("%p:%d:%d", client, live.Generation, snap.ComposerHostGeneration), phoneGitRegisteredRoots(snap.Workspaces, projects)
}

func (s *mainState) phoneGitWorkspaceRows() []remoteui.PhoneGitWorkspace {
	owner, rows := s.phoneGitWorkspaceCatalog()
	allowed := map[string]bool{}
	for _, row := range rows {
		allowed[row.Root] = true
	}
	s.mu.Lock()
	changed := false
	if s.phoneGitWorkspaces.Owner != owner {
		changed = len(s.phoneGitWorkspaces.Roots) > 0
		s.phoneGitWorkspaces = phoneGitWorkspaceScope{Owner: owner, Roots: map[string]bool{}}
	}
	for root := range s.phoneGitWorkspaces.Roots {
		if !allowed[root] {
			delete(s.phoneGitWorkspaces.Roots, root)
			changed = true
		}
	}
	for i := range rows {
		rows[i].Selected = s.phoneGitWorkspaces.Roots[rows[i].Root]
		rows[i].ID = owner + "\x00" + rows[i].ID
	}
	runtime := s.phoneRuntime
	s.mu.Unlock()
	if changed && runtime != nil {
		runtime.RevokeGit()
	}
	return rows
}
func (s *mainState) selectedPhoneGitWorkspaceRoots() []string {
	if s.cfgStore == nil || !s.cfgStore.Get().RemoteEnabled {
		return nil
	}
	rows := s.phoneGitWorkspaceRows()
	roots := []string{}
	seen := map[string]bool{}
	for _, row := range rows {
		if row.Selected && !seen[row.Root] {
			roots = append(roots, row.Root)
			seen[row.Root] = true
		}
	}
	return roots
}
func (s *mainState) setPhoneGitWorkspace(id string, enabled bool) {
	rows := s.phoneGitWorkspaceRows()
	if s.cfgStore == nil || !s.cfgStore.Get().RemoteEnabled {
		return
	}
	for _, row := range rows {
		if row.ID == id {
			s.mu.Lock()
			if s.phoneGitWorkspaces.Roots == nil {
				s.phoneGitWorkspaces.Roots = map[string]bool{}
			}
			if enabled {
				s.phoneGitWorkspaces.Roots[row.Root] = true
			} else {
				delete(s.phoneGitWorkspaces.Roots, row.Root)
			}
			runtime := s.phoneRuntime
			s.mu.Unlock()
			// Drop the old gate generation and pending reviews before any refresh.
			if runtime != nil {
				runtime.RevokeGit()
			}
			return
		}
	}
}
func (s *mainState) phoneGitRoots() []string {
	allowed, current := s.phoneGitGrant()
	roots := s.selectedPhoneGitWorkspaceRoots()
	if allowed {
		for _, root := range roots {
			if root == current {
				return roots
			}
		}
		roots = append(roots, current)
	}
	return roots
}
