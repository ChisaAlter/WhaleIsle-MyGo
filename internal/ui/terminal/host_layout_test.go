package terminal

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/hostterminal"
	"whaleisle/internal/ui/theme"
)

func hostLayoutView() hostterminal.View {
	info := hostterminal.Info{ID: "recorded-terminal", Title: "WhaleIsle-MyGo / " + strings.Repeat("长标题终端 ", 12), Cwd: `C:\work\WhaleIsle-MyGo\a-long-workspace-directory\子目录`, Cols: 96, Rows: 24, State: "running"}
	return hostterminal.View{Owner: 1, ClientReady: true, Ready: true, Writable: true, ActiveID: info.ID, Phase: "connected", Info: info, Items: []hostterminal.Info{info}}
}

func TestHostSurfaceKeepsOutputSpaceAtNormalAndNarrowSizes(t *testing.T) {
	for _, size := range [][2]int{{1000, 540}, {800, 420}, {360, 420}, {320, 300}} {
		t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
			h := &HostState{}
			store := &hostterminal.Store{}
			v := hostLayoutView()
			tt := ui.NewTester(func(c *ui.Context) {
				hostSurface(c, h, store, "layout-session", theme.Resolve(theme.IDDeepseekDark).Active(true), v, HostActions{})
			}, size[0], size[1])
			body, ok := tt.Find("Terminal")
			if !ok {
				t.Fatal("terminal body missing")
			}
			if body.H < float32(size[1]-80) || body.W < float32(size[0]-20) {
				t.Fatalf("management displaced terminal: viewport=%v body=%+v", size, body)
			}
			for _, label := range []string{terminalLabel("选择 Host 终端", "Choose Host terminal"), terminalLabel("新建 Host 终端", "New Host terminal"), terminalLabel("终端管理", "Terminal management")} {
				if _, ok := tt.Find(label); ok {
					t.Fatalf("connected terminal retained permanent toolbar: %s", label)
				}
			}
			if err := tt.RightClick("Terminal"); err != nil {
				t.Fatal(err)
			}
			if err := tt.ChooseMenuItem(terminalLabel("终端管理…", "Terminal management…")); err != nil {
				t.Fatal(err)
			}
			tt.Frame()
			after, ok := tt.Find("Terminal")
			if !ok || after != body {
				t.Fatalf("opening management resized terminal: before=%+v after=%+v", body, after)
			}
			for _, label := range []string{terminalLabel("刷新列表", "Refresh list"), terminalLabel("选择或刷新 shell", "Choose or refresh the shell"), terminalLabel("重命名终端", "Rename terminal"), terminalLabel("重连并接管输入", "Reconnect and take control"), terminalLabel("关闭终端进程", "Close terminal process")} {
				if !tt.HasText(label) {
					t.Fatal("management entry missing: " + label)
				}
			}
		})
	}
}
