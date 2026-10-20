package interaction

import (
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/dsh"
	"whaleisle/internal/store/interaction"
)

// The interaction gallery renders the pending approval and question cards that
// until now only had behavioural tests. It is software-rendered with
// ui.NewTester (theme tokens, no native text shaping or compositor).
//
//	CHAT_GALLERY_DIR=docs/review-20261020/evidence/after go test ./internal/ui/interaction -run TestTranscriptGallery
func interactionGalleryDir(t *testing.T) string {
	t.Helper()
	dir := strings.TrimSpace(os.Getenv("CHAT_GALLERY_DIR"))
	if dir == "" {
		t.Skip("set CHAT_GALLERY_DIR to render the interaction gallery")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("gallery dir: %v", err)
	}
	return dir
}

func writeInteractionGallery(t *testing.T, dir, name string, tt *ui.Tester) {
	t.Helper()
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	if err := png.Encode(f, tt.Image()); err != nil {
		_ = f.Close()
		t.Fatalf("encode %s: %v", name, err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close %s: %v", name, err)
	}
}

// galleryStore mounts one waterfall request and waits for its projection.
func galleryStore(t *testing.T, req dsh.EventsWaterfall) *interaction.Store {
	t.Helper()
	store := interaction.New()
	t.Cleanup(store.Detach)
	ready := make(chan struct{}, 1)
	store.Attach(func() {
		select {
		case ready <- struct{}{}:
		default:
		}
	})
	go func() { _, _ = store.HandleWaterfall(req) }()
	select {
	case <-ready:
	case <-time.After(time.Second):
		t.Fatal("request did not open")
	}
	return store
}

// transcript-approval-card.png — the warn approval card: state bar, "waiting for
// your approval" header with the call id, the monospace command and the
// deny/allow pair.
func TestTranscriptGalleryApprovalCard(t *testing.T) {
	dir := interactionGalleryDir(t)
	store := galleryStore(t, dsh.EventsWaterfall{
		AgentID: "conv-1", EventID: "approval-1", Event: interaction.EventApproval,
		Request: json.RawMessage(`{
			"toolName":"bash",
			"reason":"这个命令会修改工作区文件，需要你确认。",
			"displayReason":{"zh":"这个命令会修改工作区文件，需要你确认。","en":"This command writes inside the workspace."},
			"args":{"command":"go test ./internal/ui/... -count=1 && gofmt -w internal"}
		}`),
	})
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(24, 32, 24, 32).Children(func() {
			ui.Column(c).Width(720).Children(func() {
				View(c, Model{Store: store, Snapshot: store.SnapshotForAgent("conv-1"), ScopeAgentID: "conv-1", ThemeID: "deepseek.light"})
			})
		})
	}, 1200, 360)
	writeInteractionGallery(t, dir, "transcript-approval-card.png", tt)
}

// transcript-question-card.png — the question card: state dot header, the 15/24
// question, three option cards with index badges and descriptions, the custom
// answer field, the 1/3 pager and the submit/hide/deny pair.
func TestTranscriptGalleryQuestionCard(t *testing.T) {
	dir := interactionGalleryDir(t)
	questions := `[
		{"id":"q1","header":"渲染范围","question":"标题层级要落到哪一级？","options":[
			{"label":"只做 h1–h3","description":"覆盖正文里最常见的层级"},
			{"label":"h1–h6 全覆盖","description":"和原版 MarkdownText 完全一致"},
			{"label":"跟随主题设置"}
		]},
		{"id":"q2","question":"要不要同时改只读历史？"},
		{"id":"q3","question":"验收用哪张截图？","options":[{"label":"1200 宽"},{"label":"窄屏 400"},{"label":"两者都要"}]}
	]`
	store := galleryStore(t, dsh.EventsWaterfall{
		AgentID: "conv-1", EventID: "questions-1", Event: interaction.EventUserQuestions,
		Request: json.RawMessage(`{"questions":` + questions + `}`),
	})
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(24, 32, 24, 32).Children(func() {
			ui.Column(c).Width(720).Children(func() {
				View(c, Model{Store: store, Snapshot: store.SnapshotForAgent("conv-1"), ScopeAgentID: "conv-1", ThemeID: "deepseek.light"})
			})
		})
	}, 1200, 700)
	writeInteractionGallery(t, dir, "transcript-question-card.png", tt)
}
