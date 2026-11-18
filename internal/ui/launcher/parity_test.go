package launcher

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/desktop/components"
	"whaleisle/internal/ui/i18n"
)

func TestComponentListScrollKeepsActionsVisible(t *testing.T) {
	rows := make([]components.Row, 20)
	for i := range rows {
		rows[i] = components.Row{ID: fmt.Sprint(i), Name: fmt.Sprintf("Component-%02d", i), State: "available"}
	}
	tt := ui.NewTester(func(c *ui.Context) { Maintenance(c, MaintenanceComponents, Model{Components: rows}) }, 1060, 660)
	first, _ := tt.Find("Component-00")
	header, headerFound := tt.Find(i18n.T(i18n.KeyLauncherTabComponents))
	tt.Scroll(800, 450, 0, 600)
	after, _ := tt.Find("Component-00")
	headerAfter, headerAfterFound := tt.Find(i18n.T(i18n.KeyLauncherTabComponents))
	if !headerFound || !headerAfterFound {
		t.Fatalf("header brand %q not found: before=%v after=%v", i18n.T(i18n.KeyLauncherTabComponents), headerFound, headerAfterFound)
	}
	if after.Y >= first.Y {
		t.Fatalf("stage did not scroll: first=%+v after=%+v", first, after)
	}
	if headerAfter != header {
		t.Fatalf("stage scrolling moved header: before=%+v after=%+v", header, headerAfter)
	}
	tt.Scroll(800, 450, 0, 100000)
	last, ok := tt.Find("Component-19")
	if !ok || last.H <= 0 || last.Y < 56 || last.Y+last.H > 660 {
		t.Fatalf("last component cannot be reached inside stage viewport: %+v", last)
	}
}
