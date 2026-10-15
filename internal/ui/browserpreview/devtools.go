package browserpreview

import (
	"context"
	"errors"
)

func (b *Controller) openBrowserDevTools(t *Tab) {
	if t == nil || b.current() != t || t.Host == nil || b.parent == nil || b.parent.IsDestroyed() {
		return
	}
	h, epoch := t.Host, t.Host.BrowserPageVersion()
	if !b.browserToolsCurrent(t, h, epoch) {
		return
	}
	err := h.OpenBrowserDevTools(epoch) // Synchronous UI-thread dispatch, no queued old-tab callback.
	if !b.browserToolsCurrent(t, h, epoch) {
		return
	}
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			t.Error = err.Error()
		}
	} else {
		t.Error = ""
		t.ActionMessage = "已请求打开当前页开发者工具；只运行自己理解的控制台代码，关闭请使用工具窗口的关闭按钮"
	}
	b.update()
}
