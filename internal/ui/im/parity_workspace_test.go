package im

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	imstore "whaleisle/internal/store/im"
	"whaleisle/internal/store/imonboard"
)

func TestParityWorkspaceChangeHasCancellableMappingReview(t *testing.T) {
	m := Model{Store: imstore.New()}
	v := imonboard.View{WorkspaceDrafts: map[string]string{"fixture-bot": "C:/fixture/new"}}
	bot := imonboard.Bot{BotID: "fixture-bot", Workspace: "C:/fixture/original"}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			botWorkspacePicker(c, m, v, bot, false, ui.RGB(30, 30, 30), ui.RGB(100, 100, 100))
		})
	}, 640, 500)
	if err := tt.Click("保存工作区…"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("此机器人的旧聊天绑定会清除；原会话内容保留。下一条消息将使用新工作目录。") || !tt.HasText("C:/fixture/new") {
		t.Fatal("workspace change omitted target or mapping consequence", tt.Texts())
	}
	if dir := os.Getenv("WHALE_PARITY_CAPTURE_DIR"); dir != "" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(dir, "im-workspace-mapping-confirm.png"))
		if err != nil {
			t.Fatal(err)
		}
		if err = png.Encode(f, tt.Image()); err != nil {
			f.Close()
			t.Fatal(err)
		}
		f.Close()
	}
	if err := tt.Click("取消"); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("切换机器人工作目录？") {
		t.Fatal("cancel retained the mapping confirmation")
	}
}
