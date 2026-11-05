package chat

import (
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

const presentedSuccessHold = 5 * time.Second
const presentedSuccessFade = 200 * time.Millisecond

func presentedMetadata(snapshot session.TurnFilesSnapshot, state session.TurnFileResult, fallback string, now time.Time) (text string, failure bool, opacity float32, repaint time.Duration) {
	opacity = 1
	directory := true
	for _, result := range snapshot.Results {
		if result.Host != nil && result.Host.FileManager != nil {
			directory = *result.Host.FileManager == "directory"
			break
		}
	}
	if state.NativeUnavailable {
		return historyNavigationText("Host 不支持此原生操作", "Native action unavailable on this Host"), true, 1, 0
	}
	if state.Unavailable {
		return historyNavigationText("文件已不可用", "File no longer available"), true, 1, 0
	}
	if state.Busy {
		if state.Action == "reveal" {
			if directory {
				return historyNavigationText("正在打开目录…", "Opening directory…"), false, 1, 0
			}
			return historyNavigationText("正在显示文件…", "Revealing file…"), false, 1, 0
		}
		return historyNavigationText("正在打开…", "Opening…"), false, 1, 0
	}
	if state.Error != "" {
		if state.Action == "reveal" {
			if directory {
				return historyNavigationText("无法打开目录，点击重试", "Could not open directory; try again"), true, 1, 0
			}
			return historyNavigationText("无法显示文件，点击重试", "Could not reveal file; try again"), true, 1, 0
		}
		return historyNavigationText("无法打开文件，点击重试", "Could not open file; try again"), true, 1, 0
	}
	if state.Ready && !state.SettledAt.IsZero() {
		age := now.Sub(state.SettledAt)
		if age < presentedSuccessHold+presentedSuccessFade {
			text = historyNavigationText("已打开", "Opened")
			if state.Action == "reveal" {
				text = historyNavigationText("已显示文件", "File revealed")
				if directory {
					text = historyNavigationText("已打开目录", "Directory opened")
				}
			}
			if age < presentedSuccessHold {
				return text, false, 1, presentedSuccessHold - age
			}
			opacity = float32(1 - float64(age-presentedSuccessHold)/float64(presentedSuccessFade))
			return text, false, opacity, 16 * time.Millisecond
		}
	}
	return fallback, false, 1, 0
}

func presentedStatusSeat(c *ui.Context, text string, ink ui.Color, opacity float32, repaint time.Duration) {
	seat := ui.Text(c, text).FontSize(10).FixedLineHeight(16).TextColor(ink).SingleLine().Opacity(opacity).PassThrough()
	if repaint > 0 {
		seat.DrawOver(func(p *ui.Painter, _ ui.Rect) { p.After(repaint) })
	}
}
