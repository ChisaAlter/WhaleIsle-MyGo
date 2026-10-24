package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestSettingsRowMatchesNativeGeometry pins the numbers the installed settings
// dialog paints (1440×920, DPR 1.5): a row keeps 16px above and below its copy,
// the label is 14px/22 with its 12px/18 description 4px below, a 36px control
// gives the row its 68px floor, the value pill sits flush with the row's right
// edge, and the boolean switch is 36×20.
func TestSettingsRowMatchesNativeGeometry(t *testing.T) {
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
	if !ok || lang.Y != 23 || lang.H != 22 {
		t.Fatalf("label row = %+v", lang)
	}
	perm, _ := tt.Find("权限")
	if perm.Y != 84 {
		t.Fatalf("68px row floor: next row starts at %v, want 84", perm.Y)
	}
	desc, _ := tt.Find("选择新会话的默认权限模式")
	if desc.Y != 110 || desc.H != 18 {
		t.Fatalf("description = %+v, want y=110 h=18 (4px under the label)", desc)
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
