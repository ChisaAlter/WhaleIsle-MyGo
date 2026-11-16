package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/mediapreview"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/files"
)

// Child editors are durable draft consumers, but never alias an ordinary
// session ID or another child address's draft/checkpoint entry.
func (s *mainState) childComposerDraft(binding string, address session.AgentSessionAddress) *composer.DraftState {
	raw, _ := json.Marshal(address)
	key := "\x00child-draft:" + base64.RawURLEncoding.EncodeToString(raw)
	s.mu.Lock()
	defer s.mu.Unlock()
	d := s.composerDraftLocked(key)
	if s.childComposerBindings == nil {
		s.childComposerBindings = map[string]string{}
	}
	if old, ok := s.childComposerBindings[key]; ok && old != binding {
		d.RetireBinding()
	}
	s.childComposerBindings[key] = binding
	return d
}

func (s *mainState) boundReferenceQuery(sessions *session.Store, snapshot session.ViewState) func(context.Context, string, bool) ([]session.ReferenceCandidate, error) {
	current := func() bool {
		s.mu.Lock()
		same := s.sessions == sessions && !s.quitting
		s.mu.Unlock()
		v := sessions.SelectionSummary()
		return same && v.SelectedID == snapshot.SelectedID && v.ComposerHostGeneration == snapshot.ComposerHostGeneration && sessions.ComposerSessionGeneration() == snapshot.ComposerSessionGeneration
	}
	return func(ctx context.Context, q string, quoted bool) ([]session.ReferenceCandidate, error) {
		if !current() {
			return nil, context.Canceled
		}
		rows, err := sessions.QueryReferenceCandidates(ctx, q, quoted)
		if !current() {
			return nil, context.Canceled
		}
		return rows, err
	}
}

func (s *mainState) saveDefaultAgentPreset(id string) {
	s.mu.Lock()
	manager, sessions := s.agentPresetManager, s.sessions
	s.mu.Unlock()
	if manager == nil {
		return
	}
	var apply func(context.Context, string)
	if sessions != nil {
		apply = sessions.CaptureBlankAgentPresetDefaultTarget()
	}
	go func() {
		ctx := context.Background()
		manager.SaveDefault(ctx, id)
		confirmed := manager.Snapshot()
		if !confirmed.Busy && confirmed.Err == "" && confirmed.DefaultID == id && apply != nil {
			apply(ctx, id)
		}
	}()
}

type fileOpenWithOwner struct {
	sessions              *session.Store
	client                *dsh.Client
	endpoint              harness.Endpoint
	generation            int
	sessionID, root, path string
	epoch                 uint64
}

func (s *mainState) fileOpenWithSourceCurrent(o fileOpenWithOwner) bool {
	s.mu.Lock()
	valid := s.sessions == o.sessions && s.bootStatus.Generation == o.generation && s.bootStatus.Endpoint == o.endpoint && s.bootStatus.State == harness.Ready && !s.quitting
	s.mu.Unlock()
	s.dshMu.Lock()
	sameClient := s.dshClient == o.client
	s.dshMu.Unlock()
	if !valid || !sameClient || o.sessions == nil {
		return false
	}
	v := o.sessions.SelectionSummary()
	return v.SelectedID == o.sessionID && v.SelectedCwd == o.root && o.sessions.ComposerSessionGeneration() == o.epoch
}

func (s *mainState) fileOpenWithCurrent(o fileOpenWithOwner) bool {
	s.mu.Lock()
	same := s.fileOpenWithOwner == o
	s.mu.Unlock()
	return same && s.fileOpenWithSourceCurrent(o)
}

