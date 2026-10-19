package projects

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/diffsurface"
	"whaleisle/internal/ui/i18n"
)

type projectPageHeightSurface struct{ heights []float32 }

func (*projectPageHeightSurface) Present(int, int, int, int, int, int, int, int, string, map[string]string) error {
	return nil
}
func (s *projectPageHeightSurface) RenderPixels(c *ui.Context, height float32, _ string, _ map[string]string) bool {
	s.heights = append(s.heights, height)
	ui.Box(c).FillWidth().Height(height)
	return true
}

func TestProjectPublishedPageUsesMeasuredSlotWidthAndHeightLimits(t *testing.T) {
	for _, tc := range []struct {
		height int
		want   float32
	}{{79, 80}, {80, 80}, {2000, 2000}, {2001, 2000}} {
		t.Run(fmt.Sprint(tc.height), func(t *testing.T) {
			surface := &projectPageHeightSurface{}
			var widths []int
			page := projectflow.DeliveryPageReference{CallID: "owned", Height: tc.height}
			m := Model{VisualPage: func(source string, got projectflow.DeliveryPageReference, width int) VisualPagePreview {
				if source != "request:A" || got != page {
					t.Fatal("layout changed the captured publication identity")
				}
				widths = append(widths, width)
				return VisualPagePreview{Available: true, Surface: surface}
			}}
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Width(512).Children(func() { renderVisualPages(c, m, "request:A", []projectflow.DeliveryPageReference{page}) })
			}, 800, 2400)
			tt.Frame()
			if len(widths) == 0 || widths[len(widths)-1] != 512 {
				t.Fatalf("page did not receive its actual DIP slot width: %v", widths)
			}
			if len(surface.heights) == 0 || surface.heights[len(surface.heights)-1] != tc.want {
				t.Fatalf("page height clamp=%v want=%v", surface.heights, tc.want)
			}
		})
	}
}

func TestRequirementChangesKeepUnifiedLinesAndMetadataOnlyFiles(t *testing.T) {
	patch := "diff --git a/old.txt b/new.txt\nsimilarity index 50%\nrename from old.txt\nrename to new.txt\n--- a/old.txt\n+++ b/new.txt\n@@ -8,2 +9,2 @@ heading\n context\n-old\n+new\n\\ No newline at end of file\n"
	var scope projectflow.RequestChangeScope
	scope.Diff = patch
	if err := json.Unmarshal([]byte(`{"files":[{"path":"new.txt","added":99,"removed":99},{"path":"binary.dat","binary":true,"status":"A"},{"path":"scratch.txt","status":"??"}]}`), &scope); err != nil {
		t.Fatal(err)
	}
	files := requestChangeFiles(scope, diffsurface.ParseRecordedPatch(patch))
	if len(files) != 3 || files[0].Path != "new.txt" || files[0].kind != "renamed" || files[0].added != 1 || files[0].removed != 1 || !files[1].binary || files[2].kind != "untracked" {
		t.Fatalf("parsed/metadata union lost original file semantics: %+v", files)
	}
	want := []string{"@@ -8,2 +9,2 @@ heading", " context", "-old", "+new", "\\ No newline at end of file"}
	if got := requestDiffLines(files[0].Source); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("unified diff order, signs or original hunk ranges changed: %q", got)
	}
	if requestDiffLines("Binary files a/x and b/x differ\n") != nil {
		t.Fatal("binary metadata was presented as text hunks")
	}
}

