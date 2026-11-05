package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/pluginui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/recovery"
	remotestore "whaleisle/internal/store/remote"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
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

func mainParityComposerFixture(t *testing.T) (*mainState, nativePluginComposerBridge) {
	t.Helper()
	ep := harness.Endpoint{Origin: "http://127.0.0.1:41373", Cookie: "fixture=local"}
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: ep.Origin, Cookie: ep.Cookie}}
	store := session.New()
	store.Attach(client, nil)
	t.Cleanup(store.Detach)
	setDeliverableFixtureField(t, store, "selectedID", "composer-owner")
	setDeliverableFixtureField(t, store, "items", []session.Item{{SessionID: "composer-owner", Cwd: t.TempDir()}})
	state := &mainState{sessions: store, dshClient: client, bootStatus: harness.Status{State: harness.Ready, Generation: 4, Endpoint: ep}}
	state.nativePlugins.session = &nativePluginSessionState{visible: true, sessions: store, endpoint: ep, generation: 4, subject: pluginSessionStoreSubject(store)}
	return state, nativePluginComposerBridge{state: state}
}

func TestMainParityPluginComposerRejectsEveryStaleOwnerField(t *testing.T) {
	s, b := mainParityComposerFixture(t)
	o, err := b.capture(s.nativePlugins.session.subject)
	if err != nil {
		t.Fatal(err)
	}
	o.draft.Text = "preserve my native text"
	o.draft.Client.Sync()
	owner := composerPluginBinding(o)
	for _, field := range []string{"subject-session", "subject-epoch", "binding", "revision", "target", "edit"} {
		t.Run(field, func(t *testing.T) {
			stale := owner
			switch field {
			case "subject-session":
				stale.Subject.SessionID = "other"
			case "subject-epoch":
				stale.Subject.Epoch += "-old"
			case "binding":
				stale.Binding += "-old"
			case "revision":
				stale.Revision++
			case "target":
				stale.TargetID = "other"
			case "edit":
				stale.EditSeq = 9
			}
			if _, err := b.validate(stale); err == nil {
				t.Fatal("stale owner accepted")
			}
			if o.draft.Text != "preserve my native text" {
				t.Fatal("stale validation mutated draft")
			}
		})
	}
	if _, err := b.validate(owner); err != nil {
		t.Fatal(err)
	}
	o.draft.Attachments = []session.ComposerAttachment{{PreparationID: 17, Path: filepath.Join(t.TempDir(), "private.png"), Name: "private.png"}}
	if _, err := b.validate(owner); err == nil {
		t.Fatal("attachment replacement retained old binding")
	}
	current, err := b.capture(owner.Subject)
	if err != nil {
		t.Fatal(err)
	}
	if current.binding == owner.Binding {
		t.Fatal("attachment owner binding was reused")
	}
	encoded, _ := json.Marshal(pluginComposerSnapshot(current))
	privatePath, _ := json.Marshal(o.draft.Attachments[0].Path)
	if strings.Contains(string(encoded), string(privatePath)) {
		t.Fatal("plugin snapshot exposes private attachment paths")
	}
	s.bootStatus.Generation++
	if _, err := b.validate(composerPluginBinding(current)); err == nil {
		t.Fatal("old Host generation accepted")
	}
}

func mainParityPluginRetryFixture(t *testing.T) (nativePluginComposerBridge, *nativePluginComposerOwner, uint64, string) {
	t.Helper()
	s, b := mainParityComposerFixture(t)
	d := s.composerDrafts["composer-owner"]
	if d == nil {
		s.mu.Lock()
		d = s.composerDraftLocked("composer-owner")
		s.mu.Unlock()
	}
	const encoded = "R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7"
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "native-retry.gif")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	d.Text = "keep authored native text"
	d.Client.Sync()
	d.Attachments = []session.ComposerAttachment{{Path: path, Name: "native-retry.gif"}}
	a, ok := d.BeginAttachmentPreparation(0, false)
	if !ok {
		t.Fatal("initial native attempt was refused")
	}
	d.CompleteAttachmentPreparation(0, a, a, errors.New("fixture upload failed"))
	d.Attachments[0].TotalBytes = int64(len(raw))
	d.Attachments[0].MediaType = "image/gif"
	o, err := b.capture(s.nativePlugins.session.subject)
	if err != nil {
		t.Fatal(err)
	}
	return b, o, a.PreparationID, encoded
}

