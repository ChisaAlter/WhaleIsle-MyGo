package interaction

import (
	"context"
	"github.com/egoist/mygo/ui"
	"testing"
	"time"
	"whaleisle/internal/dsh"
	store "whaleisle/internal/store/interaction"
)

type uiSubmissionScope struct {
	client *dsh.Client
	valid  bool
}

func (s *uiSubmissionScope) Client() *dsh.Client      { return s.client }
func (s *uiSubmissionScope) Context() context.Context { return context.Background() }
func (s *uiSubmissionScope) AgentID() string          { return "A" }
func (s *uiSubmissionScope) Valid() bool              { return s.valid }
func TestScopedInteractionUIEnterRevalidatesAfterPaint(t *testing.T) {
	s, _, outcome := questionEditorFixture(t, `[{"id":"q","question":"Question"}]`)
	scope := &uiSubmissionScope{client: &dsh.Client{}, valid: true}
	s.AttachClient(scope.client, nil)
	facade := store.NewScopedStore(s, scope)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Children(func() {
			View(c, Model{Store: facade, Snapshot: s.SnapshotForAgent("A"), ScopeAgentID: "A", DisablePageShortcuts: true, ThemeID: "deepseek.light"})
		})
	}, 600, 900)
	if err := tt.Click("Question answer: Question"); err != nil {
		t.Fatal(err)
	}
	tt.Type("kept\nanswer")
	scope.valid = false
	tt.Key(0, ui.KeyEnter)
	select {
	case <-outcome:
		t.Fatal("stale painted editor sent")
	default:
	}
	if s.SnapshotForAgent("A").Pending[0].Sending {
		t.Fatal("revoked answer froze as Sending")
	}
	scope.valid = true
	tt.Key(0, ui.KeyEnter)
	select {
	case out := <-outcome:
		if out.Delivery == nil || !out.Delivery.Valid() {
			t.Fatal("UI bypassed scoped sender")
		}
	case <-time.After(time.Second):
		t.Fatal("authorized Enter failed")
	}
	if !s.SnapshotForAgent("A").Pending[0].Sending {
		t.Fatal("answer controls did not freeze")
	}
}
