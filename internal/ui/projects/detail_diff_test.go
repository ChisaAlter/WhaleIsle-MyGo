package projects

import (
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/projectflow"
)

func TestWorkDetailDiffEntryAndReturn(t *testing.T) {
	for _, role := range []string{"worker", "docs", "readonly"} {
		t.Run(role, func(t *testing.T) {
			var d projectflow.WorkDetail
			if err := json.Unmarshal([]byte(`{"workstream":{"id":"work-1","projectId":"project-1"},"worker":{"role":"`+role+`","sessionId":"worker-1","materialized":true}}`), &d); err != nil {
				t.Fatal(err)
			}
			opened := ""
			m := Model{TeamID: "project-1", Detail: &d, Listing: projectflow.Listing{Available: true}, OnWorkDiff: func(id string) { opened = id }}
			tt := ui.NewTester(func(c *ui.Context) { detailView(c, m) }, 1000, 900)
			label := tr("查看工作区差异", "View workspace diff")
			if tt.HasText(label) != (role != "readonly") {
				t.Fatalf("role %s diff availability: %q", role, tt.Texts())
			}
			if role != "readonly" {
				if err := tt.Click(label); err != nil {
					t.Fatal(err)
				}
				if opened != "work-1" {
					t.Fatalf("opened %q", opened)
				}
			}
		})
	}
	m := Model{TeamID: "project-1", Detail: &projectflow.WorkDetail{}, AssetDiff: &projectflow.WorkDiff{}}
	m.AssetDiff.Snapshot.Diff = "fixture-diff"
	m.OnCloseDiff = func() { m.AssetDiff = nil }
	tt := ui.NewTester(func(c *ui.Context) { detailView(c, m) }, 1000, 900)
	if !tt.HasText("fixture-diff") {
		t.Fatalf("no diff rendered: %q", tt.Texts())
	}
	if err := tt.Click(tr("返回工作详情", "Back to work details")); err != nil {
		t.Fatal(err)
	}
	if m.AssetDiff != nil {
		t.Fatal("back did not clear diff")
	}
}