func (s *mainState) refreshFileOpenWith(win *mygo.Window, o fileOpenWithOwner, force bool) {
	if o.sessions == nil || o.client == nil || o.sessionID == "" || o.root == "" || o.path == "" {
		return
	}
	if !s.fileOpenWithSourceCurrent(o) {
		return
	}
	s.mu.Lock()
	if !force && s.fileOpenWithOwner == o {
		s.mu.Unlock()
		return
	}
	s.fileOpenWithOwner = o
	s.fileOpenWithFor = o.path
	s.fileOpenWithApps = nil
	s.fileOpenWithState = files.OpenWithState{Loading: true}
	s.mu.Unlock()
	if !s.fileOpenWithCurrent(o) {
		return
	}
	target, err := o.sessions.CaptureComposerTarget(o.sessionID)
	if err != nil {
		s.mu.Lock()
		if s.fileOpenWithOwner == o {
			s.fileOpenWithState = files.OpenWithState{Error: err.Error()}
		}
		s.mu.Unlock()
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		var apps api.WorkspacePathApplicationsResult
		path := strings.ReplaceAll(o.path, `\`, "/")
		if !strings.HasPrefix(path, "/") && !(len(path) > 1 && path[1] == ':') {
			path = strings.TrimRight(strings.ReplaceAll(o.root, `\`, "/"), "/") + "/" + path
		}
		err := o.client.Call(ctx, "session/workspacePathApplications", map[string]any{"request": api.WorkspacePathApplicationsRequest{Path: path}}, &apps)
		if !s.fileOpenWithCurrent(o) || !target.NavigationCurrent() {
			return
		}
		publish := func() {
			if !s.fileOpenWithCurrent(o) || !target.NavigationCurrent() {
				return
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			s.fileOpenWithState = files.OpenWithState{}
			if err != nil {
				s.fileOpenWithState.Error = err.Error()
				return
			}
			for _, a := range apps {
				name := strings.TrimSpace(a.Name)
				if name == "" {
					name = a.Id
				}
				s.fileOpenWithApps = append(s.fileOpenWithApps, files.OpenWithApp{ID: a.Id, Name: name, Default: a.Default})
			}
		}
		if win != nil {
			win.Update(publish)
		}
	}()
}

func (s *mainState) openWorkspacePathWithOwner(o fileOpenWithOwner, path, app string) {
	if !s.fileOpenWithCurrent(o) {
		return
	}
	rel, ok := files.WorkspaceRelativePath(o.root, path)
	if !ok || strings.ReplaceAll(rel, `\`, "/") != strings.ReplaceAll(o.path, `\`, "/") {
		return
	}
	target, err := o.sessions.CaptureComposerTarget(o.sessionID)
	if err != nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		var out api.SessionOpenWorkspacePathValue
		request := api.SessionOpenWorkspacePathRequest{Path: path}
		if app != "" {
			request.Application = &app
		}
		if !s.fileOpenWithCurrent(o) || !target.NavigationCurrent() {
			return
		}
		err := o.client.Call(ctx, "session/openWorkspacePath", map[string]any{"request": request}, &out)
		if !s.fileOpenWithCurrent(o) || !target.NavigationCurrent() {
			return
		}
		if err != nil {
			o.sessions.SetLastAction(err.Error())
		} else if !out.Opened {
			o.sessions.SetLastAction(resumeText("未能使用所选应用打开文件。", "The selected application could not open the file."))
		}
	}()
}

func (s *mainState) workspaceImageReader(sessions *session.Store, client *dsh.Client, st harness.Status, sid, root string) func(context.Context, string, int64) ([]byte, error) {
	reader, valid, err := sessions.MediaPreviewSource(sid)
	return func(ctx context.Context, path string, limit int64) ([]byte, error) {
		if err != nil {
			return nil, err
		}
		current := func() bool {
			s.mu.Lock()
			ok := s.sessions == sessions && s.bootStatus.Generation == st.Generation && s.bootStatus.Endpoint == st.Endpoint && !s.quitting
			s.mu.Unlock()
			s.dshMu.Lock()
			same := s.dshClient == client
			s.dshMu.Unlock()
			v := sessions.SelectionSummary()
			return ok && same && valid() && v.SelectedID == sid && v.SelectedCwd == root
		}
		if limit <= 0 || limit > int64(^uint(0)>>1) {
			return nil, fmt.Errorf("invalid image byte limit")
		}
		return mediapreview.ReadSnapshot(ctx, reader, sid, path, int(limit), current)
	}
}

func (s *mainState) managedMarketProfile(st harness.Status) string {
	s.mu.Lock()
	sup := s.sup
	s.mu.Unlock()
	if sup == nil || st.State != harness.Ready || st.Generation == 0 {
		return ""
	}
	owned := sup.Status()
	if owned.State != harness.Ready || owned.Generation != st.Generation || owned.Endpoint != st.Endpoint || !strings.HasPrefix(owned.Source, "vendor ") {
		return ""
	}
	return paths.WebProfileDir()
}