func TestMainParityPluginComposerRetryRejectsStaleAndMissingAttachments(t *testing.T) {
	for _, change := range []string{"subject", "binding", "revision", "target", "edit", "Host", "session", "attachment-replaced", "missing", "not-failed", "mixed-action"} {
		t.Run(change, func(t *testing.T) {
			b, o, id, _ := mainParityPluginRetryFixture(t)
			edit := pluginui.ComposerEdit{Owner: composerPluginBinding(o), RetryFileID: id}
			switch change {
			case "subject":
				edit.Owner.Subject.SessionID = "other"
			case "binding":
				edit.Owner.Binding += "-old"
			case "revision":
				edit.Owner.Revision++
			case "target":
				edit.Owner.TargetID = "other"
			case "edit":
				edit.Owner.EditSeq++
			case "Host":
				b.state.bootStatus.Generation++
			case "session":
				setDeliverableFixtureField(t, o.sessions, "selectedID", "other")
			case "attachment-replaced":
				o.draft.Attachments[0].PreparationID++
			case "missing":
				edit.RetryFileID++
			case "not-failed":
				o.draft.Attachments[0].PrepareError = ""
				o, _ = b.capture(o.subject)
				edit.Owner = composerPluginBinding(o)
			case "mixed-action":
				edit.RemoveFileID = id
			}
			before := o.draft.Attachments[0]
			if err := b.applyOnUI(edit, func(func()) { t.Error("refused retry posted work") }); err == nil {
				t.Fatal("invalid retry was admitted")
			}
			if o.draft.Preparing != 0 || o.draft.Attachments[0] != before || o.draft.Text != "keep authored native text" {
				t.Fatal("refused retry changed the draft")
			}
		})
	}
}

