package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/pet"
	"whaleisle/internal/ui/theme"
)

func TestPetSettingsRenderAssetsDownloadStatesAndActions(t *testing.T) {
	t.Setenv("WHALEISLE_USER_DATA", t.TempDir())
	colors := theme.Resolve("").Active(false)
	companion := &pet.Companion{Enabled: true, Behavior: config.DefaultPetBehavior()}

	var starts, cancels int
	snap := pet.RenderAssetsSnapshot{
		State:      pet.RenderAssetsMissing,
		TotalBytes: 48063128,
	}
	access := &PetSettingsAccess{
		RenderAssetsOverride:         &snap,
		OnDownloadRenderAssets:       func() { starts++ },
		OnCancelDownloadRenderAssets: func() { cancels++ },
	}
	tt := ui.NewTester(func(c *ui.Context) {
		petPane(c, colors, companion, ui.RGB(20, 20, 20), ui.RGB(120, 120, 120), access)
	}, 900, 2400)

	btnLabel := providerCopy("下载高阶渲染包", "Download render pack")
	if _, ok := tt.Find(btnLabel); !ok {
		t.Fatalf("missing download button %q in %v", btnLabel, tt.Texts())
	}
	if err := tt.Click(btnLabel); err != nil {
		t.Fatalf("click download: %v", err)
	}
	if starts != 1 {
		t.Fatalf("expected 1 start callback, got %d", starts)
	}

	snap = pet.RenderAssetsSnapshot{
		State:           pet.RenderAssetsDownloading,
		DownloadedBytes: 24000000,
		TotalBytes:      48063128,
	}
	tt.Frame()
	cancelLabel := providerCopy("取消下载", "Cancel download")
	if _, ok := tt.Find(cancelLabel); !ok {
		t.Fatalf("missing cancel button %q in %v", cancelLabel, tt.Texts())
	}
	if err := tt.Click(cancelLabel); err != nil {
		t.Fatalf("click cancel: %v", err)
	}
	if cancels != 1 {
		t.Fatalf("expected 1 cancel callback, got %d", cancels)
	}

	snap = pet.RenderAssetsSnapshot{
		State:      pet.RenderAssetsError,
		TotalBytes: 48063128,
		Error:      "network timeout",
	}
	tt.Frame()
	retryLabel := providerCopy("重试下载", "Retry download")
	if _, ok := tt.Find(retryLabel); !ok {
		t.Fatalf("missing retry button %q in %v", retryLabel, tt.Texts())
	}
	if err := tt.Click(retryLabel); err != nil {
		t.Fatalf("click retry: %v", err)
	}
	if starts != 2 {
		t.Fatalf("expected 2 start callbacks after retry, got %d", starts)
	}

	snap = pet.RenderAssetsSnapshot{
		State:           pet.RenderAssetsReady,
		DownloadedBytes: 48063128,
		TotalBytes:      48063128,
	}
	tt.Frame()
	if _, ok := tt.Find(btnLabel); ok {
		t.Fatalf("download button should hide from top pane once Ready")
	}
	discLabel := providerCopy("本地皮肤与动画", "Local skins and animation")
	if err := tt.Click(discLabel); err != nil {
		t.Fatalf("open local skins disclosure: %v", err)
	}
	readyText := providerCopy("已就绪", "Ready")
	if !tt.HasText(readyText) {
		t.Fatalf("expected %q status inside local skins disclosure; texts=%v", readyText, tt.Texts())
	}
}
