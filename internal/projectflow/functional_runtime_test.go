package projectflow

// Explicit opt-in integration against the real isolated rc.2 Host. No provider
// credentials, prompts, queue mutations, delegation or worker tools are used.
import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
)

func TestFunctionalProjectRuntime(t *testing.T) {
	stage, base, output := os.Getenv("WHALEISLE_PROJECT_TEST_STAGE"), os.Getenv("WHALEISLE_PROJECT_TEST_ROOT"), os.Getenv("WHALEISLE_PROJECT_TEST_REPORT")
	if stage == "" || base == "" || output == "" {
		t.Skip("opt-in: set WHALEISLE_PROJECT_TEST_STAGE, ROOT and REPORT to isolated task paths")
	}
	type check struct {
		Name   string `json:"name"`
		Pass   bool   `json:"pass"`
		Count  int    `json:"count,omitempty"`
		SHA256 string `json:"sha256,omitempty"`
	}
	report := struct {
		SchemaVersion             int      `json:"schemaVersion"`
		Scope                     string   `json:"scope"`
		OwnedRoot                 string   `json:"ownedRoot"`
		Checks                    []check  `json:"checks"`
		Errors                    []string `json:"errors,omitempty"`
		Passed                    bool     `json:"passed"`
		HostStarts                int      `json:"hostStarts"`
		OwnedHostStopped          bool     `json:"ownedHostStopped"`
		OwnedListenerClosed       bool     `json:"ownedListenerClosed"`
		PromptOrWorkMutationCalls int      `json:"promptOrWorkMutationCalls"`
		DocumentSeed              string   `json:"documentSeed"`
		NotCovered                []string `json:"notCovered"`
	}{SchemaVersion: 1, Scope: "actual rc.2 Host + production Project Go client and bridge; backend integration",
		DocumentSeed: "new static document in this owned Project docs directory; actual Host list/read, not worker artifact creation",
		NotCovered:   []string{"native UI clicks/rendering", "model-backed requests, queue execution, delegation and worker artifacts"}}
	defer func() {
		report.Passed = !t.Failed()
		raw, err := json.MarshalIndent(report, "", "  ")
		if err == nil {
			err = os.WriteFile(output, append(raw, '\n'), 0600)
		}
		if err != nil {
			t.Errorf("write isolated report: %v", err)
		}
	}()
	require := func(err error) {
		t.Helper()
		if err != nil {
			report.Errors = append(report.Errors, err.Error())
			t.Fatal(err)
		}
	}
	assert := func(name string, pass bool, count int, body string) {
		t.Helper()
		c := check{Name: name, Pass: pass, Count: count}
		if body != "" {
			sum := sha256.Sum256([]byte(body))
			c.SHA256 = hex.EncodeToString(sum[:])
		}
		report.Checks = append(report.Checks, c)
		if !pass {
			t.Fatalf("%s failed", name)
		}
	}
	root, err := os.MkdirTemp(base, "project-functional-check-")
	require(err)
	report.OwnedRoot = root
	userData := filepath.Join(root, "user-data")
	home := filepath.Join(userData, "dsh-home")
	workspace := filepath.Join(root, "workspace")
	require(os.MkdirAll(workspace, 0700))
	vendor := filepath.Join(stage, "vendor")
	harnessRoot := filepath.Join(vendor, "deepseek-harness")
	node := filepath.Join(stage, "node", "node.exe")
	profile := filepath.Join(home, "profiles", "web")
	t.Setenv(paths.EnvUserData, userData)
	t.Setenv(paths.EnvVendor, vendor)
	t.Setenv("DSH_HOME", home)
	// This trusted picker callback selects only this test's already-owned path.
	// It exercises the production bridge authority and canonicalization, not UI.
	bridge := harness.NewProjectBridge(node, home, workspace, nil, func(ctx context.Context, suggested string) (string, error) {
		if suggested != workspace {
			return "", fmt.Errorf("unexpected picker scope")
		}
		return workspace, nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	assert("production-project-bridge-ready", bridge.Ready(ctx), 0, "")
	overlay, err := bridge.PrepareOverlay(ctx, filepath.Join(vendor, "dsh-project"), profile, harnessRoot)
	require(err)
	supervisor := harness.New(harness.Options{
		Install:   harness.NewInstallServer(harness.InstallServerOptions{NodeBin: node, Project: bridge.Dispatch}),
		Launch:    harness.LaunchOptions{NodeBin: node, HarnessRoot: harnessRoot, VendorDir: vendor, Host: "127.0.0.1", Port: 32183, Home: home, PatchFiles: []string{overlay}},
		Prepare:   harness.PrepareOptions{VendorDir: vendor, HarnessRoot: harnessRoot, DshHome: home, ProfileDir: profile, RemoteSet: true, Remote: false},
		Workspace: workspace, ReadyTimeout: 80 * time.Second,
	})
	var endpoint harness.Endpoint
	stop := func() {
		supervisor.Stop()
		report.OwnedHostStopped = supervisor.Status().State == harness.Stopped
		if endpoint.Origin != "" {
			u, e := url.Parse(endpoint.Origin)
			if e == nil {
				conn, e := net.DialTimeout("tcp", u.Host, time.Second)
				report.OwnedListenerClosed = e != nil
				if conn != nil {
					conn.Close()
				}
			}
		}
	}
	defer stop()
	start := func() {
		t.Helper()
		supervisor.Start()
		report.HostStarts++
		for {
			status := supervisor.Status()
			if status.State == harness.Ready {
				endpoint = status.Endpoint
				return
			}
			if status.State == harness.Failed {
				require(status.Err)
				t.Fatal("Host failed without error")
			}
			select {
			case <-ctx.Done():
				require(ctx.Err())
			case <-time.After(25 * time.Millisecond):
			}
		}
	}
	client := func() Client {
		return Client{Origin: endpoint.Origin, Cookie: endpoint.Cookie, HTTP: &http.Client{Timeout: 15 * time.Second}}
	}
	start()
	c := client()
	listing, err := c.List(ctx)
	require(err)
	assert("PROJ-01-real-empty-list", listing.Available && listing.Error == "" && len(listing.Projects) == 0, len(listing.Projects), "")
	project, err := c.Create(ctx, workspace, []string{workspace}, "owned-project-create-1")
	require(err)
	assert("PROJ-01-real-create", project.ID != "" && project.CoordinatorSessionID != "" && project.Lifecycle == "ready", 1, "")
	again, err := c.Create(ctx, workspace, []string{workspace}, "owned-project-create-2")
	require(err)
	assert("PROJ-01-directory-identity-reuse", again.ID == project.ID && again.CoordinatorSessionID == project.CoordinatorSessionID, 1, "")
	project, err = c.Rename(ctx, project.ID, "Owned local Project integration")
	require(err)
	lead, err := c.Open(ctx, project.ID)
	require(err)
	detail, err := c.LifecycleProject(ctx, project.ID)
	require(err)
	assert("PROJ-01-rename-detail-original-lead", project.Title == "Owned local Project integration" && lead == project.CoordinatorSessionID && detail.ID == project.ID && !detail.HasActiveWork(), 1, "")
	texts := map[string]string{"notes.md": "Owned notes body. No agent request.", "preferences.md": "Owned local preference.", "charter.md": "Owned project rules. No model execution."}
	for _, name := range []string{"notes.md", "preferences.md", "charter.md"} {
		before, err := c.ReadMaterial(ctx, project.ID, name)
		require(err)
		saved, err := c.WriteMaterial(ctx, project.ID, name, texts[name])
		require(err)
		read, err := c.ReadMaterial(ctx, project.ID, name)
		require(err)
		ok := saved.Editable() == texts[name] && read.Editable() == texts[name]
		if name == "notes.md" {
			ok = ok && before.GeneratedText != "" && read.GeneratedText == before.GeneratedText && strings.Contains(read.Text, NotesMarker)
		}
		assert("PROJ-02-material-roundtrip-"+name, ok, 1, read.Editable())
	}
	docs, err := c.Documents(ctx, project.ID)
	require(err)
	assert("PROJ-03-empty-documents", len(docs) == 0, len(docs), "")
	var wire struct {
		Project struct {
			StorageRoot string `json:"storageRoot"`
		} `json:"project"`
	}
	require(c.call(ctx, "detail", map[string]any{"projectId": project.ID}, &wire))
	rel, err := filepath.Rel(home, wire.Project.StorageRoot)
	require(err)
	assert("owned-document-seed-root", wire.Project.StorageRoot != "" && !filepath.IsAbs(rel) && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)), 0, "")
	documentText := "Owned Project document. Read by the actual Host.\n"
	require(os.WriteFile(filepath.Join(wire.Project.StorageRoot, "docs", "owned-check.md"), []byte(documentText), 0600))
	docs, err = c.Documents(ctx, project.ID)
	require(err)
	assert("PROJ-03-real-document-inventory", len(docs) == 1 && docs[0].Path == "docs/owned-check.md", len(docs), "")
	document, err := c.ReadDocument(ctx, project.ID, docs[0])
	require(err)
	assert("PROJ-03-real-document-read", document.Text == documentText, 1, document.Text)
	queue, err := c.RequestQueue(ctx, project.ID)
	require(err)
	requests, err := c.Requests(ctx, project.ID, "current", "")
	require(err)
	work, err := c.CurrentWork(ctx, project.ID, "")
	require(err)
	assert("PROJ-04-safe-empty-request-queue-work-reads", queue != nil && queue.Active == nil && len(queue.Queued) == 0 && len(requests.Items) == 0 && len(work.Workstreams) == 0 && len(work.Workers) == 0, 0, "")
	archived, err := c.SetProjectLifecycle(ctx, project.ID, "archive")
	require(err)
	assert("PROJ-05-real-archive", archived.Project.Lifecycle == "archived", 1, "")
	_, err = c.WriteMaterial(ctx, project.ID, "preferences.md", "must not save while archived")
	assert("PROJ-05-archive-rejects-material-write", err != nil, 0, "")
	restored, err := c.SetProjectLifecycle(ctx, project.ID, "restore")
	require(err)
	assert("PROJ-05-real-restore", restored.Project.Lifecycle == "ready" && !restored.Project.HasActiveWork(), 1, "")
	stop()
	assert("first-owned-host-listener-closed", report.OwnedHostStopped && report.OwnedListenerClosed, 0, "")
	start()
	c = client()
	listing, err = c.List(ctx)
	require(err)
	assert("PROJ-05-restart-project-list", listing.Available && len(listing.Projects) == 1 && listing.Projects[0].ID == project.ID && listing.Projects[0].Title == project.Title, 1, "")
	detail, err = c.LifecycleProject(ctx, project.ID)
	require(err)
	assert("PROJ-05-restart-original-lead-idle", detail.CoordinatorSessionID == lead && detail.Lifecycle == "ready" && detail.Paused && !detail.HasActiveWork(), 1, "")
	for _, name := range []string{"notes.md", "preferences.md", "charter.md"} {
		read, err := c.ReadMaterial(ctx, project.ID, name)
		require(err)
		assert("PROJ-05-persisted-"+name, read.Editable() == texts[name], 1, read.Editable())
	}
	docs, err = c.Documents(ctx, project.ID)
	require(err)
	assert("PROJ-05-persisted-document-list", len(docs) == 1, len(docs), "")
	document, err = c.ReadDocument(ctx, project.ID, docs[0])
	require(err)
	assert("PROJ-05-persisted-document-body", document.Text == documentText, 1, document.Text)
}
