package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	desktopgit "whaleisle/internal/desktop/git"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/diffsurface"
	"whaleisle/internal/ui/frame"
	"whaleisle/internal/ui/i18n"
)

type captionListTransport struct {
	root  string
	calls atomic.Int32
}

func (f *captionListTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	f.calls.Add(1)
	var request struct {
		Type, Method string
		RPCID        string `json:"rpcId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	if request.Type != "client-request" || request.Method != "session/list" || request.RPCID == "" {
		return nil, fmt.Errorf("caption fixture rejected unexpected Host/model request: %s", request.Method)
	}
	value := api.SessionListValue{Items: []api.SessionListValueItemsItem{{SessionId: "fixture", Cwd: &f.root}}}
	body, err := json.Marshal(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": map[string]any{"ok": true, "value": value}})
	if err != nil {
		return nil, err
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
}

func captionSelectedStore(t *testing.T, root string) (*session.Store, *captionListTransport) {
	t.Helper()
	store := session.New()
	// Preserve selection before Attach: accepting the first roster must not
	// automatically select a row and start follow/catalog/model subscriptions.
	store.Select("fixture")
	transport := &captionListTransport{root: root}
	store.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://caption.fixture.invalid"}, HTTP: &http.Client{Transport: transport}}, nil)
	t.Cleanup(store.Detach)
	store.RefreshList(context.Background())
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		snapshot := store.Snapshot()
		if !snapshot.ListLoading && snapshot.SelectedID == "fixture" && snapshot.SelectedCwd == root && snapshot.ListErr == "" {
			return store, transport
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("accepted roster did not establish the selected cwd: %+v", store.Snapshot())
	return nil, nil
}

func captionEmptyRepository(t *testing.T, dirty bool) string {
	t.Helper()
	root, hooks := t.TempDir(), t.TempDir()
	// Fixture setup runs only during the coordinated test run, writes only to
	// this temporary cwd, and never adds commits, remotes or account state.
	cmd := exec.Command("git", "-c", "core.hooksPath="+hooks, "-c", "commit.gpgsign=false", "init", "-b", "main")
	cmd.Dir = root
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(entry), "GIT_") {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	cmd.Env = append(cmd.Env, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_TERMINAL_PROMPT=0")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("temporary git init: %v: %s", err, output)
	}
	if dirty {
		if err := os.WriteFile(filepath.Join(root, "untracked.txt"), []byte("caption fixture\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestWorkspaceCaptionActionsKeepCurrentPageAndDock(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, action := range []struct {
		name, button, modalControl string
		dirty                      bool
	}{
		{"commit", "Commit", "Review selected files to commit…", true},
		{"publish", "Publish repository", "Review publish target", false},
	} {
		t.Run(action.name, func(t *testing.T) {
			root := captionEmptyRepository(t, action.dirty)
			for _, global := range []bool{false, true} {
				for _, dock := range []struct {
					name, tab string
					open, pr  bool
				}{
					{"closed-files", "files", false, true},
					{"open-files", "files", true, false},
					{"open-diff", "diff", true, true},
					{"open-agents", "agents", true, false},
				} {
					t.Run(fmt.Sprintf("global-%t/%s", global, dock.name), func(t *testing.T) {
						store, transport := captionSelectedStore(t, root)
						status := &desktopgit.RepositoryStatus{IsRepo: true, Branch: "main", Dirty: action.dirty}
						state := &mainState{filesPaneOpen: dock.open, rightPaneTab: dock.tab, gitPROpen: dock.pr, gitRepositoryStatus: status, gitCheckedRoot: root, gitCheckedAt: time.Now()}
						workbench := &diffsurface.Workbench{}
						defer workbench.ClosePane()
						model := frame.Model{Store: store, GitWorkbench: workbench, WorkspacePage: global, RightPaneOpen: dock.open, RightPaneTab: &state.rightPaneTab}
						state.bindWorkspaceTrailing(nil, &model, nil, status, false, root, "explicit fixture message")
						if model.WorkspaceTrailing == nil {
							t.Fatal("main binder omitted the caption producer")
						}
						tt := ui.NewTester(func(c *ui.Context) { model.WorkspaceTrailing(c, "compact") }, 900, 750)
						assertUnexecuted := func() {
							t.Helper()
							if workbench.Operation != nil || workbench.Progress != nil || workbench.Commit != nil && (workbench.Commit.Task != nil || workbench.Commit.Plan != nil || workbench.Commit.StackPlan != nil) || workbench.Remote != nil && (workbench.Remote.Task != nil || workbench.Remote.Generation != nil || workbench.Remote.Plan != nil) || workbench.PublishReview {
								t.Fatal("painting/opening a caption review prepared a model, execution plan or Git operation")
							}
							if transport.calls.Load() != 1 {
								t.Fatalf("caption reached extra Host/model endpoints: %d requests", transport.calls.Load())
							}
						}
						assertDock := func() {
							t.Helper()
							if state.filesPaneOpen != dock.open || state.rightPaneTab != dock.tab || state.gitPROpen != dock.pr || model.RightPaneOpen != dock.open || model.WorkspacePage != global || model.GitWorkbench != workbench {
								t.Fatalf("caption replaced the current page/dock/Workbench: open=%t tab=%s pr=%t global=%t", state.filesPaneOpen, state.rightPaneTab, state.gitPROpen, model.WorkspacePage)
							}
						}
						if workbench.CaptionReviewOpen() {
							t.Fatal("painting the caption opened a modal")
						}
						assertUnexecuted()
						if err := tt.Click(action.button); err != nil {
							t.Fatal(err)
						}
						if !workbench.CaptionReviewOpen() || !tt.HasText(action.modalControl) || workbench.Root != root {
							t.Fatal("actual titlebar button did not expose the existing Workbench review", tt.Texts())
						}
						if action.dirty && (workbench.Commit == nil || workbench.Message != "explicit fixture message" || !tt.HasText("untracked.txt")) {
							t.Fatal("Commit opened another review instance or lost its actual selected files")
						}
						if !action.dirty && (!workbench.PublishOpen || workbench.PublishName != filepath.Base(root)) {
							t.Fatal("Publish did not use the same Workbench publish form")
						}
						assertDock()
						assertUnexecuted()
						if err := tt.Click("Close"); err != nil {
							t.Fatal(err)
						}
						if workbench.CaptionReviewOpen() {
							t.Fatal("actual Close left the caption modal open")
						}
						assertDock()
						assertUnexecuted()
						if _, err := os.Stat(filepath.Join(root, ".git", "refs", "heads", "main")); !os.IsNotExist(err) {
							t.Fatal("opening/closing the review created a commit", err)
						}
					})
				}
			}
		})
	}
}
