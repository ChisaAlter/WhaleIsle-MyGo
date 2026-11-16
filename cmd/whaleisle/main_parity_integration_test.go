package main

import (
	"context"
	"errors"
	"testing"

	"whaleisle/internal/config"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/recovery"
	remotestore "whaleisle/internal/store/remote"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

func TestMainParityChildDraftHasStableIndependentCheckpoint(t *testing.T) {
	s := &mainState{}
	a := session.AgentSessionAddress{Kind: "subagent", ParentSessionID: "parent", ChildSessionID: "child", Mode: "continuable"}
	d := s.childComposerDraft("transient-client-one", a)
	d.Text = "keep this child draft"
	if s.childComposerDraft("transient-client-two", a) != d || d.Text != "keep this child draft" {
		t.Fatal("Host rebinding discarded the draft")
	}
	other := a
	other.ParentSessionID = "another-parent"
	if s.childComposerDraft("transient-client-two", other) == d {
		t.Fatal("different child addresses alias")
	}
	s.mu.Lock()
	ordinary := s.composerDraftLocked(a.ChildSessionID)
	s.mu.Unlock()
	if ordinary == d {
		t.Fatal("child and root session drafts alias")
	}
	r, err := recovery.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.resumeStore = r
	s.mu.Lock()
	saved, err := s.captureResumeOwnedLocked()
	s.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	restored := &mainState{composerDrafts: map[string]*composer.DraftState{}}
	for key, raw := range saved.Drafts {
		d, err := composer.RestoreDraftChecked(raw)
		if err != nil {
			t.Fatal(err)
		}
		restored.composerDrafts[key] = d
	}
	if got := restored.childComposerDraft("new-process-client", a).Text; got != "keep this child draft" {
		t.Fatalf("restored text %q", got)
	}
}

func mainParityComposerFixture(t *testing.T) *mainState {
	t.Helper()
	ep := harness.Endpoint{Origin: "http://127.0.0.1:41373", Cookie: "fixture=local"}
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: ep.Origin, Cookie: ep.Cookie}}
	store := session.New()
	store.Attach(client, nil)
	t.Cleanup(store.Detach)
	setDeliverableFixtureField(t, store, "selectedID", "composer-owner")
	setDeliverableFixtureField(t, store, "items", []session.Item{{SessionID: "composer-owner", Cwd: t.TempDir()}})
	return &mainState{sessions: store, dshClient: client, bootStatus: harness.Status{State: harness.Ready, Generation: 4, Endpoint: ep}}
}

func TestMainParityProjectTargetKeepsActualRoot(t *testing.T) {
	for _, tc := range []struct {
		root, path, want string
		valid            bool
	}{
		{"/project/actual", "/project/actual/report.md", "report.md", true},
		{`C:\Project\Actual`, `c:\project\actual\report.md`, "report.md", true},
		{"/project/actual", "/project/actual/../escape.md", "", false},
		{"/project/actual", "/project/actual-other/report.md", "", false},
	} {
		t.Run(tc.path, func(t *testing.T) {
			rel, err := projectTargetRelative(projectflow.FileTarget{ProjectID: "project", CoordinatorSessionID: "coordinator", Root: tc.root, CanonicalPath: tc.path})
			if (err == nil) != tc.valid || rel != tc.want {
				t.Fatalf("got %q, %v", rel, err)
			}
		})
	}
}

func TestMainParityProjectFileLifetimeDoesNotFollowChatSelection(t *testing.T) {
	s := mainParityComposerFixture(t)
	sessions, client, host := s.sessions, s.dshClient, s.bootStatus
	if !s.projectFileAccessCurrent(sessions, client, host) {
		t.Fatal("initial Project access rejected")
	}
	setDeliverableFixtureField(t, sessions, "selectedID", "another-chat")
	if !s.projectFileAccessCurrent(sessions, client, host) {
		t.Fatal("switching chat revoked retained Project file access")
	}
	s.bootStatus.Generation++
	if s.projectFileAccessCurrent(sessions, client, host) {
		t.Fatal("replaced Host retained Project file access")
	}
}

