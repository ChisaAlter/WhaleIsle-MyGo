package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestSettingsRowKeepsTheCossGrid pins the row's type-and-rhythm contract: a
// 14px label on a 22 DIP line with its 16 DIP description 4 DIP below, rows
// tiled with no gap or overlap, the value control flush with the row's right
// edge, and a plain ui.Switch keeping its 36×20 control size.
func TestSettingsRowKeepsTheCossGrid(t *testing.T) {
	fg, muted := ui.RGB(15, 17, 21), ui.RGB(129, 133, 140)
	selected := "工作区内修改"
	on := true
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Width(564).Children(func() {
			settingsFieldRow(c, "lang", "语言", "", fg, muted, nil, func() {
				ui.Select(c, &selected, []string{"中文"}).Width(82)
			})
			settingsFieldRow(c, "perm", "权限", "选择新会话的默认权限模式", fg, muted, nil, func() {
				ui.Select(c, &selected, []string{"工作区内修改"}).Width(138)
			})
			settingsFieldRow(c, "code", "显示代码工作视图", "开启后显示轨迹", fg, muted, nil, func() {
				ui.Switch(c, &on).Label("code switch")
			})
		})
	}, 720, 600)

	lang, ok := tt.Find("语言")
	if !ok || lang.H != 22 {
		t.Fatalf("label row = %+v", lang)
	}
	perm, _ := tt.Find("权限")
	if perm.Y <= lang.Y {
		t.Fatalf("rows overlap: perm %+v lang %+v", perm, lang)
	}
	desc, _ := tt.Find("选择新会话的默认权限模式")
	if desc.H != 16 || desc.Y-perm.Y-perm.H != 4 {
		t.Fatalf("description = %+v, want 16 DIP line 4px under the label %+v", desc, perm)
	}
	value, _ := tt.Find("工作区内修改")
	if value.X < 426 || value.X+value.W > 564 {
		t.Fatalf("value pill is not flush with the row's right edge: %+v", value)
	}
	sw, ok := tt.Find("code switch")
	if !ok || sw.W != 36 || sw.H != 20 {
		t.Fatalf("switch = %+v, want 36×20", sw)
	}
}
