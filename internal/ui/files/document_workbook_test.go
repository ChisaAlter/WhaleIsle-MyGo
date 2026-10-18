package files

import (
	"github.com/egoist/mygo/ui"
	"testing"
)

type workbookSurfaceFixture struct {
	fixtureDocumentSurface
	workbook bool
}

func (s *workbookSurfaceFixture) WorkbookViewerAvailable() bool { return s.workbook }
func TestWorkbookOwnerCapabilityAndNativeOverride(t *testing.T) {
	surface := &workbookSurfaceFixture{}
	state := documentSurfaceState{surface: surface, choice: 2, workbook: true}
	if documentSurfaceEnabled(state) {
		t.Fatal("enabled absent owner capability")
	}
	surface.workbook = true
	if !documentSurfaceEnabled(state) {
		t.Fatal("available workbook owner ignored")
	}
	state.choice = 1
	if documentSurfaceEnabled(state) {
		t.Fatal("native override ignored")
	}
	state.choice = 2
	surface.workbook = false
	if documentSurfaceEnabled(state) {
		t.Fatal("revoked workbook capability retained")
	}
}

func TestWorkbookOwnerViewerButton(t *testing.T) {
	m, tab := tableGridFixture()
	surface := &workbookSurfaceFixture{workbook: true}
	m.OpenDocumentViewer = func(string) (DocumentSurface, error) { return surface, nil }
	tab.document.choice = 1
	tt := ui.NewTester(func(c *ui.Context) { ui.Row(c).Children(func() { paintDocumentViewerButton(c, m) }) }, 700, 100)
	if err := tt.Click(label("完整表格预览", "Full workbook viewer")); err != nil {
		t.Fatal(err)
	}
	if !tab.document.enabled || !tab.document.workbook {
		t.Fatal("full viewer button did not route to live owner")
	}
	if err := tt.Click(label("返回原生预览", "Native preview")); err != nil {
		t.Fatal(err)
	}
	if tab.document.enabled || tab.document.choice != 1 {
		t.Fatal("native return lost")
	}
}