func TestMainParityPRViewKeepsExistingOfflineRootScope(t *testing.T) {
	s := mainParityComposerFixture(t)
	s.sessions.Detach()
	s.dshClient = nil
	s.bootStatus.State = harness.Stopped
	root := t.TempDir()
	setDeliverableFixtureField(t, s.sessions, "selectedID", "offline-chat")
	setDeliverableFixtureField(t, s.sessions, "items", []session.Item{{SessionID: "offline-chat", Cwd: root}})
	current := s.capturePRViewOwner(root)
	if !current() {
		t.Fatal("existing local Git view acquired a new Host prerequisite")
	}
	setDeliverableFixtureField(t, s.sessions, "selectedID", "replacement-chat")
	if current() {
		t.Fatal("old PR view owner survived workspace navigation")
	}
}

func mainParityRemoteChoiceFixture(t *testing.T) (*remoteWorkspaceAdoption, remotestore.DirectoryChoice) {
	t.Helper()
	s := mainParityComposerFixture(t)
	cfg, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = cfg.Update(func(c *config.Config) { c.Workspace = "/keep-current" }); err != nil {
		t.Fatal(err)
	}
	s.cfgStore = cfg
	remote := remotestore.New()
	remote.Attach(s.dshClient, nil)
	s.remoteMachines = remote
	choice := remotestore.DirectoryChoice{MachineID: "machine-one", RemotePath: "/remote/source", LocalMirror: "/host/original-mirror", Generation: remote.Snapshot().Generation, RequestID: 7}
	setDeliverableFixtureField(t, remote, "directoryRequestID", choice.RequestID)
	setDeliverableFixtureField(t, remote, "view", remotestore.Snapshot{Generation: choice.Generation, CurrentID: choice.MachineID, Directory: remotestore.DirectorySnapshot{Path: choice.RemotePath, SelectedPath: choice.RemotePath, LocalMirror: choice.LocalMirror, Loaded: true}})
	op := s.captureRemoteWorkspaceAdoption(nil, s.sessions, remote)
	if op == nil || !op.current(choice) {
		t.Fatal("initial mirror owner was not captured")
	}
	return op, choice
}

func TestMainParityRemoteMirrorCannotAdoptAfterOwnerChanges(t *testing.T) {
	for _, change := range []string{"Host", "client", "session", "machine", "request", "commit-lease"} {
		t.Run(change, func(t *testing.T) {
			op, choice := mainParityRemoteChoiceFixture(t)
			s := op.state
			switch change {
			case "Host":
				s.remoteWorkspaceLease.Add(1)
				s.bootStatus.Generation++
			case "client":
				s.remoteWorkspaceLease.Add(1)
				s.dshClient = &dsh.Client{}
			case "session":
				setDeliverableFixtureField(t, op.sessions, "selectedID", "another-session")
			case "machine":
				v := op.remote.Snapshot()
				v.CurrentID = "another-machine"
				setDeliverableFixtureField(t, op.remote, "view", v)
			case "request":
				setDeliverableFixtureField(t, op.remote, "directoryRequestID", choice.RequestID+1)
			case "commit-lease":
				s.remoteWorkspaceLease.Add(1)
			}
			if op.register(choice) {
				t.Fatal("late mirror was admitted")
			}
			if request := op.sessions.Snapshot().WorkspaceNavigation; request != nil {
				t.Fatal("late mirror opened a replacement workspace picker")
			}
			if got := op.config.Get().Workspace; got != "/keep-current" {
				t.Fatalf("workspace changed to %q", got)
			}
			if change == "commit-lease" || change == "Host" || change == "client" || change == "machine" || change == "request" {
				if err := op.persist(choice, choice.LocalMirror); !errors.Is(err, context.Canceled) {
					t.Fatalf("late commit error %v", err)
				}
				if op.config.Get().Workspace != "/keep-current" {
					t.Fatal("late commit persisted old mirror")
				}
			}
		})
	}
}

func TestMainParityRemoteMirrorCurrentReceiptMayPersist(t *testing.T) {
	op, choice := mainParityRemoteChoiceFixture(t)
	if err := op.persist(choice, choice.LocalMirror); err != nil {
		t.Fatal(err)
	}
	if got := op.config.Get().Workspace; got != choice.LocalMirror {
		t.Fatalf("current receipt persisted %q", got)
	}
}
