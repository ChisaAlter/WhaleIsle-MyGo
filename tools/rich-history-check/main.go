// Opt-in persisted-state rendering acceptance. It never prompts or answers a
// live interaction. Every record and file belongs to its new run directory.
package main

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/markdown"
)

//go:embed seed.mjs
var seedScript string

type seedManifest struct {
	Rows       []struct{ ID, Kind string }
	Attachment struct{ AttachmentID string }
}
type proof struct {
	ID, Layer string
	Messages  int
	Pass      bool
	Detail    string
}
type report struct {
	OwnedRunRoot                          string
	Results                               []proof
	Failure                               string `json:",omitempty"`
	ModelRequests                         int
	OwnedHostStopped, OwnedListenerClosed bool
	ReusedSavedState                      bool
	Limits                                []string
}

func wait(ctx context.Context, f func() (bool, error)) error {
	for {
		ok, err := f()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}
func main() {
	stage := flag.String("stage", "", "verified runtime")
	base := flag.String("root", "", "owned run parent")
	out := flag.String("out", "", "evidence JSON")
	native := flag.String("native", "", "optional native persisted-history executable")
	reuse := flag.String("reuse-run", "", "previously generated owned run; keep its exact saved artifacts")
	flag.Parse()
	if *stage == "" || *base == "" || *out == "" {
		panic("stage/root/out required")
	}
	r := report{Limits: []string{"Synthetic persisted-state rendering, not live Agent execution. No prompt, approval or user-question response RPC is invoked.", "Native captures are a fresh owned hidden Windows desktop per saved state; they do not prove real menu gestures or original pixel parity.", "Read-only disclosure/copy controls use production components and real Host snapshots in the UI tester; they are not native mouse gestures.", "Pending approval is a live waterfall, not reconstructed by its log-only audit. Continued question history is not an active timed waiter.", "Live streaming/cancellation, real workflow/subagent execution, compaction and goal changes remain unverified."}}
	if err := run(*stage, *base, *native, *reuse, &r); err != nil {
		r.Failure = err.Error()
	}
	raw, _ := json.MarshalIndent(r, "", "  ")
	if err := os.WriteFile(*out, append(raw, '\n'), 0600); err != nil {
		panic(err)
	}
	fmt.Println(string(raw))
	if r.Failure != "" {
		os.Exit(1)
	}
}
func run(stage, base, native, reuse string, r *report) error {
	root := reuse
	var err error
	if root == "" {
		root, err = os.MkdirTemp(base, "rich-history-check-")
	} else {
		rel, e := filepath.Rel(base, root)
		if e != nil || !filepath.IsAbs(root) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || !strings.HasPrefix(filepath.Base(root), "rich-history-check-") {
			return errors.New("reuse requires an existing owned rich-history run under root")
		}
		r.ReusedSavedState = true
	}
	if err != nil {
		return err
	}
	r.OwnedRunRoot = root
	data, home, cwd := filepath.Join(root, "user-data"), filepath.Join(root, "user-data", "dsh-home"), filepath.Join(root, "workspace")
	ownedUser := filepath.Join(root, "owned-userhome")
	for _, p := range []string{home, cwd, ownedUser} {
		if err = os.MkdirAll(p, 0700); err != nil {
			return err
		}
	}
	for k, v := range map[string]string{paths.EnvUserData: data, "DSH_HOME": home, "HOME": ownedUser, "USERPROFILE": ownedUser, "APPDATA": filepath.Join(ownedUser, "AppData", "Roaming"), "LOCALAPPDATA": filepath.Join(ownedUser, "AppData", "Local")} {
		if err = os.Setenv(k, v); err != nil {
			return err
		}
	}
	node, hroot := filepath.Join(stage, "node", "node.exe"), filepath.Join(stage, "vendor", "deepseek-harness")
	manifestPath := filepath.Join(root, "manifest.json")
	if reuse == "" {
		if err = os.WriteFile(filepath.Join(cwd, "owned-delivery.md"), []byte("# Owned delivered file\n\nRead-only synthetic acceptance.\n"), 0600); err != nil {
			return err
		}
		img := image.NewRGBA(image.Rect(0, 0, 96, 64))
		for y := 0; y < 64; y++ {
			for x := 0; x < 96; x++ {
				img.Set(x, y, color.RGBA{uint8(40 + x), uint8(80 + y), 180, 255})
			}
		}
		imagePath := filepath.Join(root, "owned-raster.png")
		f, err := os.Create(imagePath)
		if err != nil {
			return err
		}
		err = png.Encode(f, img)
		closeErr := f.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		script := filepath.Join(root, "seed.mjs")
		if err = os.WriteFile(script, []byte(seedScript), 0600); err != nil {
			return err
		}
		cmd := exec.Command(node, script, hroot, home, cwd, imagePath, manifestPath)
		seedOutput, err := cmd.CombinedOutput()
		if e := os.WriteFile(filepath.Join(root, "seed.log"), seedOutput, 0600); e != nil {
			return e
		}
		if err != nil {
			return fmt.Errorf("strict synthetic seed refused: %w", err)
		}
	}
	var manifest seedManifest
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(raw, &manifest); err != nil {
		return err
	}
	for _, row := range manifest.Rows {
		if row.ID != "rich-history-fixture-"+row.Kind {
			return errors.New("owned manifest identity mismatch")
		}
	}
	r.Results = append(r.Results, proof{ID: "current-v4-strict-restore", Layer: "pinned Session.fromRestore + physical catalog encoding", Pass: true, Detail: fmt.Sprintf("%d independently saved states; separate header/body zstd frames; original public attachment admission", len(manifest.Rows))})
	sup := harness.New(harness.Options{Install: harness.NewInstallServer(harness.InstallServerOptions{NodeBin: node}), Launch: harness.LaunchOptions{NodeBin: node, HarnessRoot: hroot, VendorDir: filepath.Join(stage, "vendor"), Host: "127.0.0.1", Port: 32183, Home: home}, Prepare: harness.PrepareOptions{VendorDir: filepath.Join(stage, "vendor"), HarnessRoot: hroot, DshHome: home, ProfileDir: filepath.Join(home, "profiles", "web"), RemoteSet: true, Remote: false}, Workspace: cwd, ReadyTimeout: 3 * time.Minute})
	var ep harness.Endpoint
	stop := func() {
		sup.Stop()
		r.OwnedHostStopped = sup.Status().State == harness.Stopped
		if ep.Origin != "" {
			u, _ := url.Parse(ep.Origin)
			c, e := net.DialTimeout("tcp", u.Host, time.Second)
			r.OwnedListenerClosed = e != nil
			if c != nil {
				c.Close()
			}
		}
	}
	defer stop()
	sup.Start()
	ctx, cancel := context.WithTimeout(context.Background(), 195*time.Second)
	defer cancel()
	if err = wait(ctx, func() (bool, error) {
		v := sup.Status()
		if v.State == harness.Failed {
			return false, v.Err
		}
		if v.State == harness.Ready {
			ep = v.Endpoint
			return true, nil
		}
		return false, nil
	}); err != nil {
		return err
	}
	endpoint := dsh.Endpoint{Origin: ep.Origin, Cookie: ep.Cookie}
	client := &dsh.Client{Endpoint: endpoint, HTTP: &http.Client{Timeout: 20 * time.Second}}
	mux := &dsh.Mux{Endpoint: endpoint, DialTimeout: 10 * time.Second}
	s := session.New()
	s.Attach(client, nil)
	s.AttachMux(mux)
	defer func() { s.Detach(); mux.Close() }()
	s.SetWorkspaceCwd(cwd)
	s.RefreshList(ctx)
	s.FollowWorkspace(ctx)
	if err = wait(ctx, func() (bool, error) {
		v := s.Snapshot()
		if v.ListErr != "" {
			return false, errors.New(v.ListErr)
		}
		return !v.ListLoading && len(v.Items) == len(manifest.Rows), nil
	}); err != nil {
		return err
	}
	session.SetAttachmentCacheDir(filepath.Join(root, "attachment-cache"))
	for _, row := range manifest.Rows {
		s.Select(row.ID)
		if err = wait(ctx, func() (bool, error) {
			v := s.Snapshot()
			if v.HistoryError != "" {
				return false, errors.New(v.HistoryError)
			}
			return v.SelectedID == row.ID && v.HistoryReady && !v.HistoryLoading, nil
		}); err != nil {
			return fmt.Errorf("%s follow: %w", row.Kind, err)
		}
		v := s.Snapshot()
		projection := map[string]any{"kind": row.Kind, "historyReady": v.HistoryReady, "messages": v.Messages, "workflowRuns": v.WorkflowRuns, "questions": v.QuestionProjection.Questions}
		projectionJSON, e := json.MarshalIndent(projection, "", "  ")
		if e != nil {
			return e
		}
		if e = os.WriteFile(filepath.Join(root, "projection-"+row.Kind+".json"), projectionJSON, 0600); e != nil {
			return e
		}
		if err = checkState(row.Kind, v); err != nil {
			return err
		}
		if err = readOnlyControls(row.Kind, v); err != nil {
			return err
		}
		if row.Kind == "media" {
			local, e := s.FetchAttachment(ctx, manifest.Attachment.AttachmentID)
			if e != nil {
				return e
			}
			bytes, e := os.ReadFile(local)
			if e != nil {
				return e
			}
			if _, e = ui.DecodeBitmap(bytes); e != nil {
				return e
			}
			sum := sha256.Sum256(bytes)
			r.Results = append(r.Results, proof{ID: "CHAT-14-image-read", Layer: "production session/attachment + Go image decode", Pass: true, Detail: hex.EncodeToString(sum[:])})
		}
		r.Results = append(r.Results, proof{ID: row.Kind, Layer: "production Host follow + Go Store projection + read-only UI controls", Messages: len(v.Messages), Pass: true, Detail: "Synthetic persisted state only; no live Agent or mutation callback."})
		fmt.Println("checkpoint", row.Kind)
	}
	s.Detach()
	mux.Close()
	stop()
	if !r.OwnedListenerClosed {
		return errors.New("owned Host listener remained open")
	}
	if native != "" {
		for _, row := range manifest.Rows {
			cmd := exec.Command(native, "-test.run=TestNativeRichHistory", "-test.timeout=130s")
			cmd.Env = append(os.Environ(), "WHALEISLE_ISOLATED=1", "WHALEISLE_VENDOR="+filepath.Join(stage, "vendor"), "WHALEISLE_NODE="+node, "WHALEISLE_RICH_QA_ID="+row.ID, "WHALEISLE_RICH_QA_WORKSPACE="+cwd, "WHALEISLE_RICH_QA_OUT="+filepath.Join(root, "native-"+row.Kind))
			output, e := cmd.CombinedOutput()
			if writeErr := os.WriteFile(filepath.Join(root, "native-"+row.Kind+".log"), output, 0600); writeErr != nil {
				return writeErr
			}
			if e != nil {
				return fmt.Errorf("native %s: %w", row.Kind, e)
			}
			r.Results = append(r.Results, proof{ID: row.Kind, Layer: "production hidden Windows main + real Host", Pass: true, Detail: "Fresh independent process; native PNG and safe JSON beside owned seed manifest."})
			fmt.Println("native", row.Kind)
		}
	}
	return nil
}
func checkState(kind string, v session.ViewState) error {
	if v.PromptBusy {
		return fmt.Errorf("%s synthetic saved state unexpectedly running", kind)
	}
	has := func(k session.MessageKind, text string) bool {
		for _, m := range v.Messages {
			if m.Kind == k && strings.Contains(m.Body, text) {
				return true
			}
		}
		return false
	}
	switch kind {
	case "text":
		if !has(session.MessageKindThinking, "Owned saved reasoning") || !has(session.MessageKindText, "fmt.Println") {
			return errors.New("rich text/reasoning missing")
		}
	case "media":
		if len(session.TurnDeliveries(v.Messages)) != 1 {
			return errors.New("saved deliverable missing")
		}
		imageFound := false
		for _, m := range v.Messages {
			imageFound = imageFound || len(session.ImageRefs(m.Body)) > 0
		}
		if !imageFound {
			return errors.New("standard nested image reference missing from actual Host history projection")
		}
	case "workflow":
		if len(v.WorkflowRuns) != 1 || v.WorkflowRuns[0].Status != "failed" || len(v.WorkflowRuns[0].Phases) != 2 {
			return errors.New("dedicated saved workflow fold missing")
		}
		run := v.WorkflowRuns[0]
		if run.Phases[0].Name == nil || *run.Phases[0].Name != "Review" || len(run.Phases[0].Members) != 1 || run.Phases[0].Members[0].Status != "completed" || run.Phases[1].Name == nil || *run.Phases[1].Name != "" || len(run.Phases[1].Members) != 1 || run.Phases[1].Members[0].Status != "failed" {
			return errors.New("saved workflow phases/member outcomes changed")
		}
		for _, phase := range run.Phases {
			for _, member := range phase.Members {
				if session.WorkflowMemberAvailable(v.Items, v.SelectedID, member) {
					return errors.New("deleted saved member gained navigation authority")
				}
			}
		}
	case "question-continued", "question-settled":
		var q struct {
			Active  []struct{ State string }
			Settled []struct{ CallID string }
		}
		if err := json.Unmarshal(v.QuestionProjection.Questions, &q); err != nil {
			return err
		}
		if kind == "question-continued" && (len(q.Active) != 1 || q.Active[0].State != "continued") {
			return errors.New("continued question projection missing")
		}
		if kind == "question-settled" && (len(q.Active) != 0 || len(q.Settled) != 1 || !has(session.MessageKindQuestionReply, "owned answer")) {
			return errors.New("settled question/reply projection missing")
		}
	}
	return nil
}
func readOnlyControls(kind string, v session.ViewState) error {
	for _, m := range v.Messages {
		if kind == "text" && m.Kind == session.MessageKindText && m.Role == "assistant" {
			tt := ui.NewTester(func(c *ui.Context) { markdown.View(c, m.Body, markdown.Style{FontSize: 13}) }, 900, 1000)
			if err := tt.Click("Copy code"); err != nil {
				return err
			}
			if tt.Clipboard() != "fmt.Println(\"owned code\")\n" {
				return errors.New("saved code copy mismatch")
			}
		}
		if m.Kind == session.MessageKindThinking {
			open := map[string]bool{}
			tt := ui.NewTester(func(c *ui.Context) { chat.ThinkingBlockView(c, m.ID, m.Body, &open, chat.ThinkingStyle{}) }, 900, 500)
			if err := tt.Click("\u25b8 " + i18n.T(i18n.KeyChatThinking)); err != nil {
				return err
			}
			if !open[m.ID] {
				return errors.New("saved reasoning disclosure did not open")
			}
			if err := tt.Click("Copy reasoning " + m.ID); err != nil {
				return err
			}
			if tt.Clipboard() != m.Body {
				return errors.New("saved reasoning copy mismatch")
			}
		}
		if m.Kind == session.MessageKindQuestionReply {
			tt := ui.NewTester(func(c *ui.Context) {
				chat.QuestionReplyView(c, m, v.SelectedID, theme.Resolve(theme.IDDeepseekLight).Active(false), ui.RGB(0, 0, 0), ui.RGB(80, 80, 80))
			}, 900, 700)
			if err := tt.Click("Question reply details " + m.ID); err != nil {
				return err
			}
			if !tt.HasText("Synthetic question") {
				return errors.New("saved reply disclosure absent")
			}
			if err := tt.Click("Copy question reply " + m.ID); err != nil {
				return err
			}
			if !strings.Contains(tt.Clipboard(), "owned answer") {
				return errors.New("saved reply copy mismatch")
			}
		}
	}
	return nil
}
