package browserpreview

import (
	"strings"
	"testing"
)

func TestBrowserDevToolsPlatformCopy(t *testing.T) {
	label, hint, result := browserDevToolsCopy("darwin")
	if label != "允许 Safari 检查" || !strings.Contains(hint, "不会自动打开") || !strings.Contains(result, "开发菜单") || !strings.Contains(result, "关闭 Browser 页签即可撤销") {
		t.Fatal("macOS must describe opt-in and manual Safari steps")
	}
	label, _, result = browserDevToolsCopy("windows")
	if label != "开发者工具" || strings.Contains(result, "Safari") {
		t.Fatal("Windows native inspector description changed")
	}
}