func TestMainParityPluginComposerRetryUsesNativePreparation(t *testing.T) {
	b, o, id, encoded := mainParityPluginRetryFixture(t)
	files := pluginComposerSnapshot(o).Attachments
	if len(files) != 1 || files[0].ID != id || files[0].Size == 0 || files[0].MediaType != "image/gif" || files[0].Error != "fixture upload failed" {
		t.Fatal("public attachment metadata lost native failure/known size/type")
	}
	oldBinding := composerPluginBinding(o)
	posted := make(chan func(), 4)
	if err := b.applyOnUI(pluginui.ComposerEdit{Owner: oldBinding, RetryFileID: id}, func(fn func()) { posted <- fn }); err != nil {
		t.Fatal(err)
	}
	if o.draft.Preparing != 1 || o.draft.Attachments[0].PreparationID == id || o.draft.Attachments[0].PrepareError != "" {
		t.Fatal("retry did not start a fresh native preparation attempt")
	}
	select {
	case complete := <-posted:
		complete()
	case <-time.After(3 * time.Second):
		t.Fatal("native local-image retry never completed")
	}
	a := o.draft.Attachments[0]
	if o.draft.Preparing != 0 || a.PrepareError != "" || a.SessionID != o.subject.SessionID || a.HostScope == "" || a.ImageData != encoded || a.MediaType != "image/gif" || o.draft.Text != "keep authored native text" {
		t.Fatalf("native retry lost exact bytes/owner or changed authored text: %+v", a)
	}
	if _, err := b.validate(oldBinding); err == nil {
		t.Fatal("successful retry retained the old attachment owner binding")
	}
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

func TestMainParityPadHandsOffDraftWithoutSubmitting(t *testing.T) {
	d := composer.NewDraftState("")
	if err := applyPadComposerDraft(d, "build a panel", false); err != nil {
		t.Fatal(err)
	}
	if d.Text != "build a panel" || d.Sending || d.TargetID != "" {
		t.Fatal("ask did not retain an unsubmitted draft")
	}
	before := string(d.SaveDraft())
	if err := applyPadComposerDraft(d, "replace user draft", false); err == nil {
		t.Fatal("ask replaced an existing user draft")
	}
	if string(d.SaveDraft()) != before {
		t.Fatal("refused ask changed the draft")
	}
	d = composer.NewDraftState("A😀BC")
	d.Client.SetSelectionUTF16(1, 3)
	if err := applyPadComposerDraft(d, "x", true); err != nil {
		t.Fatal(err)
	}
	if d.Text != "AxBC" || d.Sending {
		t.Fatalf("selection replacement got %q", d.Text)
	}
	d.Client.SetSelectionUTF16(2, 2)
	if err := applyPadComposerDraft(d, "y", true); err != nil {
		t.Fatal(err)
	}
	if d.Text != "AxyBC" {
		t.Fatalf("caret insertion got %q", d.Text)
	}
}

func TestMainParityProjectFileLifetimeDoesNotFollowChatSelection(t *testing.T) {
	s, _ := mainParityComposerFixture(t)
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
	s, _ := mainParityComposerFixture(t)
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
	s, _ := mainParityComposerFixture(t)
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

func TestMainParityElectedRootRetainsNativeCaptionAndExplicitRoutes(t *testing.T) {
	for _, width := range []int{1200, 400} {
		t.Run(strconv.Itoa(width), func(t *testing.T) {
			subject := pluginui.Subject{Kind: "navigation"}
			owner := &pluginui.Owner{}
			setDeliverableFixtureField(t, owner, "selected", subject)
			setDeliverableFixtureField(t, owner, "state", pluginui.Snapshot{Connected: true, Page: 1, Revision: 1, Subject: &subject,
				GlobalDocuments: []pluginui.NativeDocument{{Slot: "root", Handle: "actual-extension", Nodes: []pluginui.NativeNode{{Kind: "text", Text: "Third-party client root"}}}}})
			s, _ := mainParityComposerFixture(t)
			s.nativePlugins.navigation = &nativePluginNavigationState{owner: owner}
			ready, nativeRoute := true, false
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() {
					if !s.paintPluginRootClient(c, ready, nativeRoute, chrome.DefaultTitleBarColors()) {
						ui.Text(c, "Explicit native route")
					}
				})
			}, width, 800)
			caption, ok := tt.Find("Plugin root native caption")
			body, hasBody := tt.Find("Third-party client root")
			viewport, hasViewport := tt.Find("Plugin root client body")
			if !ok || caption.Y != 0 || caption.H != 48 || !hasBody || body.Y < 48 || !hasViewport || viewport.Y != 48 || viewport.H != 752 || viewport.W != float32(width) || tt.HasText("Explicit native route") {
				t.Fatalf("root replaced native caption or failed admission: caption=%+v body=%+v texts=%q", caption, body, tt.Texts())
			}
			saveSettingsSectionEvidence(t, tt, "main-plugin-root-"+strconv.Itoa(width))
			nativeRoute = true
			tt.Frame()
			if !tt.HasText("Explicit native route") || tt.HasText("Third-party client root") {
				t.Fatal("root swallowed explicit launcher/settings route")
			}
			nativeRoute, s.dirPickOpen = false, true
			tt.Frame()
			if !tt.HasText("Explicit native route") || tt.HasText("Third-party client root") {
				t.Fatal("root swallowed native directory picker")
			}
			s.dirPickOpen = false
			s.sessions.OpenWorkspacePicker()
			tt.Frame()
			request := s.sessions.Snapshot().WorkspaceNavigation
			if request == nil || !tt.HasText("Explicit native route") || tt.HasText("Third-party client root") {
				t.Fatal("root swallowed native workspace dialog")
			}
			s.sessions.CancelWorkspaceNavigation(request.Token)
			ready = false
			tt.Frame()
			if !tt.HasText("Explicit native route") || tt.HasText("Third-party client root") {
				t.Fatal("root replaced non-ready recovery route")
			}
			ready = true
			owner.Close()
			tt.Frame()
			if !tt.HasText("Explicit native route") || tt.HasText("Third-party client root") {
				t.Fatal("withdrawn root kept native navigation hidden")
			}
		})
	}
}
