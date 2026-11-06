package launcher

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestUpdateCenterPauseResumePreferenceAndUnavailableRoute(t *testing.T) {
	busy, paused, automatic := true, false, false
	var pauses, resumes int
	model := func() Model {
		status := releaseText("正在下载并校验安装包", "Downloading and verifying installer")
		if paused {
			status = releaseText("下载已暂停，可继续。", "Download paused. You can resume it.")
		}
		return Model{ActiveTab: TabVersions, VendorReady: true, Version: "0.3.13", ReleaseBusy: busy, ReleasePaused: paused, ReleaseAutoDownload: automatic, ReleasePhase: "downloading", ReleaseStatus: status, ReleaseReceived: 8 << 20, ReleaseTotal: 32 << 20,
			OnCancelRelease: func() {}, OnPauseRelease: func() { pauses++; busy = false; paused = true }, OnResumeRelease: func() { resumes++; busy = true; paused = false }, OnSetAutoDownload: func(v bool) { automatic = v }}
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, model()) }, 1060, 660)
	save := func(name string) {
		if dir := os.Getenv("WHALEISLE_LIFECYCLE_SHOTS"); dir != "" {
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			f, err := os.Create(filepath.Join(dir, name))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(f, tt.Image()); err != nil {
				f.Close()
				t.Fatal(err)
			}
			f.Close()
		}
	}
	save("update-center-downloading.png")
	if err := tt.Click(releaseText("有新版本时在后台自动下载", "Download new versions automatically in the background")); err != nil {
		t.Fatal(err)
	}
	if !automatic {
		t.Fatal("auto-download preference callback not consumed")
	}
	if err := tt.Click(releaseText("暂停下载", "Pause download")); err != nil {
		t.Fatal(err)
	}
	if pauses != 1 || !paused {
		t.Fatal("pause callback not consumed")
	}
	save("update-center-paused.png")
	if err := tt.Click(releaseText("继续下载", "Resume download")); err != nil {
		t.Fatal(err)
	}
	if resumes != 1 || paused {
		t.Fatal("resume callback not consumed")
	}

	route := "github"
	open := true
	reason := releaseText("CNB 尚未配置鲸屿go原生发布源，请使用 GitHub。", "CNB has no configured MyGo native release source. Use GitHub.")
	blocked := ui.NewTester(func(c *ui.Context) {
		trigger := ui.Button(c, "route")
		routePopup(c, Model{DownloadRoute: &route, RouteUnavailable: map[string]string{"cnb": reason}}, trigger, &open, false)
	}, 600, 320)
	if !blocked.HasText(reason) {
		t.Fatal("unavailable channel reason missing")
	}
	_ = blocked.Click("CNB")
	if route != "github" {
		t.Fatal("unconfigured native channel was selectable")
	}
}

func TestUpdateCenterInstallingDoesNotOfferDownloadCancellation(t *testing.T) {
	m := Model{ReleaseBusy: true, ReleasePhase: "installing", OnCancelRelease: func() { t.Fatal("install exposed download cancellation") }}
	tt := ui.NewTester(func(c *ui.Context) { UpdateCenterFooter(c, m) }, 600, 240)
	if _, ok := tt.Find(releaseText("取消下载 / 读取", "Cancel download / lookup")); ok {
		t.Fatal("the download cancel action cannot cancel installation admission")
	}
}
