package screenglance

import "runtime"

// ApprovalMessage describes the actual platform capture scope before any pixels
// are requested. The Screenshot portal, unlike Windows/macOS, owns selection.
func ApprovalMessage() string {
	if runtime.GOOS == "linux" {
		return "通过系统截图对话框截取一张图片并发送给视觉模型？请在系统界面选择要分享的显示器或区域，桌面可能截取整个屏幕。 / Capture through the system screenshot dialog and send one image to the vision model? Select the display or area to share; the desktop may capture the entire screen."
	}
	return "截取桌宠所在显示器的一张截图并发送给视觉模型？ / Capture and send one screenshot of the pet's display?"
}

func StorageDetail() string {
	switch runtime.GOOS {
	case "linux":
		return "截图可能包含私人信息，并可能保存在鲸鱼娘会话中。系统会创建临时截图；读取后尝试删除，但只读 portal 文件可能由系统保留。 / The image may contain private information and be retained in Whale chat. The system creates a temporary screenshot; removal is attempted after reading, but a read-only portal image may remain under desktop management."
	case "darwin":
		return "截图可能包含私人信息，并可能保存在鲸鱼娘会话中。macOS 截图会暂存于本机私有临时目录，读取后删除。需要你在系统设置中授予屏幕录制权限。 / The image may contain private information and be retained in Whale chat. macOS capture uses a private temporary directory, removed after reading. Screen Recording permission must be granted by you in System Settings."
	default:
		return "截图可能包含私人信息，并可能保存在鲸鱼娘会话中。本机不另存截图文件。 / The image may contain private information and may be retained in Whale chat. No separate local screenshot file is saved."
	}
}

// FailureMessage avoids showing backend errors that could contain private paths.
func FailureMessage() string {
	switch runtime.GOOS {
	case "darwin":
		return "截图不可用：请检查 macOS 系统设置 → 隐私与安全性 → 屏幕录制中的应用权限后重试 / Screenshot unavailable; check this app's Screen Recording permission in macOS System Settings and retry"
	case "linux":
		return "截图已取消、被拒绝或系统截图服务不可用；需要支持交互截图的 XDG Desktop Portal（v2+） / Screenshot cancelled, denied or unavailable; an interactive XDG Screenshot portal (v2+) is required"
	default:
		return "本次截图不可用 / Screenshot unavailable"
	}
}