func TestProjectUndoDialogKeepsItsOwnCardAndDisplayedRevision(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	var card projectflow.RequestCard
	projectJSON(t, `{"id":"A","projectId":"owned","revision":3,"status":"completed","changes":{"isGit":true,"revertible":true},"delivery":{"outcome":"completed"}}`, &card)
	currentRevision, conflict, asked, confirmed := 3, true, "", 0
	m := Model{RequestMutable: true, Listing: projectflow.Listing{Available: true}, RequestCard: &projectflow.RequestCard{ID: "B", Revision: 9}}
	m.RequestRevert = func(id string, revision int) RequestRevertState {
		state := RequestRevertState{Known: id == "A" && revision == currentRevision, Allowed: id == "A" && revision == currentRevision}
		if id == "A" && revision == 3 && conflict {
			state.Error = "pluginrpc: revert_conflict: owned A conflict"
		}
		state.Confirm = func() { confirmed++ }
		state.Ask = func() { asked = fmt.Sprintf("%s/%d", id, revision) }
		return state
	}
	tt := ui.NewTester(func(c *ui.Context) { taskDeliveryChangeActions(c, m, card) }, 900, 900)
	if err := tt.Click("Revert changes"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Could not revert automatically") || !tt.HasText("owned A conflict") {
		t.Fatalf("A's conflict was hidden by current B: %q", tt.Texts())
	}
	if err := tt.Click("Ask the coordinator"); err != nil {
		t.Fatal(err)
	}
	if asked != "A/3" || confirmed != 0 {
		t.Fatalf("conflict action changed its captured card/revision: asked=%q confirmed=%d", asked, confirmed)
	}
	conflict = false
	tt.Frame()
	if err := tt.Click("Revert changes"); err != nil {
		t.Fatal(err)
	}
	currentRevision, card.Revision = 4, 4
	tt.Frame()
	if !tt.HasText("The requirement changed; refresh and confirm again.") || confirmed != 0 {
		t.Fatal("open confirmation silently adopted the newer displayed revision")
	}
}

func TestProjectHandoverActionKeepsFailureCardWhenAnotherIsCurrent(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	card := projectflow.RequestCard{ID: "A", ProjectID: "owned", Revision: 3, HandoverError: "owned failure", Status: "summaryFailed"}
	known, recovered := true, ""
	m := Model{RequestMutable: true, Listing: projectflow.Listing{Available: true}, RequestCard: &projectflow.RequestCard{ID: "B", Revision: 9}, OnHandoverRetryDraft: func() { t.Fatal("handover fell through to B's callback") }}
	m.RequestCardActions = func(id string, revision int) RequestCardActions {
		return RequestCardActions{Known: known && id == card.ID && revision == card.Revision, HandoverRetry: func() { recovered = fmt.Sprintf("%s/%d", id, revision) }}
	}
	tt := ui.NewTester(func(c *ui.Context) { taskCardDetail(c, m, card, "overview") }, 900, 900)
	if err := tt.Click("Send handover recovery request"); err != nil {
		t.Fatal(err)
	}
	if recovered != "A/3" {
		t.Fatalf("recovery lost its failure card: %q", recovered)
	}
	known, recovered = false, ""
	tt.Frame()
	if err := tt.Click("Send handover recovery request"); err != nil {
		t.Fatal(err)
	}
	if recovered != "" {
		t.Fatal("retired failure card still invoked recovery")
	}
}

func TestRequirementChangesLongFileShowsBoundedUnifiedPreview(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	var patch strings.Builder
	patch.WriteString("diff --git a/owned.txt b/owned.txt\n--- a/owned.txt\n+++ b/owned.txt\n@@ -0,0 +1,450 @@\n")
	for i := 0; i < 450; i++ {
		fmt.Fprintf(&patch, "+owned-line-%03d\n", i)
	}
	tt := ui.NewTester(func(c *ui.Context) {
		renderRequestChangeScope(c, Model{}, projectflow.RequestChangeScope{Diff: patch.String(), Truncated: true})
	}, 900, 900)
	if !tt.HasText("@@ -0,0 +1,450 @@") || !tt.HasText("+owned-line-000") || tt.HasText("+owned-line-449") || !tt.HasText("Only the first 400 lines of each file are shown.") || !tt.HasText("Diff is truncated; inspect the working directory for complete contents.") {
		t.Fatalf("long unified preview lost hunk/cap/truncation behavior: %q", tt.Texts())
	}
}
